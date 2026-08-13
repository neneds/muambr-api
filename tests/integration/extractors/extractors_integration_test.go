package extractors_test

import (
	"os"
	"testing"
	"time"

	other "muambr-api/extractors/other"
	"muambr-api/models"
)

// Ensure models is used
var _ models.ProductComparison

// TestExtractorIntegration tests the actual integration with external websites
// These tests require internet connectivity and may be slow
func TestExtractorIntegration(t *testing.T) {
	// Skip integration tests if not in integration mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests (set INTEGRATION_TESTS=true to run)")
	}

	t.Run("AcharPromoIntegration", func(t *testing.T) {
		extractor := other.NewAcharPromoExtractorV2()
		
		// Chat API uses AI reasoning, needs more time
		timeout := 90 * time.Second
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

	t.Run("KuantoKustaIntegration", func(t *testing.T) {
		extractor := other.NewKuantoKustaExtractorV2()
		
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

	t.Run("WalmartUSAIntegration", func(t *testing.T) {
		extractor := other.NewWalmartUSAExtractor()
		
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
				t.Logf("Walmart USA extraction failed (this may be expected due to anti-bot protection): %v", err)
			} else {
				t.Logf("Walmart USA extraction succeeded with %d results", len(results))
			}
		case <-time.After(timeout):
			t.Errorf("Walmart USA extraction timed out after %v", timeout)
		}
	})
}

