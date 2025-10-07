package handlers

import (
	"net/http"
	"sort"
	"strconv"
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

// GetComparisons handles GET /api/comparisons?name=productName&country=PT&currency=EUR&currentCountry=US&limit=10
//
// Query Parameters:
// - name (required): Product name to search for
// - country (required): User's base/home country ISO code (PT, US, ES, DE, GB, BR)
// - currentCountry (optional): User's current location ISO code - if different from base country
// - currency (optional): Target currency for price conversion - defaults to base country's currency
// - limit (optional): Maximum number of results to return - defaults to 10
//
// Extractor Selection Rules:
// 1. Always use extractors from the base country (country parameter)
// 2. If currentCountry is provided and different from base country, append extractors from current country
// 3. This allows users to see products from both their home country and current location
//
// Currency Conversion:
// - Products with different currencies than the target currency will include a convertedPrice field
// - Store names include availability context: "Store (Available for BaseCountry) - Browsing from CurrentCountry"
func (h *ComparisonHandler) GetComparisons(c *gin.Context) {
	// Parse query parameters
	productName := c.Query("name")
	countryParam := c.Query("country")
	currentCountryParam := c.Query("currentCountry")
	baseCurrency := c.Query("currency")
	
	// Parse limit parameter with default value of 10
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Validate required parameters
	if productName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product name is required"})
		return
	}

	if countryParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Country ISO code is required (e.g., PT, US, ES, DE, GB, BR)"})
		return
	}

	// Parse and validate ISO country code for user's base country
	baseCountry, err := models.ParseCountryFromISO(strings.ToUpper(countryParam))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid country ISO code. Supported codes: PT, US, ES, DE, GB, BR"})
		return
	}

	// Detect and validate current country (where user is currently located) using ExtractorHandler
	currentCountry, err := h.extractorHandler.DetectCountryCode(strings.ToUpper(currentCountryParam))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Use base country's default currency if not provided
	if baseCurrency == "" {
		baseCurrency = baseCountry.GetCurrencyCode()
	}

	// Get product comparisons using the ExtractorHandler
	comparisons, err := h.extractorHandler.GetProductComparisons(productName, baseCountry, &currentCountry, baseCurrency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product comparisons"})
		return
	}

	// If no extractors available yet, return empty response
	if len(comparisons) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []models.ProductComparison{}})
		return
	}

	// Sort products by price (smallest first)
	sort.Slice(comparisons, func(i, j int) bool {
		priceI, errI := strconv.ParseFloat(comparisons[i].Price, 64)
		priceJ, errJ := strconv.ParseFloat(comparisons[j].Price, 64)
		
		// If parsing fails, put those items at the end
		if errI != nil && errJ != nil {
			return false
		}
		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}
		
		return priceI < priceJ
	})

	// Apply limit
	if limit > 0 && len(comparisons) > limit {
		comparisons = comparisons[:limit]
	}

	response := models.ComparisonResponse(comparisons)
	c.JSON(http.StatusOK, response)
}