package handlers

import (
	"muambr-api/htmlparser"
	"muambr-api/models"
	"muambr-api/utils"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// LinkPreviewHandler handles link preview related endpoints
type LinkPreviewHandler struct {
	extractorHandler *ExtractorHandler
}

// NewLinkPreviewHandler creates a new LinkPreviewHandler
func NewLinkPreviewHandler() *LinkPreviewHandler {
	return &LinkPreviewHandler{
		extractorHandler: NewExtractorHandler(),
	}
}

// LinkPreviewRequest represents the request for link preview
type LinkPreviewRequest struct {
	URL              string `form:"url" binding:"required"`
	BaseCountry      string `form:"baseCountry"`
	AddComparisons   bool   `form:"addComparisons"`
}

// LinkPreviewResponse represents the response with parsed data
type LinkPreviewResponse struct {
	ProductData *htmlparser.ParsedProductData `json:"productData"`
}

// GetLinkPreview handles GET /api/v1/linkpreview?url=...&baseCountry=PT&addComparisons=true
//
// Query Parameters:
// - url (required): URL to parse
// - baseCountry (optional): User's base country ISO code (PT, US, ES, DE, GB, BR)
// - addComparisons (optional): Whether to add product comparisons (default: false) - NOT YET IMPLEMENTED
func (h *LinkPreviewHandler) GetLinkPreview(c *gin.Context) {
	var req LinkPreviewRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.LogError("❌ Error binding query parameters", utils.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request parameters",
			"details": err.Error(),
		})
		return
	}

	utils.Info("📥 Received link preview request",
		utils.String("url", req.URL),
		utils.String("baseCountry", req.BaseCountry),
		utils.Bool("addComparisons", req.AddComparisons))

	// Validate URL
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		utils.LogError("❌ Invalid URL", utils.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid URL",
			"details": err.Error(),
		})
		return
	}

	// Parse the URL and get product data
	productData, err := htmlparser.ParseURL(req.URL)
	if err != nil {
		utils.LogError("❌ Error parsing URL", utils.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to parse URL",
			"details": err.Error(),
		})
		return
	}

	utils.Info("✅ Successfully parsed product data",
		utils.String("title", productData.Title),
		utils.String("currency", productData.Currency))

	// Prepare response
	response := LinkPreviewResponse{
		ProductData: productData,
	}

	// TODO: Add comparison feature integration when needed
	if req.AddComparisons {
		utils.Info("ℹ️ Comparison feature requested but not yet implemented for link preview")
		// This would integrate with the comparison handler in the future
		_ = parsedURL // Using this to avoid unused variable warning
	}

	c.JSON(http.StatusOK, response)
}

// guessCountryFromURL tries to determine the country from the URL
func guessCountryFromURL(pageURL *url.URL) models.Country {
	host := pageURL.Host

	if containsAny(host, []string{".br", "brazil"}) {
		return models.CountryBrazil
	} else if containsAny(host, []string{".pt", "portugal"}) {
		return models.CountryPortugal
	} else if containsAny(host, []string{".es", "spain", "espana"}) {
		return models.CountrySpain
	} else if containsAny(host, []string{".uk", ".gb", "britain"}) {
		return models.CountryUK
	} else if containsString(host, ".de") {
		return models.CountryGermany
	}

	// Default to US
	return models.CountryUS
}

// containsAny checks if a string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if containsString(s, substr) {
			return true
		}
	}
	return false
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
