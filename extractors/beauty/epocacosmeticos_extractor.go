package beauty

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// epocaCosmeticosNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// EpocaCosmeticosExtractor overrides GetComparisons entirely and uses the VTEX JSON API,
// so the parser methods are never called.
type epocaCosmeticosNoopParser struct{ *extractors.BaseHTMLParser }

func (p *epocaCosmeticosNoopParser) GetProductSelectors() []string                   { return nil }
func (p *epocaCosmeticosNoopParser) GetNameSelectors() []string                      { return nil }
func (p *epocaCosmeticosNoopParser) GetPriceSelectors() []string                     { return nil }
func (p *epocaCosmeticosNoopParser) GetURLSelectors() []string                       { return nil }
func (p *epocaCosmeticosNoopParser) ParseProductName(html string) string             { return "" }
func (p *epocaCosmeticosNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *epocaCosmeticosNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *epocaCosmeticosNoopParser) ParseStore(html string) string               { return "" }

// EpocaCosmeticosExtractor extracts beauty products from Época Cosméticos (Brazil).
//
// The storefront is a Next.js/VTEX FastStore application that renders product
// listings entirely client-side, so scraping the initial HTML yields only loading
// skeletons. Instead this extractor calls the VTEX catalog search API which
// returns product data as JSON and is publicly accessible without authentication.
//
// API endpoint: GET /api/catalog_system/pub/products/search?ft={query}&_from=0&_to=19
type EpocaCosmeticosExtractor struct {
	*extractors.BaseGoExtractor
}

// NewEpocaCosmeticosExtractor creates a new Época Cosméticos extractor.
func NewEpocaCosmeticosExtractor() *EpocaCosmeticosExtractor {
	parser := &epocaCosmeticosNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("epocacosmeticos")}
	base := extractors.NewBaseGoExtractor(
		"https://www.epocacosmeticos.com.br",
		models.CountryBrazil,
		"epocacosmeticos_v1",
		parser,
	)
	return &EpocaCosmeticosExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *EpocaCosmeticosExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the VTEX catalog search API URL.
func (e *EpocaCosmeticosExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf("%s/api/catalog_system/pub/products/search?ft=%s&_from=0&_to=19", e.GetBaseURL(), encoded), nil
}

// GetComparisons fetches products from the VTEX search API and returns comparisons.
func (e *EpocaCosmeticosExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("epocacosmeticos: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("epocacosmeticos: failed to fetch API: %w", err)
	}

	return e.parseVTEXResponse(body)
}

// vtexProduct mirrors the VTEX catalog search API response structure.
type vtexProduct struct {
	ProductName string     `json:"productName"`
	Brand       string     `json:"brand"`
	Link        string     `json:"link"`
	Description string     `json:"description"`
	Items       []vtexItem `json:"items"`
}

type vtexItem struct {
	Images  []vtexImage  `json:"images"`
	Sellers []vtexSeller `json:"sellers"`
}

type vtexImage struct {
	ImageURL string `json:"imageUrl"`
}

type vtexSeller struct {
	CommercialOffer vtexOffer `json:"commertialOffer"`
}

type vtexOffer struct {
	Price       float64 `json:"Price"`
	IsAvailable bool    `json:"IsAvailable"`
}

// parseVTEXResponse unmarshals the JSON body and converts each product into a
// ProductComparison, skipping unavailable or zero-price items.
func (e *EpocaCosmeticosExtractor) parseVTEXResponse(body string) ([]models.ProductComparison, error) {
	var products []vtexProduct
	if err := json.Unmarshal([]byte(body), &products); err != nil {
		return nil, fmt.Errorf("epocacosmeticos: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	storeName := "Época Cosméticos"
	results := make([]models.ProductComparison, 0, len(products))

	for _, p := range products {
		if len(p.Items) == 0 {
			continue
		}
		item := p.Items[0]

		// Pick the lowest available price across sellers.
		var price float64
		for _, seller := range item.Sellers {
			offer := seller.CommercialOffer
			if !offer.IsAvailable || offer.Price <= 0 {
				continue
			}
			if price == 0 || offer.Price < price {
				price = offer.Price
			}
		}
		if price == 0 {
			continue
		}

		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: strings.TrimSpace(p.ProductName),
			Price:       price,
			Currency:    "BRL",
			StoreName:   storeName,
			Country:     string(models.CountryBrazil),
			Category:    &category,
		}

		if p.Link != "" {
			link := p.Link
			comparison.StoreURL = &link
		}

		if len(item.Images) > 0 && item.Images[0].ImageURL != "" {
			img := item.Images[0].ImageURL
			comparison.ImageURL = &img
		}

		if desc := strings.TrimSpace(p.Description); desc != "" {
			comparison.Description = &desc
		}

		results = append(results, comparison)
	}

	utils.Info("Época Cosméticos extraction completed", utils.Int("results", len(results)))
	return results, nil
}
