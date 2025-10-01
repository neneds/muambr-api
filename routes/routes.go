package routes

import (
	"muambr-api/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the API routes
func SetupRoutes(r *gin.Engine) {
	// Initialize handlers
	comparisonHandler := handlers.NewComparisonHandler()

	// API group
	api := r.Group("/api")
	{
		// GET /api/comparisons?name=productName&country=PT&currentCountry=US
		api.GET("/comparisons", comparisonHandler.GetComparisons)
	}
}
