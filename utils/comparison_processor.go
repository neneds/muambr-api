package utils

import (
	"sort"
	"muambr-api/models"
)

// ComparisonProcessor handles the processing and filtering of product comparisons
type ComparisonProcessor struct {
	priceOutlierThreshold float64 // Threshold for filtering outlier prices (e.g., 0.6 for 60% below)
}

// NewComparisonProcessor creates a new ComparisonProcessor with default settings
func NewComparisonProcessor() *ComparisonProcessor {
	return &ComparisonProcessor{
		priceOutlierThreshold: 0.6, // 60% below average price threshold
	}
}

// ProcessComparisons processes raw comparisons and returns organized country sections
func (cp *ComparisonProcessor) ProcessComparisons(comparisons []models.ProductComparison, limit int) []models.CountrySection {
	if len(comparisons) == 0 {
		return []models.CountrySection{}
	}

	// Step 1: Filter out price outliers (values 60% below average)
	filteredComparisons := cp.filterPriceOutliers(comparisons)
	
	// Step 2: Group comparisons by country
	countryGroups := cp.groupComparisonsByCountry(filteredComparisons)
	
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

// filterPriceOutliers removes products with prices that are 60% below the average price
func (cp *ComparisonProcessor) filterPriceOutliers(comparisons []models.ProductComparison) []models.ProductComparison {
	if len(comparisons) <= 2 {
		// Don't filter if we have too few comparisons
		return comparisons
	}
	
	// Calculate average price using effective prices (converted if available)
	var totalPrice float64
	var validPrices []float64
	
	for _, comparison := range comparisons {
		effectivePrice := cp.getEffectivePrice(comparison)
		if effectivePrice > 0 { // Only consider positive prices
			totalPrice += effectivePrice
			validPrices = append(validPrices, effectivePrice)
		}
	}
	
	if len(validPrices) == 0 {
		return comparisons // Return original if no valid prices found
	}
	
	averagePrice := totalPrice / float64(len(validPrices))
	minAcceptablePrice := averagePrice * cp.priceOutlierThreshold
	
	// Filter out products with prices below the threshold
	var filteredComparisons []models.ProductComparison
	for _, comparison := range comparisons {
		effectivePrice := cp.getEffectivePrice(comparison)
		if effectivePrice >= minAcceptablePrice {
			filteredComparisons = append(filteredComparisons, comparison)
		} else {
			// Log filtered out products for debugging
			Info("Filtering out price outlier",
				String("product_name", comparison.ProductName),
				String("store_name", comparison.StoreName),
				String("country", comparison.Country),
				Float64("effective_price", effectivePrice),
				Float64("average_price", averagePrice),
				Float64("min_acceptable_price", minAcceptablePrice))
		}
	}
	
	return filteredComparisons
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

// SetPriceOutlierThreshold allows customization of the price filtering threshold
func (cp *ComparisonProcessor) SetPriceOutlierThreshold(threshold float64) {
	if threshold > 0 && threshold <= 1.0 {
		cp.priceOutlierThreshold = threshold
	}
}