package extractors_test

import (
	"os"
	"path/filepath"
	"testing"

	"muambr-api/extractors"
	beauty "muambr-api/extractors/beauty"
	"muambr-api/models"
)

func loadWellsPTFixture(filename string) (string, error) {
	path := filepath.Join("..", "..", "mocks", "extractors", "sample_responses", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestWellsPTExtractor(t *testing.T) {
	extractor := beauty.NewWellsPTExtractor()

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
		if got := extractor.GetCategory(); got != models.CategoryBeauty {
			t.Errorf("expected %s, got %s", models.CategoryBeauty, got)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		if got := extractor.GetIdentifier(); got != "wells_pt_v1" {
			t.Errorf("expected wells_pt_v1, got %s", got)
		}
	})

	t.Run("BaseURL", func(t *testing.T) {
		if got := extractor.BaseURL(); got != "https://wells.pt" {
			t.Errorf("expected https://wells.pt, got %s", got)
		}
	})

	t.Run("InterfaceImplementation", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
}

func TestWellsPTGetComparisonsFromHTML(t *testing.T) {
	extractor := beauty.NewWellsPTExtractor()

	html, err := loadWellsPTFixture("wells_pt_steampod.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	t.Run("ExtractsInStockTilesAndSkipsNotify", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) != 3 {
			t.Fatalf("expected 3 products, got %d", len(comparisons))
		}
	})

	t.Run("UsesPVPAsSellingPrice", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) == 0 {
			t.Fatal("no comparisons returned")
		}

		c := comparisons[0]
		if c.ProductName != "L'Oréal Professionnel Steampod 4.0" {
			t.Errorf("unexpected ProductName: %q", c.ProductName)
		}
		if c.Price != 387.03 {
			t.Errorf("expected selling price 387.03 (pvp), got %f", c.Price)
		}
		if c.Currency != "EUR" {
			t.Errorf("expected currency EUR, got %s", c.Currency)
		}
		if c.StoreName != "Wells" {
			t.Errorf("expected store Wells, got %s", c.StoreName)
		}
		if c.Country != "PT" {
			t.Errorf("expected country PT, got %s", c.Country)
		}
		if c.Category == nil || *c.Category != models.CategoryBeauty {
			t.Errorf("expected category beauty")
		}
		if c.ID != "7658368" {
			t.Errorf("expected id 7658368, got %s", c.ID)
		}
		if c.StoreURL == nil {
			t.Error("expected non-nil StoreURL")
		} else if *c.StoreURL != "https://wells.pt/loreal-professionnel-steampod-4.0-7658368.html" {
			t.Errorf("unexpected StoreURL: %s", *c.StoreURL)
		}
	})

	t.Run("FallsBackToListPriceWhenPVPMissing", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) < 2 {
			t.Fatal("expected a second product with list-price fallback")
		}
		c := comparisons[1]
		if c.ProductName != "Steampod Professional Smoothing Treatment" {
			t.Errorf("unexpected ProductName: %q", c.ProductName)
		}
		if c.Price != 50.78 {
			t.Errorf("expected fallback price 50.78, got %f", c.Price)
		}
	})

	t.Run("ParsesStringPVP", func(t *testing.T) {
		comparisons, err := extractor.GetComparisonsFromHTML(html)
		if err != nil {
			t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
		}
		if len(comparisons) < 3 {
			t.Fatal("expected a third product with string pvp")
		}
		c := comparisons[2]
		if c.ProductName != "Armani Code Eau de Toilette" {
			t.Errorf("unexpected ProductName: %q", c.ProductName)
		}
		if c.Price != 64.95 {
			t.Errorf("expected string pvp 64.95, got %f", c.Price)
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
