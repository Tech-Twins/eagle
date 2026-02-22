package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/eaglebank/shared/events"
	"github.com/eaglebank/shared/middleware"
	redisClient "github.com/eaglebank/shared/redis"
	txcmd "github.com/eaglebank/transaction-service/internal/command"
	"github.com/eaglebank/transaction-service/internal/handler"
	txqry "github.com/eaglebank/transaction-service/internal/query"
	"github.com/eaglebank/transaction-service/internal/repository"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/eagle_transactions?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis connection
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redis, err := redisClient.NewClient(redisAddr, "", 0)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Initialize event publisher
	publisher := events.NewPublisher(redis.Client)

	// CQRS: write repo, read repo, account read cache
	writeRepo := repository.NewTransactionWriteRepository(db)
	readRepo := repository.NewTransactionReadRepository(db, redis.Client)
	accountRepo := repository.NewAccountRepository(db, redis.Client)

	// Command + Query services
	commandSvc := txcmd.NewTransactionCommandService(writeRepo, readRepo, accountRepo, publisher)
	querySvc := txqry.NewTransactionQueryService(readRepo, accountRepo)

	transactionHandler := handler.NewTransactionHandler(commandSvc, querySvc)

	// Setup router
	router := gin.Default()
	router.Use(middleware.LoggingMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Transaction routes
	v1 := router.Group("/v1/accounts/:accountNumber/transactions", middleware.AuthMiddleware())
	{
		v1.POST("", transactionHandler.CreateTransaction)
		v1.GET("", transactionHandler.ListTransactions)
		v1.GET("/:transactionId", transactionHandler.GetTransaction)
	}
	port := getEnv("PORT", "8084")
	log.Printf("Transaction service starting on port %s", port)
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
