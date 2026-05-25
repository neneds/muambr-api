package extractors_test

import (
	"testing"

	other "muambr-api/extractors/other"
	"muambr-api/models"
)

// TestIdealoExtractorV2 is skipped - idealo extractor not yet implemented

func TestKuantoKustaExtractor(t *testing.T) {
	extractor := other.NewKuantoKustaExtractorV2()

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
		expected := "kuantokusta_v2"
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
}

func TestMercadoLivreExtractor(t *testing.T) {
	extractor := other.NewMercadoLivreExtractorV2()

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
		expected := "mercadolivre_v2"
		if identifier != expected {
			t.Errorf("Expected identifier %s, got %s", expected, identifier)
		}
	})
}

