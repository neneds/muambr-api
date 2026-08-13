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

// pecPTNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// PerfumesECompanhiaPTExtractor overrides GetComparisons entirely and uses the
// Doofinder search JSON API, so the parser methods are never called.
type pecPTNoopParser struct{ *extractors.BaseHTMLParser }

func (p *pecPTNoopParser) GetProductSelectors() []string       { return nil }
func (p *pecPTNoopParser) GetNameSelectors() []string          { return nil }
func (p *pecPTNoopParser) GetPriceSelectors() []string         { return nil }
func (p *pecPTNoopParser) GetURLSelectors() []string           { return nil }
func (p *pecPTNoopParser) ParseProductName(html string) string { return "" }
func (p *pecPTNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *pecPTNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *pecPTNoopParser) ParseStore(html string) string               { return "" }

const pecDoofinderSearchURL = "https://eu1-search.doofinder.com/5/search"
const pecDoofinderHashID = "85c5ffc7ebcc63b687fde301eeae18df"

// PerfumesECompanhiaPTExtractor extracts beauty products from Perfumes e Companhia (Portugal).
//
// The storefront search page (/pt/pesquisa/) is client-rendered by Doofinder, so
// HTML scraping returns no product tiles. Search results come from Doofinder's
// public JSON API (hashid from the store's Doofinder config). Do not scrape the
// SFCC HTML search page.
//
// Endpoint: GET https://eu1-search.doofinder.com/5/search?hashid=...&query={q}&rpp=20
type PerfumesECompanhiaPTExtractor struct {
	*extractors.BaseGoExtractor
}

// NewPerfumesECompanhiaPTExtractor creates a new Perfumes e Companhia Portugal extractor.
func NewPerfumesECompanhiaPTExtractor() *PerfumesECompanhiaPTExtractor {
	parser := &pecPTNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("perfumes_e_companhia_pt")}
	base := extractors.NewBaseGoExtractor(
		"https://www.perfumesecompanhia.pt",
		models.CountryPortugal,
		"perfumes_e_companhia_pt_v1",
		parser,
	)
	return &PerfumesECompanhiaPTExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *PerfumesECompanhiaPTExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Doofinder v5 search API URL for the Portugal store.
func (e *PerfumesECompanhiaPTExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("hashid", pecDoofinderHashID)
	params.Set("query", productName)
	params.Set("rpp", "20")
	return pecDoofinderSearchURL + "?" + params.Encode(), nil
}

// FetchHTML overrides the base client because Doofinder rejects requests that
// omit the storefront Origin ("request not authenticated"). This is a CORS-style
// gate, not a WAF challenge.
func (e *PerfumesECompanhiaPTExtractor) FetchHTML(targetURL string) (string, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("perfumes_e_companhia_pt: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.GetRandomUserAgent())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.perfumesecompanhia.pt")
	req.Header.Set("Referer", "https://www.perfumesecompanhia.pt/")

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("perfumes_e_companhia_pt: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("perfumes_e_companhia_pt: failed to read response: %w", err)
	}
	return string(body), nil
}

// GetComparisons fetches products from the Doofinder search API and returns comparisons.
func (e *PerfumesECompanhiaPTExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("perfumes_e_companhia_pt: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("perfumes_e_companhia_pt: failed to fetch API: %w", err)
	}

	return e.parseDoofinderResponse(body)
}

type pecSearchResponse struct {
	Results []pecProduct `json:"results"`
}

type pecProduct struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Price        float64 `json:"price"`
	SalePrice    float64 `json:"sale_price"`
	BestPrice    float64 `json:"best_price"`
	Link         string  `json:"link"`
	ImageLink    string  `json:"image_link"`
	Availability string  `json:"availability"`
}

func (e *PerfumesECompanhiaPTExtractor) parseDoofinderResponse(body string) ([]models.ProductComparison, error) {
	var resp pecSearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("perfumes_e_companhia_pt: failed to parse API response: %w", err)
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

		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Perfumes e Companhia",
			Country:     string(models.CountryPortugal),
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

	utils.Info("Perfumes e Companhia PT extraction completed", utils.Int("results", len(results)))
	return results, nil
}
