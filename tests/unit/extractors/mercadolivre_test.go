package extractors_test

import (
	"testing"

	"muambr-api/extractors"
	other "muambr-api/extractors/other"
	"muambr-api/models"
)

func TestMercadoLivreExtractorMetadata(t *testing.T) {
	extractor := other.NewMercadoLivreExtractorV2()

	t.Run("GetCountryCode", func(t *testing.T) {
		if got := extractor.GetCountryCode(); got != models.CountryBrazil {
			t.Errorf("Expected %s, got %s", models.CountryBrazil, got)
		}
	})

	t.Run("GetMacroRegion", func(t *testing.T) {
		if got := extractor.GetMacroRegion(); got != models.MacroRegionLATAM {
			t.Errorf("Expected %s, got %s", models.MacroRegionLATAM, got)
		}
	})

	t.Run("GetIdentifier", func(t *testing.T) {
		if got := extractor.GetIdentifier(); got != "mercadolivre_v2" {
			t.Errorf("Expected mercadolivre_v2, got %s", got)
		}
	})

	t.Run("BaseURL", func(t *testing.T) {
		if got := extractor.GetBaseURL(); got != "https://lista.mercadolivre.com.br" {
			t.Errorf("Expected https://lista.mercadolivre.com.br, got %s", got)
		}
	})

	t.Run("BuildSearchURL", func(t *testing.T) {
		url, err := extractor.BuildSearchURL("iPhone 15")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := "https://lista.mercadolivre.com.br/iphone-15"
		if url != expected {
			t.Errorf("Expected %s, got %s", expected, url)
		}
	})

	t.Run("ImplementsExtractorInterface", func(t *testing.T) {
		var _ extractors.Extractor = extractor
	})
}

func TestMercadoLivreJSONLDExtraction(t *testing.T) {
	html, err := loadSampleResponse("mercadolivre_search.html")
	if err != nil {
		t.Fatalf("Failed to load sample response: %v", err)
	}

	extractor := other.NewMercadoLivreExtractorV2()
	comparisons, err := extractor.GetComparisonsFromHTML(html)
	if err != nil {
		t.Fatalf("GetComparisonsFromHTML failed: %v", err)
	}

	t.Run("ExtractsProducts", func(t *testing.T) {
		if len(comparisons) == 0 {
			t.Fatal("Expected products, got 0")
		}
		t.Logf("Extracted %d products", len(comparisons))
	})

	t.Run("ProductFields", func(t *testing.T) {
		if len(comparisons) == 0 {
			t.Skip("No comparisons to validate")
		}
		c := comparisons[0]
		if c.ProductName == "" {
			t.Error("Expected non-empty ProductName")
		}
		if c.Price <= 0 {
			t.Errorf("Expected positive price, got %f", c.Price)
		}
		if c.Currency != "BRL" {
			t.Errorf("Expected currency BRL, got %s", c.Currency)
		}
		if c.StoreName != "MercadoLivre" {
			t.Errorf("Expected store MercadoLivre, got %s", c.StoreName)
		}
		if c.Country != string(models.CountryBrazil) {
			t.Errorf("Expected country BR, got %s", c.Country)
		}
		if c.StoreURL == nil || *c.StoreURL == "" {
			t.Error("Expected non-empty StoreURL")
		}
		if c.ID == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("AllProductsHaveRequiredFields", func(t *testing.T) {
		for i, c := range comparisons {
			if c.ProductName == "" {
				t.Errorf("Product %d: empty name", i)
			}
			if c.Price <= 0 {
				t.Errorf("Product %d: invalid price %f", i, c.Price)
			}
		}
	})
}

func TestMercadoLivreEmptyHTML(t *testing.T) {
	extractor := other.NewMercadoLivreExtractorV2()
	comparisons, err := extractor.GetComparisonsFromHTML("")
	if err != nil {
		t.Fatalf("Unexpected error on empty HTML: %v", err)
	}
	if len(comparisons) != 0 {
		t.Errorf("Expected 0 comparisons for empty HTML, got %d", len(comparisons))
	}
}

func TestMercadoLivreHTMLFallback(t *testing.T) {
	// Minimal HTML with poly-card structure but no JSON-LD
	html := `<!DOCTYPE html><html><body>
<ol class="ui-search-layout">
<li class="ui-search-layout__item">
  <div class="poly-card">
    <div class="poly-card__content">
      <h3 class="poly-component__title-wrapper">
        <a href="https://www.mercadolivre.com.br/iphone-15" class="poly-component__title">Apple iPhone 15 128GB</a>
      </h3>
      <div class="poly-price__current">
        <span class="andes-money-amount__fraction">4.699</span>
      </div>
    </div>
  </div>
</li>
<li class="ui-search-layout__item">
  <div class="poly-card">
    <div class="poly-card__content">
      <h3 class="poly-component__title-wrapper">
        <a href="https://www.mercadolivre.com.br/iphone-15-pro" class="poly-component__title">Apple iPhone 15 Pro 256GB</a>
      </h3>
      <div class="poly-price__current">
        <span class="andes-money-amount__fraction">6.299</span>
        <span class="andes-money-amount__cents">90</span>
      </div>
    </div>
  </div>
</li>
</ol>
</body></html>`

	extractor := other.NewMercadoLivreExtractorV2()
	comparisons, err := extractor.GetComparisonsFromHTML(html)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Run("ExtractsFromHTML", func(t *testing.T) {
		if len(comparisons) != 2 {
			t.Fatalf("Expected 2 products, got %d", len(comparisons))
		}
	})

	t.Run("ParsesPrice", func(t *testing.T) {
		if comparisons[0].Price != 4699 {
			t.Errorf("Expected price 4699, got %f", comparisons[0].Price)
		}
	})

	t.Run("ParsesPriceWithCents", func(t *testing.T) {
		expected := 6299.90
		if comparisons[1].Price != expected {
			t.Errorf("Expected price %f, got %f", expected, comparisons[1].Price)
		}
	})

	t.Run("ParsesProductURL", func(t *testing.T) {
		if comparisons[0].StoreURL == nil || *comparisons[0].StoreURL != "https://www.mercadolivre.com.br/iphone-15" {
			t.Errorf("Expected URL https://www.mercadolivre.com.br/iphone-15, got %v", comparisons[0].StoreURL)
		}
	})
}
