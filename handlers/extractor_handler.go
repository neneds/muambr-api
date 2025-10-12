package handlers

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// ExtractorConfig holds configuration for extractor execution
type ExtractorConfig struct {
	Timeout           time.Duration
	RetryAttempts     int
	EnableParallel    bool
	MaxConcurrency    int
}

// ExtractorHandler handles country detection and price extraction coordination
type ExtractorHandler struct {
	extractorRegistry   *extractors.ExtractorRegistry
	exchangeRateService *utils.ExchangeRateService
	config             ExtractorConfig
}

// NewExtractorHandler creates a new ExtractorHandler with initialized extractors
func NewExtractorHandler() *ExtractorHandler {
	registry := extractors.NewExtractorRegistry()
	initializeExtractors(registry)
	
	return &ExtractorHandler{
		extractorRegistry:   registry,
		exchangeRateService: utils.NewExchangeRateService(),
		config: ExtractorConfig{
			Timeout:        30 * time.Second, // Default timeout per extractor
			RetryAttempts:  1,                // No retries by default
			EnableParallel: true,             // Enable parallel execution
			MaxConcurrency: 5,                // Max concurrent extractors
		},
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
	
	// Register AcharPromo extractor for Brazil only
	registry.RegisterExtractor(extractors.NewAcharPromoExtractor())
	
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

	// Execute all selected extractors with timeout handling and parallel processing
	if h.config.EnableParallel && len(extractorsToUse) > 1 {
		allResults = h.executeExtractorsInParallel(extractorsToUse, searchTerm, baseCountry, targetCurrency)
	} else {
		allResults = h.executeExtractorsSequentially(extractorsToUse, searchTerm, baseCountry, targetCurrency)
	}	// Apply currency conversion if needed
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

// ExtractorResult represents the result of an extractor execution
type ExtractorResult struct {
	ExtractorName    string
	ExtractorCountry string
	Results          []models.ProductComparison
	Error            error
	Duration         time.Duration
}

// executeExtractorsInParallel executes extractors concurrently with timeout and error handling
func (h *ExtractorHandler) executeExtractorsInParallel(extractorList []extractors.Extractor, searchTerm string, baseCountry models.Country, targetCurrency string) []models.ProductComparison {
	var allResults []models.ProductComparison
	resultChan := make(chan ExtractorResult, len(extractorList))
	
	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, h.config.MaxConcurrency)
	var wg sync.WaitGroup

	// Launch extractors in parallel
	for _, extractor := range extractorList {
		wg.Add(1)
		go func(ext extractors.Extractor) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			result := h.executeExtractorWithTimeout(ext, searchTerm, baseCountry, targetCurrency)
			resultChan <- result
		}(extractor)
	}

	// Close the result channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results from all extractors
	for result := range resultChan {
		if result.Error != nil {
			utils.Warn("Extractor failed during parallel product search - continuing with remaining extractors",
				utils.String("search_term", searchTerm),
				utils.String("extractor_name", result.ExtractorName),
				utils.String("extractor_country", result.ExtractorCountry),
				utils.String("base_country", string(baseCountry)),
				utils.String("target_currency", targetCurrency),
				utils.String("duration", result.Duration.String()),
				utils.Error(result.Error))
		} else {
			utils.Info("Extractor successfully completed parallel product search",
				utils.String("search_term", searchTerm),
				utils.String("extractor_name", result.ExtractorName),
				utils.String("extractor_country", result.ExtractorCountry),
				utils.String("duration", result.Duration.String()),
				utils.Int("results_count", len(result.Results)))
			
			allResults = append(allResults, result.Results...)
		}
	}

	return allResults
}

// executeExtractorsSequentially executes extractors one by one with timeout handling
func (h *ExtractorHandler) executeExtractorsSequentially(extractorList []extractors.Extractor, searchTerm string, baseCountry models.Country, targetCurrency string) []models.ProductComparison {
	var allResults []models.ProductComparison

	for _, extractor := range extractorList {
		result := h.executeExtractorWithTimeout(extractor, searchTerm, baseCountry, targetCurrency)
		
		if result.Error != nil {
			utils.Warn("Extractor failed during sequential product search - continuing with remaining extractors",
				utils.String("search_term", searchTerm),
				utils.String("extractor_name", result.ExtractorName),
				utils.String("extractor_country", result.ExtractorCountry),
				utils.String("base_country", string(baseCountry)),
				utils.String("target_currency", targetCurrency),
				utils.String("duration", result.Duration.String()),
				utils.Error(result.Error))
		} else {
			utils.Info("Extractor successfully completed sequential product search",
				utils.String("search_term", searchTerm),
				utils.String("extractor_name", result.ExtractorName),
				utils.String("extractor_country", result.ExtractorCountry),
				utils.String("duration", result.Duration.String()),
				utils.Int("results_count", len(result.Results)))
			
			allResults = append(allResults, result.Results...)
		}
	}

	return allResults
}

// executeExtractorWithTimeout executes a single extractor with timeout handling
func (h *ExtractorHandler) executeExtractorWithTimeout(extractor extractors.Extractor, searchTerm string, baseCountry models.Country, targetCurrency string) ExtractorResult {
	start := time.Now()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	// Channel to receive the result
	resultChan := make(chan ExtractorResult, 1)

	// Execute extractor in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- ExtractorResult{
					ExtractorName:    extractor.GetIdentifier(),
					ExtractorCountry: string(extractor.GetCountryCode()),
					Error:           fmt.Errorf("extractor panicked: %v", r),
					Duration:        time.Since(start),
				}
			}
		}()

		results, err := extractor.GetComparisons(searchTerm)
		resultChan <- ExtractorResult{
			ExtractorName:    extractor.GetIdentifier(),
			ExtractorCountry: string(extractor.GetCountryCode()),
			Results:          results,
			Error:           err,
			Duration:        time.Since(start),
		}
	}()

	// Wait for either completion or timeout
	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return ExtractorResult{
			ExtractorName:    extractor.GetIdentifier(),
			ExtractorCountry: string(extractor.GetCountryCode()),
			Error:           fmt.Errorf("extractor timeout after %v: %w", h.config.Timeout, ctx.Err()),
			Duration:        time.Since(start),
		}
	}
}

// applyCountryContextAndCurrencyConversion applies country context and currency conversion to comparison results
func (h *ExtractorHandler) applyCountryContextAndCurrencyConversion(comparisons []models.ProductComparison, baseCountry models.Country, currentCountry *models.Country, targetCurrency string) []models.ProductComparison {
	
	// Apply country context and currency conversion
	for i := range comparisons {
		// Add simple country context: "StoreName - CountryCode" (use product's actual country, not base country)
		comparisons[i].StoreName += " - " + comparisons[i].Country
		
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