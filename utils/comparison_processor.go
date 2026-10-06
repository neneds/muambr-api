package utils

import (
	"sort"
	"strings"

	"muambr-api/models"
)

const (
	// Keep prices within this band of the country median. A phone case is far
	// below a phone median; an import-shop listing is far above a soda median.
	priceOutlierLowFactor  = 0.20
	priceOutlierHighFactor = 3.0
	// Need several name matches before those offers, rather than every offer, set the median.
	priceOutlierMinMatches = 3
)

// ComparisonProcessor handles the processing and filtering of product comparisons
type ComparisonProcessor struct{}

// NewComparisonProcessor creates a new ComparisonProcessor with default settings
func NewComparisonProcessor() *ComparisonProcessor {
	return &ComparisonProcessor{}
}

// ProcessComparisons processes raw comparisons and returns organized country sections.
// query is the searched product name, used so the price band follows matching offers.
func (cp *ComparisonProcessor) ProcessComparisons(comparisons []models.ProductComparison, query string, limit int) []models.CountrySection {
	if len(comparisons) == 0 {
		return []models.CountrySection{}
	}

	// Group first so one country's prices do not set the band for another.
	countryGroups := cp.groupComparisonsByCountry(comparisons)
	for countryCode, countryComparisons := range countryGroups {
		countryGroups[countryCode] = cp.filterPriceOutliers(countryComparisons, query)
	}

	// Step 3: Process each country group: sort by price and apply per-country limit
	var sections []models.CountrySection
	for countryCode, countryComparisons := range countryGroups {
		// Sort by price (smallest first) and apply limit
		processedComparisons := cp.sortAndLimitCountryComparisons(countryComparisons, limit)

		// Create country section
		section := models.CountrySection{
			Country:      countryCode,
			CountryName:  cp.getCountryName(countryCode),
			Comparisons:  processedComparisons,
			ResultsCount: len(processedComparisons),
		}
		sections = append(sections, section)
	}

	return sections
}

// filterPriceOutliers drops prices far below or far above the median.
// The median comes from offers that match the query when there are enough of them,
// so a few expensive imports do not hide a cheap real product, and a cheap
// accessory still falls outside a cluster of expensive matches.
func (cp *ComparisonProcessor) filterPriceOutliers(comparisons []models.ProductComparison, query string) []models.ProductComparison {
	if len(comparisons) <= 2 {
		return comparisons
	}

	type pricedOffer struct {
		comparison models.ProductComparison
		price      float64
		confidence float64
	}

	offers := make([]pricedOffer, 0, len(comparisons))
	for _, comparison := range comparisons {
		price := cp.getEffectivePrice(comparison)
		if price <= 0 {
			continue
		}
		confidence := 0.5
		if strings.TrimSpace(query) != "" {
			confidence = MatchConfidence(query, comparison.ProductName)
		}
		offers = append(offers, pricedOffer{comparison: comparison, price: price, confidence: confidence})
	}
	if len(offers) <= 2 {
		return comparisons
	}

	sample := make([]float64, 0, len(offers))
	for _, offer := range offers {
		if offer.confidence >= MatchConfidenceBestPriceMin {
			sample = append(sample, offer.price)
		}
	}
	if len(sample) < priceOutlierMinMatches {
		sample = sample[:0]
		for _, offer := range offers {
			sample = append(sample, offer.price)
		}
	}

	mid := medianPrice(sample)
	if mid <= 0 {
		return comparisons
	}
	minAcceptable := mid * priceOutlierLowFactor
	maxAcceptable := mid * priceOutlierHighFactor

	filtered := make([]models.ProductComparison, 0, len(offers))
	for _, offer := range offers {
		if offer.price >= minAcceptable && offer.price <= maxAcceptable {
			filtered = append(filtered, offer.comparison)
			continue
		}
		Info("Filtering out price outlier",
			String("product_name", offer.comparison.ProductName),
			String("store_name", offer.comparison.StoreName),
			String("country", offer.comparison.Country),
			Float64("effective_price", offer.price),
			Float64("median_price", mid),
			Float64("min_acceptable_price", minAcceptable),
			Float64("max_acceptable_price", maxAcceptable))
	}
	if len(filtered) == 0 {
		return comparisons
	}
	return filtered
}

func medianPrice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// groupComparisonsByCountry groups product comparisons by country using the Country field
func (cp *ComparisonProcessor) groupComparisonsByCountry(comparisons []models.ProductComparison) map[string][]models.ProductComparison {
	countryGroups := make(map[string][]models.ProductComparison)

	for _, comparison := range comparisons {
		// Use the Country field directly from the ProductComparison model
		countryCode := comparison.Country
		if countryCode == "" {
			countryCode = "Unknown" // Fallback for empty country
		}
		countryGroups[countryCode] = append(countryGroups[countryCode], comparison)
	}

	return countryGroups
}

// sortAndLimitCountryComparisons sorts a country's comparisons by price (smallest first) and applies per-country limit
func (cp *ComparisonProcessor) sortAndLimitCountryComparisons(comparisons []models.ProductComparison, limit int) []models.ProductComparison {
	// Sort by price (smallest first), using converted price if available
	sort.Slice(comparisons, func(i, j int) bool {
		priceI := cp.getEffectivePrice(comparisons[i])
		priceJ := cp.getEffectivePrice(comparisons[j])
		return priceI < priceJ
	})

	// Apply per-country limit
	if limit > 0 && len(comparisons) > limit {
		return comparisons[:limit]
	}

	return comparisons
}

// getEffectivePrice returns the converted price if available, otherwise the original price
func (cp *ComparisonProcessor) getEffectivePrice(comparison models.ProductComparison) float64 {
	if comparison.ConvertedPrice != nil {
		return comparison.ConvertedPrice.Price
	}
	return comparison.Price
}

// getCountryName returns the human-readable country name for a country code
func (cp *ComparisonProcessor) getCountryName(countryCode string) string {
	if country, err := models.ParseCountryFromISO(countryCode); err == nil {
		return country.GetCountryName()
	}
	return countryCode // Fallback to country code if not found
}
