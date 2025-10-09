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

// ComparisonRequest represents the validated request parameters for product comparison
type ComparisonRequest struct {
	ProductName    string
	BaseCountry    models.Country
	CurrentCountry models.Country
	Currency       string
	Limit          int
	UseMacroRegion bool // When true, use macro region of currentCountry for extractor selection
}

// ComparisonError represents an error with HTTP status code and message
type ComparisonError struct {
	StatusCode int
	Message    string
}

// NewComparisonHandler creates a new ComparisonHandler
func NewComparisonHandler() *ComparisonHandler {
	return &ComparisonHandler{
		extractorHandler: NewExtractorHandler(),
	}
}

// GetComparisons handles GET /api/v1/comparisons/search?name=productName&baseCountry=PT&currency=EUR&currentUserCountry=US&limit=10&useMacroRegion=true
//
// Query Parameters (matching Swift client expectations):
// - name (required): Product name to search for
// - baseCountry (required): User's base/home country ISO code (PT, US, ES, DE, GB, BR)
// - currentUserCountry (optional): User's current location ISO code - if different from base country
// - currency (optional): Target currency for price conversion - defaults to base country's currency
// - limit (optional): Maximum number of results to return - defaults to 10
// - useMacroRegion (optional): When "true", use macro region of currentUserCountry for extractor selection - defaults to false
//
// Extractor Selection Rules:
// 1. Always use extractors from the base country (baseCountry parameter)
// 2. If currentUserCountry is provided and different from base country:
//    - If useMacroRegion=true: use extractors from the macro region of currentUserCountry
//    - If useMacroRegion=false (default): use extractors from currentUserCountry only
// 3. This allows users to see products from their home country and either specific location or broader regional availability
//
// Currency Conversion:
// - Products with different currencies than the target currency will include a convertedPrice field
// - Store names include availability context: "Store (Available for BaseCountry) - Browsing from CurrentCountry"
func (h *ComparisonHandler) GetComparisons(c *gin.Context) {
	// Parse and validate request parameters
	params, validationErr := h.parseAndValidateRequest(c)
	if validationErr != nil {
		h.sendErrorResponse(c, validationErr)
		return
	}

	// Get product comparisons from extractors
	comparisons, err := h.extractorHandler.GetProductComparisons(
		params.ProductName, 
		params.BaseCountry, 
		&params.CurrentCountry, 
		params.Currency,
		params.UseMacroRegion,
	)
	if err != nil {
		h.sendInternalErrorResponse(c, "Failed to get product comparisons")
		return
	}

	// Handle empty results
	if len(comparisons) == 0 {
		h.sendEmptyResultsResponse(c)
		return
	}

	// Apply per-country sorting and limiting
	processedComparisons := h.processComparisonsByCountry(comparisons, params.Limit)

	// Return successful response
	h.sendSuccessResponse(c, processedComparisons)
}

// parseAndValidateRequest extracts and validates all request parameters
func (h *ComparisonHandler) parseAndValidateRequest(c *gin.Context) (*ComparisonRequest, *ComparisonError) {
	// Parse query parameters
	productName := c.Query("name")
	baseCountryParam := c.Query("baseCountry")
	currentUserCountryParam := c.Query("currentUserCountry")
	currency := c.Query("currency")
	useMacroRegionParam := c.Query("useMacroRegion")
	
	// Parse limit parameter with default value of 10
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Parse useMacroRegion parameter with default value of false
	useMacroRegion := false
	if useMacroRegionParam != "" {
		useMacroRegion = strings.ToLower(useMacroRegionParam) == "true"
	}

	// Validate required parameters
	if productName == "" {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    "Product name is required",
		}
	}

	if baseCountryParam == "" {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    "Base country ISO code is required (e.g., PT, US, ES, DE, GB, BR)",
		}
	}

	// Parse and validate ISO country code for user's base country
	baseCountry, err := models.ParseCountryFromISO(strings.ToUpper(baseCountryParam))
	if err != nil {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    "Invalid country ISO code. Supported codes: PT, US, ES, DE, GB, BR",
		}
	}

	// Detect and validate current country (where user is currently located)
	currentCountry, err := h.extractorHandler.DetectCountryCode(strings.ToUpper(currentUserCountryParam))
	if err != nil {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		}
	}
	
	// Use base country's default currency if not provided
	if currency == "" {
		currency = baseCountry.GetCurrencyCode()
	}

	return &ComparisonRequest{
		ProductName:    productName,
		BaseCountry:    baseCountry,
		CurrentCountry: currentCountry,
		Currency:       currency,
		Limit:          limit,
		UseMacroRegion: useMacroRegion,
	}, nil
}

