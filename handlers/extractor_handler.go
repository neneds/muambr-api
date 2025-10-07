package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"muambr-api/extractors"
	"muambr-api/models"
)

// ExtractorHandler handles country detection and price extraction coordination
type ExtractorHandler struct {
	extractorRegistry *extractors.ExtractorRegistry
}

// NewExtractorHandler creates a new ExtractorHandler with initialized extractors
func NewExtractorHandler() *ExtractorHandler {
	registry := extractors.NewExtractorRegistry()
	initializeExtractors(registry)
	
	return &ExtractorHandler{
		extractorRegistry: registry,
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

// convertCurrency converts a price from one currency to another
// TODO: This is a placeholder implementation. In production, integrate with a real currency conversion API
func (h *ExtractorHandler) convertCurrency(price string, fromCurrency string, toCurrency string) *models.ConvertedPrice {
	if fromCurrency == toCurrency {
		return nil // No conversion needed
	}
	
	// TODO: Implement real currency conversion using an external API like:
	// - ExchangeRate-API
	// - Fixer.io
	// - CurrencyLayer
	// - European Central Bank API
	
	// For now, return a placeholder conversion
	// This should be replaced with actual API calls
	mockExchangeRates := map[string]map[string]float64{
		"BRL": {
			"EUR": 0.18,  // 1 BRL = 0.18 EUR (approximate)
			"USD": 0.20,  // 1 BRL = 0.20 USD (approximate)
		},
		"EUR": {
			"BRL": 5.55,  // 1 EUR = 5.55 BRL (approximate)
			"USD": 1.10,  // 1 EUR = 1.10 USD (approximate)
		},
		"USD": {
			"BRL": 5.00,  // 1 USD = 5.00 BRL (approximate)
			"EUR": 0.91,  // 1 USD = 0.91 EUR (approximate)
		},
	}
	
	// Parse the price (remove any non-numeric characters except decimal point)
	priceFloat := parsePrice(price)
	if priceFloat <= 0 {
		return nil
	}
	
	// Get exchange rate
	if rates, exists := mockExchangeRates[fromCurrency]; exists {
		if rate, rateExists := rates[toCurrency]; rateExists {
			convertedAmount := priceFloat * rate
			return &models.ConvertedPrice{
				Price:    fmt.Sprintf("%.2f", convertedAmount),
				Currency: toCurrency,
			}
		}
	}
	
	// If no exchange rate found, return nil
	return nil
}

// parsePrice parses a price string and returns the numeric value
func parsePrice(priceStr string) float64 {
	// Remove common currency symbols and formatting
	cleaned := priceStr
	
	// Remove currency symbols
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "R$", "")
	cleaned = strings.ReplaceAll(cleaned, "£", "")
	cleaned = strings.ReplaceAll(cleaned, "¥", "")
	
	// Remove spaces
	cleaned = strings.TrimSpace(cleaned)
	
	// Handle different decimal separators
	// Convert European format (1.234,56) to US format (1234.56)
	if strings.Contains(cleaned, ",") && strings.Contains(cleaned, ".") {
		// Format like 1.234,56 -> 1234.56
		parts := strings.Split(cleaned, ",")
		if len(parts) == 2 {
			integerPart := strings.ReplaceAll(parts[0], ".", "")
			cleaned = integerPart + "." + parts[1]
		}
	} else if strings.Contains(cleaned, ",") && !strings.Contains(cleaned, ".") {
		// Format like 1234,56 -> 1234.56
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}
	
	// Parse as float
	if value, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return value
	}
	
	return 0
}

// CountryValidationError represents an error in country code validation
type CountryValidationError struct {
	Code            string   `json:"code"`
	SupportedCodes  []string `json:"supportedCodes"`
}

func (e *CountryValidationError) Error() string {
	return "Invalid country ISO code: " + e.Code
}