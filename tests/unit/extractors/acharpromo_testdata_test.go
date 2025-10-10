package extractors_test

import (
	"testing"

	"muambr-api/tests/testhelpers"
)

func TestAcharPromoExtractorWithTestData(t *testing.T) {
	// Example of how to use test helpers for more comprehensive testing
	
	t.Run("LoadSampleHTML", func(t *testing.T) {
		// Load sample HTML from testdata
		htmlContent := testhelpers.LoadHTMLTestData(t, "acharpromo_sample.html")
		
		if len(htmlContent) == 0 {
			t.Error("Expected non-empty HTML content")
		}
		
		// Verify the HTML contains expected elements
		if !containsString(htmlContent, "iPhone 15 Pro") {
			t.Error("Expected HTML to contain iPhone 15 Pro")
		}
		
		if !containsString(htmlContent, "R$ 8.999,00") {
			t.Error("Expected HTML to contain price R$ 8.999,00")
		}
	})

	t.Run("LoadSampleJSONResponse", func(t *testing.T) {
		// Load and parse JSON response
		var response struct {
			Products []struct {
				Name      string  `json:"name"`
				PriceText string  `json:"price_text"`
				Price     float64 `json:"price"`
				StoreURL  string  `json:"store_url"`
				ImageURL  string  `json:"image_url"`
			} `json:"products"`
			Total  int    `json:"total"`
			Source string `json:"source"`
		}
		
		testhelpers.LoadJSONTestData(t, "acharpromo_sample_response.json", &response)
		
		// Verify the response structure
		if response.Total != 3 {
			t.Errorf("Expected total 3, got %d", response.Total)
		}
		
		if response.Source != "achar.promo" {
			t.Errorf("Expected source 'achar.promo', got %s", response.Source)
		}
		
		if len(response.Products) != 3 {
			t.Errorf("Expected 3 products, got %d", len(response.Products))
		}
		
		// Verify first product
		firstProduct := response.Products[0]
		if firstProduct.Name != "iPhone 15 Pro 256GB" {
			t.Errorf("Expected first product name 'iPhone 15 Pro 256GB', got %s", firstProduct.Name)
		}
		
		if firstProduct.Price != 8999.00 {
			t.Errorf("Expected first product price 8999.00, got %f", firstProduct.Price)
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && 
	       (haystack == needle || 
	        haystack[:len(needle)] == needle ||
	        haystack[len(haystack)-len(needle):] == needle ||
	        containsStringInMiddle(haystack, needle))
}

func containsStringInMiddle(haystack, needle string) bool {
	for i := 1; i < len(haystack)-len(needle)+1; i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}