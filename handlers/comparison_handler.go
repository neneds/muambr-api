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

// GetComparisons handles GET /api/comparisons?name=productName&country=PT&currency=EUR&currentCountry=US&macroregion=true&limit=10
func (h *ComparisonHandler) GetComparisons(c *gin.Context) {
	// Parse query parameters
	productName := c.Query("name")
	countryParam := c.Query("country")
	currentCountryParam := c.Query("currentCountry")
	baseCurrency := c.Query("currency")
	macroRegionParam := c.Query("macroregion") == "true"
	
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