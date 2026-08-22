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
	Product             models.ProductIdentityInput   `json:"product"`
	ObservedPrice       *models.ObservedPriceInput    `json:"observedPrice,omitempty"`
	CurrentCountry      string                        `json:"currentCountry"`
	BaseCountry         string                        `json:"baseCountry"`
	Currency            string                        `json:"currency,omitempty"`
	Limit               int                           `json:"limit,omitempty"`
	UseMacroRegion      bool                          `json:"useMacroRegion,omitempty"`
	Source              *models.ComparisonSourceInput `json:"source,omitempty"`
	ProductURL          string                        `json:"productURL,omitempty"`
	ProductLocation     *models.ProductLocationInput  `json:"productLocation,omitempty"`
	ComparisonCountries []string                      `json:"comparisonCountries,omitempty"`
	ComparisonScope     *models.ComparisonScopeInput  `json:"comparisonScope,omitempty"`
}

// CreateProductComparison handles POST /api/v1/product-comparisons
//
// Primary comparison contract for camera / manual / URL / barcode flows.
func (h *ProductComparisonHandler) CreateProductComparison(c *gin.Context) {
	var req CreateProductComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeInvalidRequest, "api.errors.invalid_request_body")
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
				h.sendError(c, http.StatusBadRequest, models.ErrorCodeProductNotFound, "api.errors.product_not_found_url")
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
			// Infer current country from URL only when the client did not send
			// productLocation.country or currentCountry. URL country must not
			// override product location.
			if req.CurrentCountry == "" && (req.ProductLocation == nil || strings.TrimSpace(req.ProductLocation.Country) == "") {
				if u, err := url.Parse(productURL); err == nil {
					req.CurrentCountry = string(guessCountryFromURL(u))
				}
			}
		}
	}

	if productName == "" {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeInvalidRequest, "api.errors.product_name_required")
		return
	}
	if req.BaseCountry == "" {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, "api.errors.base_country_required")
		return
	}

	baseCountry, err := models.ParseCountryFromISO(strings.ToUpper(req.BaseCountry))
	if err != nil {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, "api.errors.invalid_country_code")
		return
	}

	currentCountry := baseCountry
	if req.ProductLocation != nil && strings.TrimSpace(req.ProductLocation.Country) != "" {
		detected, err := h.extractorHandler.DetectCountryCode(req.ProductLocation.Country)
		if err != nil || detected == "" {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, "api.errors.invalid_country_code")
			return
		}
		currentCountry = detected
	} else if req.CurrentCountry != "" {
		detected, err := h.extractorHandler.DetectCountryCode(req.CurrentCountry)
		if err != nil || detected == "" {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, "api.errors.invalid_country_code")
			return
		}
		currentCountry = detected
	}

	explicitCountries, err := parseCountryList(req.ComparisonCountries)
	if err != nil {
		h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, "api.errors.invalid_country_code")
		return
	}
	var journeyCountries []models.Country
	if req.ComparisonScope != nil {
		journeyCountries, err = parseCountryList(req.ComparisonScope.JourneyCountries)
		if err != nil {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeCountryUnknown, "api.errors.invalid_country_code")
			return
		}
	}
	expandMicroregion := req.UseMacroRegion
	if req.ComparisonScope != nil && req.ComparisonScope.MicroregionEnabled != nil {
		expandMicroregion = expandMicroregion || *req.ComparisonScope.MicroregionEnabled
	}
	comparisonSpecs := models.ResolveComparisonSet(models.ComparisonSetInput{
		BaseCountry:       baseCountry,
		ProductLocation:   currentCountry,
		ExplicitCountries: explicitCountries,
		JourneyCountries:  journeyCountries,
		ExpandMicroregion: expandMicroregion,
	})

	normalizedCurrency := req.Currency
	if normalizedCurrency == "" {
		normalizedCurrency = baseCountry.GetCurrencyCode()
	}
	normalizedCurrency = strings.ToUpper(normalizedCurrency)

	if req.ObservedPrice != nil {
		if req.ObservedPrice.Amount <= 0 {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodePriceNotFound, "api.errors.observed_price_amount")
			return
		}
		if strings.TrimSpace(req.ObservedPrice.Currency) == "" {
			h.sendError(c, http.StatusBadRequest, models.ErrorCodeCurrencyUnknown, "api.errors.observed_price_currency")
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
	if req.Product.Category != nil && strings.TrimSpace(string(*req.Product.Category)) != "" {
		parsed, parseErr := models.ParseCategoryFromString(string(*req.Product.Category))
		if parseErr != nil {
			// Unknown category (e.g. grocery) → generic providers
			other := models.CategoryOther
			category = &other
			categoryConfidence = 0.5
		} else {
			category = &parsed
			categoryConfidence = 1.0
		}
	} else {
		other := models.CategoryOther
		category = &other
		categoryConfidence = 0.5
	}

	currentCountryPtr := &currentCountry
	outcome, err := h.extractorHandler.GetProductComparisonsForCountries(
		productName,
		comparisonSpecs,
		baseCountry,
		currentCountryPtr,
		normalizedCurrency,
		category,
	)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, models.ErrorCodeInternalError, "api.errors.failed_get_comparisons")
		return
	}

	sections := h.comparisonProcessor.ProcessComparisons(outcome.Comparisons, limit)

	exchangeRateInfo := h.buildExchangeRateInfo(req.ObservedPrice, normalizedCurrency)

	// If observed currency differs and conversion failed, surface a clear error
	if req.ObservedPrice != nil &&
		!strings.EqualFold(req.ObservedPrice.Currency, normalizedCurrency) &&
		exchangeRateInfo == nil {
		h.sendError(c, http.StatusServiceUnavailable, models.ErrorCodeExchangeRateUnavailable, "api.errors.exchange_rate_unavailable")
		return
	}

	result := h.comparisonEngine.BuildResult(utils.ComparisonEngineInput{
		ProductName:         productName,
		Brand:               req.Product.Brand,
		Model:               req.Product.Model,
		Category:            category,
		CategoryConfidence:  categoryConfidence,
		BaseCountry:         baseCountry,
		CurrentCountry:      currentCountry,
		NormalizedCurrency:  normalizedCurrency,
		Observed:            req.ObservedPrice,
		Sections:            sections,
		Comparisons:         outcome.Comparisons,
		ComparisonCountries: comparisonSpecs,
		Meta:                outcome.Meta,
		EntryMethod:         entryMethod,
		ExchangeRate:        exchangeRateInfo,
		Now:                 time.Now().UTC(),
	})

	localizeEmptyComparisonMessage(c, &result)

	c.JSON(http.StatusOK, result)
}

func parseCountryList(codes []string) ([]models.Country, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	out := make([]models.Country, 0, len(codes))
	for _, raw := range codes {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		country, err := models.ParseCountryFromISO(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, country)
	}
	return out, nil
}

func localizeEmptyComparisonMessage(c *gin.Context, result *models.ProductComparisonResult) {
	if result == nil || result.Message == nil || result.Code == nil {
		return
	}
	if *result.Code != models.ErrorCodeProductNotFound {
		return
	}
	msg := localization.TAccept(c.GetHeader("Accept-Language"), "api.errors.no_comparable_prices")
	result.Message = &msg
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

func (h *ProductComparisonHandler) sendError(c *gin.Context, status int, code models.APIErrorCode, messageKey string) {
	message := localization.TAccept(c.GetHeader("Accept-Language"), messageKey)
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
