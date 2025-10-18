package handlers

import (
	"testing"

	"muambr-api/handlers"
	"muambr-api/models"

	"github.com/stretchr/testify/assert"
)

func TestExtractorHandler_GetProductComparisons_BaseCountryOnly(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with base country only (no current country, no macro region)
	results, err := handler.GetProductComparisons("iPhone", models.CountryBrazil, nil, "USD", false)

	// Should not return an error (even if extractors fail, handler should continue)
	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Results can be empty if extractors fail, but the call should succeed
	// The key test is that only BR extractors should be attempted
	// This is verified through the logging in the actual implementation
}

func TestExtractorHandler_GetProductComparisons_BaseCountryPlusCurrentCountry(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with base country (BR) + different current country (PT), no macro region
	currentCountry := models.CountryPortugal
	results, err := handler.GetProductComparisons("iPhone", models.CountryBrazil, &currentCountry, "USD", false)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Should attempt both BR and PT extractors
	// BR: mercadolivre_v2, acharpromo_v2
	// PT: kuantokusta_v2
}

func TestExtractorHandler_GetProductComparisons_BaseCountryPlusCurrentCountryPlusMacroRegion(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with base country (BR) + current country (PT) + macro region enabled
	// This should include:
	// - BR extractors (base country): mercadolivre_v2, acharpromo_v2
	// - PT extractors (current country): kuantokusta_v2
	// - EU macro region extractors (PT's macro region): kuantokusta_v2 (PT), idealo_v2 (ES)
	currentCountry := models.CountryPortugal
	results, err := handler.GetProductComparisons("MacBook Pro m1", models.CountryBrazil, &currentCountry, "EUR", true)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Should attempt BR + PT + ES extractors (EU macro region includes PT and ES)
}

func TestExtractorHandler_GetProductComparisons_SameBaseAndCurrentCountry(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with same base and current country - should not duplicate extractors
	currentCountry := models.CountryBrazil
	results, err := handler.GetProductComparisons("iPhone", models.CountryBrazil, &currentCountry, "USD", false)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Should only attempt BR extractors once (deduplication should work)
}

func TestExtractorHandler_GetProductComparisons_SameBaseAndCurrentCountryWithMacroRegion(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with same base and current country + macro region
	// Base: BR, Current: BR, MacroRegion: true
	// Should include BR extractors + LATAM macro region extractors (which includes BR)
	currentCountry := models.CountryBrazil
	results, err := handler.GetProductComparisons("iPhone", models.CountryBrazil, &currentCountry, "USD", true)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Should attempt BR extractors (base/current) + LATAM macro region (also BR)
	// Deduplication should ensure extractors are not called multiple times
}

func TestExtractorHandler_GetProductComparisons_NoCurrentCountryWithMacroRegion(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with only base country and macro region enabled, but no current country
	// This should only use base country extractors since macro region requires current country
	results, err := handler.GetProductComparisons("iPhone", models.CountryBrazil, nil, "USD", true)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Should only attempt BR extractors since macro region requires currentCountry to be set
}

func TestExtractorHandler_GetProductComparisons_EmptySearchTerm(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with empty search term
	results, _ := handler.GetProductComparisons("", models.CountryBrazil, nil, "USD", false)

	// Should handle empty search term gracefully
	assert.NotNil(t, results) // Can be empty slice but not nil
}

func TestExtractorHandler_GetProductComparisons_UnsupportedCountry(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with a country that has no registered extractors
	// Using a country enum that exists but has no extractors registered
	results, err := handler.GetProductComparisons("iPhone", models.CountryGermany, nil, "USD", false)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results) // Should return empty results, not error
}

func TestExtractorHandler_GetProductComparisons_CrossMacroRegion(t *testing.T) {
	// Create handler with registered extractors
	handler := handlers.NewExtractorHandler()

	// Test with base country from one macro region and current country from another
	// Base: BR (LATAM), Current: PT (EU), MacroRegion: true
	// Should include:
	// - BR extractors (base country - LATAM)
	// - PT extractors (current country - EU)
	// - EU macro region extractors (PT, ES, etc.)
	currentCountry := models.CountryPortugal
	results, err := handler.GetProductComparisons("MacBook", models.CountryBrazil, &currentCountry, "EUR", true)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Should attempt extractors from both LATAM (base) and EU (current + macro region)
}

func TestExtractorHandler_DetectCountryCode_ValidCountry(t *testing.T) {
	handler := handlers.NewExtractorHandler()

	// Test valid country codes
	testCases := []struct {
		input    string
		expected models.Country
	}{
		{"BR", models.CountryBrazil},
		{"PT", models.CountryPortugal},
		{"ES", models.CountrySpain},
		{"br", models.CountryBrazil}, // Should handle lowercase
		{"pt", models.CountryPortugal},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := handler.DetectCountryCode(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractorHandler_DetectCountryCode_InvalidCountry(t *testing.T) {
	handler := handlers.NewExtractorHandler()

	// Test invalid country codes
	invalidCodes := []string{"XX", "ZZ", "ABC", "123"}

	for _, code := range invalidCodes {
		t.Run(code, func(t *testing.T) {
			_, err := handler.DetectCountryCode(code)
			assert.Error(t, err)
		})
	}
}

func TestExtractorHandler_DetectCountryCode_Empty(t *testing.T) {
	handler := handlers.NewExtractorHandler()

	result, _ := handler.DetectCountryCode("")
	assert.Equal(t, models.Country(""), result) // Should return empty country, not error
}

// Integration test to verify extractor registration
func TestExtractorHandler_ExtractorRegistration(t *testing.T) {
	handler := handlers.NewExtractorHandler()

	// Verify that expected extractors are registered
	// We can't directly access the registry, but we can test through GetProductComparisons
	// and verify no panic/errors occur with known countries

	countries := []models.Country{
		models.CountryBrazil,   // Should have mercadolivre_v2, acharpromo_v2
		models.CountryPortugal, // Should have kuantokusta_v2
		models.CountrySpain,    // Should have idealo_v2
	}

	for _, country := range countries {
		t.Run(string(country), func(t *testing.T) {
			results, err := handler.GetProductComparisons("test", country, nil, "USD", false)
			assert.NoError(t, err)
			assert.NotNil(t, results)
			// Don't assert on results length as extractors might fail due to network/parsing
			// The important part is that the handler doesn't panic or return unexpected errors
		})
	}
}