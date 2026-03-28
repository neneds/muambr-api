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

func TestAcharPromoExtractorFromHTML(t *testing.T) {
	extractor := extractors.NewAcharPromoExtractorV2()

	html, err := loadSampleResponse("acharpromo_deals.html")
	if err != nil {
		t.Fatalf("Failed to load sample response: %v", err)
	}

	t.Run("ExtractsAllDeals", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) != 5 {
			t.Errorf("Expected 5 deals, got %d", len(comparisons))
		}
	})

	t.Run("ParsesProductFields", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) == 0 {
			t.Fatal("No comparisons returned")
		}

		// Check first product: Micro-ondas Electrolux
		c := comparisons[0]
		if c.ProductName != "Micro-ondas Electrolux Branco 23L Efficient" {
			t.Errorf("Expected product name 'Micro-ondas Electrolux Branco 23L Efficient', got %q", c.ProductName)
		}
		if c.Price != 502 {
			t.Errorf("Expected price 502, got %f", c.Price)
		}
		if c.Currency != "BRL" {
			t.Errorf("Expected currency BRL, got %s", c.Currency)
		}
		if c.StoreName != "Mercado Livre" {
			t.Errorf("Expected store name 'Mercado Livre', got %s", c.StoreName)
		}
		if c.Country != "BR" {
			t.Errorf("Expected country BR, got %s", c.Country)
		}
		if c.StoreURL == nil || *c.StoreURL != "https://meli.la/2aZJVVx" {
			t.Errorf("Expected store URL https://meli.la/2aZJVVx, got %v", c.StoreURL)
		}
		if c.ImageURL == nil || *c.ImageURL != "https://http2.mlstatic.com/D_NQ_NP_742060-MLA99449598594_112025-O.webp" {
			t.Errorf("Unexpected image URL: %v", c.ImageURL)
		}
		if c.ID == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("ParsesBrazilianPriceFormats", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}

		// Product with comma decimal: "125,10"
		if comparisons[1].Price != 125.10 {
			t.Errorf("Expected price 125.10, got %f", comparisons[1].Price)
		}

		// Product with dot thousand sep and comma decimal: "2.199,00"
		if comparisons[2].Price != 2199.00 {
			t.Errorf("Expected price 2199.00, got %f", comparisons[2].Price)
		}

		// Product with dot thousand sep and comma decimal: "4.999,00"
		if comparisons[3].Price != 4999.00 {
			t.Errorf("Expected price 4999.00, got %f", comparisons[3].Price)
		}
	})

	t.Run("HandlesEmptyHTML", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML("<html></html>")
		if err != nil {
			t.Fatalf("Expected no error for empty HTML, got: %v", err)
		}
		if len(comparisons) != 0 {
			t.Errorf("Expected 0 comparisons for empty HTML, got %d", len(comparisons))
		}
	})
}

func TestAcharPromoSearchFiltering(t *testing.T) {
	extractor := extractors.NewAcharPromoExtractorV2()

	html, err := loadSampleResponse("acharpromo_deals.html")
	if err != nil {
		t.Fatalf("Failed to load sample response: %v", err)
	}

	// GetComparisonsFromHTML returns ALL deals (no filtering)
	allDeals, err := extractor.GetComparisonsFromHTML(html)
	if err != nil {
		t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
	}

	t.Run("GetComparisonsFromHTML returns all deals unfiltered", func(t *testing.T) {
		if len(allDeals) != 5 {
			t.Errorf("Expected 5 total deals, got %d", len(allDeals))
		}
	})
}