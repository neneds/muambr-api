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

// sephoraBRNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// SephoraBRExtractor overrides GetComparisons entirely and parses a GTM dataLayer
// block from the SFCC Search-Show AJAX response, so these methods are never called.
type sephoraBRNoopParser struct{ *extractors.BaseHTMLParser }

func (p *sephoraBRNoopParser) GetProductSelectors() []string                   { return nil }
func (p *sephoraBRNoopParser) GetNameSelectors() []string                      { return nil }
func (p *sephoraBRNoopParser) GetPriceSelectors() []string                     { return nil }
func (p *sephoraBRNoopParser) GetURLSelectors() []string                       { return nil }
func (p *sephoraBRNoopParser) ParseProductName(html string) string             { return "" }
func (p *sephoraBRNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *sephoraBRNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *sephoraBRNoopParser) ParseStore(html string) string               { return "" }

// SephoraBRExtractor extracts beauty products from Sephora Brazil.
//
// The storefront is a Salesforce Commerce Cloud (Demandware) application. The
// standard SFCC Search-Show controller accepts a ?format=ajax parameter and
// returns an HTML fragment that embeds a window.dataLayer.push() call containing
// a JSON products array. This extractor fetches that fragment, locates the
// 'products': [...] value by bracket-matching, and unmarshals it directly.
//
// Endpoint: GET /on/demandware.store/Sites-Sephora_BR-Site/pt_BR/Search-Show
//
//	?q={query}&format=ajax&start=0&sz=20
type SephoraBRExtractor struct {
	*extractors.BaseGoExtractor
}

// NewSephoraBRExtractor creates a new Sephora Brazil extractor.
func NewSephoraBRExtractor() *SephoraBRExtractor {
	parser := &sephoraBRNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("sephora_br")}
	base := extractors.NewBaseGoExtractor(
		"https://www.sephora.com.br",
		models.CountryBrazil,
		"sephora_br_v1",
		parser,
	)
	return &SephoraBRExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *SephoraBRExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the SFCC Search-Show AJAX URL.
func (e *SephoraBRExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf(
		"%s/on/demandware.store/Sites-Sephora_BR-Site/pt_BR/Search-Show?q=%s&format=ajax&start=0&sz=20",
		e.GetBaseURL(), encoded,
	), nil
}

// GetComparisons fetches the SFCC search page and extracts products from the
// embedded dataLayer block.
func (e *SephoraBRExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("sephora_br: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("sephora_br: failed to fetch search page: %w", err)
	}

	return e.GetComparisonsFromHTML(body)
}

// GetComparisonsFromHTML parses the SFCC AJAX HTML fragment and extracts product
// data from the embedded GTM dataLayer products array. Exposed for unit testing.
func (e *SephoraBRExtractor) GetComparisonsFromHTML(body string) ([]models.ProductComparison, error) {
	products, err := extractSephoraBRProducts(body)
	if err != nil {
		return nil, fmt.Errorf("sephora_br: %w", err)
	}

	category := models.CategoryBeauty
	results := make([]models.ProductComparison, 0, len(products))

	for _, p := range products {
		if p.Price <= 0 || strings.TrimSpace(p.Name) == "" {
			continue
		}

		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: strings.TrimSpace(p.Name),
			Price:       p.Price,
			Currency:    "BRL",
			StoreName:   "Sephora",
			Country:     string(models.CountryBrazil),
			Category:    &category,
		}

		if p.Link != "" {
			link := p.Link
			comparison.StoreURL = &link
		}

		results = append(results, comparison)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no products found in dataLayer response")
	}

	return results, nil
}

// sephoraProduct mirrors the product objects in Sephora BR's GTM dataLayer push.
type sephoraProduct struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"` // may be int or decimal (e.g. 226.1)
	Link  string  `json:"link"`
}

// extractSephoraBRProducts locates the 'products': [...] value inside the
// window.dataLayer.push block and unmarshals it. The array is found by
// bracket-matching starting from the first '[' after the key, making the
// extraction robust to whitespace and ordering changes in surrounding fields.
func extractSephoraBRProducts(html string) ([]sephoraProduct, error) {
	const marker = "'products': ["

	idx := strings.Index(html, marker)
	if idx == -1 {
		return nil, fmt.Errorf("dataLayer products array not found in response")
	}

	start := idx + len(marker) - 1 // points to '['
	depth := 0
	end := -1

	for i := start; i < len(html); i++ {
		switch html[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				end = i + 1
			}
		}
		if end > 0 {
			break
		}
	}

	if end == -1 {
		return nil, fmt.Errorf("dataLayer products array is malformed (unmatched brackets)")
	}

	var products []sephoraProduct
	if err := json.Unmarshal([]byte(html[start:end]), &products); err != nil {
		return nil, fmt.Errorf("failed to parse dataLayer products JSON: %w", err)
	}

	return products, nil
}
