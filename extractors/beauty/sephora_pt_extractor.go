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

// sephoraPTNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// SephoraPTExtractor overrides GetComparisons entirely and uses the Constructor.io
// search JSON API, so the parser methods are never called.
type sephoraPTNoopParser struct{ *extractors.BaseHTMLParser }

func (p *sephoraPTNoopParser) GetProductSelectors() []string       { return nil }
func (p *sephoraPTNoopParser) GetNameSelectors() []string          { return nil }
func (p *sephoraPTNoopParser) GetPriceSelectors() []string         { return nil }
func (p *sephoraPTNoopParser) GetURLSelectors() []string           { return nil }
func (p *sephoraPTNoopParser) ParseProductName(html string) string { return "" }
func (p *sephoraPTNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *sephoraPTNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *sephoraPTNoopParser) ParseStore(html string) string               { return "" }

const sephoraPTConstructorSearchURL = "https://ac.cnstrc.com/search/"
const sephoraPTConstructorKey = "key_rLqnH7Y10YHnrl7W"

// SephoraPTExtractor extracts beauty products from Sephora Portugal.
//
// Storefront search (/procurar) is a Next.js page whose JSON-LD ItemList is
// incomplete (often a single product). Results come from Constructor.io's
// public JSON search API using the storefront key. Do not scrape the HTML
// search page; it is not the same as sephora_br_v1 (SFCC Search-Show ajax).
//
// Endpoint: GET https://ac.cnstrc.com/search/{query}?key=...&c=ciojs-2.1458.0
type SephoraPTExtractor struct {
	*extractors.BaseGoExtractor
}

// NewSephoraPTExtractor creates a new Sephora Portugal extractor.
func NewSephoraPTExtractor() *SephoraPTExtractor {
	parser := &sephoraPTNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("sephora_pt")}
	base := extractors.NewBaseGoExtractor(
		"https://www.sephora.pt",
		models.CountryPortugal,
		"sephora_pt_v1",
		parser,
	)
	return &SephoraPTExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *SephoraPTExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Constructor.io search API URL for Sephora Portugal.
func (e *SephoraPTExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.PathEscape(productName)
	params := url.Values{}
	params.Set("key", sephoraPTConstructorKey)
	params.Set("c", "ciojs-2.1458.0")
	return sephoraPTConstructorSearchURL + encoded + "?" + params.Encode(), nil
}

// GetComparisons fetches products from Constructor.io and returns comparisons.
func (e *SephoraPTExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("sephora_pt: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("sephora_pt: failed to fetch API: %w", err)
	}

	return e.parseConstructorResponse(body)
}

type sephoraPTSearchResponse struct {
	Response sephoraPTResponse `json:"response"`
}

type sephoraPTResponse struct {
	Results []sephoraPTResult `json:"results"`
}

type sephoraPTResult struct {
	Value string           `json:"value"`
	Data  sephoraPTProduct `json:"data"`
}

type sephoraPTProduct struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	URL      string         `json:"url"`
	ImageURL string         `json:"image_url"`
	Price    sephoraPTPrice `json:"price"`
}

type sephoraPTPrice struct {
	Currency     string  `json:"currency"`
	SalesPrice   float64 `json:"salesPrice"`
	InitialPrice float64 `json:"initialPrice"`
	MinPrice     float64 `json:"minPrice"`
}

func (e *SephoraPTExtractor) parseConstructorResponse(body string) ([]models.ProductComparison, error) {
	var resp sephoraPTSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("sephora_pt: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	results := make([]models.ProductComparison, 0, len(resp.Response.Results))

	for _, r := range resp.Response.Results {
		p := r.Data
		price := p.Price.SalesPrice
		if price <= 0 {
			price = p.Price.InitialPrice
		}
		if price <= 0 {
			price = p.Price.MinPrice
		}
		if price <= 0 {
			continue
		}

		name := strings.TrimSpace(r.Value)
		if name == "" {
			name = strings.TrimSpace(p.Name)
		}
		if name == "" {
			continue
		}

		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = utils.GenerateUUID()
		}

		currency := strings.TrimSpace(p.Price.Currency)
		if currency == "" {
			currency = "EUR"
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    currency,
			StoreName:   "Sephora",
			Country:     string(models.CountryPortugal),
			Category:    &category,
		}

		if p.URL != "" {
			link := p.URL
			if !strings.HasPrefix(link, "http") {
				link = e.GetBaseURL() + link
			}
			comparison.StoreURL = &link
		}
		if p.ImageURL != "" {
			img := p.ImageURL
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Sephora PT extraction completed", utils.Int("results", len(results)))
	return results, nil
}
