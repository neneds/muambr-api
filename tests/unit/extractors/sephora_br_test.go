package extractors_test

import (
	"os"
	"path/filepath"
	"testing"

	"muambr-api/extractors"
	beauty "muambr-api/extractors/beauty"
	"muambr-api/models"
)

func loadSephoraBRFixture(filename string) (string, error) {
	path := filepath.Join("..", "..", "mocks", "extractors", "sample_responses", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestSephoraBRExtractor(t *testing.T) {
	extractor := beauty.NewSephoraBRExtractor()

	t.Run("GetCountryCode", func(t *testing.T) {
		if got := extractor.GetCountryCode(); got != models.CountryBrazil {
			t.Errorf("expected %s, got %s", models.CountryBrazil, got)
		}
	})

	t.Run("GetMacroRegion", func(t *testing.T) {
		if got := extractor.GetMacroRegion(); got != models.MacroRegionLATAM {
			t.Errorf("expected %s, got %s", models.MacroRegionLATAM, got)
		}
	})

	t.Run("GetCategory", func(t *testing.T) {
		if got := extractor.GetCategory(); got != models.CategoryBeauty {
			t.Errorf("expected %s, got %s", models.CategoryBeauty, got)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		if got := extractor.GetIdentifier(); got != "sephora_br_v1" {
			t.Errorf("expected sephora_br_v1, got %s", got)
		}
	})

	t.Run("BaseURL", func(t *testing.T) {
		if got := extractor.BaseURL(); got != "https://www.sephora.com.br" {
			t.Errorf("expected https://www.sephora.com.br, got %s", got)
		}
	})

	t.Run("InterfaceImplementation", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
}

func TestSephoraBRGetComparisonsFromHTML(t *testing.T) {
	extractor := beauty.NewSephoraBRExtractor()

	html, err := loadSephoraBRFixture("sephora_br_armani_code.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	t.Run("ExtractsExpectedCount", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) != 7 {
			t.Errorf("expected 7 products, got %d", len(comparisons))
		}
	})

	t.Run("ParsesFirstProductFields", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) == 0 {
			t.Fatal("no comparisons returned")
		}

		c := comparisons[0]

		if c.ProductName != "Perfume Armani Code Masculino Eau de Parfum" {
			t.Errorf("unexpected ProductName: %q", c.ProductName)
		}
		if c.Price != 1079 {
			t.Errorf("expected price 1079, got %f", c.Price)
		}
		if c.Currency != "BRL" {
			t.Errorf("expected currency BRL, got %s", c.Currency)
		}
		if c.StoreName != "Sephora" {
			t.Errorf("expected store Sephora, got %s", c.StoreName)
		}
		if c.Country != "BR" {
			t.Errorf("expected country BR, got %s", c.Country)
		}
		if c.Category == nil || *c.Category != models.CategoryBeauty {
			t.Errorf("expected category beauty")
		}
		if c.StoreURL == nil {
			t.Error("expected non-nil StoreURL")
		} else if *c.StoreURL != "https://www.sephora.com.br/perfume-armani-code-masculino-eau-de-parfum-9090711656-9090711656.html" {
			t.Errorf("unexpected StoreURL: %s", *c.StoreURL)
		}
		if c.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("ReturnsErrorOnEmptyHTML", func(t *testing.T) {
		_, err := extractor.GetComparisonsFromHTML("")
		if err == nil {
			t.Error("expected error for empty HTML, got nil")
		}
	})

	t.Run("ReturnsErrorOnMissingDataLayer", func(t *testing.T) {
		_, err := extractor.GetComparisonsFromHTML("<html><body>no products here</body></html>")
		if err == nil {
			t.Error("expected error when dataLayer is absent, got nil")
		}
	})
}
