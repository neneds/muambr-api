package extractors

import (
	"muambr-api/models"
)

// Extractor defines the interface that all price extractors must implement
type Extractor interface {
	// GetCountryCode returns the ISO country code this extractor supports
	GetCountryCode() models.Country

	// GetMacroRegion returns the macro region this extractor supports
	// e.g., "EU", "NA", "LATAM"
	// This can be used for broader regional support if needed
	GetMacroRegion() models.MacroRegion
	
	// GetIdentifier returns a static string identifier for this extractor
	GetIdentifier() string

	// BaseURL returns the base URL for the extractor's website
	BaseURL() string

	// GetComparisons extracts product comparisons for the given product name
	// Returns an array of ProductComparison objects
	GetComparisons(productName string) ([]models.ProductComparison, error)
}

// ExtractorRegistry manages all available extractors
type ExtractorRegistry struct {
	extractors map[models.Country][]Extractor
}

// NewExtractorRegistry creates a new registry for managing extractors
func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		extractors: make(map[models.Country][]Extractor),
	}
}

// RegisterExtractor registers an extractor for a specific country
func (r *ExtractorRegistry) RegisterExtractor(extractor Extractor) {
	country := extractor.GetCountryCode()
	r.extractors[country] = append(r.extractors[country], extractor)
}

// GetExtractorsForCountry returns all extractors available for a given country
func (r *ExtractorRegistry) GetExtractorsForCountry(country models.Country) []Extractor {
	return r.extractors[country]
}

// GetAllExtractors returns all registered extractors grouped by country
func (r *ExtractorRegistry) GetAllExtractors() map[models.Country][]Extractor {
	return r.extractors
}

// GetSupportedCountries returns a list of all countries that have registered extractors
func (r *ExtractorRegistry) GetSupportedCountries() []models.Country {
	var countries []models.Country
	for country := range r.extractors {
		countries = append(countries, country)
	}
	return countries
}