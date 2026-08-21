package beauty

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// druniESNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// DruniESExtractor overrides GetComparisons entirely and uses the Doofinder
// search JSON API, so the parser methods are never called.
type druniESNoopParser struct{ *extractors.BaseHTMLParser }

func (p *druniESNoopParser) GetProductSelectors() []string       { return nil }
func (p *druniESNoopParser) GetNameSelectors() []string          { return nil }
func (p *druniESNoopParser) GetPriceSelectors() []string         { return nil }
func (p *druniESNoopParser) GetURLSelectors() []string           { return nil }
func (p *druniESNoopParser) ParseProductName(html string) string { return "" }
func (p *druniESNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *druniESNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *druniESNoopParser) ParseStore(html string) string               { return "" }

const druniESDoofinderSearchURL = "https://eu1-search.doofinder.com/5/search"
const druniESDoofinderHashID = "8ed7450f44117ffe10dedbb105484e0e"

// DruniESExtractor extracts beauty products from Druni (Spain).
//
// Magento catalogsearch returns HTTP 406 from a plain client. Search results
// come from Doofinder's public JSON API (hashid from the storefront layer).
// Origin is required. Do not scrape Magento HTML search.
//
// Endpoint: GET https://eu1-search.doofinder.com/5/search?hashid=...&query={q}&rpp=20
type DruniESExtractor struct {
	*extractors.BaseGoExtractor
}

// NewDruniESExtractor creates a new Druni Spain extractor.
func NewDruniESExtractor() *DruniESExtractor {
	parser := &druniESNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("druni_es")}
	base := extractors.NewBaseGoExtractor(
		"https://www.druni.es",
		models.CountrySpain,
		"druni_es_v1",
		parser,
	)
	return &DruniESExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *DruniESExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Doofinder v5 search API URL for Druni Spain.
func (e *DruniESExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("hashid", druniESDoofinderHashID)
	params.Set("query", productName)
	params.Set("rpp", "20")
	return druniESDoofinderSearchURL + "?" + params.Encode(), nil
}

// FetchHTML overrides the base client because Doofinder rejects requests that
// omit the storefront Origin ("request not authenticated"). This is a CORS-style
// gate, not a WAF challenge.
func (e *DruniESExtractor) FetchHTML(targetURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("druni_es: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.druni.es")
	req.Header.Set("Referer", "https://www.druni.es/")

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("druni_es: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("druni_es: failed to read response: %w", err)
	}
	return string(body), nil
}

// GetComparisons fetches products from Doofinder and returns comparisons.
func (e *DruniESExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("druni_es: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("druni_es: failed to fetch API: %w", err)
	}

	return e.parseDoofinderResponse(body)
}

type druniESSearchResponse struct {
	Results []druniESProduct `json:"results"`
}

type druniESProduct struct {
	ID           json.Number `json:"id"`
	Title        string      `json:"title"`
	Price        float64     `json:"price"`
	SalePrice    float64     `json:"sale_price"`
	BestPrice    float64     `json:"best_price"`
	Link         string      `json:"link"`
	ImageLink    string      `json:"image_link"`
	Availability string      `json:"availability"`
}

func (e *DruniESExtractor) parseDoofinderResponse(body string) ([]models.ProductComparison, error) {
	var resp druniESSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("druni_es: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	results := make([]models.ProductComparison, 0, len(resp.Results))

	for _, p := range resp.Results {
		if strings.EqualFold(p.Availability, "out_of_stock") {
			continue
		}

		price := p.SalePrice
		if price <= 0 {
			price = p.BestPrice
		}
		if price <= 0 {
			price = p.Price
		}
		if price <= 0 {
			continue
		}

		name := strings.TrimSpace(p.Title)
		if name == "" {
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
			StoreName:   "Druni",
			Country:     string(models.CountrySpain),
			Category:    &category,
		}

		if p.Link != "" {
			link := p.Link
			comparison.StoreURL = &link
		}
		if p.ImageLink != "" {
			img := p.ImageLink
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Druni ES extraction completed", utils.Int("results", len(results)))
	return results, nil
}
