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

// ExchangeRateResponse represents the response from exchangerate-api.com V6 (paid)
type ExchangeRateResponse struct {
	Result           string             `json:"result"`
	Documentation    string             `json:"documentation"`
	TermsOfUse       string             `json:"terms_of_use"`
	TimeLastUpdate   int64              `json:"time_last_update_unix"`
	TimeNextUpdate   int64              `json:"time_next_update_unix"`
	BaseCode         string             `json:"base_code"`
	ConversionRates  map[string]float64 `json:"conversion_rates"`
}

// ExchangeRateV4Response represents the response from exchangerate-api.com V4 (free)
type ExchangeRateV4Response struct {
	Provider       string             `json:"provider"`
	Terms          string             `json:"terms"`
	Base           string             `json:"base"`
	Date           string             `json:"date"`
	TimeLastUpdate int64              `json:"time_last_updated"`
	Rates          map[string]float64 `json:"rates"`
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
	var baseURL string
	
	Debug("Exchange Rate Service initialization", String("apiKeyLength", fmt.Sprintf("%d", len(apiKey))), String("apiKeyPresent", fmt.Sprintf("%t", apiKey != "")))
	
	if apiKey == "" {
		// Log info - will use free API instead of mock rates
		Info("EXCHANGE_RATE_API_KEY not set, using free exchangerate-api.com v4 endpoint")
		baseURL = "https://api.exchangerate-api.com/v4"
	} else {
		Info("Using exchangerate-api.com v6 endpoint with API key", String("keyPrefix", apiKey[:8]+"..."))
		Info("Note: If API key is inactive, service will automatically fall back to free v4 endpoint")
		baseURL = "https://v6.exchangerate-api.com/v6"
	}

	return &ExchangeRateService{
		apiKey:   apiKey,
		baseURL:  baseURL,
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
	// Try paid API first if API key is available
	if s.apiKey != "" {
		rates, err := s.tryV6API(baseCurrency)
		if err == nil {
			return rates, nil
		}
		
		// Log the V6 failure and fall back to free API
		Warn("V6 API failed (possibly inactive API key), falling back to free V4 API", 
			String("baseCurrency", baseCurrency), 
			String("error", err.Error()))
	}
	
	// Try free V4 API (no API key required)
	return s.tryV4API(baseCurrency)
}

// tryV6API attempts to use the paid V6 API
func (s *ExchangeRateService) tryV6API(baseCurrency string) (map[string]float64, error) {
	url := fmt.Sprintf("%s/%s/latest/%s", s.baseURL, s.apiKey, baseCurrency)
	Debug("Trying V6 API", String("url", url), String("baseCurrency", baseCurrency))
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("V6 API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("V6 API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read V6 response: %w", err)
	}

	var v6Response ExchangeRateResponse
	if err := json.Unmarshal(body, &v6Response); err != nil {
		return nil, fmt.Errorf("failed to parse V6 response: %w", err)
	}

	if v6Response.Result != "success" {
		return nil, fmt.Errorf("V6 API error: %s", v6Response.Result)
	}

	Info("Successfully fetched rates from V6 API", String("baseCurrency", baseCurrency))
	return v6Response.ConversionRates, nil
}

// tryV4API attempts to use the free V4 API
func (s *ExchangeRateService) tryV4API(baseCurrency string) (map[string]float64, error) {
	url := fmt.Sprintf("https://api.exchangerate-api.com/v4/latest/%s", baseCurrency)
	Debug("Trying free V4 API", String("url", url), String("baseCurrency", baseCurrency))
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("V4 API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("V4 API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read V4 response: %w", err)
	}

	var v4Response ExchangeRateV4Response
	if err := json.Unmarshal(body, &v4Response); err != nil {
		return nil, fmt.Errorf("failed to parse V4 response: %w", err)
	}

	Info("Successfully fetched rates from free V4 API", String("baseCurrency", baseCurrency))
	return v4Response.Rates, nil
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
	
	// Remove currency symbols (order matters: remove multi-char symbols first)
	cleaned = strings.ReplaceAll(cleaned, "R$", "")  // Brazilian Real - must be before "$"
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
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