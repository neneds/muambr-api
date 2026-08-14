package handlers

import (
	"context"
	"fmt"
	"muambr-api/extractors"
	appliances "muambr-api/extractors/appliances"
	beauty "muambr-api/extractors/beauty"
	other "muambr-api/extractors/other"
	"muambr-api/models"
	"muambr-api/utils"
	"strconv"
	"sync"
	"time"
)

// ExtractorConfig holds configuration for extractor execution
type ExtractorConfig struct {
	Timeout        time.Duration
	RetryAttempts  int
	EnableParallel bool
	MaxConcurrency int
}

// ExtractorHandler handles country detection and price extraction coordination
type ExtractorHandler struct {
	extractorRegistry   *extractors.ExtractorRegistry
	exchangeRateService *utils.ExchangeRateService
	config              ExtractorConfig
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
	utils.Info("Registering V2 extractors architecture")

	// other category — generic extractors used when no category is specified
	registry.RegisterExtractor(other.NewKuantoKustaExtractorV2())
	registry.RegisterExtractor(other.NewAcharPromoExtractorV2())
	registry.RegisterExtractor(other.NewAmericanasBRExtractor())
	registry.RegisterExtractor(other.NewWalmartUSAExtractor())

	// beauty category extractors
	registry.RegisterExtractor(beauty.NewEpocaCosmeticosExtractor())
	registry.RegisterExtractor(beauty.NewSephoraBRExtractor())
	registry.RegisterExtractor(beauty.NewPrimorPTExtractor())
	registry.RegisterExtractor(beauty.NewPerfumesECompanhiaPTExtractor())

	// appliances category extractors
	registry.RegisterExtractor(appliances.NewCarrefourBRExtractor())
	registry.RegisterExtractor(appliances.NewFastshopBRExtractor())

	utils.Info("All V2 extractors registered successfully")
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
			Code:           countryParam,
			SupportedCodes: []string{"PT", "US", "ES", "DE", "GB", "BR"},
		}
	}

	return country, nil
}

// ExtractionOutcome is the result of running extractors for a comparison request.
type ExtractionOutcome struct {
	Comparisons []models.ProductComparison
	Meta        utils.ExtractionMeta
}

// GetProductComparisons retrieves product comparisons using available extractors
func (h *ExtractorHandler) GetProductComparisons(searchTerm string, baseCountry models.Country, currentCountry *models.Country, targetCurrency string, useMacroRegion bool, category *models.ProductCategory) ([]models.ProductComparison, error) {
	outcome, err := h.GetProductComparisonsWithMeta(searchTerm, baseCountry, currentCountry, targetCurrency, useMacroRegion, category)
	if err != nil {
		return nil, err
	}
	return outcome.Comparisons, nil
}

