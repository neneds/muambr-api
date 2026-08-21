package other

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

// auchanPTNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// AuchanPTExtractor overrides GetComparisons entirely and parses SFCC product-tile
// GTM JSON from the search HTML, so these methods are never called.
type auchanPTNoopParser struct{ *extractors.BaseHTMLParser }

func (p *auchanPTNoopParser) GetProductSelectors() []string       { return nil }
func (p *auchanPTNoopParser) GetNameSelectors() []string          { return nil }
func (p *auchanPTNoopParser) GetPriceSelectors() []string         { return nil }
func (p *auchanPTNoopParser) GetURLSelectors() []string           { return nil }
func (p *auchanPTNoopParser) ParseProductName(html string) string { return "" }
func (p *auchanPTNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *auchanPTNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *auchanPTNoopParser) ParseStore(html string) string               { return "" }

// auchanTileAttrRe matches data-urls then data-gtm on the same product-tile opening tag.
var auchanTileAttrRe = regexp.MustCompile(`data-urls="([^"]+)"[^>]*data-gtm="([^"]+)"`)

// AuchanPTExtractor extracts generic retail products from Auchan Portugal.
//
// The storefront is Salesforce Commerce Cloud. Search-Show?format=ajax redirects
// to the full search page. /pt/pesquisa?q= embeds name, id, and price on each
// tile as data-gtm JSON, plus absoluteProductUrl in data-urls.
//
// Endpoint: GET https://www.auchan.pt/pt/pesquisa?q={query}
type AuchanPTExtractor struct {
	*extractors.BaseGoExtractor
}

// NewAuchanPTExtractor creates a new Auchan Portugal extractor.
func NewAuchanPTExtractor() *AuchanPTExtractor {
	parser := &auchanPTNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("auchan_pt")}
	base := extractors.NewBaseGoExtractor(
		"https://www.auchan.pt",
		models.CountryPortugal,
		"auchan_pt_v1",
		parser,
	)
	return &AuchanPTExtractor{BaseGoExtractor: base}
}

// GetCategory returns the generic/other category for registry filtering.
func (e *AuchanPTExtractor) GetCategory() models.ProductCategory {
	return models.CategoryOther
}

// BuildSearchURL constructs the Auchan SFCC search page URL.
func (e *AuchanPTExtractor) BuildSearchURL(productName string) (string, error) {
	encoded := url.QueryEscape(productName)
	return fmt.Sprintf("%s/pt/pesquisa?q=%s", e.GetBaseURL(), encoded), nil
}

// GetComparisons fetches the Auchan search page and extracts product tiles.
func (e *AuchanPTExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("auchan_pt: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("auchan_pt: failed to fetch search page: %w", err)
	}

	return e.GetComparisonsFromHTML(body)
}

// GetComparisonsFromHTML parses product tiles from Auchan search HTML.
// Exposed for unit testing.
func (e *AuchanPTExtractor) GetComparisonsFromHTML(body string) ([]models.ProductComparison, error) {
	matches := auchanTileAttrRe.FindAllStringSubmatch(body, -1)
	category := models.CategoryOther
	results := make([]models.ProductComparison, 0, len(matches))
	seen := make(map[string]struct{})

	for _, m := range matches {
		var urls auchanTileURLs
		if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &urls); err != nil {
			utils.Warn("auchan_pt: skipped malformed tile urls", utils.Error(err))
			continue
		}
		var tile auchanTileGTM
		if err := json.Unmarshal([]byte(html.UnescapeString(m[2])), &tile); err != nil {
			utils.Warn("auchan_pt: skipped malformed tile gtm", utils.Error(err))
			continue
		}

		name := strings.TrimSpace(tile.Name)
		price := tile.Price.Float64()
		if name == "" || price <= 0 {
			continue
		}

		id := strings.TrimSpace(tile.ID)
		if id == "" {
			id = utils.GenerateUUID()
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Auchan",
			Country:     string(models.CountryPortugal),
			Category:    &category,
		}

		link := strings.TrimSpace(urls.AbsoluteProductURL)
		if link == "" {
			link = strings.TrimSpace(urls.ProductURL)
		}
		if link != "" {
			if !strings.HasPrefix(link, "http") {
				link = e.GetBaseURL() + link
			}
			comparison.StoreURL = &link
		}

		results = append(results, comparison)
	}

	utils.Info("Auchan PT extraction completed", utils.Int("results", len(results)))
	return results, nil
}

type auchanTileURLs struct {
	ProductURL         string `json:"productUrl"`
	AbsoluteProductURL string `json:"absoluteProductUrl"`
}

type auchanTileGTM struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Price auchanAmount `json:"price"`
}

// auchanAmount accepts JSON numbers or numeric strings (Auchan GTM uses strings).
type auchanAmount float64

func (a auchanAmount) Float64() float64 { return float64(a) }

func (a *auchanAmount) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*a = 0
		return nil
	}
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*a = auchanAmount(n)
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
	*a = auchanAmount(n)
	return nil
}
