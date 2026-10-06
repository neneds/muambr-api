package extractors_test

import (
	"testing"

	other "muambr-api/extractors/other"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZoomBRProductPageOffers(t *testing.T) {
	html := `<script id="__NEXT_DATA__" type="application/json">{
		"props": {"initialReduxState": {"offers": {"offerList": [
			{"id":"1431608720","name":"Geladeira Electrolux 480L","price":3399,"imageUrl":"https://i.zst.com.br/fridge.jpg","sellerName":"Magazine Luiza"},
			{"id":"2","name":"Geladeira Electrolux 480L","price":0,"sellerName":"Empty"},
			{"id":"3","name":"Geladeira Electrolux 480L","price":3499,"sellerName":"Fast Shop"}
		]}}}
	}</script>`

	got, err := other.NewZoomBRExtractor().ParseProductHTML(html)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "1431608720", got[0].ID)
	assert.Equal(t, "Geladeira Electrolux 480L", got[0].ProductName)
	assert.Equal(t, 3399.0, got[0].Price)
	assert.Equal(t, "BRL", got[0].Currency)
	assert.Equal(t, "BR", got[0].Country)
	assert.Equal(t, "Magazine Luiza", got[0].StoreName)
	require.NotNil(t, got[0].StoreURL)
	assert.Equal(t, "https://www.zoom.com.br/lead?oid=1431608720&channel=1", *got[0].StoreURL)
	require.NotNil(t, got[0].ImageURL)
	assert.Equal(t, "https://i.zst.com.br/fridge.jpg", *got[0].ImageURL)
	assert.Equal(t, "Fast Shop", got[1].StoreName)
	assert.Equal(t, 3499.0, got[1].Price)
}
