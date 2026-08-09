package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"muambr-api/linkparsers"
	"muambr-api/localization"
	"muambr-api/models"
	"muambr-api/utils"

	"github.com/gin-gonic/gin"
)

// ProductComparisonHandler handles the product-aligned comparison endpoint.
type ProductComparisonHandler struct {
	extractorHandler    *ExtractorHandler
	comparisonProcessor *utils.ComparisonProcessor
	comparisonEngine    *utils.ComparisonEngine
}

// NewProductComparisonHandler creates a ProductComparisonHandler.
func NewProductComparisonHandler() *ProductComparisonHandler {
	return &ProductComparisonHandler{
		extractorHandler:    NewExtractorHandler(),
		comparisonProcessor: utils.NewComparisonProcessor(),
		comparisonEngine:    utils.NewComparisonEngine(),
	}
}

// CreateProductComparisonRequest is the POST /api/v1/product-comparisons body.
type CreateProductComparisonRequest struct {
	Product         models.ProductIdentityInput  `json:"product"`
	ObservedPrice   *models.ObservedPriceInput   `json:"observedPrice,omitempty"`
	CurrentCountry  string                       `json:"currentCountry"`
	BaseCountry     string                       `json:"baseCountry"`
	Currency        string                       `json:"currency,omitempty"`
	Limit           int                          `json:"limit,omitempty"`
	UseMacroRegion  bool                         `json:"useMacroRegion,omitempty"`
	Source          *models.ComparisonSourceInput `json:"source,omitempty"`
	ProductURL      string                       `json:"productURL,omitempty"`
}

// CreateProductComparison handles POST /api/v1/product-comparisons
//
// Primary comparison contract for camera / manual / URL / barcode flows.
func (h *ProductComparisonHandler) CreateProductComparison(c *gin.Context) {
	var req CreateProductComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeInvalidRequest, "Invalid request body")
		return
	}

	productName := strings.TrimSpace(req.Product.Name)
	entryMethod := "manual"
	if req.Source != nil && req.Source.Type != "" {
		entryMethod = req.Source.Type
	}

	// URL flow: extract product identity / observed price from the page when needed
	if req.ProductURL != "" || (req.Source != nil && req.Source.URL != "") {
		productURL := req.ProductURL
		if productURL == "" {
			productURL = req.Source.URL
		}
		parsed, err := linkparsers.ParseURL(productURL)
		if err != nil {
			utils.Warn("Failed to parse product URL for comparison", utils.Error(err), utils.String("url", productURL))
			if productName == "" {
				h.sendError(c, http.StatusBadRequest, models.ErrorCodeProductNotFound, "Could not extract product from URL")
				return
			}
		} else {
			if productName == "" && parsed.Title != "" {
				productName = parsed.Title
			}
			if req.ObservedPrice == nil && parsed.Price != nil && *parsed.Price > 0 {
				currency := strings.ToUpper(parsed.Currency)
				if currency == "" {
					currency = "USD"
				}
				req.ObservedPrice = &models.ObservedPriceInput{
					Amount:   *parsed.Price,
					Currency: currency,
				}
			}
			if entryMethod == "manual" {
				entryMethod = "url"
			}
			// Infer current country from URL host when not provided
			if req.CurrentCountry == "" {
				if u, err := url.Parse(productURL); err == nil {
					req.CurrentCountry = string(guessCountryFromURL(u))
				}
			}
		}
	}

	if productName == "" {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeInvalidRequest, localization.T("api.errors.product_name_required"))
		return
	}
	if req.BaseCountry == "" {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, localization.T("api.errors.base_country_required"))
		return
	}

	baseCountry, err := models.ParseCountryFromISO(strings.ToUpper(req.BaseCountry))
	if err != nil {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, localization.T("api.errors.invalid_country_code"))
		return
	}

	currentCountry := baseCountry
	if req.CurrentCountry != "" {
		detected, err := h.extractorHandler.DetectCountryCode(strings.ToUpper(req.CurrentCountry))
		if err != nil || detected == "" {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, localization.T("api.errors.invalid_country_code"))
			return
		}
		currentCountry = detected
	}

	normalizedCurrency := req.Currency
	if normalizedCurrency == "" {
		normalizedCurrency = baseCountry.GetCurrencyCode()
	}
	normalizedCurrency = strings.ToUpper(normalizedCurrency)

	if req.ObservedPrice != nil {
		if req.ObservedPrice.Amount <= 0 {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodePriceNotFound, "observedPrice.amount must be greater than 0")
			return
		}
		if strings.TrimSpace(req.ObservedPrice.Currency) == "" {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeCurrencyUnknown, "observedPrice.currency is required")
			return
		}
		req.ObservedPrice.Currency = strings.ToUpper(req.ObservedPrice.Currency)
		if req.ObservedPrice.Country == "" {
			req.ObservedPrice.Country = string(currentCountry)
		}
		if req.ObservedPrice.Store == "" && req.Source != nil {
			req.ObservedPrice.Store = req.Source.Store
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var category *models.ProductCategory
	categoryConfidence := 0.0
	if req.Product.Category != nil {
		parsed, err := models.ParseCategoryFromString(string(*req.Product.Category))
		if err != nil {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeInvalidRequest, "invalid category: must be one of electronics, beauty, appliances, fashion, other")
			return
		}
		category = &parsed
		categoryConfidence = 1.0
	} else {
		// Unknown category → generic (other) providers
		other := models.CategoryOther
		category = &other
		categoryConfidence = 0.5
	}

	currentCountryPtr := &currentCountry
	outcome, err := h.extractorHandler.GetProductComparisonsWithMeta(
		productName,
		baseCountry,
		currentCountryPtr,
		normalizedCurrency,
		req.UseMacroRegion,
		category,
	)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, models.ErrorCodeInternalError, localization.T("api.errors.failed_get_comparisons"))
		return
	}

	if outcome.Meta.ProvidersAttempted == 0 {
		h.sendError(c, http.StatusNotFound, models.ErrorCodeNoComparisonSources, "No comparison sources available for this country/category")
		return
	}

	sections := h.comparisonProcessor.ProcessComparisons(outcome.Comparisons, limit)

	exchangeRateInfo := h.buildExchangeRateInfo(req.ObservedPrice, normalizedCurrency)

	// If observed currency differs and conversion failed, surface a clear error
	if req.ObservedPrice != nil &&
		!strings.EqualFold(req.ObservedPrice.Currency, normalizedCurrency) &&
		exchangeRateInfo == nil {
		h.sendError(c, http.StatusServiceUnavailable, models.ErrorCodeExchangeRateUnavailable, "Exchange rate unavailable for currency conversion")
		return
	}

	result := h.comparisonEngine.BuildResult(utils.ComparisonEngineInput{
		ProductName:        productName,
		Brand:              req.Product.Brand,
		Model:              req.Product.Model,
		Category:           category,
		CategoryConfidence: categoryConfidence,
		BaseCountry:        baseCountry,
		CurrentCountry:     currentCountry,
		NormalizedCurrency: normalizedCurrency,
		Observed:           req.ObservedPrice,
		Sections:           sections,
		Comparisons:        outcome.Comparisons,
		Meta:               outcome.Meta,
		EntryMethod:        entryMethod,
		ExchangeRate:       exchangeRateInfo,
		Now:                time.Now().UTC(),
	})

	c.JSON(http.StatusOK, result)
}

