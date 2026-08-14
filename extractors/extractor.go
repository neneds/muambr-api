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
	GetMacroRegion() models.MacroRegion

	// GetCategory returns the product category this extractor is optimised for
	GetCategory() models.ProductCategory

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

// GetExtractorsForCountry returns all extractors available for a given country.
// When category is non-nil only extractors matching that category are returned.
func (r *ExtractorRegistry) GetExtractorsForCountry(country models.Country, category *models.ProductCategory) []Extractor {
	all := r.extractors[country]
	if category == nil {
		return all
	}
	return filterByCategory(all, *category)
}

// filterByCategory returns only the extractors whose category matches the given value
func filterByCategory(extractors []Extractor, category models.ProductCategory) []Extractor {
	var result []Extractor
	for _, e := range extractors {
		if e.GetCategory() == category {
			result = append(result, e)
		}
	}
	return result
}
