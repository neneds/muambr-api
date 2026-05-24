package extractors_test

import (
	other "muambr-api/extractors/other"
	"muambr-api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalmartUSAExtractor(t *testing.T) {
	extractor := other.NewWalmartUSAExtractor()

	t.Run("Extractor Properties", func(t *testing.T) {
		assert.Equal(t, models.CountryUS, extractor.GetCountryCode())
		assert.Equal(t, models.MacroRegionNA, extractor.GetMacroRegion())
		assert.Equal(t, "walmart_usa", extractor.GetIdentifier())
		assert.Equal(t, "https://www.walmart.com", extractor.BaseURL())
	})

	t.Run("BuildSearchURL", func(t *testing.T) {
		testCases := []struct {
			productName string
			expectedURL string
		}{
			{
				productName: "beard oil",
				expectedURL: "https://www.walmart.com/search?q=beard+oil",
			},
			{
				productName: "iPhone 15",
				expectedURL: "https://www.walmart.com/search?q=iPhone+15",
			},
			{
				productName: "gaming chair",
				expectedURL: "https://www.walmart.com/search?q=gaming+chair",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.productName, func(t *testing.T) {
				url, err := extractor.BuildSearchURL(tc.productName)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedURL, url)
			})
		}
	})
}

func TestWalmartUSAParser(t *testing.T) {
	parser := other.NewWalmartUSAParser()

	t.Run("GetSelectors", func(t *testing.T) {
		productSelectors := parser.GetProductSelectors()
		assert.NotEmpty(t, productSelectors)
		assert.Contains(t, productSelectors[0], "data-automation-id")

		nameSelectors := parser.GetNameSelectors()
		assert.NotEmpty(t, nameSelectors)
		assert.Contains(t, nameSelectors[0], "product-title")

		priceSelectors := parser.GetPriceSelectors()
		assert.NotEmpty(t, priceSelectors)
		assert.Contains(t, priceSelectors[0], "product-price")

		urlSelectors := parser.GetURLSelectors()
		assert.NotEmpty(t, urlSelectors)
		assert.Contains(t, urlSelectors[0], "href")
	})

	t.Run("ParseProductName", func(t *testing.T) {
		testCases := []struct {
			name         string
			html         string
			expectedName string
		}{
			{
				name:         "Product title with data-automation-id",
				html:         `<span data-automation-id="product-title">Honest Amish Classic Beard Oil - 2 oz</span>`,
				expectedName: "Honest Amish Classic Beard Oil - 2 oz",
			},
			{
				name:         "H3 product title",
				html:         `<h3 data-automation-id="product-title">Beard Growth Oil for Men</h3>`,
				expectedName: "Beard Growth Oil for Men",
			},
			{
				name:         "Link with product title",
				html:         `<a data-testid="product-title">Premium Beard Care Kit</a>`,
				expectedName: "Premium Beard Care Kit",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := parser.ParseProductName(tc.html)
				assert.Equal(t, tc.expectedName, result)
			})
		}
	})

	t.Run("ParsePrice", func(t *testing.T) {
		testCases := []struct {
			name             string
			html             string
			expectedPrice    float64
			expectedCurrency string
			shouldError      bool
		}{
			{
				name:             "Price with data-automation-id",
				html:             `<span data-automation-id="product-price">$19.99</span>`,
				expectedPrice:    19.99,
				expectedCurrency: "USD",
				shouldError:      false,
			},
			{
				name:             "Price without dollar sign",
				html:             `<div data-testid="price-current">24.95</div>`,
				expectedPrice:    24.95,
				expectedCurrency: "USD",
				shouldError:      false,
			},
			{
				name:             "Price with comma",
				html:             `<span class="price current">$1,299.00</span>`,
				expectedPrice:    1299.00,
				expectedCurrency: "USD",
				shouldError:      false,
			},
			{
				name:        "No price found",
				html:        `<div>No price information</div>`,
				shouldError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				price, currency, err := parser.ParsePrice(tc.html)
				if tc.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expectedPrice, price)
					assert.Equal(t, tc.expectedCurrency, currency)
				}
			})
		}
	})

	t.Run("ParseURL", func(t *testing.T) {
		testCases := []struct {
			name        string
			html        string
			baseURL     string
			expectedURL string
		}{
			{
				name:        "Full URL",
				html:        `<a data-automation-id="product-title" href="https://www.walmart.com/ip/Honest-Amish-Classic-Beard-Oil/123456789">Test Product</a>`,
				baseURL:     "https://www.walmart.com",
				expectedURL: "https://www.walmart.com/ip/Honest-Amish-Classic-Beard-Oil/123456789",
			},
			{
				name:        "Relative URL",
				html:        `<a href="/ip/Beard-Oil-Kit/987654321" data-testid="product-title">Test Product</a>`,
				baseURL:     "https://www.walmart.com",
				expectedURL: "https://www.walmart.com/ip/Beard-Oil-Kit/987654321",
			},
			{
				name:        "Fallback to base URL",
				html:        `<div>No URL found</div>`,
				baseURL:     "https://www.walmart.com",
				expectedURL: "https://www.walmart.com",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := parser.ParseURL(tc.html, tc.baseURL)
				assert.Equal(t, tc.expectedURL, result)
			})
		}
	})

	t.Run("ParseStore", func(t *testing.T) {
		testCases := []struct {
			name          string
			html          string
			expectedStore string
		}{
			{
				name:          "Third-party seller",
				html:          `<span class="seller">BeardCare Co.</span>`,
				expectedStore: "BeardCare Co.",
			},
			{
				name:          "Default to Walmart",
				html:          `<div>No seller info</div>`,
				expectedStore: "Walmart",
			},
			{
				name:          "Sold by text pattern",
				html:          `<div>sold by Premium Brands</div>`,
				expectedStore: "Premium Brands",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := parser.ParseStore(tc.html)
				assert.Equal(t, tc.expectedStore, result)
			})
		}
	})
}