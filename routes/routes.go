package routes

import (
	"muambr-api/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the API routes
func SetupRoutes(r *gin.Engine) {
	productComparisonHandler := handlers.NewProductComparisonHandler()
	exchangeRateHandler := handlers.NewExchangeRateHandler()
	linkPreviewHandler := handlers.NewLinkPreviewHandler()

	v1 := r.Group("/api/v1")
	{
		// POST /api/v1/product-comparisons — primary comparison contract
		v1.POST("/product-comparisons", productComparisonHandler.CreateProductComparison)

		// GET /api/v1/linkpreview?url=...&baseCountry=PT&addComparisons=true
		v1.GET("/linkpreview", linkPreviewHandler.GetLinkPreview)
	}

	rates := r.Group("/rates")
	{
		// GET /rates/exchange-rates?baseCurrency=USD
		rates.GET("/exchange-rates", exchangeRateHandler.GetExchangeRates)
	}
}
