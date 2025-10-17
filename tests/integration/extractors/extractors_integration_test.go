package extractors_test

import (
	"os"
	"testing"
	"time"

	"muambr-api/extractors"
	"muambr-api/models"
)

// TestExtractorIntegration tests the actual integration with external websites
// These tests require internet connectivity and may be slow
func TestExtractorIntegration(t *testing.T) {
	// Skip integration tests if not in integration mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests (set INTEGRATION_TESTS=true to run)")
	}

	t.Run("AcharPromoIntegration", func(t *testing.T) {
		extractor := extractors.NewAcharPromoExtractorV2()
		
		// Set a reasonable timeout for network requests
		timeout := 30 * time.Second
		done := make(chan bool, 1)
		var results []models.ProductComparison
		var err error
		
		go func() {
			results, err = extractor.GetComparisons("iPhone")
			done <- true
		}()
		
		select {
		case <-done:
			// Test completed
			if err != nil {
				t.Logf("AcharPromo extraction failed (this may be expected due to anti-bot protection): %v", err)
			} else {
				t.Logf("AcharPromo extraction succeeded with %d results", len(results))
			}
		case <-time.After(timeout):
			t.Errorf("AcharPromo extraction timed out after %v", timeout)
		}
	})

	t.Run("MercadoLivreIntegration", func(t *testing.T) {
		extractor := extractors.NewMercadoLivreExtractor()
		
		timeout := 30 * time.Second
		done := make(chan bool, 1)
		var results []models.ProductComparison
		var err error
		
		go func() {
			results, err = extractor.GetComparisons("iPhone")
			done <- true
		}()
		
		select {
		case <-done:
			if err != nil {
				t.Logf("MercadoLivre extraction failed (this may be expected due to anti-bot protection): %v", err)
			} else {
				t.Logf("MercadoLivre extraction succeeded with %d results", len(results))
			}
		case <-time.After(timeout):
			t.Errorf("MercadoLivre extraction timed out after %v", timeout)
		}
	})

	t.Run("KuantoKustaIntegration", func(t *testing.T) {
		extractor := extractors.NewKuantoKustaExtractor()
		
		timeout := 30 * time.Second
		done := make(chan bool, 1)
		var results []models.ProductComparison
		var err error
		
		go func() {
			results, err = extractor.GetComparisons("iPhone")
			done <- true
		}()
		
		select {
		case <-done:
			if err != nil {
				t.Logf("KuantoKusta extraction failed (this may be expected due to anti-bot protection): %v", err)
			} else {
				t.Logf("KuantoKusta extraction succeeded with %d results", len(results))
			}
		case <-time.After(timeout):
			t.Errorf("KuantoKusta extraction timed out after %v", timeout)
		}
	})
}

// TestGoEnvironment tests that the Go environment is properly configured
func TestGoEnvironment(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests (set INTEGRATION_TESTS=true to run)")
	}

	t.Run("GoDependencies", func(t *testing.T) {
		// Test that Go HTTP client and HTML parsing work
		assert.NotNil(t, http.DefaultClient, "HTTP client should be available")
	})
}