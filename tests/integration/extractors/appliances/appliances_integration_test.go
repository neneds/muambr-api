package appliances_integration_test

import (
	"os"
	"testing"
	"time"

	appliances "muambr-api/extractors/appliances"
	"muambr-api/models"
)

// runAppliancesExtractorTest is a shared helper that calls GetComparisons on an
// extractor, waits for a result (or timeout), and logs the outcome. It does not
// fail on extraction errors because anti-bot protection may cause transient failures.
func runAppliancesExtractorTest(t *testing.T, name string, timeout time.Duration, fn func() ([]models.ProductComparison, error)) {
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

func TestAppliancesExtractorsIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests (set INTEGRATION_TESTS=true to run)")
	}

	const searchQuery = "air fryer 12l"

	runAppliancesExtractorTest(t, "CarrefourBRIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return appliances.NewCarrefourBRExtractor().GetComparisons(searchQuery)
	})
	runAppliancesExtractorTest(t, "FastshopBRIntegration", 30*time.Second, func() ([]models.ProductComparison, error) {
		return appliances.NewFastshopBRExtractor().GetComparisons(searchQuery)
	})
}
