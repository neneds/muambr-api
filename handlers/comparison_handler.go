package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"muambr-api/models"
	"muambr-api/localization"
	"muambr-api/utils"
	"github.com/gin-gonic/gin"
)

// ComparisonHandler handles product comparison related endpoints
type ComparisonHandler struct {
	extractorHandler     *ExtractorHandler
	comparisonProcessor  *utils.ComparisonProcessor
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
		extractorHandler:    NewExtractorHandler(),
		comparisonProcessor: utils.NewComparisonProcessor(),
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
	// Parse and validate request parameters (using English for validation messages)
	params, validationErr := h.parseAndValidateRequest(c)
	if validationErr != nil {
		h.sendErrorResponse(c, validationErr)
		return
	}

	// Create request-scoped localizer based on baseCountry
	language := params.BaseCountry.GetLanguageCode()
	localizer, err := localization.NewLocalizedContext(language)
	if err != nil {
		// Fallback to English if language loading fails
		localizer, _ = localization.NewLocalizedContext("en")
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
		h.sendInternalErrorResponseWithLocalizer(c, localizer, "api.errors.failed_get_comparisons")
		return
	}

	// Handle empty results
	if len(comparisons) == 0 {
		h.sendEmptyResultsResponseWithLocalizer(c, localizer)
		return
	}

	// Process comparisons using the dedicated processor
	sections := h.comparisonProcessor.ProcessComparisons(comparisons, params.Limit)

	// Return successful response
	h.sendSuccessResponseWithSections(c, sections)
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
			Message:    localization.T("api.errors.product_name_required"),
		}
	}

	if baseCountryParam == "" {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    localization.T("api.errors.base_country_required"),
		}
	}

	// Parse and validate ISO country code for user's base country
	baseCountry, err := models.ParseCountryFromISO(strings.ToUpper(baseCountryParam))
	if err != nil {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    localization.T("api.errors.invalid_country_code"),
		}
	}

	// Detect and validate current country (where user is currently located)
	currentCountry, err := h.extractorHandler.DetectCountryCode(strings.ToUpper(currentUserCountryParam))
	if err != nil {
		return nil, &ComparisonError{
			StatusCode: http.StatusBadRequest,
			Message:    localization.T("api.errors.invalid_country_code"),
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



// sendErrorResponse sends a standardized error response
func (h *ComparisonHandler) sendErrorResponse(c *gin.Context, compErr *ComparisonError) {
	c.JSON(compErr.StatusCode, models.ProductComparisonResponse{
		Success:      false,
		Message:      &compErr.Message,
		Sections:     []models.CountrySection{},
		TotalResults: 0,
	})
}

// sendInternalErrorResponse sends a 500 internal server error response (deprecated)
func (h *ComparisonHandler) sendInternalErrorResponse(c *gin.Context, messageKey string) {
	message := localization.T(messageKey)
	c.JSON(http.StatusInternalServerError, models.ProductComparisonResponse{
		Success:      false,
		Message:      &message,
		Sections:     []models.CountrySection{},
		TotalResults: 0,
	})
}

// sendInternalErrorResponseWithLocalizer sends a 500 internal server error response using request-scoped localizer
func (h *ComparisonHandler) sendInternalErrorResponseWithLocalizer(c *gin.Context, localizer *localization.RequestLocalizer, messageKey string) {
	message := localization.TR(localizer, messageKey)
	c.JSON(http.StatusInternalServerError, models.ProductComparisonResponse{
		Success:      false,
		Message:      &message,
		Sections:     []models.CountrySection{},
		TotalResults: 0,
	})
}

// sendEmptyResultsResponse sends a successful response with no results found (deprecated)
func (h *ComparisonHandler) sendEmptyResultsResponse(c *gin.Context) {
	message := localization.T("api.success.no_comparisons_found")
	c.JSON(http.StatusOK, models.ProductComparisonResponse{
		Success:      true,
		Message:      &message,
		Sections:     []models.CountrySection{},
		TotalResults: 0,
	})
}

// sendEmptyResultsResponseWithLocalizer sends a successful response with no results found using request-scoped localizer
func (h *ComparisonHandler) sendEmptyResultsResponseWithLocalizer(c *gin.Context, localizer *localization.RequestLocalizer) {
	message := localization.TR(localizer, "api.success.no_comparisons_found")
	c.JSON(http.StatusOK, models.ProductComparisonResponse{
		Success:      true,
		Message:      &message,
		Sections:     []models.CountrySection{},
		TotalResults: 0,
	})
}



// sendSuccessResponseWithSections sends a successful response with processed country sections
func (h *ComparisonHandler) sendSuccessResponseWithSections(c *gin.Context, sections []models.CountrySection) {
	// Calculate total results across all sections
	totalResults := 0
	for _, section := range sections {
		totalResults += section.ResultsCount
	}
	
	c.JSON(http.StatusOK, models.ProductComparisonResponse{
		Success:      true,
		Message:      nil,
		Sections:     sections,
		TotalResults: totalResults,
	})
}

// sendSuccessResponse sends a successful response with comparison results (legacy method for backward compatibility)
func (h *ComparisonHandler) sendSuccessResponse(c *gin.Context, comparisons []models.ProductComparison) {
	sections := h.comparisonProcessor.ProcessComparisons(comparisons, 10) // Default limit for legacy calls
	h.sendSuccessResponseWithSections(c, sections)
}