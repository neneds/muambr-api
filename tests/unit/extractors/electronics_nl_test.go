package extractors_test

import (
	"os"
	"path/filepath"
	"testing"

	"muambr-api/extractors"
	electronics "muambr-api/extractors/electronics"
	"muambr-api/models"
)

func loadElectronicsFixture(filename string) (string, error) {
	path := filepath.Join("..", "..", "mocks", "extractors", "sample_responses", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestParadigitNLExtractor(t *testing.T) {
	extractor := electronics.NewParadigitNLExtractor()

	t.Run("GetCountryCode", func(t *testing.T) {
		if got := extractor.GetCountryCode(); got != models.CountryNetherlands {
			t.Errorf("expected %s, got %s", models.CountryNetherlands, got)
		}
	})

	t.Run("GetCategory", func(t *testing.T) {
		if got := extractor.GetCategory(); got != models.CategoryElectronics {
			t.Errorf("expected %s, got %s", models.CategoryElectronics, got)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		if got := extractor.GetIdentifier(); got != "paradigit_nl_v1" {
			t.Errorf("expected paradigit_nl_v1, got %s", got)
		}
	})

	t.Run("InterfaceImplementation", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
}

func TestParadigitNLGetComparisonsFromHTML(t *testing.T) {
	extractor := electronics.NewParadigitNLExtractor()
	html, err := loadElectronicsFixture("paradigit_nl_iphone.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	comparisons, err := extractor.GetComparisonsFromHTML(html)
	if err != nil {
		t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
	}
	if len(comparisons) != 1 {
		t.Fatalf("expected 1 in-stock product, got %d", len(comparisons))
	}
	got := comparisons[0]
	if got.ProductName == "" || got.Price != 799 {
		t.Errorf("unexpected product: name=%q price=%v", got.ProductName, got.Price)
	}
	if got.Currency != "EUR" || got.StoreName != "Paradigit" {
		t.Errorf("unexpected store/currency: %+v", got)
	}
	if got.ID != "80071442" {
		t.Errorf("expected sku id 80071442, got %s", got.ID)
	}
}

func TestAlternateNLExtractor(t *testing.T) {
	extractor := electronics.NewAlternateNLExtractor()

	t.Run("GetCountryCode", func(t *testing.T) {
		if got := extractor.GetCountryCode(); got != models.CountryNetherlands {
			t.Errorf("expected %s, got %s", models.CountryNetherlands, got)
		}
	})

	t.Run("GetCategory", func(t *testing.T) {
		if got := extractor.GetCategory(); got != models.CategoryElectronics {
			t.Errorf("expected %s, got %s", models.CategoryElectronics, got)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		if got := extractor.GetIdentifier(); got != "alternate_nl_v1" {
			t.Errorf("expected alternate_nl_v1, got %s", got)
		}
	})

	t.Run("InterfaceImplementation", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
}

func TestAlternateNLGetComparisonsFromHTML(t *testing.T) {
	extractor := electronics.NewAlternateNLExtractor()
	html, err := loadElectronicsFixture("alternate_nl_iphone.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	comparisons, err := extractor.GetComparisonsFromHTML(html)
	if err != nil {
		t.Fatalf("GetComparisonsFromHTML returned error: %v", err)
	}
	if len(comparisons) != 2 {
		t.Fatalf("expected 2 products, got %d", len(comparisons))
	}
	if comparisons[0].Price != 139.90 {
		t.Errorf("expected 139.90, got %v", comparisons[0].Price)
	}
	if comparisons[1].Price != 1864 {
		t.Errorf("expected 1864, got %v", comparisons[1].Price)
	}
	if comparisons[0].ID != "100156407" {
		t.Errorf("expected product id 100156407, got %s", comparisons[0].ID)
	}
	if comparisons[0].StoreName != "Alternate" || comparisons[0].Currency != "EUR" {
		t.Errorf("unexpected store/currency: %+v", comparisons[0])
	}
}
