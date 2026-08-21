package electronics_integration_test

import (
	"os"
	"testing"
	"time"

	electronics "muambr-api/extractors/electronics"
	"muambr-api/models"
)

func runElectronicsExtractorTest(t *testing.T, name string, timeout time.Duration, fn func() ([]models.ProductComparison, error)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		done := make(chan struct{}, 1)
		var results []models.ProductComparison
		var err error

		go func() {
			results, err = fn()
			close(done)
		}()

		select {
		case <-done:
			if err != nil {
				t.Logf("%s extraction failed (may be transient/anti-bot): %v", name, err)
			} else {
				t.Logf("%s extraction succeeded with %d results", name, len(results))
			}
		case <-time.After(timeout):
			t.Errorf("%s extraction timed out after %v", name, timeout)
		}
	})
}

func TestElectronicsExtractorsIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests (set INTEGRATION_TESTS=true to run)")
	}

	const searchQuery = "iphone"

	runElectronicsExtractorTest(t, "ParadigitNLIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return electronics.NewParadigitNLExtractor().GetComparisons(searchQuery)
	})
	runElectronicsExtractorTest(t, "AlternateNLIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return electronics.NewAlternateNLExtractor().GetComparisons(searchQuery)
	})
}
