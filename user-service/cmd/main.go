package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eaglebank/shared/events"
	"github.com/eaglebank/shared/middleware"
	redisClient "github.com/eaglebank/shared/redis"
	usercmd "github.com/eaglebank/user-service/internal/command"
	"github.com/eaglebank/user-service/internal/handler"
	userqry "github.com/eaglebank/user-service/internal/query"
	"github.com/eaglebank/user-service/internal/repository"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	middleware.MustInitJWTSecret()

	// Database connection (write store)
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/eagle_users?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis connection (read model store + event streaming)
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redis, err := redisClient.NewClient(redisAddr, "", 0)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// --- CQRS wiring ---
	publisher := events.NewPublisher(redis.Client)

	writeRepo := repository.NewUserWriteRepository(db)
	readRepo := repository.NewUserReadRepository(db, redis.Client)

	commandSvc := usercmd.NewUserCommandService(writeRepo, readRepo, publisher)
	querySvc := userqry.NewUserQueryService(readRepo)

	userHandler := handler.NewUserHandler(commandSvc, querySvc)

	// Setup router
	router := gin.Default()
	router.Use(middleware.LoggingMiddleware())

	v1 := router.Group("/v1/users")
	{
		v1.POST("", userHandler.CreateUser)
		v1.GET("/:userId", middleware.AuthMiddleware(), userHandler.GetUser)
		v1.PATCH("/:userId", middleware.AuthMiddleware(), userHandler.UpdateUser)
		v1.DELETE("/:userId", middleware.AuthMiddleware(), userHandler.DeleteUser)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start event subscriber — handled by the command service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		subscriber := events.NewSubscriber(redis.Client, events.SubscriberConfig{
			Group:    "user-service-group",
			Consumer: "user-consumer-1",
			Stream:   events.AccountEventsStream,
			Handler:  commandSvc.HandleAccountEvent,
		})
		if err := subscriber.Start(ctx); err != nil {
			log.Printf("Subscriber stopped: %v", err)
		}
	}()

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	port := getEnv("PORT", "8082")
	log.Printf("User service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
