package handlers

import (
	"fmt"
	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// ExtractorHandler handles country detection and price extraction coordination
type ExtractorHandler struct {
	extractorRegistry   *extractors.ExtractorRegistry
	exchangeRateService *utils.ExchangeRateService
}

// NewExtractorHandler creates a new ExtractorHandler with initialized extractors
func NewExtractorHandler() *ExtractorHandler {
	registry := extractors.NewExtractorRegistry()
	initializeExtractors(registry)
	
	return &ExtractorHandler{
		extractorRegistry:   registry,
		exchangeRateService: utils.NewExchangeRateService(),
	}
}

// initializeExtractors initializes and registers all available extractors
func initializeExtractors(registry *extractors.ExtractorRegistry) {
	// Register Kelkoo extractor for Spain only
	registry.RegisterExtractor(extractors.NewKelkooExtractor())
	
	// Register KuantoKusta extractor for Portugal only
	registry.RegisterExtractor(extractors.NewKuantoKustaExtractor())
	
	// Register Mercado Livre extractor for Brazil only
	registry.RegisterExtractor(extractors.NewMercadoLivreExtractor())
	
	// TODO: Add more extractors as they are implemented
}

// DetectCountryCode detects and validates the country code for the currentCountry parameter
func (h *ExtractorHandler) DetectCountryCode(countryParam string) (models.Country, error) {
	if countryParam == "" {
		// TODO: Implement IP-based country detection as fallback
		// For now, return empty country (no detection)
		return "", nil
	}
	
	// Parse and validate the provided country code
	country, err := models.ParseCountryFromISO(countryParam)
	if err != nil {
		return "", &CountryValidationError{
			Code:            countryParam,
			SupportedCodes: []string{"PT", "US", "ES", "DE", "GB", "BR"},
		}
	}
	
	return country, nil
}

// GetProductComparisons retrieves product comparisons using available extractors
func (h *ExtractorHandler) GetProductComparisons(searchTerm string, baseCountry models.Country, currentCountry *models.Country, targetCurrency string) ([]models.ProductComparison, error) {
	var allResults []models.ProductComparison
	
	// Always use extractors from the base country (country parameter)
	extractorsToUse := h.extractorRegistry.GetExtractorsForCountry(baseCountry)
	
	// If currentCountry is available and different from baseCountry, append extractors from current country
	if currentCountry != nil && *currentCountry != baseCountry {
		currentCountryExtractors := h.extractorRegistry.GetExtractorsForCountry(*currentCountry)
		extractorsToUse = append(extractorsToUse, currentCountryExtractors...)
	}

	// Execute all selected extractors
	for _, extractor := range extractorsToUse {
		results, err := extractor.GetComparisons(searchTerm)
		if err != nil {
			// Log error but continue with other extractors
			continue
		}
		allResults = append(allResults, results...)
	}
	
	// Apply currency conversion if needed
	if targetCurrency != "" {
		allResults = h.applyCountryContextAndCurrencyConversion(allResults, baseCountry, currentCountry, targetCurrency)
	}
	
	return allResults, nil
}

// applyCountryContextAndCurrencyConversion applies country context and currency conversion to comparison results
func (h *ExtractorHandler) applyCountryContextAndCurrencyConversion(comparisons []models.ProductComparison, baseCountry models.Country, currentCountry *models.Country, targetCurrency string) []models.ProductComparison {
	
	// Apply country context and currency conversion
	for i := range comparisons {
		// Add availability context based on base country (user's home country)
		comparisons[i].Store += " (Available for " + baseCountry.GetCountryName() + ")"
		
		// If user is in a different country than their base country, add location context
		if currentCountry != nil && *currentCountry != baseCountry {
			comparisons[i].Store += " - Browsing from " + currentCountry.GetCountryName()
		}
		
		// Apply currency conversion: convert from product's currency to target currency
		if targetCurrency != "" && comparisons[i].Currency != targetCurrency {
			convertedPrice := h.convertCurrency(comparisons[i].Price, comparisons[i].Currency, targetCurrency)
			if convertedPrice != nil {
				comparisons[i].ConvertedPrice = convertedPrice
			}
		}
	}
	
	return comparisons
}

// convertCurrency converts a price from one currency to another using real exchange rates with caching
func (h *ExtractorHandler) convertCurrency(price string, fromCurrency string, toCurrency string) *models.ConvertedPrice {
	if fromCurrency == toCurrency {
		return nil // No conversion needed
	}
	
	// Use the exchange rate service to convert the price
	convertedPriceStr, err := h.exchangeRateService.ConvertPriceString(price, fromCurrency, toCurrency)
	if err != nil {
		// Log error and return nil - this will allow the product to still be shown without conversion
		fmt.Printf("Currency conversion failed from %s to %s for price %s: %v\n", fromCurrency, toCurrency, price, err)
		return nil
	}
	
	return &models.ConvertedPrice{
		Price:    convertedPriceStr,
		Currency: toCurrency,
	}
}

// CountryValidationError represents an error in country code validation
type CountryValidationError struct {
	Code            string   `json:"code"`
	SupportedCodes  []string `json:"supportedCodes"`
}

func (e *CountryValidationError) Error() string {
	return "Invalid country ISO code: " + e.Code
}