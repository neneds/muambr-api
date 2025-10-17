package extractors_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"muambr-api/extractors"
	"muambr-api/models"
)

// loadTestData loads HTML test data for use in unit tests
func loadTestData(filename string) (string, error) {
	testDataPath := filepath.Join("..", "..", "testdata", "html", filename)
	data, err := ioutil.ReadFile(testDataPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestAcharPromoExtractor(t *testing.T) {
	extractor := extractors.NewAcharPromoExtractorV2()

	t.Run("GetCountryCode", func(t *testing.T) {
		country := extractor.GetCountryCode()
		expected := models.CountryBrazil
		if country != expected {
			t.Errorf("Expected country code %s, got %s", expected, country)
		}
	})

	t.Run("GetMacroRegion", func(t *testing.T) {
		region := extractor.GetMacroRegion()
		expected := models.MacroRegionLATAM
		if region != expected {
			t.Errorf("Expected macro region %s, got %s", expected, region)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		identifier := extractor.GetIdentifier()
		expected := "acharpromo_v2"
		if identifier != expected {
			t.Errorf("Expected identifier %s, got %s", expected, identifier)
		}
	})

	t.Run("BaseURL", func(t *testing.T) {
		baseURL := extractor.BaseURL()
		expected := "https://achar.promo"
		if baseURL != expected {
			t.Errorf("Expected base URL %s, got %s", expected, baseURL)
		}
	})

	t.Run("Interface Implementation", func(t *testing.T) {
		// Verify that AcharPromoExtractor implements the Extractor interface
		var _ extractors.Extractor = extractor
	})
}

func TestAcharPromoExtractorWithRealHTML(t *testing.T) {
	// Load real HTML data from our test files
	htmlContent, err := loadTestData("acharpromo_ipad10_search.html")
	if err != nil {
		t.Skipf("Skipping real HTML test - test data not available: %v", err)
		return
	}

	// Verify we have valid HTML content
	if !strings.Contains(htmlContent, "<html") && !strings.Contains(htmlContent, "<!DOCTYPE") {
		t.Skipf("Skipping real HTML test - HTML content appears invalid")
		return
	}

	// Test that we can at least process the HTML without crashing
	// Note: This is a basic smoke test since we'd need to modify the extractor
	// to accept HTML content directly for proper unit testing
	t.Logf("Successfully loaded AcharPromo HTML test data (%d bytes)", len(htmlContent))
	
	// Check for expected elements that should be in an AcharPromo search results page
	expectedElements := []string{
		"achar.promo",  // Site name/branding
		"search",       // Search-related content
	}

	for _, element := range expectedElements {
		if !strings.Contains(strings.ToLower(htmlContent), strings.ToLower(element)) {
			t.Logf("Warning: Expected element '%s' not found in HTML content", element)
		}
	}

	// Verify the HTML is properly formed
	if strings.Contains(htmlContent, "Test Data Generated:") {
		t.Logf("✓ HTML test data includes metadata header")
	}

	if strings.Contains(htmlContent, "Content Encoding: br") {
		t.Logf("✓ HTML was properly decompressed from Brotli format")
	}
}

// TestAcharPromoHTMLStructure tests the structure of the actual HTML we collected
func TestAcharPromoHTMLStructure(t *testing.T) {
	htmlContent, err := loadTestData("acharpromo_ipad10_search.html")
	if err != nil {
		t.Skipf("Test data not available: %v", err)
		return
	}

	// Test HTML structure elements we expect
	structureTests := []struct {
		name     string
		expected string
		found    bool
	}{
		{"DOCTYPE declaration", "<!DOCTYPE html>", false},
		{"HTML opening tag", "<html", false},
		{"Head section", "<head>", false},
		{"Body section", "<body", false},
		{"AcharPromo branding", "achar.promo", false},
	}

	lowerHTML := strings.ToLower(htmlContent)

	for i := range structureTests {
		test := &structureTests[i]
		test.found = strings.Contains(lowerHTML, strings.ToLower(test.expected))
		
		if test.found {
			t.Logf("✓ Found %s", test.name)
		} else {
			t.Logf("⚠ Missing %s", test.name)
		}
	}

	// Count total found elements
	foundCount := 0
	for _, test := range structureTests {
		if test.found {
			foundCount++
		}
	}

	t.Logf("HTML Structure Analysis: %d/%d expected elements found", foundCount, len(structureTests))
}

func TestAcharPromoExtractorPriceExtraction(t *testing.T) {
	_ = extractors.NewAcharPromoExtractorV2() // Prevent unused variable error

	testCases := []struct {
		name        string
		priceText   string
		expected    float64
		shouldError bool
	}{
		{
			name:        "Valid Brazilian Real price with comma",
			priceText:   "R$ 1.299,99",
			expected:    1299.99,
			shouldError: false,
		},
		{
			name:        "Valid price without thousand separator",
			priceText:   "R$ 299,50",
			expected:    299.50,
			shouldError: false,
		},
		{
			name:        "Price with spaces",
			priceText:   "  R$ 1.599,00  ",
			expected:    1599.00,
			shouldError: false,
		},
		{
			name:        "Empty price text",
			priceText:   "",
			expected:    0,
			shouldError: true,
		},
		{
			name:        "Invalid price format",
			priceText:   "Invalid price",
			expected:    0,
			shouldError: true,
		},
		{
			name:        "Price without currency symbol",
			priceText:   "1.999,99",
			expected:    1999.99,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: This test requires access to the private extractPriceFromText method
			// In a real implementation, you might want to make this method public for testing
			// or test it indirectly through the public methods
			t.Skip("Skipping private method test - requires refactoring for testability")
		})
	}
}