// processComparisonsByCountry groups results by country, sorts by price, and applies per-country limits
func (h *ComparisonHandler) processComparisonsByCountry(comparisons []models.ProductComparison, limit int) []models.ProductComparison {
	// Group comparisons by country using the Country field from ProductComparison model
	countryGroups := h.groupComparisonsByCountry(comparisons)
	
	// Process each country group: sort by price and apply limit
	var finalResults []models.ProductComparison
	for _, countryComparisons := range countryGroups {
		processedCountryResults := h.sortAndLimitCountryComparisons(countryComparisons, limit)
		finalResults = append(finalResults, processedCountryResults...)
	}
	
	return finalResults
}

// groupComparisonsByCountry groups product comparisons by country using the Country field
func (h *ComparisonHandler) groupComparisonsByCountry(comparisons []models.ProductComparison) map[string][]models.ProductComparison {
	countryGroups := make(map[string][]models.ProductComparison)
	
	for _, comparison := range comparisons {
		// Use the Country field directly from the ProductComparison model
		countryCode := comparison.Country
		if countryCode == "" {
			countryCode = "Unknown" // Fallback for empty country
		}
		countryGroups[countryCode] = append(countryGroups[countryCode], comparison)
	}
	
	return countryGroups
}

// sortAndLimitCountryComparisons sorts a country's comparisons by price (smallest first) and applies limit
func (h *ComparisonHandler) sortAndLimitCountryComparisons(comparisons []models.ProductComparison, limit int) []models.ProductComparison {
	// Sort by price (smallest first), using converted price if available
	sort.Slice(comparisons, func(i, j int) bool {
		priceI := h.getEffectivePrice(comparisons[i])
		priceJ := h.getEffectivePrice(comparisons[j])
		return priceI < priceJ
	})
	
	// Apply limit
	if limit > 0 && len(comparisons) > limit {
		return comparisons[:limit]
	}
	
	return comparisons
}

// getEffectivePrice returns the converted price if available, otherwise the original price
func (h *ComparisonHandler) getEffectivePrice(comparison models.ProductComparison) float64 {
	if comparison.ConvertedPrice != nil {
		return comparison.ConvertedPrice.Price
	}
	return comparison.Price
}

// sendErrorResponse sends a standardized error response
func (h *ComparisonHandler) sendErrorResponse(c *gin.Context, compErr *ComparisonError) {
	c.JSON(compErr.StatusCode, models.ProductComparisonResponse{
		Success:      false,
		Message:      &compErr.Message,
		Comparisons:  []models.ProductComparison{},
		TotalResults: 0,
	})
}

// sendInternalErrorResponse sends a 500 internal server error response
func (h *ComparisonHandler) sendInternalErrorResponse(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, models.ProductComparisonResponse{
		Success:      false,
		Message:      &message,
		Comparisons:  []models.ProductComparison{},
		TotalResults: 0,
	})
}

// sendEmptyResultsResponse sends a successful response with no results found
func (h *ComparisonHandler) sendEmptyResultsResponse(c *gin.Context) {
	message := "No comparisons found for this product"
	c.JSON(http.StatusOK, models.ProductComparisonResponse{
		Success:      true,
		Message:      &message,
		Comparisons:  []models.ProductComparison{},
		TotalResults: 0,
	})
}

// sendSuccessResponse sends a successful response with comparison results
func (h *ComparisonHandler) sendSuccessResponse(c *gin.Context, comparisons []models.ProductComparison) {
	c.JSON(http.StatusOK, models.ProductComparisonResponse{
		Success:      true,
		Message:      nil,
		Comparisons:  comparisons,
		TotalResults: len(comparisons),
	})
}