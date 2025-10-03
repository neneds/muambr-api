package handlers

import (
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

// GetProductComparisons retrieves product comparisons using all available extractors
func (h *ExtractorHandler) GetProductComparisons(productName string, targetCountry models.Country, currentCountry models.Country, includeMacroRegion bool) ([]models.ProductComparison, error) {
	var allComparisons []models.ProductComparison
	
	var extractorsToUse []extractors.Extractor
	
	if includeMacroRegion && currentCountry != "" {
		// Get all extractors from the same macro region as currentCountry
		currentMacroRegion := currentCountry.GetMacroRegion()
		extractorsToUse = h.extractorRegistry.GetExtractorsForMacroRegion(currentMacroRegion)
	} else {
		// Get extractors for the target country only
		extractorsToUse = h.extractorRegistry.GetExtractorsForCountry(targetCountry)
	}
	
	// Iterate through selected extractors
	for _, extractor := range extractorsToUse {
		// Extract comparisons from this extractor
		comparisons, err := extractor.GetComparisons(productName)
		if err != nil {
			// Log error but continue with other extractors
			// TODO: Add proper logging
			continue
		}
		
		// Apply current country context to results
		comparisons = h.applyCurrentCountryContext(comparisons, currentCountry)
		
		// Add to overall results
		allComparisons = append(allComparisons, comparisons...)
	}
	
	return allComparisons, nil
}

// applyCurrentCountryContext applies current country context to comparison results
func (h *ExtractorHandler) applyCurrentCountryContext(comparisons []models.ProductComparison, currentCountry models.Country) []models.ProductComparison {
	if currentCountry == "" {
		return comparisons
	}
	
	// Apply current country context (shipping, availability, etc.)
	for i := range comparisons {
		// TODO: Implement actual current country logic:
		// - Check shipping availability to current country
		// - Calculate shipping costs
		// - Apply regional restrictions
		// - Convert currencies if needed
		
		// For now, just add availability context to store name
		comparisons[i].Store += " (Available in " + currentCountry.GetCountryName() + ")"
	}
	
	return comparisons
}

// CountryValidationError represents an error in country code validation
type CountryValidationError struct {
	Code            string   `json:"code"`
	SupportedCodes  []string `json:"supportedCodes"`
}

func (e *CountryValidationError) Error() string {
	return "Invalid country ISO code: " + e.Code
}