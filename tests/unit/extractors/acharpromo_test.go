package extractors_test

import (
	"os"
	"path/filepath"
	"testing"

	"muambr-api/extractors"
	"muambr-api/models"
)

// loadTestData loads HTML test data for use in unit tests
func loadTestData(filename string) (string, error) {
	testDataPath := filepath.Join("..", "..", "testdata", "html", filename)
	data, err := os.ReadFile(testDataPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loadSampleResponse loads an HTML sample response for unit tests
func loadSampleResponse(filename string) (string, error) {
	testDataPath := filepath.Join("..", "..", "mocks", "extractors", "sample_responses", filename)
	data, err := os.ReadFile(testDataPath)
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
		var _ extractors.Extractor = extractor
	})
}

func TestAcharPromoExtractorFromSSE(t *testing.T) {
	extractor := extractors.NewAcharPromoExtractorV2()

	sseData, err := loadSampleResponse("acharpromo_chat.txt")
	if err != nil {
		t.Fatalf("Failed to load sample response: %v", err)
	}

	t.Run("ExtractsAllProducts", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(sseData)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) != 5 {
			t.Errorf("Expected 5 products, got %d", len(comparisons))
		}
	})

	t.Run("ParsesProductFields", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(sseData)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) == 0 {
			t.Fatal("No comparisons returned")
		}

		c := comparisons[0]
		if c.ProductName != "Apple iPhone 15 128GB Preto" {
			t.Errorf("Expected product name 'Apple iPhone 15 128GB Preto', got %q", c.ProductName)
		}
		if c.Price != 4199 {
			t.Errorf("Expected price 4199, got %f", c.Price)
		}
		if c.Currency != "BRL" {
			t.Errorf("Expected currency BRL, got %s", c.Currency)
		}
		if c.StoreName != "Magazine Luiza" {
			t.Errorf("Expected store name 'Magazine Luiza', got %s", c.StoreName)
		}
		if c.Country != "BR" {
			t.Errorf("Expected country BR, got %s", c.Country)
		}
		if c.StoreURL == nil {
			t.Error("Expected non-nil StoreURL")
		} else if *c.StoreURL != "https://example.com/store/iphone15" {
			t.Errorf("Expected store URL https://example.com/store/iphone15, got %s", *c.StoreURL)
		}
		if c.ImageURL == nil {
			t.Error("Expected non-nil ImageURL")
		} else if *c.ImageURL != "https://example.com/iphone15.jpg" {
			t.Errorf("Unexpected image URL: %s", *c.ImageURL)
		}
		if c.ID == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("ParsesExtractedPrices", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(sseData)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}

		expectedPrices := []float64{4199, 5499, 3899.9, 5219.1, 2800}
		for i, expected := range expectedPrices {
			if i >= len(comparisons) {
				break
			}
			if comparisons[i].Price != expected {
				t.Errorf("Product %d: expected price %f, got %f", i, expected, comparisons[i].Price)
			}
		}
	})

	t.Run("ParsesStoreNames", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(sseData)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}

		expectedStores := []string{"Magazine Luiza", "Mercado Livre", "Shopee", "Amazon", "OLX"}
		for i, expected := range expectedStores {
			if i >= len(comparisons) {
				break
			}
			if comparisons[i].StoreName != expected {
				t.Errorf("Product %d: expected store %q, got %q", i, expected, comparisons[i].StoreName)
			}
		}
	})

	t.Run("HandlesEmptySSEResponse", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML("")
		if err != nil {
			t.Fatalf("Expected no error for empty response, got: %v", err)
		}
		if len(comparisons) != 0 {
			t.Errorf("Expected 0 comparisons for empty response, got %d", len(comparisons))
		}
	})

	t.Run("HandlesSSEWithNoToolOutput", func(t *testing.T) {
		noProducts := `data: {"type":"start"}
data: {"type":"text-delta","textDelta":"Hello"}
data: {"type":"finish","finishReason":"stop"}
`
		comparisons, err := extractor.GetComparisonsFromHTML(noProducts)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(comparisons) != 0 {
			t.Errorf("Expected 0 comparisons, got %d", len(comparisons))
		}
	})

	t.Run("SkipsProductsWithZeroPrice", func(t *testing.T) {
		sseWithBadProduct := `data: {"type":"tool-output-available","toolCallId":"call_1","output":{"status":"success","searchId":"s1","query":"test","category":"electronics","products":[{"id":"1","title":"Good Product","price":"R$ 100,00","extracted_price":100,"image":"","url":"","source":"Store","product_id":"1","isRecommended":false},{"id":"2","title":"Bad Product","price":"","extracted_price":0,"image":"","url":"","source":"Store","product_id":"2","isRecommended":false}],"metadata":{"totalProducts":2,"totalShops":1,"topShops":["Store"],"minPrice":0}}}
`
		comparisons, err := extractor.GetComparisonsFromHTML(sseWithBadProduct)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(comparisons) != 1 {
			t.Errorf("Expected 1 comparison (skipping zero-price), got %d", len(comparisons))
		}
	})
}