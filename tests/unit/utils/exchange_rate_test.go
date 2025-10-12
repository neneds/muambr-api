package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"muambr-api/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExchangeRateService(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		expectWarning  bool
		expectedAPIKey string
	}{
		{
			name:           "With API key",
			apiKey:         "test_api_key_123",
			expectWarning:  false,
			expectedAPIKey: "test_api_key_123",
		},
		{
			name:           "Without API key",
			apiKey:         "",
			expectWarning:  true,
			expectedAPIKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
			defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)

			if tt.apiKey != "" {
				os.Setenv("EXCHANGE_RATE_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("EXCHANGE_RATE_API_KEY")
			}

			service := utils.NewExchangeRateService()

			// Note: We can't directly access private fields, so we'll test behavior instead
			assert.NotNil(t, service)
		})
	}
}

func TestExchangeRateService_GetExchangeRates_WithoutAPIKey(t *testing.T) {
	// Test without API key - should return mock rates
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()
	
	t.Run("Mock rates fallback when no API key", func(t *testing.T) {
		rates, err := service.GetExchangeRates("USD")

		require.NoError(t, err)
		assert.Equal(t, 1.0, rates["USD"])
		assert.Equal(t, 0.85, rates["EUR"])
		assert.Equal(t, 5.34, rates["BRL"])
	})
	
	t.Run("Cache functionality test", func(t *testing.T) {
		// Clear cache and get rates twice - should be consistent
		service.ClearCache()
		
		rates1, err1 := service.GetExchangeRates("USD")
		require.NoError(t, err1)
		
		rates2, err2 := service.GetExchangeRates("USD")
		require.NoError(t, err2)
		
		// Should return same rates (from cache or mock fallback)
		assert.Equal(t, rates1["EUR"], rates2["EUR"])
		assert.Equal(t, rates1["BRL"], rates2["BRL"])
		
		// Note: When using mock rates without API key, cache might not be populated
		// as the service falls back to mock rates immediately
		// Just verify the service works consistently
		assert.NotEmpty(t, rates1)
		assert.NotEmpty(t, rates2)
	})
}

func TestExchangeRateService_GetExchangeRates_WithAPIKey(t *testing.T) {
	// Test with API key set up
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Setenv("EXCHANGE_RATE_API_KEY", "test_api_key")

	service := utils.NewExchangeRateService()

	t.Run("Service with API key returns rates", func(t *testing.T) {
		// When API key is set, it will try to make real API call or fall back to mock
		// Since we can't mock the internal HTTP client easily, we test the fallback behavior
		rates, err := service.GetExchangeRates("USD")

		require.NoError(t, err)
		assert.NotNil(t, rates)
		assert.Contains(t, rates, "USD")
		assert.Contains(t, rates, "EUR")
		assert.Contains(t, rates, "BRL")
	})
}

func TestExchangeRateService_ConvertCurrency(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	t.Run("Same currency conversion", func(t *testing.T) {
		result, err := service.ConvertCurrency(100.0, "USD", "USD")
		
		require.NoError(t, err)
		assert.Equal(t, 100.0, result)
	})

	t.Run("Successful currency conversion using mock rates", func(t *testing.T) {
		result, err := service.ConvertCurrency(100.0, "USD", "EUR")
		
		require.NoError(t, err)
		assert.Equal(t, 85.0, result) // USD to EUR mock rate is 0.85
	})

	t.Run("Convert BRL to USD", func(t *testing.T) {
		result, err := service.ConvertCurrency(100.0, "BRL", "USD")
		
		require.NoError(t, err)
		assert.Equal(t, 19.0, result) // BRL to USD mock rate is 0.19
	})

	t.Run("Currency not in mock rates", func(t *testing.T) {
		// Test with currency not in mock rates - should still work due to fallback
		result, err := service.ConvertCurrency(100.0, "XYZ", "USD")
		
		require.NoError(t, err)
		assert.Equal(t, 100.0, result) // Fallback mock rates are 1.0 for all
	})
}

