package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ExchangeRateResponse represents the response from exchangerate-api.com
type ExchangeRateResponse struct {
	Result           string             `json:"result"`
	Documentation    string             `json:"documentation"`
	TermsOfUse       string             `json:"terms_of_use"`
	TimeLastUpdate   int64              `json:"time_last_update_unix"`
	TimeNextUpdate   int64              `json:"time_next_update_unix"`
	BaseCode         string             `json:"base_code"`
	ConversionRates  map[string]float64 `json:"conversion_rates"`
}

// CachedExchangeRate represents a cached exchange rate entry
type CachedExchangeRate struct {
	Rates     map[string]float64
	Timestamp time.Time
	BaseCode  string
}

// ExchangeRateService handles currency conversion with caching
type ExchangeRateService struct {
	apiKey    string
	baseURL   string
	cache     map[string]*CachedExchangeRate
	cacheTTL  time.Duration
	mutex     sync.RWMutex
}

// NewExchangeRateService creates a new exchange rate service with API key from environment
func NewExchangeRateService() *ExchangeRateService {
	apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	if apiKey == "" {
		// Log warning but don't fail - fall back to mock rates
		Warn("EXCHANGE_RATE_API_KEY not set, using mock exchange rates")
	}

	return &ExchangeRateService{
		apiKey:   apiKey,
		baseURL:  "https://v6.exchangerate-api.com/v6",
		cache:    make(map[string]*CachedExchangeRate),
		cacheTTL: 5 * time.Hour, // Cache for 5 hours as requested
		mutex:    sync.RWMutex{},
	}
}

// GetExchangeRates retrieves exchange rates for a base currency, using cache when available
func (s *ExchangeRateService) GetExchangeRates(baseCurrency string) (map[string]float64, error) {
	s.mutex.RLock()
	cached, exists := s.cache[baseCurrency]
	s.mutex.RUnlock()

	// Check if we have valid cached data
	if exists && time.Since(cached.Timestamp) < s.cacheTTL {
		return cached.Rates, nil
	}

	// Cache miss or expired - fetch new rates
	rates, err := s.fetchExchangeRates(baseCurrency)
	if err != nil {
		// If API fails and we have expired cache, use it anyway
		if exists {
			Warn("Exchange rate API failed, using expired cache", String("baseCurrency", baseCurrency))
			return cached.Rates, nil
		}
		
		// If no cache and API fails, return mock rates
		Warn("Exchange rate API failed and no cache available, using mock rates", String("baseCurrency", baseCurrency))
		return s.getMockRates(baseCurrency), nil
	}

	// Cache the new rates
	s.mutex.Lock()
	s.cache[baseCurrency] = &CachedExchangeRate{
		Rates:     rates,
		Timestamp: time.Now(),
		BaseCode:  baseCurrency,
	}
	s.mutex.Unlock()

	return rates, nil
}

// fetchExchangeRates makes an API call to get exchange rates
func (s *ExchangeRateService) fetchExchangeRates(baseCurrency string) (map[string]float64, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("exchange rate API key not configured")
	}

	url := fmt.Sprintf("%s/%s/latest/%s", s.baseURL, s.apiKey, baseCurrency)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response ExchangeRateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Result != "success" {
		return nil, fmt.Errorf("exchange rate API returned error result: %s", response.Result)
	}

	return response.ConversionRates, nil
}

// ConvertCurrency converts an amount from one currency to another
func (s *ExchangeRateService) ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	// Get exchange rates with the from currency as base
	rates, err := s.GetExchangeRates(fromCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get exchange rates for %s: %w", fromCurrency, err)
	}

	// Get the conversion rate to target currency
	rate, exists := rates[toCurrency]
	if !exists {
		return 0, fmt.Errorf("conversion rate not found for %s to %s", fromCurrency, toCurrency)
	}

	return amount * rate, nil
}

// ConvertPriceString converts a price string from one currency to another, returning formatted result
func (s *ExchangeRateService) ConvertPriceString(priceStr, fromCurrency, toCurrency string) (string, error) {
	// Parse price string to float using our custom parser
	price := parsePrice(priceStr)
	if price <= 0 {
		return "", fmt.Errorf("invalid price format: %s", priceStr)
	}

	// Convert currency
	convertedPrice, err := s.ConvertCurrency(price, fromCurrency, toCurrency)
	if err != nil {
		return "", err
	}

	// Format to 2 decimal places
	return fmt.Sprintf("%.2f", convertedPrice), nil
}

// getMockRates returns fallback mock exchange rates when API is unavailable
func (s *ExchangeRateService) getMockRates(baseCurrency string) map[string]float64 {
	// Mock rates based on approximate real-world values (October 2025)
	mockRates := map[string]map[string]float64{
		"USD": {
			"USD": 1.0,
			"EUR": 0.85,
			"BRL": 5.34,
			"GBP": 0.74,
			"JPY": 150.0,
		},
		"EUR": {
			"USD": 1.17,
			"EUR": 1.0,
			"BRL": 6.27,
			"GBP": 0.87,
			"JPY": 176.5,
		},
		"BRL": {
			"USD": 0.19,
			"EUR": 0.16,
			"BRL": 1.0,
			"GBP": 0.14,
			"JPY": 28.1,
		},
	}

	if rates, exists := mockRates[baseCurrency]; exists {
		return rates
	}

	// Default fallback - return identity rates
	return map[string]float64{
		"USD": 1.0,
		"EUR": 1.0,
		"BRL": 1.0,
		"GBP": 1.0,
		"JPY": 1.0,
	}
}

// ClearCache clears all cached exchange rates (useful for testing)
func (s *ExchangeRateService) ClearCache() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cache = make(map[string]*CachedExchangeRate)
}

// GetCacheStatus returns information about cached currencies
func (s *ExchangeRateService) GetCacheStatus() map[string]time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status := make(map[string]time.Time)
	for currency, cached := range s.cache {
		status[currency] = cached.Timestamp
	}
	return status
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