func (h *ProductComparisonHandler) buildExchangeRateInfo(observed *models.ObservedPriceInput, targetCurrency string) *models.ExchangeRateInfo {
	if observed == nil {
		// Still expose a self-rate so clients can show "updated" metadata when available
		ts, ok := h.extractorHandler.GetExchangeRateService().GetRateTimestamp(targetCurrency)
		if !ok {
			// Warm the cache
			_, _ = h.extractorHandler.GetExchangeRateService().GetExchangeRates(targetCurrency)
			ts, ok = h.extractorHandler.GetExchangeRateService().GetRateTimestamp(targetCurrency)
		}
		if !ok {
			return nil
		}
		return &models.ExchangeRateInfo{
			Base:      targetCurrency,
			Target:    targetCurrency,
			Rate:      1.0,
			Source:    "exchangerate-api",
			Timestamp: ts.UTC().Format(time.RFC3339),
		}
	}

	from := strings.ToUpper(observed.Currency)
	to := strings.ToUpper(targetCurrency)
	conversion, err := h.extractorHandler.GetExchangeRateService().ConvertCurrencyWithMeta(1.0, from, to)
	if err != nil {
		utils.Warn("Failed to resolve exchange rate for comparison",
			utils.String("from", from),
			utils.String("to", to),
			utils.Error(err))
		return nil
	}

	return &models.ExchangeRateInfo{
		Base:      from,
		Target:    to,
		Rate:      conversion.Rate,
		Source:    conversion.Source,
		Timestamp: conversion.Timestamp.UTC().Format(time.RFC3339),
	}
}

func (h *ProductComparisonHandler) sendError(c *gin.Context, status int, code models.APIErrorCode, message string) {
	c.JSON(status, models.ProductComparisonResult{
		Success:      false,
		Code:         &code,
		Message:      &message,
		ComparisonID: utils.GenerateUUID(),
		Status:       models.ComparisonStatusEmpty,
		Prices:       []models.PriceOffer{},
		Sections:     []models.CountrySection{},
		Metadata: models.ComparisonMetadata{
			PriceType: "retail",
		},
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:  time.Now().UTC().Format(time.RFC3339),
	})
}
