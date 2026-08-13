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

// primorPTNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// PrimorPTExtractor overrides GetComparisons entirely and uses the Empathy search JSON API,
// so the parser methods are never called.
type primorPTNoopParser struct{ *extractors.BaseHTMLParser }

func (p *primorPTNoopParser) GetProductSelectors() []string                   { return nil }
func (p *primorPTNoopParser) GetNameSelectors() []string                      { return nil }
func (p *primorPTNoopParser) GetPriceSelectors() []string                     { return nil }
func (p *primorPTNoopParser) GetURLSelectors() []string                       { return nil }
func (p *primorPTNoopParser) ParseProductName(html string) string             { return "" }
func (p *primorPTNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *primorPTNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *primorPTNoopParser) ParseStore(html string) string               { return "" }

const primorEmpathySearchURL = "https://api.empathy.co/search/v1/query/primor/search"

// PrimorPTExtractor extracts beauty products from Primor Portugal.
//
// The storefront (pt.primor.eu) is behind AWS WAF, so HTML search cannot be used.
// Search results are served by Empathy's public JSON API, which returns catalog
// hits with title, salePrice, product URL, and image — no auth required.
//
// Endpoint: GET https://api.empathy.co/search/v1/query/primor/search
//
//	?internal=true&query={q}&origin=search_box:none&start=0&rows=20
//	&instance=primor&scope=desktop&lang=pt&store=pt
type PrimorPTExtractor struct {
	*extractors.BaseGoExtractor
}

// NewPrimorPTExtractor creates a new Primor Portugal extractor.
func NewPrimorPTExtractor() *PrimorPTExtractor {
	parser := &primorPTNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("primor_pt")}
	base := extractors.NewBaseGoExtractor(
		"https://pt.primor.eu",
		models.CountryPortugal,
		"primor_pt_v1",
		parser,
	)
	return &PrimorPTExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *PrimorPTExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Empathy search API URL for the Portugal store.
func (e *PrimorPTExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("internal", "true")
	params.Set("query", productName)
	params.Set("origin", "search_box:none")
	params.Set("start", "0")
	params.Set("rows", "20")
	params.Set("instance", "primor")
	params.Set("scope", "desktop")
	params.Set("lang", "pt")
	params.Set("store", "pt")
	return primorEmpathySearchURL + "?" + params.Encode(), nil
}

// GetComparisons fetches products from the Empathy search API and returns comparisons.
func (e *PrimorPTExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("primor_pt: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("primor_pt: failed to fetch API: %w", err)
	}

	return e.parseEmpathyResponse(body)
}

type primorSearchResponse struct {
	Catalog primorCatalog `json:"catalog"`
}

type primorCatalog struct {
	Content []primorProduct `json:"content"`
}

type primorProduct struct {
	ID           string       `json:"id"`
	SKU          string       `json:"sku"`
	Title        string       `json:"title"`
	Name         string       `json:"__name"`
	SalePrice    float64      `json:"salePrice"`
	Prices       primorPrices `json:"__prices"`
	Link         string       `json:"link"`
	URL          string       `json:"__url"`
	ImageLink    string       `json:"imageLink"`
	Availability string       `json:"availability"`
	VariantValue string       `json:"variantValue"`
	Brand        string       `json:"brand"`
}

type primorPrices struct {
	Current primorPriceValue `json:"current"`
}

type primorPriceValue struct {
	Value float64 `json:"value"`
}

func (e *PrimorPTExtractor) parseEmpathyResponse(body string) ([]models.ProductComparison, error) {
	var resp primorSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("primor_pt: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	results := make([]models.ProductComparison, 0, len(resp.Catalog.Content))

	for _, p := range resp.Catalog.Content {
		if strings.EqualFold(p.Availability, "false") {
			continue
		}

		price := p.SalePrice
		if price <= 0 {
			price = p.Prices.Current.Value
		}
		if price <= 0 {
			continue
		}

		name := strings.TrimSpace(p.Title)
		if name == "" {
			name = strings.TrimSpace(p.Name)
		}
		if name == "" {
			continue
		}
		if variant := strings.TrimSpace(p.VariantValue); variant != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(variant)) {
			name = name + " " + variant
		}

		id := p.SKU
		if id == "" {
			id = p.ID
		}
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Primor",
			Country:     string(models.CountryPortugal),
			Category:    &category,
		}

		productURL := p.Link
		if productURL == "" {
			productURL = p.URL
		}
		if productURL != "" {
			comparison.StoreURL = &productURL
		}

		if p.ImageLink != "" {
			img := p.ImageLink
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Primor PT extraction completed", utils.Int("results", len(results)))
	return results, nil
}
