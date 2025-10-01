package main

import (
	"log"
	"muambr-api/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create Gin router with default middleware (logger and recovery)
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Product Comparison API is running",
		})
	})

	// Setup API routes
	routes.SetupRoutes(r)

	// Start server on port 8080
	log.Println("Starting Product Comparison API server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