// GetProductComparisonsWithMeta retrieves comparisons plus provider success/failure metadata.
func (h *ExtractorHandler) GetProductComparisonsWithMeta(searchTerm string, baseCountry models.Country, currentCountry *models.Country, targetCurrency string, useMacroRegion bool, category *models.ProductCategory) (*ExtractionOutcome, error) {
	if sanitized := utils.SanitizeSearchQuery(searchTerm); sanitized != searchTerm {
		utils.Info("Sanitized search query",
			utils.String("original", searchTerm),
			utils.String("sanitized", sanitized))
		searchTerm = sanitized
	}

	utils.Info("GetProductComparisons called",
		utils.String("searchTerm", searchTerm),
		utils.String("baseCountry", string(baseCountry)),
		utils.Any("currentCountry", currentCountry),
		utils.String("targetCurrency", targetCurrency),
		utils.Bool("useMacroRegion", useMacroRegion),
		utils.Any("category", category))

	allResults := make([]models.ProductComparison, 0)

	// Use a map to track extractors by their identifier to prevent duplicates
	extractorMap := make(map[string]extractors.Extractor)

	// Always use extractors from the base country (country parameter)
	baseCountryExtractors := h.extractorRegistry.GetExtractorsForCountry(baseCountry, category)
	for _, extractor := range baseCountryExtractors {
		extractorMap[extractor.GetIdentifier()] = extractor
	}
	utils.Info("Added base country extractors",
		utils.String("baseCountry", string(baseCountry)),
		utils.Int("count", len(baseCountryExtractors)))

	// Add extractors from current country if different from base country
	if currentCountry != nil && *currentCountry != baseCountry {
		currentCountryExtractors := h.extractorRegistry.GetExtractorsForCountry(*currentCountry, category)
		for _, extractor := range currentCountryExtractors {
			extractorMap[extractor.GetIdentifier()] = extractor
		}
		utils.Info("Added current country extractors",
			utils.String("currentCountry", string(*currentCountry)),
			utils.Int("count", len(currentCountryExtractors)))
	}

	// If macro region is enabled, add extractors from all countries in the current user's macro region
	if useMacroRegion && currentCountry != nil {
		macroRegion := currentCountry.GetMacroRegion()
		countriesInRegion := models.GetCountriesInMacroRegion(macroRegion)
		utils.Info("Processing macro region",
			utils.String("macroRegion", string(macroRegion)),
			utils.String("currentCountry", string(*currentCountry)),
			utils.Int("countries", len(countriesInRegion)))

		for _, country := range countriesInRegion {
			regionExtractors := h.extractorRegistry.GetExtractorsForCountry(country, category)
			for _, extractor := range regionExtractors {
				extractorMap[extractor.GetIdentifier()] = extractor
			}
		}
		utils.Info("Added macro region extractors",
			utils.String("macroRegion", string(macroRegion)),
			utils.Int("totalCountries", len(countriesInRegion)),
			utils.Int("totalExtractors", len(extractorMap)))
	}

	// If a specific category has no registered providers, fall back to generic ("other").
	// Matches product rule: unknown / unavailable category → generic comparison strategy.
	if len(extractorMap) == 0 && category != nil && *category != models.CategoryOther {
		utils.Info("No extractors for requested category — falling back to generic providers",
			utils.String("requestedCategory", string(*category)))
		fallback := models.CategoryOther
		return h.GetProductComparisonsWithMeta(searchTerm, baseCountry, currentCountry, targetCurrency, useMacroRegion, &fallback)
	}

	extractorsToUse := make([]extractors.Extractor, 0, len(extractorMap))
	for _, extractor := range extractorMap {
		extractorsToUse = append(extractorsToUse, extractor)
	}

	extractorNames := make([]string, len(extractorsToUse))
	extractorCountries := make([]string, len(extractorsToUse))
	for i, extractor := range extractorsToUse {
		extractorNames[i] = extractor.GetIdentifier()
		extractorCountries[i] = string(extractor.GetCountryCode())
	}

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

	var runResults []ExtractorResult
	if h.config.EnableParallel && len(extractorsToUse) > 1 {
		runResults = h.runExtractorsInParallel(extractorsToUse, searchTerm, baseCountry, targetCurrency)
	} else {
		runResults = h.runExtractorsSequentially(extractorsToUse, searchTerm, baseCountry, targetCurrency)
	}

	meta := utils.ExtractionMeta{
		ProvidersAttempted: len(extractorsToUse),
		FailedProviders:    make([]string, 0),
	}
	for _, result := range runResults {
		if result.Error != nil {
			meta.ProvidersFailed++
			meta.FailedProviders = append(meta.FailedProviders, result.ExtractorName)
			utils.Warn("Extractor failed during product search - continuing with remaining extractors",
				utils.String("search_term", searchTerm),
				utils.String("extractor_name", result.ExtractorName),
				utils.String("extractor_country", result.ExtractorCountry),
				utils.String("base_country", string(baseCountry)),
				utils.String("target_currency", targetCurrency),
				utils.String("duration", result.Duration.String()),
				utils.Error(result.Error))
			continue
		}
		meta.ProvidersSucceeded++
		utils.Info("Extractor successfully completed product search",
			utils.String("search_term", searchTerm),
			utils.String("extractor_name", result.ExtractorName),
			utils.String("extractor_country", result.ExtractorCountry),
			utils.String("duration", result.Duration.String()),
			utils.Int("results_count", len(result.Results)))
		allResults = append(allResults, result.Results...)
	}

	if targetCurrency != "" {
		allResults = h.applyCountryContextAndCurrencyConversion(allResults, baseCountry, currentCountry, targetCurrency)
	}

	utils.Info("Product comparison search completed",
		utils.String("search_term", searchTerm),
		utils.String("base_country", string(baseCountry)),
		utils.Any("current_country", currentCountry),
		utils.String("target_currency", targetCurrency),
		utils.Int("total_results", len(allResults)),
		utils.Int("extractors_attempted", len(extractorsToUse)),
		utils.Int("extractors_succeeded", meta.ProvidersSucceeded),
		utils.Int("extractors_failed", meta.ProvidersFailed))

	if allResults == nil {
		allResults = make([]models.ProductComparison, 0)
	}
	return &ExtractionOutcome{
		Comparisons: allResults,
		Meta:        meta,
	}, nil
}

// GetExchangeRateService exposes the shared FX service for comparison enrichment.
func (h *ExtractorHandler) GetExchangeRateService() *utils.ExchangeRateService {
	return h.exchangeRateService
}

// ExtractorResult represents the result of an extractor execution
type ExtractorResult struct {
	ExtractorName    string
	ExtractorCountry string
	Results          []models.ProductComparison
	Error            error
	Duration         time.Duration
}

// runExtractorsInParallel executes extractors concurrently and returns per-provider results.
func (h *ExtractorHandler) runExtractorsInParallel(extractorList []extractors.Extractor, searchTerm string, baseCountry models.Country, targetCurrency string) []ExtractorResult {
	resultChan := make(chan ExtractorResult, len(extractorList))
	semaphore := make(chan struct{}, h.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, extractor := range extractorList {
		wg.Add(1)
		go func(ext extractors.Extractor) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			resultChan <- h.executeExtractorWithTimeout(ext, searchTerm, baseCountry, targetCurrency)
		}(extractor)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make([]ExtractorResult, 0, len(extractorList))
	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// runExtractorsSequentially executes extractors one by one and returns per-provider results.
func (h *ExtractorHandler) runExtractorsSequentially(extractorList []extractors.Extractor, searchTerm string, baseCountry models.Country, targetCurrency string) []ExtractorResult {
	results := make([]ExtractorResult, 0, len(extractorList))
	for _, extractor := range extractorList {
		results = append(results, h.executeExtractorWithTimeout(extractor, searchTerm, baseCountry, targetCurrency))
	}
	return results
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
					Error:            fmt.Errorf("extractor panicked: %v", r),
					Duration:         time.Since(start),
				}
			}
		}()

		results, err := extractor.GetComparisons(searchTerm)
		resultChan <- ExtractorResult{
			ExtractorName:    extractor.GetIdentifier(),
			ExtractorCountry: string(extractor.GetCountryCode()),
			Results:          results,
			Error:            err,
			Duration:         time.Since(start),
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
			Error:            fmt.Errorf("extractor timeout after %v: %w", h.config.Timeout, ctx.Err()),
			Duration:         time.Since(start),
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
	Code           string   `json:"code"`
	SupportedCodes []string `json:"supportedCodes"`
}

func (e *CountryValidationError) Error() string {
	return "Invalid country ISO code: " + e.Code
}
