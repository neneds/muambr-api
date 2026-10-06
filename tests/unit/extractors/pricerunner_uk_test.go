package extractors_test

import (
	"testing"

	other "muambr-api/extractors/other"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceRunnerUKCategoryListing(t *testing.T) {
	html := `<script type="application/json">{"__DEHYDRATED_QUERY_STATE__":{"queries":[{"queryKey":["cl-list-data-query",{"categoryId":"1"}],"state":{"data":{"pages":[{"products":[
		{"id":"3431242042","name":"Apple iPhone 17 Pro Max, 256GB Deep Blue","url":"/pl/1-3431242042/Mobile-Phones/Apple-iPhone-17-Pro-Max","lowestPrice":{"amount":"1069.99","currency":"GBP"},"image":{"path":"/product/3239077511/phone.jpg"},"outOfStock":false},
		{"id":"9","name":"Sold out phone","url":"/pl/9","lowestPrice":{"amount":"10.00","currency":"GBP"},"outOfStock":true}
	]}]}}}]}}</script>`

	got, err := other.NewPriceRunnerUKExtractor().GetComparisonsFromHTML(html)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "3431242042", got[0].ID)
	assert.Equal(t, "Apple iPhone 17 Pro Max, 256GB Deep Blue", got[0].ProductName)
	assert.Equal(t, 1069.99, got[0].Price)
	assert.Equal(t, "GBP", got[0].Currency)
	assert.Equal(t, "GB", got[0].Country)
	require.NotNil(t, got[0].StoreURL)
	assert.Equal(t, "https://www.pricerunner.com/pl/1-3431242042/Mobile-Phones/Apple-iPhone-17-Pro-Max", *got[0].StoreURL)
	require.NotNil(t, got[0].ImageURL)
	assert.Equal(t, "https://owp.klarna.com/product/3239077511/phone.jpg", *got[0].ImageURL)
}

func TestPriceRunnerUKSuggestJSONStillParses(t *testing.T) {
	body := `{"products":[{"id":"1","name":"Apple iPhone 17, 256GB Black","url":"/pl/1","lowestPrice":{"amount":"699.00","currency":"GBP"},"outOfStock":false}],"suggestions":[{"name":"Apple iPhone 17","url":"/cl/1/Mobile-Phones?attr_60501678=100019743","type":"FEATURE"}]}`

	got, err := other.NewPriceRunnerUKExtractor().GetComparisonsFromHTML(body)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 699.0, got[0].Price)
}
