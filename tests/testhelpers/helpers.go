package testhelpers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"muambr-api/models"
)

// TestDataDir returns the path to the testdata directory
func TestDataDir() string {
	// Get the current working directory and navigate to testdata
	wd, _ := os.Getwd()
	// Go up until we find the tests directory
	for {
		if _, err := os.Stat(filepath.Join(wd, "tests")); err == nil {
			return filepath.Join(wd, "tests", "testdata")
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			// Reached root, fallback to relative path
			return "../../testdata"
		}
		wd = parent
	}
}

// LoadHTMLTestData loads HTML test data from the testdata/html directory
func LoadHTMLTestData(t *testing.T, filename string) string {
	t.Helper()
	
	filePath := filepath.Join(TestDataDir(), "html", filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load HTML test data %s: %v", filename, err)
	}
	
	return string(content)
}

// LoadJSONTestData loads and parses JSON test data from the testdata/json directory
func LoadJSONTestData(t *testing.T, filename string, target interface{}) {
	t.Helper()
	
	filePath := filepath.Join(TestDataDir(), "json", filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load JSON test data %s: %v", filename, err)
	}
	
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatalf("Failed to parse JSON test data %s: %v", filename, err)
	}
}

// CreateTempHTMLFile creates a temporary HTML file for testing
func CreateTempHTMLFile(t *testing.T, content string) string {
	t.Helper()
	
	tempFile, err := ioutil.TempFile("", "test_*.html")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	
	tempFile.Close()
	
	// Register cleanup
	t.Cleanup(func() {
		os.Remove(tempFile.Name())
	})
	
	return tempFile.Name()
}

// AssertProductComparison asserts that a ProductComparison has expected values
func AssertProductComparison(t *testing.T, actual models.ProductComparison, expected ProductComparisonMatcher) {
	t.Helper()
	
	if expected.ID != nil && actual.ID != *expected.ID {
		t.Errorf("Expected ID %s, got %s", *expected.ID, actual.ID)
	}
	
	if expected.ProductName != nil && actual.ProductName != *expected.ProductName {
		t.Errorf("Expected ProductName %s, got %s", *expected.ProductName, actual.ProductName)
	}
	
	if expected.Price != nil && actual.Price != *expected.Price {
		t.Errorf("Expected Price %f, got %f", *expected.Price, actual.Price)
	}
	
	if expected.Currency != nil && actual.Currency != *expected.Currency {
		t.Errorf("Expected Currency %s, got %s", *expected.Currency, actual.Currency)
	}
	
	if expected.StoreName != nil && actual.StoreName != *expected.StoreName {
		t.Errorf("Expected StoreName %s, got %s", *expected.StoreName, actual.StoreName)
	}
	
	if expected.Country != nil && actual.Country != *expected.Country {
		t.Errorf("Expected Country %s, got %s", *expected.Country, actual.Country)
	}
}

// ProductComparisonMatcher provides flexible matching for ProductComparison
type ProductComparisonMatcher struct {
	ID          *string
	ProductName *string
	Price       *float64
	Currency    *string
	StoreName   *string
	Country     *string
}

// StringPtr returns a pointer to a string (helper for test setup)
func StringPtr(s string) *string {
	return &s
}

// Float64Ptr returns a pointer to a float64 (helper for test setup)
func Float64Ptr(f float64) *float64 {
	return &f
}

// SkipIfIntegrationDisabled skips the test if integration tests are disabled
func SkipIfIntegrationDisabled(t *testing.T) {
	t.Helper()
	
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test (set INTEGRATION_TESTS=true to run)")
	}
}

// WithTestTimeout runs a function with a timeout, useful for integration tests
func WithTestTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	
	done := make(chan bool, 1)
	go func() {
		fn()
		done <- true
	}()
	
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(timeout):
		t.Fatalf("Test timed out after %v", timeout)
	}
}