package beauty

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// boozyshopNLNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// BoozyshopNLExtractor overrides GetComparisons entirely and uses Shopify's
// predictive-search JSON API, so the parser methods are never called.
type boozyshopNLNoopParser struct{ *extractors.BaseHTMLParser }

func (p *boozyshopNLNoopParser) GetProductSelectors() []string       { return nil }
func (p *boozyshopNLNoopParser) GetNameSelectors() []string          { return nil }
func (p *boozyshopNLNoopParser) GetPriceSelectors() []string         { return nil }
func (p *boozyshopNLNoopParser) GetURLSelectors() []string           { return nil }
func (p *boozyshopNLNoopParser) ParseProductName(html string) string { return "" }
func (p *boozyshopNLNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *boozyshopNLNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *boozyshopNLNoopParser) ParseStore(html string) string               { return "" }

// BoozyshopNLExtractor extracts beauty products from Boozyshop (Netherlands).
//
// Search HTML is parseable but Shopify's public predictive-search JSON is the
// stable source. Do not scrape /search HTML.
//
// Endpoint: GET https://www.boozyshop.nl/search/suggest.json?q={q}&resources[type]=product&resources[limit]=20
type BoozyshopNLExtractor struct {
	*extractors.BaseGoExtractor
}

// NewBoozyshopNLExtractor creates a new Boozyshop Netherlands extractor.
func NewBoozyshopNLExtractor() *BoozyshopNLExtractor {
	parser := &boozyshopNLNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("boozyshop_nl")}
	base := extractors.NewBaseGoExtractor(
		"https://www.boozyshop.nl",
		models.CountryNetherlands,
		"boozyshop_nl_v1",
		parser,
	)
	return &BoozyshopNLExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *BoozyshopNLExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Shopify predictive-search JSON URL.
func (e *BoozyshopNLExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("q", productName)
	params.Set("resources[type]", "product")
	params.Set("resources[limit]", "20")
	return e.GetBaseURL() + "/search/suggest.json?" + params.Encode(), nil
}

// GetComparisons fetches products from Shopify predictive search and returns comparisons.
func (e *BoozyshopNLExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("boozyshop_nl: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("boozyshop_nl: failed to fetch API: %w", err)
	}

	return e.parseSuggestResponse(body)
}

type boozyshopSuggestResponse struct {
	Resources boozyshopResources `json:"resources"`
}

type boozyshopResources struct {
	Results boozyshopResults `json:"results"`
}

type boozyshopResults struct {
	Products []boozyshopProduct `json:"products"`
}

type boozyshopProduct struct {
	ID        json.Number     `json:"id"`
	Title     string          `json:"title"`
	Price     json.Number     `json:"price"`
	URL       string          `json:"url"`
	Image     json.RawMessage `json:"image"`
	Available bool            `json:"available"`
}

func (e *BoozyshopNLExtractor) parseSuggestResponse(body string) ([]models.ProductComparison, error) {
	var resp boozyshopSuggestResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("boozyshop_nl: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	products := resp.Resources.Results.Products
	results := make([]models.ProductComparison, 0, len(products))

	for _, p := range products {
		if !p.Available {
			continue
		}
		name := strings.TrimSpace(p.Title)
		price, err := strconv.ParseFloat(string(p.Price), 64)
		if name == "" || err != nil || price <= 0 {
			continue
		}

		id := strings.TrimSpace(p.ID.String())
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Boozyshop",
			Country:     string(models.CountryNetherlands),
			Category:    &category,
		}

		if link := e.absoluteURL(p.URL); link != "" {
			comparison.StoreURL = &link
		}
		if img := shopifyImageURL(p.Image); img != "" {
			if strings.HasPrefix(img, "//") {
				img = "https:" + img
			}
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Boozyshop NL extraction completed", utils.Int("results", len(results)))
	return results, nil
}

func (e *BoozyshopNLExtractor) absoluteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return e.GetBaseURL() + raw
	}
	return e.GetBaseURL() + "/" + raw
}

func shopifyImageURL(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		URL string `json:"url"`
		Src string `json:"src"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	if strings.TrimSpace(obj.URL) != "" {
		return strings.TrimSpace(obj.URL)
	}
	return strings.TrimSpace(obj.Src)
}
