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

// GetComparisons handles GET /api/comparisons?name=productName&country=PT&currency=EUR&currentCountry=US
func (h *ComparisonHandler) GetComparisons(c *gin.Context) {
	// Parse query parameters
	productName := c.Query("name")
	countryParam := c.Query("country")
	currentCountryParam := c.Query("currentCountry")
	baseCurrency := c.Query("currency")

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
	country, valid := models.ParseCountryFromISO(strings.ToUpper(countryParam))
	if !valid {
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
	comparisons, err := h.extractorHandler.GetProductComparisons(productName, country, currentCountry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product comparisons"})
		return
	}

	// If no extractors available yet, return mock data
	if len(comparisons) == 0 {
		// TODO: Remove this mock data once actual extractors are implemented
		var mockStoreSuffix string
		if currentCountry != "" {
			mockStoreSuffix = " (Available in " + currentCountry.GetCountryName() + ")"
		}

		comparisons = []models.ProductComparison{
			{
				Name:     "Sony WH-1000XM6 - Auriculares Bluetooth con cancelación activa de ruido - Negro nuevo",
				Price:    "342.87",
				Store:    "Store audio" + mockStoreSuffix,
				Currency: baseCurrency,
				URL:      "https://www.idealo.es/relocator/relocate?categoryId=2520&offerKey=3755ad8b44312650d3c97c23bb8c93b1&offerListId=206509478-27E3E95F56F555BCADFDD6FB2FBB0E79&pos=3&price=342.87&productid=206509477&sid=335485&type=offer",
			},
			{
				Name:     productName + " - Premium variant",
				Price:    "289.99",
				Store:    "Electronics Store Plus" + mockStoreSuffix,
				Currency: baseCurrency,
				URL:      "https://example.com/product2",
			},
		}
	}

	response := models.ComparisonResponse(comparisons)
	c.JSON(http.StatusOK, response)
}