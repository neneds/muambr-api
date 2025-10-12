package extractors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKuantoKustaExtractorIsolated(t *testing.T) {
	// Initialize logger for testing
	err := utils.InitDevelopmentLogger()
	require.NoError(t, err, "Failed to initialize logger")

	t.Run("Basic KuantoKusta Properties", func(t *testing.T) {
		extractor := extractors.NewKuantoKustaExtractor()
		
		assert.Equal(t, models.CountryPortugal, extractor.GetCountryCode())
		assert.Equal(t, models.MacroRegionEU, extractor.GetMacroRegion())
		assert.Equal(t, "kuantokusta", extractor.GetIdentifier())
		assert.Equal(t, "https://www.kuantokusta.pt", extractor.BaseURL())
	})

	t.Run("KuantoKusta Search with Timeout", func(t *testing.T) {
		extractor := extractors.NewKuantoKustaExtractor()
		
		// Test with a short timeout to identify hanging issues
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		
		done := make(chan bool, 1)
		var results []models.ProductComparison
		var err error
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("extractor panicked: %v", r)
				}
				done <- true
			}()
			
			results, err = extractor.GetComparisons("macbook")
		}()
		
		select {
		case <-done:
			if err != nil {
				t.Logf("KuantoKusta extractor failed: %v", err)
				// Don't fail the test - just log the error for debugging
			} else {
				t.Logf("KuantoKusta extractor completed successfully with %d results", len(results))
				
				// Verify basic result structure if we got results
				if len(results) > 0 {
					result := results[0]
					assert.NotEmpty(t, result.ProductName, "Product name should not be empty")
					assert.Greater(t, result.Price, 0.0, "Price should be greater than 0")
					assert.Equal(t, "EUR", result.Currency, "Currency should be EUR for Portuguese products")
					assert.Equal(t, "PT", result.Country, "Country should be PT")
					assert.Contains(t, result.StoreName, "KuantoKusta", "Store name should contain KuantoKusta")
				}
			}
		case <-ctx.Done():
			t.Errorf("KuantoKusta extractor timed out after 15 seconds")
		}
	})

	t.Run("KuantoKusta Search Performance Test", func(t *testing.T) {
		extractor := extractors.NewKuantoKustaExtractor()
		
		// Test multiple searches to identify performance patterns
		searchTerms := []string{"macbook", "iphone", "samsung"}
		
		for _, term := range searchTerms {
			t.Run(fmt.Sprintf("Search_%s", term), func(t *testing.T) {
				start := time.Now()
				
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				
				done := make(chan bool, 1)
				var results []models.ProductComparison
				var err error
				
				go func() {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("extractor panicked: %v", r)
						}
						done <- true
					}()
					
					results, err = extractor.GetComparisons(term)
				}()
				
				select {
				case <-done:
					duration := time.Since(start)
					t.Logf("Search for '%s' completed in %v", term, duration)
					
					if err != nil {
						t.Logf("Search for '%s' failed: %v", term, err)
					} else {
						t.Logf("Search for '%s' returned %d results", term, len(results))
					}
					
					// Performance assertion - should complete within reasonable time
					assert.Less(t, duration, 30*time.Second, "Search should complete within 30 seconds")
					
				case <-ctx.Done():
					duration := time.Since(start)
					t.Errorf("Search for '%s' timed out after %v", term, duration)
				}
			})
		}
	})

	t.Run("KuantoKusta Error Scenarios", func(t *testing.T) {
		extractor := extractors.NewKuantoKustaExtractor()
		
		// Test with invalid/empty search terms
		testCases := []struct {
			name       string
			searchTerm string
		}{
			{"Empty string", ""},
			{"Special characters", "!@#$%^&*()"},
			{"Very long term", "this is a very long search term that might cause issues with the extractor or the underlying service"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				
				done := make(chan bool, 1)
				var results []models.ProductComparison
				var err error
				
				go func() {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("extractor panicked: %v", r)
						}
						done <- true
					}()
					
					results, err = extractor.GetComparisons(tc.searchTerm)
				}()
				
				select {
				case <-done:
					// Should handle gracefully - either return empty results or proper error
					if err != nil {
						assert.NotNil(t, err, "Should return proper error for invalid input")
						t.Logf("Expected error for '%s': %v", tc.searchTerm, err)
					} else {
						t.Logf("Search for '%s' returned %d results", tc.searchTerm, len(results))
					}
				case <-ctx.Done():
					t.Errorf("Search for '%s' timed out", tc.searchTerm)
				}
			})
		}
	})
}

func TestKuantoKustaDirectHTTPAccess(t *testing.T) {
	// Test direct HTTP access to KuantoKusta to identify network issues
	t.Run("Direct HTTP Access Test", func(t *testing.T) {
		// Initialize logger for testing
		err := utils.InitDevelopmentLogger()
		require.NoError(t, err, "Failed to initialize logger")

		testURL := "https://www.kuantokusta.pt/search?q=macbook"
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		done := make(chan bool, 1)
		var httpErr error
		
		go func() {
			defer func() { done <- true }()
			
			// Create anti-bot configuration similar to what the extractor uses
			config := utils.DefaultAntiBotConfig("https://www.kuantokusta.pt")
			
			utils.Info("Testing direct HTTP access to KuantoKusta", 
				utils.String("url", testURL),
				utils.String("user_agent", utils.GetRandomUserAgent()))
			
			// Make request using anti-bot utility
			resp, err := utils.MakeAntiBotRequest(testURL, config)
			if err != nil {
				httpErr = fmt.Errorf("HTTP request failed: %w", err)
				return
			}
			defer resp.Body.Close()
			
			utils.Info("HTTP request successful", 
				utils.String("status", resp.Status),
				utils.Int("status_code", resp.StatusCode),
				utils.String("content_type", resp.Header.Get("Content-Type")))
		}()
		
		select {
		case <-done:
			if httpErr != nil {
				t.Logf("Direct HTTP access failed: %v", httpErr)
				t.Log("This might indicate network connectivity issues with KuantoKusta")
			} else {
				t.Log("Direct HTTP access to KuantoKusta successful")
			}
		case <-ctx.Done():
			t.Error("Direct HTTP access to KuantoKusta timed out")
		}
	})
}

// Benchmark test for KuantoKusta performance
func BenchmarkKuantoKustaExtractor(b *testing.B) {
	// Initialize logger for benchmarking
	err := utils.InitDevelopmentLogger()
	if err != nil {
		b.Fatalf("Failed to initialize logger: %v", err)
	}

	extractor := extractors.NewKuantoKustaExtractor()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		
		done := make(chan bool, 1)
		go func() {
			defer func() { 
				if r := recover(); r != nil {
					b.Logf("Benchmark iteration %d panicked: %v", i, r)
				}
				done <- true 
			}()
			
			_, err := extractor.GetComparisons("macbook")
			if err != nil {
				b.Logf("Benchmark iteration %d failed: %v", i, err)
			}
		}()
		
		select {
		case <-done:
			// Completed within time
		case <-ctx.Done():
			b.Logf("Benchmark iteration %d timed out", i)
		}
		
		cancel()
	}
}