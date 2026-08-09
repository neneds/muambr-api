package utils

import (
	"testing"
	"time"

	"muambr-api/models"
	"muambr-api/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchConfidence_ExactTokens(t *testing.T) {
	score := utils.MatchConfidence("Sony WH-1000XM6", "Sony WH-1000XM6 Wireless Headphones")
	assert.GreaterOrEqual(t, score, 0.75)
}

func TestMatchConfidence_Unrelated(t *testing.T) {
	score := utils.MatchConfidence("iPhone 15 Pro", "Kitchen Blender 500W")
	assert.Less(t, score, 0.5)
}

func TestComparisonEngine_BuildResult_SavingsAndDealScore(t *testing.T) {
	engine := utils.NewComparisonEngine()
	category := models.CategoryElectronics
	conf := 0.95

	sections := []models.CountrySection{
		{
			Country:     "BR",
			CountryName: "Brazil",
			Comparisons: []models.ProductComparison{
				{
					ID:          "1",
					ProductName: "Sony WH-1000XM6",
					Price:       2499,
					Currency:    "BRL",
					StoreName:   "Amazon BR",
					Country:     "BR",
					MatchConfidence: &conf,
				},
			},
			ResultsCount: 1,
		},
		{
			Country:     "US",
			CountryName: "United States",
			Comparisons: []models.ProductComparison{
				{
					ID:          "2",
					ProductName: "Sony WH-1000XM6",
					Price:       349,
					Currency:    "USD",
					ConvertedPrice: &models.ConvertedPrice{
						Price:    1891.58,
						Currency: "BRL",
					},
					StoreName:       "Amazon US",
					Country:         "US",
					MatchConfidence: &conf,
				},
			},
			ResultsCount: 1,
		},
	}

	result := engine.BuildResult(utils.ComparisonEngineInput{
		ProductName:        "Sony WH-1000XM6",
		Brand:              "Sony",
		Model:              "WH-1000XM6",
		Category:           &category,
		CategoryConfidence: 0.96,
		BaseCountry:        models.CountryBrazil,
		CurrentCountry:     models.CountryUS,
		NormalizedCurrency: "BRL",
		Observed: &models.ObservedPriceInput{
			Amount:   349,
			Currency: "USD",
			Country:  "US",
			Store:    "Amazon US",
		},
		Sections: sections,
		Meta: utils.ExtractionMeta{
			ProvidersAttempted: 2,
			ProvidersSucceeded: 2,
		},
		EntryMethod: "manual",
		ExchangeRate: &models.ExchangeRateInfo{
			Base:      "USD",
			Target:    "BRL",
			Rate:      5.42,
			Source:    "exchangerate-api",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Now: time.Date(2026, 8, 9, 3, 0, 0, 0, time.UTC),
	})

	require.True(t, result.Success)
	assert.Equal(t, models.ComparisonStatusComplete, result.Status)
	assert.NotEmpty(t, result.ComparisonID)
	assert.Equal(t, "2026-08-09T03:00:00Z", result.CapturedAt)
	require.NotNil(t, result.Observed)
	assert.InDelta(t, 1891.58, result.Observed.NormalizedAmount, 0.01)
	require.NotNil(t, result.BestBaseCountryPrice)
	assert.Equal(t, 2499.0, result.BestBaseCountryPrice.NormalizedAmount)
	require.NotNil(t, result.BestCurrentCountryPrice)
	assert.InDelta(t, 1891.58, result.BestCurrentCountryPrice.NormalizedAmount, 0.01)
	require.NotNil(t, result.Savings)
	assert.True(t, result.Savings.IsCheaper)
	assert.InDelta(t, 607.42, result.Savings.Amount, 0.1)
	assert.Greater(t, result.Savings.Percentage, 20.0)
	require.NotNil(t, result.DealScore)
	assert.GreaterOrEqual(t, result.DealScore.Value, 70)
	assert.True(t, result.DealScore.IsDefinitive)
	assert.Equal(t, "retail", result.Metadata.PriceType)
	require.NotNil(t, result.Category)
	assert.Equal(t, models.CategoryElectronics, result.Category.ID)
}

func TestComparisonEngine_BuildResult_PartialStatus(t *testing.T) {
	engine := utils.NewComparisonEngine()
	sections := []models.CountrySection{
		{
			Country:     "BR",
			CountryName: "Brazil",
			Comparisons: []models.ProductComparison{
				{
					ID:          "1",
					ProductName: "iPhone 15",
					Price:       5000,
					Currency:    "BRL",
					StoreName:   "ML",
					Country:     "BR",
				},
			},
			ResultsCount: 1,
		},
	}

	result := engine.BuildResult(utils.ComparisonEngineInput{
		ProductName:        "iPhone 15",
		BaseCountry:        models.CountryBrazil,
		CurrentCountry:     models.CountryBrazil,
		NormalizedCurrency: "BRL",
		Sections:           sections,
		Meta: utils.ExtractionMeta{
			ProvidersAttempted: 3,
			ProvidersSucceeded: 1,
			ProvidersFailed:    2,
			FailedProviders:    []string{"a", "b"},
		},
		Now: time.Now().UTC(),
	})

	assert.Equal(t, models.ComparisonStatusPartial, result.Status)
	assert.Equal(t, 2, result.Metadata.ProvidersFailed)
}

func TestComparisonEngine_BuildResult_Empty(t *testing.T) {
	engine := utils.NewComparisonEngine()
	result := engine.BuildResult(utils.ComparisonEngineInput{
		ProductName:        "Unknown Gadget",
		BaseCountry:        models.CountryBrazil,
		CurrentCountry:     models.CountryUS,
		NormalizedCurrency: "BRL",
		Sections:           nil,
		Meta:               utils.ExtractionMeta{ProvidersAttempted: 2, ProvidersSucceeded: 2},
		Now:                time.Now().UTC(),
	})

	assert.Equal(t, models.ComparisonStatusEmpty, result.Status)
	require.NotNil(t, result.Code)
	assert.Equal(t, models.ErrorCodeProductNotFound, *result.Code)
}

func TestComparisonEngine_SavingsWithoutObservedUsesBestCurrent(t *testing.T) {
	engine := utils.NewComparisonEngine()
	sections := []models.CountrySection{
		{
			Country: "BR",
			Comparisons: []models.ProductComparison{
				{ProductName: "X", Price: 2000, Currency: "BRL", Country: "BR", StoreName: "BR Store"},
			},
			ResultsCount: 1,
		},
		{
			Country: "US",
			Comparisons: []models.ProductComparison{
				{
					ProductName: "X",
					Price:       300,
					Currency:    "USD",
					ConvertedPrice: &models.ConvertedPrice{Price: 1500, Currency: "BRL"},
					Country:     "US",
					StoreName:   "US Store",
				},
			},
			ResultsCount: 1,
		},
	}

	result := engine.BuildResult(utils.ComparisonEngineInput{
		ProductName:        "X",
		BaseCountry:        models.CountryBrazil,
		CurrentCountry:     models.CountryUS,
		NormalizedCurrency: "BRL",
		Sections:           sections,
		Meta:               utils.ExtractionMeta{ProvidersAttempted: 2, ProvidersSucceeded: 2},
		Now:                time.Now().UTC(),
	})

	require.NotNil(t, result.Savings)
	assert.Equal(t, "best_current", result.Savings.ComparedAgainst)
	assert.True(t, result.Savings.IsCheaper)
	assert.InDelta(t, 500.0, result.Savings.Amount, 0.01)
}
