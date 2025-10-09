package main

import (
	"os"
	"muambr-api/routes"
	"muambr-api/utils"
	"muambr-api/localization"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize logger
	if err := utils.InitDevelopmentLogger(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// Initialize localization with English as default
	if err := localization.InitLocalizer("en"); err != nil {
		utils.Warn("Failed to initialize localizer, using fallback strings", utils.Error(err))
	}

	// Load .env file if it exists (for local development)
	if err := godotenv.Load(); err != nil {
		utils.Info("No .env file found or error loading .env file (this is OK in production)")
	}

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
			"status":  localization.T("api.health.status_ok"),
			"message": localization.T("api.health.message"),
		})
	})

	// Setup API routes
	routes.SetupRoutes(r)

	// Get port from environment variable (required for Render.com)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default for local development
	}

	// Start server
	utils.Info("Starting Product Comparison API server", utils.String("port", port))
	if err := r.Run(":" + port); err != nil {
		utils.Fatal("Failed to start server", utils.Error(err))
	}
}
