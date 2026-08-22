package models_test

import (
	"testing"

	"muambr-api/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveComparisonSet_ExplicitIncludesBaseAndRoles(t *testing.T) {
	specs := models.ResolveComparisonSet(models.ComparisonSetInput{
		BaseCountry:       models.CountryBrazil,
		ProductLocation:   models.CountryUK,
		ExplicitCountries: []models.Country{models.CountryUK, models.CountryNetherlands, models.CountryPortugal, models.CountryUK},
		JourneyCountries:  []models.Country{models.CountryUK, models.CountryNetherlands, models.CountryPortugal},
	})

	require.Len(t, specs, 4)
	assert.Equal(t, models.CountryUK, specs[0].Country)
	assert.Equal(t, []string{"current", "journey"}, specs[0].Roles)
	assert.Equal(t, models.CountryNetherlands, specs[1].Country)
	assert.Equal(t, []string{"journey"}, specs[1].Roles)
	assert.Equal(t, models.CountryPortugal, specs[2].Country)
	assert.Equal(t, []string{"journey"}, specs[2].Roles)
	assert.Equal(t, models.CountryBrazil, specs[3].Country)
	assert.Equal(t, []string{"base"}, specs[3].Roles)
}

func TestResolveComparisonSet_DoesNotAddUnselectedJourneyCountries(t *testing.T) {
	specs := models.ResolveComparisonSet(models.ComparisonSetInput{
		BaseCountry:       models.CountryBrazil,
		ProductLocation:   models.CountryUK,
		ExplicitCountries: []models.Country{models.CountryBrazil, models.CountryUK},
		JourneyCountries:  []models.Country{models.CountryUK, models.CountryNetherlands, models.CountryPortugal},
	})

	codes := models.CountriesFromSpecs(specs)
	assert.Equal(t, []models.Country{models.CountryBrazil, models.CountryUK}, codes)
}

func TestResolveComparisonSet_LegacyFallbackAndMicroregionAddsOnly(t *testing.T) {
	specs := models.ResolveComparisonSet(models.ComparisonSetInput{
		BaseCountry:       models.CountryBrazil,
		ProductLocation:   models.CountryUK,
		ExpandMicroregion: true,
	})

	codes := models.CountriesFromSpecs(specs)
	assert.Contains(t, codes, models.CountryBrazil)
	assert.Contains(t, codes, models.CountryUK)
	assert.Contains(t, codes, models.CountryNetherlands)
	assert.Contains(t, codes, models.CountryPortugal)

	byCode := map[models.Country][]string{}
	for _, s := range specs {
		byCode[s.Country] = s.Roles
	}
	assert.Equal(t, []string{"base"}, byCode[models.CountryBrazil])
	assert.Equal(t, []string{"current"}, byCode[models.CountryUK])
	assert.Contains(t, byCode[models.CountryNetherlands], "microregion")
	assert.NotContains(t, byCode[models.CountryUK], "microregion")
}

func TestResolveComparisonSet_NoCurrentLeavesOnlyBase(t *testing.T) {
	specs := models.ResolveComparisonSet(models.ComparisonSetInput{
		BaseCountry:       models.CountryBrazil,
		ExpandMicroregion: true,
	})
	require.Len(t, specs, 1)
	assert.Equal(t, models.CountryBrazil, specs[0].Country)
	assert.Equal(t, []string{"base"}, specs[0].Roles)
}