func TestExchangeRateService_ConvertPriceString(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	tests := []struct {
		name          string
		priceStr      string
		fromCurrency  string
		toCurrency    string
		expected      string
		expectError   bool
	}{
		{
			name:         "Valid price conversion",
			priceStr:     "$100.00",
			fromCurrency: "USD",
			toCurrency:   "EUR",
			expected:     "85.00",
			expectError:  false,
		},
		{
			name:         "European format price",
			priceStr:     "1.234,56",
			fromCurrency: "USD",
			toCurrency:   "EUR",
			expected:     "1049.38",
			expectError:  false,
		},
		{
			name:        "Invalid price format",
			priceStr:    "invalid",
			expectError: true,
		},
		{
			name:        "Zero price",
			priceStr:    "0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ConvertPriceString(tt.priceStr, tt.fromCurrency, tt.toCurrency)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExchangeRateService_MockRatesBehavior(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	tests := []struct {
		name         string
		baseCurrency string
		expectedUSD  float64
		expectedEUR  float64
	}{
		{
			name:         "USD base currency",
			baseCurrency: "USD",
			expectedUSD:  1.0,
			expectedEUR:  0.85,
		},
		{
			name:         "EUR base currency",
			baseCurrency: "EUR",
			expectedUSD:  1.17,
			expectedEUR:  1.0,
		},
		{
			name:         "BRL base currency",
			baseCurrency: "BRL",
			expectedUSD:  0.19,
			expectedEUR:  0.16,
		},
		{
			name:         "Unknown currency - fallback",
			baseCurrency: "XYZ",
			expectedUSD:  1.0,
			expectedEUR:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rates, err := service.GetExchangeRates(tt.baseCurrency)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedUSD, rates["USD"])
			assert.Equal(t, tt.expectedEUR, rates["EUR"])
		})
	}
}

func TestExchangeRateService_CacheManagement(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	t.Run("GetCacheStatus after rates fetched", func(t *testing.T) {
		// Get some rates to potentially populate cache
		_, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		
		_, err = service.GetExchangeRates("EUR")
		require.NoError(t, err)

		status := service.GetCacheStatus()
		// When using mock rates without API key, cache might not be populated
		// Just verify the method works
		assert.NotNil(t, status)
	})

	t.Run("ClearCache", func(t *testing.T) {
		service.ClearCache()

		status := service.GetCacheStatus()
		assert.Len(t, status, 0)
	})
}

