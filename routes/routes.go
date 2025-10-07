package routes

import (
	"muambr-api/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the API routes
func SetupRoutes(r *gin.Engine) {
	// Initialize handlers
	comparisonHandler := handlers.NewComparisonHandler()
	adminHandler := handlers.NewAdminHandler()

	// API group
	api := r.Group("/api")
	{
		// GET /api/comparisons?name=productName&country=PT&currentCountry=US
		api.GET("/comparisons", comparisonHandler.GetComparisons)
	}

	// Admin group for utility endpoints
	admin := r.Group("/admin")
	{
		// GET /admin/exchange-rates/status - Check cache status
		admin.GET("/exchange-rates/status", adminHandler.GetExchangeRateStatus)
		
		// DELETE /admin/exchange-rates/cache - Clear cache
		admin.DELETE("/exchange-rates/cache", adminHandler.ClearExchangeRateCache)
		
		// GET /admin/exchange-rates/test?currency=USD - Test API
		admin.GET("/exchange-rates/test", adminHandler.TestExchangeRateAPI)
	}
}
