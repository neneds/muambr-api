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

// GetComparisons handles GET /api/v1/comparisons/search?name=productName&baseCountry=PT&currency=EUR&currentUserCountry=US&limit=10
//
// Query Parameters (matching Swift client expectations):
// - name (required): Product name to search for
// - baseCountry (required): User's base/home country ISO code (PT, US, ES, DE, GB, BR)
// - currentUserCountry (optional): User's current location ISO code - if different from base country
// - currency (optional): Target currency for price conversion - defaults to base country's currency
// - limit (optional): Maximum number of results to return - defaults to 10
//
// Extractor Selection Rules:
// 1. Always use extractors from the base country (baseCountry parameter)
// 2. If currentUserCountry is provided and different from base country, append extractors from current country
// 3. This allows users to see products from both their home country and current location
//
// Currency Conversion:
// - Products with different currencies than the target currency will include a convertedPrice field
// - Store names include availability context: "Store (Available for BaseCountry) - Browsing from CurrentCountry"
func (h *ComparisonHandler) GetComparisons(c *gin.Context) {
	// Parse query parameters (matching Swift API expectations)
	productName := c.Query("name")
	baseCountryParam := c.Query("baseCountry")
	currentUserCountryParam := c.Query("currentUserCountry")
	currency := c.Query("currency")
	
	// Parse limit parameter with default value of 10
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Validate required parameters
	if productName == "" {
		errorMsg := "Product name is required"
		c.JSON(http.StatusBadRequest, models.ProductComparisonResponse{
			Success:      false,
			Message:      &errorMsg,
			Comparisons:  []models.ProductComparison{},
			TotalResults: 0,
		})
		return
	}

	if baseCountryParam == "" {
		errorMsg := "Base country ISO code is required (e.g., PT, US, ES, DE, GB, BR)"
		c.JSON(http.StatusBadRequest, models.ProductComparisonResponse{
			Success:      false,
			Message:      &errorMsg,
			Comparisons:  []models.ProductComparison{},
			TotalResults: 0,
		})
		return
	}

	// Parse and validate ISO country code for user's base country
	baseCountry, err := models.ParseCountryFromISO(strings.ToUpper(baseCountryParam))
	if err != nil {
		errorMsg := "Invalid country ISO code. Supported codes: PT, US, ES, DE, GB, BR"
		c.JSON(http.StatusBadRequest, models.ProductComparisonResponse{
			Success:      false,
			Message:      &errorMsg,
			Comparisons:  []models.ProductComparison{},
			TotalResults: 0,
		})
		return
	}

	// Detect and validate current country (where user is currently located) using ExtractorHandler
	currentCountry, err := h.extractorHandler.DetectCountryCode(strings.ToUpper(currentUserCountryParam))
	if err != nil {
		errorMsg := err.Error()
		c.JSON(http.StatusBadRequest, models.ProductComparisonResponse{
			Success:      false,
			Message:      &errorMsg,
			Comparisons:  []models.ProductComparison{},
			TotalResults: 0,
		})
		return
	}
	
	// Use base country's default currency if not provided
	if currency == "" {
		currency = baseCountry.GetCurrencyCode()
	}

	// Get product comparisons using the ExtractorHandler
	comparisons, err := h.extractorHandler.GetProductComparisons(productName, baseCountry, &currentCountry, currency)
	if err != nil {
		errorMsg := "Failed to get product comparisons"
		c.JSON(http.StatusInternalServerError, models.ProductComparisonResponse{
			Success:      false,
			Message:      &errorMsg,
			Comparisons:  []models.ProductComparison{},
			TotalResults: 0,
		})
		return
	}

	// If no extractors available yet, return empty successful response
	if len(comparisons) == 0 {
		successMsg := "No comparisons found for this product"
		c.JSON(http.StatusOK, models.ProductComparisonResponse{
			Success:      true,
			Message:      &successMsg,
			Comparisons:  []models.ProductComparison{},
			TotalResults: 0,
		})
		return
	}

	// Sort products by price (smallest first)
	// Use converted price if available, otherwise use original price
	sort.Slice(comparisons, func(i, j int) bool {
		priceI := comparisons[i].Price
		priceJ := comparisons[j].Price
		
		// Use converted price if available for comparison i
		if comparisons[i].ConvertedPrice != nil {
			if convertedI, err := strconv.ParseFloat(comparisons[i].ConvertedPrice.Price, 64); err == nil {
				priceI = convertedI
			}
		}
		
		// Use converted price if available for comparison j
		if comparisons[j].ConvertedPrice != nil {
			if convertedJ, err := strconv.ParseFloat(comparisons[j].ConvertedPrice.Price, 64); err == nil {
				priceJ = convertedJ
			}
		}
		
		return priceI < priceJ
	})

	// Apply limit
	if limit > 0 && len(comparisons) > limit {
		comparisons = comparisons[:limit]
	}

	// Return response in expected Swift API format
	c.JSON(http.StatusOK, models.ProductComparisonResponse{
		Success:      true,
		Message:      nil,
		Comparisons:  comparisons,
		TotalResults: len(comparisons),
	})
}