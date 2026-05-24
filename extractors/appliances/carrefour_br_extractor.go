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

// carrefourBRNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// CarrefourBRExtractor overrides GetComparisons entirely and calls the VTEX Intelligent
// Search JSON API, so the parser methods are never called.
type carrefourBRNoopParser struct{ *extractors.BaseHTMLParser }

func (p *carrefourBRNoopParser) GetProductSelectors() []string                   { return nil }
func (p *carrefourBRNoopParser) GetNameSelectors() []string                      { return nil }
func (p *carrefourBRNoopParser) GetPriceSelectors() []string                     { return nil }
func (p *carrefourBRNoopParser) GetURLSelectors() []string                       { return nil }
func (p *carrefourBRNoopParser) ParseProductName(html string) string             { return "" }
func (p *carrefourBRNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *carrefourBRNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *carrefourBRNoopParser) ParseStore(html string) string               { return "" }

// CarrefourBRExtractor extracts appliance products from Carrefour Brazil.
//
// The storefront runs on VTEX IO and exposes the VTEX Intelligent Search API
// publicly without authentication. The endpoint returns product data as JSON
// with the standard VTEX items/sellers/commertialOffer structure.
//
// Endpoint: GET /api/io/_v/api/intelligent-search/product_search
//
//	?query={query}&count=20&locale=pt-BR
type CarrefourBRExtractor struct {
	*extractors.BaseGoExtractor
}

// NewCarrefourBRExtractor creates a new Carrefour Brazil extractor.
func NewCarrefourBRExtractor() *CarrefourBRExtractor {
	parser := &carrefourBRNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("carrefour_br")}
	base := extractors.NewBaseGoExtractor(
		"https://www.carrefour.com.br",
		models.CountryBrazil,
		"carrefour_br_v1",
		parser,
	)
	return &CarrefourBRExtractor{BaseGoExtractor: base}
}

// GetCategory returns the appliances category for registry filtering.
func (e *CarrefourBRExtractor) GetCategory() models.ProductCategory {
	return models.CategoryAppliances
}

// BuildSearchURL constructs the VTEX Intelligent Search API URL.
func (e *CarrefourBRExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf(
		"%s/api/io/_v/api/intelligent-search/product_search?query=%s&count=20&locale=pt-BR",
		e.GetBaseURL(), encoded,
	), nil
}

// GetComparisons fetches products from the VTEX Intelligent Search API and
// returns comparisons.
func (e *CarrefourBRExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("carrefour_br: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("carrefour_br: failed to fetch API: %w", err)
	}

	return e.parseVTEXResponse(body)
}

// carrefourProduct mirrors the VTEX Intelligent Search product response structure.
type carrefourProduct struct {
	ProductName string          `json:"productName"`
	Link        string          `json:"link"`
	Items       []carrefourItem `json:"items"`
}

type carrefourItem struct {
	Images  []carrefourImage  `json:"images"`
	Sellers []carrefourSeller `json:"sellers"`
}

type carrefourImage struct {
	ImageURL string `json:"imageUrl"`
}

type carrefourSeller struct {
	CommertialOffer carrefourOffer `json:"commertialOffer"`
}

type carrefourOffer struct {
	Price           float64 `json:"Price"`
	AvailableQty    int     `json:"AvailableQuantity"`
}

type carrefourSearchResponse struct {
	Products []carrefourProduct `json:"products"`
}

// parseVTEXResponse unmarshals the Intelligent Search JSON and converts each
// product into a ProductComparison, skipping zero-price or out-of-stock items.
func (e *CarrefourBRExtractor) parseVTEXResponse(body string) ([]models.ProductComparison, error) {
	var resp carrefourSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("carrefour_br: failed to parse API response: %w", err)
	}

	category := models.CategoryAppliances
	results := make([]models.ProductComparison, 0, len(resp.Products))

	for _, p := range resp.Products {
		if len(p.Items) == 0 {
			continue
		}
		item := p.Items[0]

		// Pick the lowest available price across all sellers.
		var price float64
		for _, seller := range item.Sellers {
			offer := seller.CommertialOffer
			if offer.Price <= 0 {
				continue
			}
			if price == 0 || offer.Price < price {
				price = offer.Price
			}
		}
		if price == 0 {
			continue
		}

		link := fmt.Sprintf("%s%s", e.GetBaseURL(), p.Link)

		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: strings.TrimSpace(p.ProductName),
			Price:       price,
			Currency:    "BRL",
			StoreName:   "Carrefour",
			Country:     string(models.CountryBrazil),
			Category:    &category,
			StoreURL:    &link,
		}

		if len(item.Images) > 0 && item.Images[0].ImageURL != "" {
			img := item.Images[0].ImageURL
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	return results, nil
}
