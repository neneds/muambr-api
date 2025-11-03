package routes

import (
	"muambr-api/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the API routes
func SetupRoutes(r *gin.Engine) {
	// Initialize handlers
	comparisonHandler := handlers.NewComparisonHandler()
	exchangeRateHandler := handlers.NewExchangeRateHandler()
	linkPreviewHandler := handlers.NewLinkPreviewHandler()

	// API v1 group - matches Swift client expectations
	v1 := r.Group("/api/v1")
	{
		comparisons := v1.Group("/comparisons")
		{
			// GET /api/v1/comparisons/search?name=productName&baseCountry=PT&currentUserCountry=US
			comparisons.GET("/search", comparisonHandler.GetComparisons)
		}

		// GET /api/v1/linkpreview?url=...&baseCountry=PT&addComparisons=true
		v1.GET("/linkpreview", linkPreviewHandler.GetLinkPreview)
	}

	// Exchange rates endpoints	
	rates := r.Group("/rates")
	{
		// GET /rates/exchange-rates?baseCurrency=USD - Get exchange rates for base currency
		rates.GET("/exchange-rates", exchangeRateHandler.GetExchangeRates)
	}
}
