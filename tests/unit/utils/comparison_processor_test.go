package utils

import (
	"testing"

	"muambr-api/models"
	"muambr-api/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterPriceOutliers_KeepsCheapClusterAndDropsExpensiveImports(t *testing.T) {
	processor := utils.NewComparisonProcessor()
	sections := processor.ProcessComparisons([]models.ProductComparison{
		offer("Fanta Orange 500ml", 3.99),
		offer("Fanta Orange 500ml", 6.99),
		offer("Fanta Orange 500ml", 10.02),
		offer("Fanta Orange 500ml", 11.92),
		offer("Fanta Orange 500ml", 14.88),
		offer("Fanta Orange 500ml", 16.90),
		offer("Imported Fanta Orange 500ml hamper", 120),
		offer("Imported Fanta Orange 500ml hamper", 140),
	}, "fanta orange 500ml", 20)

	require.Len(t, sections, 1)
	prices := pricesOf(sections[0].Comparisons)
	assert.Contains(t, prices, 3.99)
	assert.Contains(t, prices, 11.92)
	assert.NotContains(t, prices, 120.0)
	assert.NotContains(t, prices, 140.0)
}

func TestFilterPriceOutliers_DropsCheapAccessoryAmongPhones(t *testing.T) {
	processor := utils.NewComparisonProcessor()
	sections := processor.ProcessComparisons([]models.ProductComparison{
		offer("Apple iPhone 15 Pro Max", 1099),
		offer("Apple iPhone 15 Pro", 1199),
		offer("Apple iPhone 15", 1299),
		offer("Apple Siliconenhoesje met MagSafe voor iPhone 15 Plus telefoonhoesje", 29.99),
	}, "iphone", 20)

	require.Len(t, sections, 1)
	prices := pricesOf(sections[0].Comparisons)
	assert.Contains(t, prices, 1099.0)
	assert.Contains(t, prices, 1199.0)
	assert.Contains(t, prices, 1299.0)
	assert.NotContains(t, prices, 29.99)
}

func TestFilterPriceOutliers_LeavesTwoOffersUntouched(t *testing.T) {
	processor := utils.NewComparisonProcessor()
	sections := processor.ProcessComparisons([]models.ProductComparison{
		offer("Fanta Orange 500ml", 4),
		offer("Fanta Orange 500ml", 400),
	}, "fanta orange 500ml", 20)

	require.Len(t, sections, 1)
	assert.Len(t, sections[0].Comparisons, 2)
}

func offer(name string, price float64) models.ProductComparison {
	return models.ProductComparison{
		ProductName: name,
		Price:       price,
		Currency:    "BRL",
		Country:     "BR",
		StoreName:   "Store",
	}
}

func pricesOf(comparisons []models.ProductComparison) []float64 {
	prices := make([]float64, 0, len(comparisons))
	for _, comparison := range comparisons {
		prices = append(prices, comparison.Price)
	}
	return prices
}
