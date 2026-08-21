package beauty

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// wellsPTNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// WellsPTExtractor overrides GetComparisons entirely and parses SFCC product-tile
// impression JSON from the search HTML, so these methods are never called.
type wellsPTNoopParser struct{ *extractors.BaseHTMLParser }

func (p *wellsPTNoopParser) GetProductSelectors() []string       { return nil }
func (p *wellsPTNoopParser) GetNameSelectors() []string          { return nil }
func (p *wellsPTNoopParser) GetPriceSelectors() []string         { return nil }
func (p *wellsPTNoopParser) GetURLSelectors() []string           { return nil }
func (p *wellsPTNoopParser) ParseProductName(html string) string { return "" }
func (p *wellsPTNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *wellsPTNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *wellsPTNoopParser) ParseStore(html string) string               { return "" }

// wellsTileAttrRe matches a product tile's relative URL and GTM impression JSON.
// Both attributes live on the same opening tag; url comes first in the live HTML.
var wellsTileAttrRe = regexp.MustCompile(`data-product-tile-url="([^"]+)"[^>]*data-product-tile-impression="([^"]+)"`)

// WellsPTExtractor extracts beauty products from Wells Portugal.
//
// The storefront is Salesforce Commerce Cloud. Search-Show?format=ajax redirects
// to a fragment with no product tiles. The full search page at
// /resultados-pesquisa-wells?q= embeds name, SKU, and prices on each tile as
// data-product-tile-impression JSON. pvp is the current selling price
// (w-sales-price); price is the struck-through PVPR list price.
//
// Endpoint: GET https://wells.pt/resultados-pesquisa-wells?q={query}
type WellsPTExtractor struct {
	*extractors.BaseGoExtractor
}

// NewWellsPTExtractor creates a new Wells Portugal extractor.
func NewWellsPTExtractor() *WellsPTExtractor {
	parser := &wellsPTNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("wells_pt")}
	base := extractors.NewBaseGoExtractor(
		"https://wells.pt",
		models.CountryPortugal,
		"wells_pt_v1",
		parser,
	)
	return &WellsPTExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *WellsPTExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Wells SFCC search page URL.
func (e *WellsPTExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf("%s/resultados-pesquisa-wells?q=%s", e.GetBaseURL(), encoded), nil
}

// GetComparisons fetches the Wells search page and extracts product tiles.
func (e *WellsPTExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("wells_pt: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("wells_pt: failed to fetch search page: %w", err)
	}

	return e.GetComparisonsFromHTML(body)
}

// GetComparisonsFromHTML parses product tiles from Wells search HTML.
// Exposed for unit testing.
func (e *WellsPTExtractor) GetComparisonsFromHTML(body string) ([]models.ProductComparison, error) {
	matches := wellsTileAttrRe.FindAllStringSubmatch(body, -1)
	category := models.CategoryBeauty
	results := make([]models.ProductComparison, 0, len(matches))

	for _, m := range matches {
		rawURL := html.UnescapeString(m[1])
		var tile wellsTileImpression
		if err := json.Unmarshal([]byte(html.UnescapeString(m[2])), &tile); err != nil {
			utils.Warn("wells_pt: skipped malformed tile impression", utils.Error(err))
			continue
		}
		if tile.Notify {
			continue
		}

		name := strings.TrimSpace(tile.Name)
		if name == "" {
			continue
		}

		price := tile.PVP.Float64()
		if price <= 0 {
			price = tile.Price.Float64()
		}
		if price <= 0 {
			continue
		}

		id := strings.TrimSpace(tile.SKU)
		if id == "" {
			id = strings.TrimSpace(tile.ID)
		}
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Wells",
			Country:     string(models.CountryPortugal),
			Category:    &category,
		}

		if rawURL != "" {
			link := rawURL
			if !strings.HasPrefix(link, "http") {
				link = e.GetBaseURL() + link
			}
			comparison.StoreURL = &link
		}

		results = append(results, comparison)
	}

	utils.Info("Wells PT extraction completed", utils.Int("results", len(results)))
	return results, nil
}

// wellsTileImpression mirrors data-product-tile-impression JSON on Wells product tiles.
type wellsTileImpression struct {
	ID     string      `json:"id"`
	SKU    string      `json:"sku"`
	Name   string      `json:"name"`
	Price  wellsAmount `json:"price"`
	PVP    wellsAmount `json:"pvp"`
	Notify bool        `json:"notify"`
}

// wellsAmount accepts JSON numbers or numeric strings (Wells mixes both for pvp).
type wellsAmount float64

func (a wellsAmount) Float64() float64 { return float64(a) }

func (a *wellsAmount) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*a = 0
		return nil
	}
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*a = wellsAmount(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		*a = 0
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*a = wellsAmount(n)
	return nil
}
