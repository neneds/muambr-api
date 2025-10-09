package handlers

import (
	"fmt"
	"strconv"
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
	
	// Register Magazine Luiza extractor for Brazil only
	registry.RegisterExtractor(extractors.NewMagaluExtractor())
	
	// TODO: Add more extractors as they are implemented
}

// DetectCountryCode detects and validates the country code for the currentCountry parameter
func (h *ExtractorHandler) DetectCountryCode(countryParam string) (models.Country, error) {
	if countryParam == "" {
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
func (h *ExtractorHandler) GetProductComparisons(searchTerm string, baseCountry models.Country, currentCountry *models.Country, targetCurrency string, useMacroRegion bool) ([]models.ProductComparison, error) {
	var allResults []models.ProductComparison
	
	// Use a map to track extractors by their identifier to prevent duplicates
	extractorMap := make(map[string]extractors.Extractor)
	
	// Always use extractors from the base country (country parameter)
	baseCountryExtractors := h.extractorRegistry.GetExtractorsForCountry(baseCountry)
	for _, extractor := range baseCountryExtractors {
		extractorMap[extractor.GetIdentifier()] = extractor
	}
	
	// If currentCountry is available and different from baseCountry, append extractors from current country or its macro region
	if currentCountry != nil && *currentCountry != baseCountry {
		if useMacroRegion {
			// Use extractors from all countries in the macro region of the current country
			macroRegion := currentCountry.GetMacroRegion()
			countriesInRegion := models.GetCountriesInMacroRegion(macroRegion)
			for _, country := range countriesInRegion {
				regionExtractors := h.extractorRegistry.GetExtractorsForCountry(country)
				for _, extractor := range regionExtractors {
					// Add extractor only if it's not already in the map (deduplication)
					extractorMap[extractor.GetIdentifier()] = extractor
				}
			}
		} else {
			// Use extractors from current country only (existing behavior)
			currentCountryExtractors := h.extractorRegistry.GetExtractorsForCountry(*currentCountry)
			for _, extractor := range currentCountryExtractors {
				// Add extractor only if it's not already in the map (deduplication)
				extractorMap[extractor.GetIdentifier()] = extractor
			}
		}
	}
	
	// Convert map back to slice for execution
	extractorsToUse := make([]extractors.Extractor, 0, len(extractorMap))
	for _, extractor := range extractorMap {
		extractorsToUse = append(extractorsToUse, extractor)
	}
	
	// Log which extractors will be used for this search
	extractorNames := make([]string, len(extractorsToUse))
	extractorCountries := make([]string, len(extractorsToUse))
	for i, extractor := range extractorsToUse {
		extractorNames[i] = extractor.GetIdentifier()
		extractorCountries[i] = string(extractor.GetCountryCode())
	}
	
	// Log macro region info if applicable
	var macroRegionInfo string
	if currentCountry != nil && useMacroRegion {
		macroRegion := currentCountry.GetMacroRegion()
		countriesInRegion := models.GetCountriesInMacroRegion(macroRegion)
		macroRegionInfo = fmt.Sprintf("%s (%v)", macroRegion, countriesInRegion)
	}
	
	utils.Info("Starting product comparison search with deduplicated extractors",
		utils.String("search_term", searchTerm),
		utils.String("base_country", string(baseCountry)),
		utils.Any("current_country", currentCountry),
		utils.Bool("use_macro_region", useMacroRegion),
		utils.String("macro_region_info", macroRegionInfo),
		utils.String("target_currency", targetCurrency),
		utils.Int("unique_extractor_count", len(extractorsToUse)),
		utils.Any("extractor_names", extractorNames),
		utils.Any("extractor_countries", extractorCountries))

	// Execute all selected extractors independently - errors in one extractor don't affect others
	for _, extractor := range extractorsToUse {
		results, err := extractor.GetComparisons(searchTerm)
		if err != nil {
			// Log error but continue with other extractors to ensure independence
			utils.Warn("Extractor failed during product search - continuing with remaining extractors",
				utils.String("search_term", searchTerm),
				utils.String("extractor_name", extractor.GetIdentifier()),
				utils.String("extractor_country", string(extractor.GetCountryCode())),
				utils.String("base_country", string(baseCountry)),
				utils.String("target_currency", targetCurrency),
				utils.Error(err))
			continue
		}
		
		// Log successful extraction
		utils.Info("Extractor successfully completed product search",
			utils.String("search_term", searchTerm),
			utils.String("extractor_name", extractor.GetIdentifier()),
			utils.String("extractor_country", string(extractor.GetCountryCode())),
			utils.Int("results_count", len(results)))
		
		allResults = append(allResults, results...)
	}
	
	// Apply currency conversion if needed
	if targetCurrency != "" {
		allResults = h.applyCountryContextAndCurrencyConversion(allResults, baseCountry, currentCountry, targetCurrency)
	}
	
	// Log final results summary
	utils.Info("Product comparison search completed",
		utils.String("search_term", searchTerm),
		utils.String("base_country", string(baseCountry)),
		utils.Any("current_country", currentCountry),
		utils.String("target_currency", targetCurrency),
		utils.Int("total_results", len(allResults)),
		utils.Int("extractors_attempted", len(extractorsToUse)))
	
	return allResults, nil
}

// applyCountryContextAndCurrencyConversion applies country context and currency conversion to comparison results
func (h *ExtractorHandler) applyCountryContextAndCurrencyConversion(comparisons []models.ProductComparison, baseCountry models.Country, currentCountry *models.Country, targetCurrency string) []models.ProductComparison {
	
	// Apply country context and currency conversion
	for i := range comparisons {
		// Add simple country context: "StoreName - CountryCode"
		comparisons[i].StoreName += " - " + string(baseCountry)
		
		// Apply currency conversion: convert from product's currency to target currency
		if targetCurrency != "" && comparisons[i].Currency != targetCurrency {
			convertedPrice := h.convertCurrency(comparisons[i].Price, comparisons[i].Currency, targetCurrency)
			if convertedPrice != nil {
				// Set the converted price field instead of modifying the original price
				comparisons[i].ConvertedPrice = &models.ConvertedPrice{
					Price:    convertedPrice.Amount,
					Currency: convertedPrice.Currency,
				}
			}
		}
	}
	
	return comparisons
}

// ConvertedPriceResult represents the result of currency conversion
type ConvertedPriceResult struct {
	Amount   float64
	Currency string
}

// convertCurrency converts a price from one currency to another using real exchange rates with caching
func (h *ExtractorHandler) convertCurrency(price float64, fromCurrency string, toCurrency string) *ConvertedPriceResult {
	if fromCurrency == toCurrency {
		return nil // No conversion needed
	}
	
	// Convert float64 to string for the exchange rate service
	priceStr := strconv.FormatFloat(price, 'f', 2, 64)
	
	// Use the exchange rate service to convert the price
	convertedPriceStr, err := h.exchangeRateService.ConvertPriceString(priceStr, fromCurrency, toCurrency)
	if err != nil {
		// Log error and return nil - this will allow the product to still be shown without conversion
		utils.Warn("Currency conversion failed", 
			utils.String("fromCurrency", fromCurrency),
			utils.String("toCurrency", toCurrency),
			utils.Float64("price", price),
			utils.Error(err),
		)
		return nil
	}
	
	// Convert back to float64
	convertedPrice, err := strconv.ParseFloat(convertedPriceStr, 64)
	if err != nil {
		utils.LogError("Failed to parse converted price", 
			utils.String("convertedPriceStr", convertedPriceStr),
			utils.Error(err),
		)
		return nil
	}
	
	return &ConvertedPriceResult{
		Amount:   convertedPrice,
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