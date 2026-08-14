package handlers

import (
	"muambr-api/linkparsers"
	"muambr-api/localization"
	"muambr-api/models"
	"muambr-api/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LinkPreviewHandler handles link preview related endpoints
type LinkPreviewHandler struct {
	extractorHandler    *ExtractorHandler
	comparisonProcessor *utils.ComparisonProcessor
	comparisonEngine    *utils.ComparisonEngine
}

// NewLinkPreviewHandler creates a new LinkPreviewHandler
func NewLinkPreviewHandler() *LinkPreviewHandler {
	return &LinkPreviewHandler{
		extractorHandler:    NewExtractorHandler(),
		comparisonProcessor: utils.NewComparisonProcessor(),
		comparisonEngine:    utils.NewComparisonEngine(),
	}
}

// LinkPreviewRequest represents the request for link preview
type LinkPreviewRequest struct {
	URL            string `form:"url" binding:"required"`
	BaseCountry    string `form:"baseCountry"`
	AddComparisons bool   `form:"addComparisons"`
	Category       string `form:"category"`
	Limit          int    `form:"limit"`
}

// LinkPreviewResponse represents the response with parsed data
type LinkPreviewResponse struct {
	ProductData *linkparsers.ParsedProductData   `json:"productData"`
	Country     string                           `json:"country,omitempty"`
	Comparison  *models.ProductComparisonResult  `json:"comparison,omitempty"`
}

// GetLinkPreview handles GET /api/v1/linkpreview?url=...&baseCountry=PT&addComparisons=true
//
// Query Parameters:
// - url (required): URL to parse
// - baseCountry (optional): User's base country ISO code (PT, US, ES, DE, GB, BR)
// - addComparisons (optional): When true, run a full product comparison using extracted title/price
// - category (optional): electronics | beauty | appliances | fashion | other
// - limit (optional): max results per country (default 10)
func (h *LinkPreviewHandler) GetLinkPreview(c *gin.Context) {
	var req LinkPreviewRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.LogError("❌ Error binding query parameters", utils.Error(err))
		code := models.ErrorCodeInvalidRequest
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   localization.TAccept(c.GetHeader("Accept-Language"), "api.errors.invalid_request_parameters"),
			"code":    code,
			"details": err.Error(),
		})
		return
	}

	utils.Info("📥 Received link preview request",
		utils.String("url", req.URL),
		utils.String("baseCountry", req.BaseCountry),
		utils.Bool("addComparisons", req.AddComparisons))

	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		utils.LogError("❌ Invalid URL", utils.Error(err))
		code := models.ErrorCodeInvalidRequest
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   localization.TAccept(c.GetHeader("Accept-Language"), "api.errors.invalid_url"),
			"code":    code,
			"details": err.Error(),
		})
		return
	}

	productData, err := linkparsers.ParseURL(req.URL)
	if err != nil {
		utils.LogError("❌ Error parsing URL", utils.Error(err))
		code := models.ErrorCodeProductNotFound
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   localization.TAccept(c.GetHeader("Accept-Language"), "api.errors.failed_parse_url"),
			"code":    code,
			"details": err.Error(),
		})
		return
	}

	urlCountry := guessCountryFromURL(parsedURL)

	utils.Info("✅ Successfully parsed product data",
		utils.String("title", productData.Title),
		utils.String("currency", productData.Currency),
		utils.String("country", string(urlCountry)))

	response := LinkPreviewResponse{
		ProductData: productData,
		Country:     string(urlCountry),
	}

	if req.AddComparisons {
		comparison, err := h.buildComparisonFromPreview(req, productData, urlCountry)
		if err != nil {
			utils.Warn("Link preview comparison failed", utils.Error(err))
			code := models.ErrorCodeInternalError
			msg := localization.TAccept(c.GetHeader("Accept-Language"), "api.errors.internal_server_error")
			response.Comparison = &models.ProductComparisonResult{
				Success:      false,
				Code:         &code,
				Message:      &msg,
				ComparisonID: utils.GenerateUUID(),
				Status:       models.ComparisonStatusEmpty,
				Prices:       []models.PriceOffer{},
				Sections:     []models.CountrySection{},
				Metadata:     models.ComparisonMetadata{PriceType: "retail", EntryMethod: "url"},
				CapturedAt:   time.Now().UTC().Format(time.RFC3339),
				ExpiresAt:    time.Now().UTC().Format(time.RFC3339),
			}
		} else {
			localizeEmptyComparisonMessage(c, comparison)
			response.Comparison = comparison
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *LinkPreviewHandler) buildComparisonFromPreview(
	req LinkPreviewRequest,
	productData *linkparsers.ParsedProductData,
	urlCountry models.Country,
) (*models.ProductComparisonResult, error) {
	if productData.Title == "" {
		return nil, errString("PRODUCT_NOT_FOUND: no product title extracted from URL")
	}

	baseCountry := urlCountry
	if req.BaseCountry != "" {
		parsed, err := models.ParseCountryFromISO(strings.ToUpper(req.BaseCountry))
		if err != nil {
			return nil, errString("COUNTRY_UNKNOWN: invalid baseCountry")
		}
		baseCountry = parsed
	}

	currentCountry := urlCountry
	normalizedCurrency := baseCountry.GetCurrencyCode()

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var category *models.ProductCategory
	categoryConfidence := 0.5
	if req.Category != "" {
		parsed, err := models.ParseCategoryFromString(strings.ToLower(req.Category))
		if err != nil {
			return nil, errString("INVALID_REQUEST: invalid category")
		}
		category = &parsed
		categoryConfidence = 1.0
	} else {
		other := models.CategoryOther
		category = &other
	}

	outcome, err := h.extractorHandler.GetProductComparisonsWithMeta(
		productData.Title,
		baseCountry,
		&currentCountry,
		normalizedCurrency,
		false,
		category,
	)
	if err != nil {
		return nil, err
	}

	sections := h.comparisonProcessor.ProcessComparisons(outcome.Comparisons, limit)

	var observed *models.ObservedPriceInput
	if productData.Price != nil && *productData.Price > 0 {
		currency := strings.ToUpper(productData.Currency)
		if currency == "" {
			currency = currentCountry.GetCurrencyCode()
		}
		observed = &models.ObservedPriceInput{
			Amount:   *productData.Price,
			Currency: currency,
			Country:  string(currentCountry),
		}
	}

	var exchangeRate *models.ExchangeRateInfo
	if observed != nil {
		conversion, convErr := h.extractorHandler.GetExchangeRateService().ConvertCurrencyWithMeta(1.0, observed.Currency, normalizedCurrency)
		if convErr == nil {
			exchangeRate = &models.ExchangeRateInfo{
				Base:      observed.Currency,
				Target:    normalizedCurrency,
				Rate:      conversion.Rate,
				Source:    conversion.Source,
				Timestamp: conversion.Timestamp.UTC().Format(time.RFC3339),
			}
		}
	}

	result := h.comparisonEngine.BuildResult(utils.ComparisonEngineInput{
		ProductName:        productData.Title,
		Category:           category,
		CategoryConfidence: categoryConfidence,
		BaseCountry:        baseCountry,
		CurrentCountry:     currentCountry,
		NormalizedCurrency: normalizedCurrency,
		Observed:           observed,
		Sections:           sections,
		Comparisons:        outcome.Comparisons,
		Meta:               outcome.Meta,
		EntryMethod:        "url",
		ExchangeRate:       exchangeRate,
		Now:                time.Now().UTC(),
	})
	return &result, nil
}

type stringError string

func (e stringError) Error() string { return string(e) }

func errString(msg string) error { return stringError(msg) }

// guessCountryFromURL tries to determine the country from the URL
func guessCountryFromURL(pageURL *url.URL) models.Country {
	host := pageURL.Host

	if containsAny(host, []string{".br", "brazil", "mercadolivre.com.br"}) {
		return models.CountryBrazil
	} else if containsAny(host, []string{".pt", "portugal", "kuantokusta"}) {
		return models.CountryPortugal
	} else if containsAny(host, []string{".es", "spain", "espana"}) {
		return models.CountrySpain
	} else if containsAny(host, []string{".uk", ".gb", "britain", "co.uk"}) {
		return models.CountryUK
	} else if containsString(host, ".de") {
		return models.CountryGermany
	}

	return models.CountryUS
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if containsString(s, substr) {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
