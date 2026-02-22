package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/eaglebank/shared/middleware"
	"github.com/gin-gonic/gin"
)

var (
	authServiceURL        = getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	userServiceURL        = getEnv("USER_SERVICE_URL", "http://localhost:8082")
	accountServiceURL     = getEnv("ACCOUNT_SERVICE_URL", "http://localhost:8083")
	transactionServiceURL = getEnv("TRANSACTION_SERVICE_URL", "http://localhost:8084")
)

func main() {
	router := gin.Default()
	router.Use(middleware.LoggingMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "api-gateway"})
	})

	// Auth routes (no authentication required)
	router.POST("/v1/auth/login", proxyTo(authServiceURL))
	router.POST("/v1/auth/refresh", proxyTo(authServiceURL))

	// User routes
	router.POST("/v1/users", proxyTo(userServiceURL))                                         // No auth for registration
	router.GET("/v1/users/:userId", middleware.AuthMiddleware(), proxyTo(userServiceURL))
	router.PATCH("/v1/users/:userId", middleware.AuthMiddleware(), proxyTo(userServiceURL))
	router.DELETE("/v1/users/:userId", middleware.AuthMiddleware(), proxyTo(userServiceURL))

	// Account routes
	router.POST("/v1/accounts", middleware.AuthMiddleware(), proxyTo(accountServiceURL))
	router.GET("/v1/accounts", middleware.AuthMiddleware(), proxyTo(accountServiceURL))
	router.GET("/v1/accounts/:accountNumber", middleware.AuthMiddleware(), proxyTo(accountServiceURL))
	router.PATCH("/v1/accounts/:accountNumber", middleware.AuthMiddleware(), proxyTo(accountServiceURL))
	router.DELETE("/v1/accounts/:accountNumber", middleware.AuthMiddleware(), proxyTo(accountServiceURL))

	// Transaction routes
	router.POST("/v1/accounts/:accountNumber/transactions", middleware.AuthMiddleware(), proxyTo(transactionServiceURL))
	router.GET("/v1/accounts/:accountNumber/transactions", middleware.AuthMiddleware(), proxyTo(transactionServiceURL))
	router.GET("/v1/accounts/:accountNumber/transactions/:transactionId", middleware.AuthMiddleware(), proxyTo(transactionServiceURL))

	port := getEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func proxyTo(serviceURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build target URL
		targetURL := serviceURL + c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		// Read request body
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Create new request
		req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create request"})
			return
		}

		// Copy headers
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Forward user context from JWT middleware if authenticated
		if userID, exists := c.Get("userId"); exists {
			req.Header.Set("X-User-ID", userID.(string))
		}
		if email, exists := c.Get("email"); exists {
			req.Header.Set("X-User-Email", email.(string))
		}

		// Make request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error proxying request: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"message": "Service unavailable"})
			return
		}
		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to read response"})
			return
		}

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Forward response
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		// Remove trailing slash if present
		return strings.TrimSuffix(value, "/")
	}
	return fallback
}
