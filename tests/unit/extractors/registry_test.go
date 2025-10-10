package extractors_test

import (
	"testing"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/tests/mocks"
)

func TestExtractorRegistry(t *testing.T) {
	t.Run("NewExtractorRegistry", func(t *testing.T) {
		registry := extractors.NewExtractorRegistry()
		if registry == nil {
			t.Fatal("Expected registry to be created, got nil")
		}
	})

	t.Run("RegisterExtractor", func(t *testing.T) {
		registry := extractors.NewExtractorRegistry()
		
		// Create a mock extractor
		mockExtractor := mocks.NewMockExtractor(models.CountryBrazil, "mock_extractor")
		
		// Register the extractor
		registry.RegisterExtractor(mockExtractor)
		
		// Verify it's registered
		extractorsList := registry.GetExtractorsForCountry(models.CountryBrazil)
		if len(extractorsList) != 1 {
			t.Errorf("Expected 1 extractor for Brazil, got %d", len(extractorsList))
		}
		
		if extractorsList[0] != mockExtractor {
			t.Error("Expected registered extractor to match mock extractor")
		}
	})

	t.Run("GetExtractorsForCountry", func(t *testing.T) {
		registry := extractors.NewExtractorRegistry()
		
		// Test with no extractors registered
		extractorsList := registry.GetExtractorsForCountry(models.CountryBrazil)
		if extractorsList != nil {
			t.Errorf("Expected nil for unregistered country, got %v", extractorsList)
		}
		
		// Register multiple extractors for the same country
		mockExtractor1 := mocks.NewMockExtractor(models.CountryBrazil, "mock_extractor_1")
		mockExtractor2 := mocks.NewMockExtractor(models.CountryBrazil, "mock_extractor_2")
		
		registry.RegisterExtractor(mockExtractor1)
		registry.RegisterExtractor(mockExtractor2)
		
		extractorsList = registry.GetExtractorsForCountry(models.CountryBrazil)
		if len(extractorsList) != 2 {
			t.Errorf("Expected 2 extractors for Brazil, got %d", len(extractorsList))
		}
	})

	t.Run("MultipleCountriesSupport", func(t *testing.T) {
		registry := extractors.NewExtractorRegistry()
		
		// Register extractors for different countries
		brazilExtractor := mocks.NewMockExtractor(models.CountryBrazil, "brazil_extractor")
		portugalExtractor := mocks.NewMockExtractor(models.CountryPortugal, "portugal_extractor")
		usExtractor := mocks.NewMockExtractor(models.CountryUS, "us_extractor")
		
		registry.RegisterExtractor(brazilExtractor)
		registry.RegisterExtractor(portugalExtractor)
		registry.RegisterExtractor(usExtractor)
		
		// Test Brazil extractors
		brazilExtractors := registry.GetExtractorsForCountry(models.CountryBrazil)
		if len(brazilExtractors) != 1 {
			t.Errorf("Expected 1 extractor for Brazil, got %d", len(brazilExtractors))
		}
		
		// Test Portugal extractors
		portugalExtractors := registry.GetExtractorsForCountry(models.CountryPortugal)
		if len(portugalExtractors) != 1 {
			t.Errorf("Expected 1 extractor for Portugal, got %d", len(portugalExtractors))
		}
		
		// Test US extractors
		usExtractors := registry.GetExtractorsForCountry(models.CountryUS)
		if len(usExtractors) != 1 {
			t.Errorf("Expected 1 extractor for US, got %d", len(usExtractors))
		}
		
		// Test non-existent country
		germanyExtractors := registry.GetExtractorsForCountry(models.CountryGermany)
		if germanyExtractors != nil {
			t.Errorf("Expected nil for Germany (no extractors), got %v", germanyExtractors)
		}
	})
}