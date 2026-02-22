package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	accountcmd "github.com/eaglebank/account-service/internal/command"
	"github.com/eaglebank/account-service/internal/handler"
	accountqry "github.com/eaglebank/account-service/internal/query"
	"github.com/eaglebank/account-service/internal/repository"
	"github.com/eaglebank/shared/events"
	"github.com/eaglebank/shared/middleware"
	redisClient "github.com/eaglebank/shared/redis"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection (write store)
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/eagle_accounts?sslmode=disable")
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

	writeRepo := repository.NewAccountWriteRepository(db)
	readRepo := repository.NewAccountReadRepository(db, redis.Client)

	commandSvc := accountcmd.NewAccountCommandService(writeRepo, readRepo, publisher)
	querySvc := accountqry.NewAccountQueryService(readRepo)

	accountHandler := handler.NewAccountHandler(commandSvc, querySvc)

	// Setup router
	router := gin.Default()
	router.Use(middleware.LoggingMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := router.Group("/v1/accounts", middleware.AuthMiddleware())
	{
		v1.POST("", accountHandler.CreateAccount)
		v1.GET("", accountHandler.ListAccounts)
		v1.GET("/:accountNumber", accountHandler.GetAccount)
		v1.PATCH("/:accountNumber", accountHandler.UpdateAccount)
		v1.DELETE("/:accountNumber", accountHandler.DeleteAccount)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		subscriber := events.NewSubscriber(redis.Client, events.SubscriberConfig{
			Group:    "account-service-group",
			Consumer: "account-consumer-1",
			Stream:   events.TransactionEventsStream,
			Handler:  commandSvc.HandleTransactionEvent,
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

	port := getEnv("PORT", "8083")
	log.Printf("Account service starting on port %s", port)
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
