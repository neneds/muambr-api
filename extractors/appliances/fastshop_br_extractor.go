package appliances

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// fastshopBRNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// FastshopBRExtractor overrides GetComparisons entirely and calls the VTEX Intelligent
// Search JSON API, so the parser methods are never called.
type fastshopBRNoopParser struct{ *extractors.BaseHTMLParser }

func (p *fastshopBRNoopParser) GetProductSelectors() []string       { return nil }
func (p *fastshopBRNoopParser) GetNameSelectors() []string          { return nil }
func (p *fastshopBRNoopParser) GetPriceSelectors() []string         { return nil }
func (p *fastshopBRNoopParser) GetURLSelectors() []string           { return nil }
func (p *fastshopBRNoopParser) ParseProductName(html string) string { return "" }
func (p *fastshopBRNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *fastshopBRNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *fastshopBRNoopParser) ParseStore(html string) string               { return "" }

// FastshopBRExtractor extracts appliance products from Fast Shop (Brazil).
//
// The storefront is VTEX FastStore. Search HTML / Next.js s.json is a shell;
// the public Intelligent Search API returns product name + BRL price without auth.
//
// Endpoint: GET /api/io/_v/api/intelligent-search/product_search
//
//	?query={query}&count=20&locale=pt-BR
type FastshopBRExtractor struct {
	*extractors.BaseGoExtractor
}

// NewFastshopBRExtractor creates a new Fast Shop Brazil extractor.
func NewFastshopBRExtractor() *FastshopBRExtractor {
	parser := &fastshopBRNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("fastshop_br")}
	base := extractors.NewBaseGoExtractor(
		"https://site.fastshop.com.br",
		models.CountryBrazil,
		"fastshop_br_v1",
		parser,
	)
	return &FastshopBRExtractor{BaseGoExtractor: base}
}

// GetCategory returns the appliances category for registry filtering.
func (e *FastshopBRExtractor) GetCategory() models.ProductCategory {
	return models.CategoryAppliances
}

// BuildSearchURL constructs the VTEX Intelligent Search API URL.
func (e *FastshopBRExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf(
		"%s/api/io/_v/api/intelligent-search/product_search?query=%s&count=20&locale=pt-BR",
		e.GetBaseURL(), encoded,
	), nil
}

// GetComparisons fetches products from the VTEX Intelligent Search API and
// returns comparisons.
func (e *FastshopBRExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("fastshop_br: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("fastshop_br: failed to fetch API: %w", err)
	}

	return e.parseVTEXResponse(body)
}

type fastshopProduct struct {
	ProductName string         `json:"productName"`
	Link        string         `json:"link"`
	Items       []fastshopItem `json:"items"`
}

type fastshopItem struct {
	Images  []fastshopImage  `json:"images"`
	Sellers []fastshopSeller `json:"sellers"`
}

type fastshopImage struct {
	ImageURL string `json:"imageUrl"`
}

type fastshopSeller struct {
	CommertialOffer fastshopOffer `json:"commertialOffer"`
}

type fastshopOffer struct {
	Price        float64 `json:"Price"`
	SpotPrice    float64 `json:"spotPrice"`
	AvailableQty int     `json:"AvailableQuantity"`
}

type fastshopSearchResponse struct {
	Products []fastshopProduct `json:"products"`
}

func (e *FastshopBRExtractor) parseVTEXResponse(body string) ([]models.ProductComparison, error) {
	var resp fastshopSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("fastshop_br: failed to parse API response: %w", err)
	}

	category := models.CategoryAppliances
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
			StoreName:   "Fast Shop",
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
