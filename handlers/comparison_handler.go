package handlers

import (
	"net/http"
	"strings"
	"muambr-api/models"
	"github.com/gin-gonic/gin"
)

// ComparisonHandler handles product comparison related endpoints
type ComparisonHandler struct {
	extractorHandler *ExtractorHandler
}

// NewComparisonHandler creates a new ComparisonHandler
func NewComparisonHandler() *ComparisonHandler {
	return &ComparisonHandler{
		extractorHandler: NewExtractorHandler(),
	}
}

// GetComparisons handles GET /api/comparisons?name=productName&country=PT&currency=EUR&currentCountry=US&macroregion=true
func (h *ComparisonHandler) GetComparisons(c *gin.Context) {
	// Parse query parameters
	productName := c.Query("name")
	countryParam := c.Query("country")
	currentCountryParam := c.Query("currentCountry")
	baseCurrency := c.Query("currency")
	macroRegionParam := c.Query("macroregion") == "true"

	// Validate required parameters
	if productName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product name is required"})
		return
	}

	if countryParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Country ISO code is required (e.g., PT, US, ES, DE, GB, BR)"})
		return
	}

	// Parse and validate ISO country code for comparison target
	country, err := models.ParseCountryFromISO(strings.ToUpper(countryParam))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid country ISO code. Supported codes: PT, US, ES, DE, GB, BR"})
		return
	}

	// Detect and validate current country using ExtractorHandler
	currentCountry, err := h.extractorHandler.DetectCountryCode(strings.ToUpper(currentCountryParam))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Use country's default currency if not provided
	if baseCurrency == "" {
		baseCurrency = country.GetCurrencyCode()
	}

	// Get product comparisons using the ExtractorHandler
	comparisons, err := h.extractorHandler.GetProductComparisons(productName, country, currentCountry, macroRegionParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product comparisons"})
		return
	}

	// If no extractors available yet, return empty response
	if len(comparisons) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []models.ProductComparison{}})
		return
	}

	response := models.ComparisonResponse(comparisons)
	c.JSON(http.StatusOK, response)
}