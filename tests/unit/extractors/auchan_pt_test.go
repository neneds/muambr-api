package extractors_test

import (
	"os"
	"path/filepath"
	"testing"

	"muambr-api/extractors"
	other "muambr-api/extractors/other"
	"muambr-api/models"
)

func loadAuchanPTFixture(filename string) (string, error) {
	path := filepath.Join("..", "..", "mocks", "extractors", "sample_responses", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestAuchanPTExtractor(t *testing.T) {
	extractor := other.NewAuchanPTExtractor()

	t.Run("GetCountryCode", func(t *testing.T) {
		if got := extractor.GetCountryCode(); got != models.CountryPortugal {
			t.Errorf("expected %s, got %s", models.CountryPortugal, got)
		}
	})

	t.Run("GetMacroRegion", func(t *testing.T) {
		if got := extractor.GetMacroRegion(); got != models.MacroRegionEU {
			t.Errorf("expected %s, got %s", models.MacroRegionEU, got)
		}
	})

	t.Run("GetCategory", func(t *testing.T) {
		if got := extractor.GetCategory(); got != models.CategoryOther {
			t.Errorf("expected %s, got %s", models.CategoryOther, got)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		if got := extractor.GetIdentifier(); got != "auchan_pt_v1" {
			t.Errorf("expected auchan_pt_v1, got %s", got)
		}
	})

	t.Run("BaseURL", func(t *testing.T) {
		if got := extractor.BaseURL(); got != "https://www.auchan.pt" {
			t.Errorf("expected https://www.auchan.pt, got %s", got)
		}
	})

	t.Run("InterfaceImplementation", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
}

func TestAuchanPTGetComparisonsFromHTML(t *testing.T) {
	extractor := other.NewAuchanPTExtractor()

	html, err := loadAuchanPTFixture("auchan_pt_leite.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	t.Run("DedupesAndSkipsZeroPrice", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) != 2 {
			t.Fatalf("expected 2 products, got %d", len(comparisons))
		}
	})

	t.Run("ParsesStringPriceAndAbsoluteURL", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) == 0 {
			t.Fatal("no comparisons returned")
		}

		c := comparisons[0]
		if c.ProductName != "LEITE UHT AUCHAN MEIO GORDO 1L" {
			t.Errorf("unexpected ProductName: %q", c.ProductName)
		}
		if c.Price != 0.86 {
			t.Errorf("expected price 0.86, got %f", c.Price)
		}
		if c.Currency != "EUR" {
			t.Errorf("expected currency EUR, got %s", c.Currency)
		}
		if c.StoreName != "Auchan" {
			t.Errorf("expected store Auchan, got %s", c.StoreName)
		}
		if c.Country != "PT" {
			t.Errorf("expected country PT, got %s", c.Country)
		}
		if c.Category == nil || *c.Category != models.CategoryOther {
			t.Errorf("expected category other")
		}
		if c.ID != "3010403" {
			t.Errorf("expected id 3010403, got %s", c.ID)
		}
		wantURL := "https://www.auchan.pt/pt/alimentacao/produtos-lacteos/leites/leite-uht/leite-uht-auchan-meio-gordo-1l/3010403.html"
		if c.StoreURL == nil {
			t.Error("expected non-nil StoreURL")
		} else if *c.StoreURL != wantURL {
			t.Errorf("unexpected StoreURL: %s", *c.StoreURL)
		}
	})

	t.Run("FallsBackToRelativeProductURL", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) < 2 {
			t.Fatal("expected a second product")
		}
		c := comparisons[1]
		if c.ProductName != "LEITE MIMOSA UHT MEIO GORDO 1L" {
			t.Errorf("unexpected ProductName: %q", c.ProductName)
		}
		if c.Price != 1 {
			t.Errorf("expected numeric price 1.00, got %f", c.Price)
		}
		wantURL := "https://www.auchan.pt/pt/alimentacao/produtos-lacteos/leites/leite-uht/leite-mimosa-uht-meio-gordo-1l/11885.html"
		if c.StoreURL == nil {
			t.Error("expected non-nil StoreURL")
		} else if *c.StoreURL != wantURL {
			t.Errorf("unexpected StoreURL: %s", *c.StoreURL)
		}
	})

	t.Run("ReturnsEmptyOnMissingTiles", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML("<html><body>no products here</body></html>")
		if err != nil {
			t.Fatalf("expected empty result, got error: %v", err)
		}
		if len(comparisons) != 0 {
			t.Errorf("expected 0 products, got %d", len(comparisons))
		}
	})
}
