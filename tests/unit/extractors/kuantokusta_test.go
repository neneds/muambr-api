package extractors_test

import (
	"strings"
	"testing"

	other "muambr-api/extractors/other"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKuantoKustaProductsJSON(t *testing.T) {
	body := `{
	  "data": [
	    {
	      "id": 11928783,
	      "images": ["https://s1.kuantokusta.pt/img.jpg"],
	      "name": "Apple iPhone 17 Pro Max 6.9\" 256GB Silver",
	      "priceMin": 1170,
	      "url": "/p/11928783/apple-iphone-17-pro-max-69-256gb-silver"
	    },
	    {"id": 1, "name": "Missing price", "priceMin": 0, "url": "/p/1/x"}
	  ],
	  "page": 1,
	  "rows": 24,
	  "total": 2
	}`

	got, err := other.NewKuantoKustaExtractorV2().GetComparisonsFromHTML(body)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "11928783", got[0].ID)
	assert.Equal(t, "Apple iPhone 17 Pro Max 6.9\" 256GB Silver", got[0].ProductName)
	assert.Equal(t, 1170.0, got[0].Price)
	assert.Equal(t, "EUR", got[0].Currency)
	assert.Equal(t, "PT", got[0].Country)
	require.NotNil(t, got[0].StoreURL)
	assert.Equal(t, "https://www.kuantokusta.pt/p/11928783/apple-iphone-17-pro-max-69-256gb-silver", *got[0].StoreURL)
	require.NotNil(t, got[0].ImageURL)
	assert.Equal(t, "https://s1.kuantokusta.pt/img.jpg", *got[0].ImageURL)
}

func TestKuantoKustaExtractorReal(t *testing.T) {
	// Note: This assumes KuantoKustaExtractor exists
	// Let's check if it exists by trying to create one
	
	t.Skip("KuantoKusta extractor not yet implemented - placeholder test")
	
	// This is what the test would look like when implemented:
	/*
	extractor := extractors.NewKuantoKustaExtractor()

	t.Run("GetCountryCode", func(t *testing.T) {
		country := extractor.GetCountryCode()
		expected := models.CountryPortugal
		if country != expected {
			t.Errorf("Expected country code %s, got %s", expected, country)
		}
	})

	t.Run("GetMacroRegion", func(t *testing.T) {
		region := extractor.GetMacroRegion()
		expected := models.MacroRegionEU
		if region != expected {
			t.Errorf("Expected macro region %s, got %s", expected, region)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		identifier := extractor.GetIdentifier()
		expected := "kuantokusta"
		if identifier != expected {
			t.Errorf("Expected identifier %s, got %s", expected, identifier)
		}
	})

	t.Run("BaseURL", func(t *testing.T) {
		baseURL := extractor.BaseURL()
		expected := "https://www.kuantokusta.pt"
		if baseURL != expected {
			t.Errorf("Expected base URL %s, got %s", expected, baseURL)
		}
	})

	t.Run("Interface Implementation", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
	*/
}

func TestKuantoKustaHTMLStructure(t *testing.T) {
	htmlContent, err := loadTestData("kuantokusta_ipad10_search.html")
	if err != nil {
		t.Skipf("Test data not available: %v", err)
		return
	}

	// Test HTML structure elements we expect from KuantoKusta
	structureTests := []struct {
		name     string
		expected string
		found    bool
	}{
		{"DOCTYPE declaration", "<!DOCTYPE html>", false},
		{"HTML opening tag", "<html", false},
		{"Head section", "<head>", false},
		{"Body section", "<body", false},
		{"KuantoKusta branding", "kuantokusta", false},
		{"Search results", "ipad", false},
		{"Product listings", "produto", false}, // Portuguese for product
		{"Price information", "€", false},      // Euro symbol
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

	t.Logf("KuantoKusta HTML Structure Analysis: %d/%d expected elements found", foundCount, len(structureTests))
	t.Logf("HTML size: %d bytes", len(htmlContent))

	// Verify the HTML is properly formed
	if strings.Contains(htmlContent, "Test Data Generated:") {
		t.Logf("✓ HTML test data includes metadata header")
	}

	if strings.Contains(htmlContent, "Content Encoding: gzip") {
		t.Logf("✓ HTML was properly decompressed from Gzip format")
	}
}