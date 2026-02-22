package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/eaglebank/auth-service/internal/handler"
	authqry "github.com/eaglebank/auth-service/internal/query"
	"github.com/eaglebank/auth-service/internal/repository"
	"github.com/eaglebank/shared/middleware"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/eagle_users?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// CQRS: auth is read-only; no CommandService needed
	userRepo := repository.NewUserRepository(db)
	querySvc := authqry.NewAuthQueryService(userRepo)
	authHandler := handler.NewAuthHandler(querySvc)

	// Setup router
	router := gin.Default()
	router.Use(middleware.LoggingMiddleware())

	// Auth routes
	v1 := router.Group("/v1/auth")
	{
		v1.POST("/login", authHandler.Login)
		v1.POST("/refresh", authHandler.RefreshToken)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := getEnv("PORT", "8081")
	log.Printf("Auth service starting on port %s", port)
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