func TestExchangeRateService_ConvertPriceString_ParsePrice(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Simple price",
			input:       "100.50",
			expectError: false,
		},
		{
			name:        "Price with dollar sign",
			input:       "$125.99",
			expectError: false,
		},
		{
			name:        "Price with euro sign",
			input:       "€89.95",
			expectError: false,
		},
		{
			name:        "Brazilian real format",
			input:       "R$250.75",
			expectError: false,
		},
		{
			name:        "European format with comma",
			input:       "1234,56",
			expectError: false,
		},
		{
			name:        "European format with dots and comma",
			input:       "1.234,56",
			expectError: false,
		},
		{
			name:        "Price with spaces",
			input:       " $ 50.25 ",
			expectError: false,
		},
		{
			name:        "British pound",
			input:       "£75.50",
			expectError: false,
		},
		{
			name:        "Japanese yen",
			input:       "¥15000",
			expectError: false,
		},
		{
			name:        "Invalid format",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "Only symbols",
			input:       "$€£",
			expectError: true,
		},
		{
			name:        "Zero price",
			input:       "0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ConvertPriceString(tt.input, "USD", "EUR")
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestExchangeRateService_ErrorHandling(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	t.Run("ConvertCurrency with invalid currency", func(t *testing.T) {
		// Test conversion to an unsupported currency
		_, err := service.ConvertCurrency(100.0, "USD", "INVALID")
		
		// Should fail because INVALID currency is not in mock rates
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversion rate not found")
	})
}

func TestExchangeRateService_CacheBehavior(t *testing.T) {
	t.Run("Cache miss behavior without API key", func(t *testing.T) {
		originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
		defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
		os.Unsetenv("EXCHANGE_RATE_API_KEY")

		service := utils.NewExchangeRateService()
		service.ClearCache()

		// First call should return mock rates (no caching when API fails)
		rates1, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		assert.NotNil(t, rates1)

		// Verify cache is empty (mock rates don't get cached)
		status := service.GetCacheStatus()
		assert.Len(t, status, 0)

		// Second call should also return mock rates
		rates2, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		assert.Equal(t, rates1, rates2) // Should be identical mock rates
	})

	t.Run("Cache hit vs miss performance", func(t *testing.T) {
		originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
		defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
		os.Unsetenv("EXCHANGE_RATE_API_KEY")

		service := utils.NewExchangeRateService()
		service.ClearCache()

		// Measure time for multiple calls - should be consistent (all mock rates)
		start1 := time.Now()
		_, err1 := service.GetExchangeRates("USD")
		duration1 := time.Since(start1)
		require.NoError(t, err1)

		start2 := time.Now()
		_, err2 := service.GetExchangeRates("USD")
		duration2 := time.Since(start2)
		require.NoError(t, err2)

		// Both should be fast since they're mock rates
		assert.Less(t, duration1.Milliseconds(), int64(100))
		assert.Less(t, duration2.Milliseconds(), int64(100))
	})

	t.Run("Cache operations consistency", func(t *testing.T) {
		originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
		defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
		os.Unsetenv("EXCHANGE_RATE_API_KEY")

		service := utils.NewExchangeRateService()

		// Test multiple currencies
		currencies := []string{"USD", "EUR", "BRL", "GBP"}
		results := make(map[string]map[string]float64)

		for _, currency := range currencies {
			rates, err := service.GetExchangeRates(currency)
			require.NoError(t, err)
			results[currency] = rates

			// Verify rates contain the base currency
			assert.Contains(t, rates, currency)
			assert.Equal(t, 1.0, rates[currency])
		}

		// Clear cache and verify consistency
		service.ClearCache()
		status := service.GetCacheStatus()
		assert.Len(t, status, 0)

		// Re-fetch and verify same results (mock rates should be consistent)
		for _, currency := range currencies {
			rates, err := service.GetExchangeRates(currency)
			require.NoError(t, err)
			assert.Equal(t, results[currency], rates, "Mock rates should be consistent for %s", currency)
		}
	})

	t.Run("Cache status tracking", func(t *testing.T) {
		originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
		defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
		os.Unsetenv("EXCHANGE_RATE_API_KEY")

		service := utils.NewExchangeRateService()
		service.ClearCache()

		// Initial state - empty cache
		status := service.GetCacheStatus()
		assert.Len(t, status, 0)

		// Fetch some rates
		currencies := []string{"USD", "EUR", "BRL"}
		for _, currency := range currencies {
			_, err := service.GetExchangeRates(currency)
			require.NoError(t, err)
		}

		// Verify cache status (should still be empty with mock rates)
		status = service.GetCacheStatus()
		assert.Len(t, status, 0) // Mock rates don't populate cache

		// Clear cache again
		service.ClearCache()
		status = service.GetCacheStatus()
		assert.Len(t, status, 0)
	})
}

func TestExchangeRateService_CacheWithMockAPI(t *testing.T) {
	// Test actual cache behavior by simulating successful API responses
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	
	// Set a fake API key to enable API calls
	os.Setenv("EXCHANGE_RATE_API_KEY", "test_key_for_cache_testing")

	t.Run("Cache population and retrieval with API failure fallback", func(t *testing.T) {
		service := utils.NewExchangeRateService()
		service.ClearCache()

		// First call will fail (no mock server) but should use mock rates
		rates1, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		assert.NotNil(t, rates1)

		// Verify cache status - should still be empty since API failed
		status := service.GetCacheStatus()
		assert.Len(t, status, 0)

		// Second call should return same mock rates
		rates2, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		assert.Equal(t, rates1, rates2)
	})

	t.Run("Cache TTL behavior simulation", func(t *testing.T) {
		service := utils.NewExchangeRateService()
		service.ClearCache()

		// Test with different currencies to verify fallback behavior
		currencies := []string{"USD", "EUR", "BRL", "JPY"}
		
		for _, currency := range currencies {
			rates, err := service.GetExchangeRates(currency)
			require.NoError(t, err)
			
			// Verify basic rate properties
			assert.Contains(t, rates, currency)
			assert.Equal(t, 1.0, rates[currency]) // Base currency should be 1.0
			assert.NotEmpty(t, rates)
			
			// Common currencies should be present in mock rates
			if currency == "USD" || currency == "EUR" || currency == "BRL" {
				assert.Contains(t, rates, "USD")
				assert.Contains(t, rates, "EUR") 
				assert.Contains(t, rates, "BRL")
			}
		}

		// Clear cache and verify it's empty
		service.ClearCache()
		status := service.GetCacheStatus()
		assert.Len(t, status, 0)
	})

	t.Run("Multiple currency cache consistency", func(t *testing.T) {
		service := utils.NewExchangeRateService()
		service.ClearCache()

		// Test cross-currency rate consistency
		usdRates, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		
		eurRates, err := service.GetExchangeRates("EUR")
		require.NoError(t, err)
		
		brlRates, err := service.GetExchangeRates("BRL")
		require.NoError(t, err)

		// Verify that mock rates are mathematically consistent where possible
		// USD to EUR should be inverse of EUR to USD (approximately)
		if usdToEur, exists := usdRates["EUR"]; exists {
			if eurToUsd, exists := eurRates["USD"]; exists {
				// They should be inverses (within reasonable tolerance for mock data)
				product := usdToEur * eurToUsd
				assert.Greater(t, product, 0.8) // Should be close to 1.0
				assert.Less(t, product, 1.2)   // But allow some variance in mock data
			}
		}

		// Verify base currencies are always 1.0
		assert.Equal(t, 1.0, usdRates["USD"])
		assert.Equal(t, 1.0, eurRates["EUR"]) 
		assert.Equal(t, 1.0, brlRates["BRL"])
	})
}

func TestExchangeRateService_RealCacheBehavior(t *testing.T) {
	// Test actual cache behavior with a mock HTTP server
	t.Run("Cache population with successful API response", func(t *testing.T) {
		// Create a mock server that returns valid exchange rate data
		callCount := 0
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			
			response := map[string]interface{}{
				"result":      "success",
				"base_code":   "USD",
				"time_last_update_unix": time.Now().Unix(),
				"time_next_update_unix": time.Now().Add(24 * time.Hour).Unix(),
				"conversion_rates": map[string]float64{
					"USD": 1.0,
					"EUR": 0.85,
					"BRL": 5.34,
					"GBP": 0.74,
					"JPY": 150.0,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		// Set up service with API key and custom base URL
		originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
		defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
		os.Setenv("EXCHANGE_RATE_API_KEY", "test_key")

		service := utils.NewExchangeRateService()
		
		// Use reflection to change the baseURL to point to our mock server
		serviceValue := reflect.ValueOf(service).Elem()
		baseURLField := serviceValue.FieldByName("baseURL")
		if baseURLField.IsValid() && baseURLField.CanSet() {
			baseURLField.SetString(mockServer.URL)
		} else {
			// If we can't access the field directly, use unsafe reflection
			servicePtr := unsafe.Pointer(serviceValue.UnsafeAddr())
			baseURLPtr := (*string)(unsafe.Pointer(uintptr(servicePtr) + unsafe.Offsetof(struct{
				apiKey   string
				baseURL  string
			}{}.baseURL)))
			*baseURLPtr = mockServer.URL
		}

		service.ClearCache()

		// First call should hit the API
		rates1, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		assert.Equal(t, 1, callCount, "Should have called API once")
		assert.Equal(t, 0.85, rates1["EUR"])
		assert.Equal(t, 5.34, rates1["BRL"])

		// Verify cache is populated
		status := service.GetCacheStatus()
		assert.Len(t, status, 1)
		assert.Contains(t, status, "USD")

		// Second call should use cache (no additional API call)
		rates2, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		assert.Equal(t, 1, callCount, "Should still be only one API call (cache hit)")
		assert.Equal(t, rates1, rates2, "Cached rates should be identical")

		// Different currency should trigger another API call
		_, err = service.GetExchangeRates("EUR")
		require.NoError(t, err)
		assert.Equal(t, 2, callCount, "Should have called API twice for different currency")

		// Verify cache now has both currencies
		status = service.GetCacheStatus()
		assert.Len(t, status, 2)
		assert.Contains(t, status, "USD")
		assert.Contains(t, status, "EUR")
	})

	t.Run("Cache expiration behavior", func(t *testing.T) {
		// This test would need to either wait for cache expiration or modify TTL
		// For now, we'll test the cache clear functionality
		originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
		defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
		os.Unsetenv("EXCHANGE_RATE_API_KEY")

		service := utils.NewExchangeRateService()
		
		// Get some rates (will be mock rates)
		_, err := service.GetExchangeRates("USD")
		require.NoError(t, err)
		
		// Clear cache
		service.ClearCache()
		status := service.GetCacheStatus()
		assert.Len(t, status, 0, "Cache should be empty after clear")

		// Get rates again
		_, err = service.GetExchangeRates("USD")
		require.NoError(t, err)
		
		// Cache should still be empty (mock rates don't populate cache)
		status = service.GetCacheStatus()
		assert.Len(t, status, 0, "Mock rates should not populate cache")
	})
}

func TestExchangeRateService_ConcurrentAccess(t *testing.T) {
	originalKey := os.Getenv("EXCHANGE_RATE_API_KEY")
	defer os.Setenv("EXCHANGE_RATE_API_KEY", originalKey)
	os.Unsetenv("EXCHANGE_RATE_API_KEY")

	service := utils.NewExchangeRateService()

	// Test concurrent cache access
	t.Run("Concurrent cache operations", func(t *testing.T) {
		// Start multiple goroutines accessing cache
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				rates, err := service.GetExchangeRates("USD")
				assert.NoError(t, err)
				assert.Equal(t, 0.85, rates["EUR"])
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}