package other

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// americanasBRNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// AmericanasBRExtractor overrides GetComparisons entirely and calls the VTEX Intelligent
// Search JSON API, so the parser methods are never called.
type americanasBRNoopParser struct{ *extractors.BaseHTMLParser }

func (p *americanasBRNoopParser) GetProductSelectors() []string       { return nil }
func (p *americanasBRNoopParser) GetNameSelectors() []string          { return nil }
func (p *americanasBRNoopParser) GetPriceSelectors() []string         { return nil }
func (p *americanasBRNoopParser) GetURLSelectors() []string           { return nil }
func (p *americanasBRNoopParser) ParseProductName(html string) string { return "" }
func (p *americanasBRNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *americanasBRNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *americanasBRNoopParser) ParseStore(html string) string               { return "" }

// AmericanasBRExtractor extracts generic retail products from Americanas (Brazil).
//
// The storefront is VTEX FastStore. Search HTML is a client-rendered shell; the
// public Intelligent Search API returns product name + BRL price without auth.
// Marketplace sellers often hold the in-stock offer while seller "1" is Price 0.
//
// Endpoint: GET /api/io/_v/api/intelligent-search/product_search
//
//	?query={query}&count=20&locale=pt-BR
type AmericanasBRExtractor struct {
	*extractors.BaseGoExtractor
}

// NewAmericanasBRExtractor creates a new Americanas Brazil extractor.
func NewAmericanasBRExtractor() *AmericanasBRExtractor {
	parser := &americanasBRNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("americanas_br")}
	base := extractors.NewBaseGoExtractor(
		"https://www.americanas.com.br",
		models.CountryBrazil,
		"americanas_br_v1",
		parser,
	)
	return &AmericanasBRExtractor{BaseGoExtractor: base}
}

// GetCategory returns the generic/other category for registry filtering.
func (e *AmericanasBRExtractor) GetCategory() models.ProductCategory {
	return models.CategoryOther
}

// BuildSearchURL constructs the VTEX Intelligent Search API URL.
func (e *AmericanasBRExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf(
		"%s/api/io/_v/api/intelligent-search/product_search?query=%s&count=20&locale=pt-BR",
		e.GetBaseURL(), encoded,
	), nil
}

// GetComparisons fetches products from the VTEX Intelligent Search API and
// returns comparisons.
func (e *AmericanasBRExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("americanas_br: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("americanas_br: failed to fetch API: %w", err)
	}

	return e.parseVTEXResponse(body)
}

type americanasProduct struct {
	ProductName string           `json:"productName"`
	Link        string           `json:"link"`
	Items       []americanasItem `json:"items"`
}

type americanasItem struct {
	Images  []americanasImage  `json:"images"`
	Sellers []americanasSeller `json:"sellers"`
}

type americanasImage struct {
	ImageURL string `json:"imageUrl"`
}

type americanasSeller struct {
	CommertialOffer americanasOffer `json:"commertialOffer"`
}

type americanasOffer struct {
	Price        float64 `json:"Price"`
	SpotPrice    float64 `json:"spotPrice"`
	AvailableQty int     `json:"AvailableQuantity"`
}

type americanasSearchResponse struct {
	Products []americanasProduct `json:"products"`
}

func (e *AmericanasBRExtractor) parseVTEXResponse(body string) ([]models.ProductComparison, error) {
	var resp americanasSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("americanas_br: failed to parse API response: %w", err)
	}

	category := models.CategoryOther
	results := make([]models.ProductComparison, 0, len(resp.Products))

	for _, p := range resp.Products {
		if len(p.Items) == 0 {
			continue
		}
		item := p.Items[0]

		var price float64
		for _, seller := range item.Sellers {
			offer := seller.CommertialOffer
			if offer.AvailableQty <= 0 {
				continue
			}
			offerPrice := offer.Price
			if offer.SpotPrice > 0 {
				offerPrice = offer.SpotPrice
			}
			if offerPrice <= 0 {
				continue
			}
			if price == 0 || offerPrice < price {
				price = offerPrice
			}
		}
		if price == 0 {
			continue
		}

		link := p.Link
		if link != "" && !strings.HasPrefix(link, "http") {
			link = e.GetBaseURL() + link
		}

		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: strings.TrimSpace(p.ProductName),
			Price:       price,
			Currency:    "BRL",
			StoreName:   "Americanas",
			Country:     string(models.CountryBrazil),
			Category:    &category,
		}
		if link != "" {
			comparison.StoreURL = &link
		}
		if len(item.Images) > 0 && item.Images[0].ImageURL != "" {
			img := item.Images[0].ImageURL
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	return results, nil
}
