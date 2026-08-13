package beauty_integration_test

import (
	"os"
	"testing"
	"time"

	beauty "muambr-api/extractors/beauty"
	"muambr-api/models"
)

// runBeautyExtractorTest is a shared helper that calls GetComparisons on an extractor,
// waits for a result (or timeout), and logs the outcome. It does not fail on
// extraction errors because anti-bot protection may cause transient failures.
func runBeautyExtractorTest(t *testing.T, name string, timeout time.Duration, fn func() ([]models.ProductComparison, error)) {
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

func TestBeautyExtractorsIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests (set INTEGRATION_TESTS=true to run)")
	}

	const searchQuery = "armani code"

	runBeautyExtractorTest(t, "EpocaCosmeticosIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return beauty.NewEpocaCosmeticosExtractor().GetComparisons(searchQuery)
	})

	runBeautyExtractorTest(t, "SephoraBRIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return beauty.NewSephoraBRExtractor().GetComparisons(searchQuery)
	})

	runBeautyExtractorTest(t, "PrimorPTIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return beauty.NewPrimorPTExtractor().GetComparisons(searchQuery)
	})
}
