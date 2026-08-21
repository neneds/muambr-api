package electronics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// paradigitNLNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// ParadigitNLExtractor overrides GetComparisons entirely and parses JSON-LD from
// the getproducts HTML fragment, so the parser methods are never called.
type paradigitNLNoopParser struct{ *extractors.BaseHTMLParser }

func (p *paradigitNLNoopParser) GetProductSelectors() []string       { return nil }
func (p *paradigitNLNoopParser) GetNameSelectors() []string          { return nil }
func (p *paradigitNLNoopParser) GetPriceSelectors() []string         { return nil }
func (p *paradigitNLNoopParser) GetURLSelectors() []string           { return nil }
func (p *paradigitNLNoopParser) ParseProductName(html string) string { return "" }
func (p *paradigitNLNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *paradigitNLNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *paradigitNLNoopParser) ParseStore(html string) string               { return "" }

// paradigitSearchContentID is the Umbraco node id of /zoekresultaten/.
const paradigitSearchContentID = "2720"

// ParadigitNLExtractor extracts electronics products from Paradigit (Netherlands).
//
// Storefront search is an Umbraco hash-routed page. Results come from POST
// /search/getproducts, which embeds one schema.org Product JSON-LD block per
// hit. Skip OutOfStock. Do not scrape /zoekresultaten/ HTML (no product tiles).
//
// Endpoint: POST https://www.paradigit.nl/search/getproducts
type ParadigitNLExtractor struct {
	*extractors.BaseGoExtractor
}

// NewParadigitNLExtractor creates a new Paradigit Netherlands extractor.
func NewParadigitNLExtractor() *ParadigitNLExtractor {
	parser := &paradigitNLNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("paradigit_nl")}
	base := extractors.NewBaseGoExtractor(
		"https://www.paradigit.nl",
		models.CountryNetherlands,
		"paradigit_nl_v1",
		parser,
	)
	return &ParadigitNLExtractor{BaseGoExtractor: base}
}

// GetCategory returns the electronics category for registry filtering.
func (e *ParadigitNLExtractor) GetCategory() models.ProductCategory {
	return models.CategoryElectronics
}

// BuildSearchURL returns the getproducts endpoint (query is sent as POST form).
func (e *ParadigitNLExtractor) BuildSearchURL(productName string) (string, error) {
	return e.GetBaseURL() + "/search/getproducts", nil
}

// GetComparisons POSTs the search form and parses JSON-LD products.
func (e *ParadigitNLExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	body, err := e.postSearch(productName)
	if err != nil {
		return nil, err
	}
	return e.GetComparisonsFromHTML(body)
}

func (e *ParadigitNLExtractor) postSearch(productName string) (string, error) {
	form := url.Values{}
	form.Set("ContentId", paradigitSearchContentID)
	form.Set("SearchText", productName)
	form.Set("CurrentPage", "1")

	req, err := http.NewRequest(http.MethodPost, e.GetBaseURL()+"/search/getproducts", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("paradigit_nl: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "text/html, */*;q=0.8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://www.paradigit.nl")
	req.Header.Set("Referer", "https://www.paradigit.nl/zoekresultaten/")

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("paradigit_nl: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("paradigit_nl: failed to read response: %w", err)
	}
	return string(body), nil
}

// GetComparisonsFromHTML parses schema.org Product JSON-LD from getproducts HTML.
func (e *ParadigitNLExtractor) GetComparisonsFromHTML(body string) ([]models.ProductComparison, error) {
	items, err := e.ExtractJSONLD(body)
	if err != nil {
		return nil, fmt.Errorf("paradigit_nl: failed to parse JSON-LD: %w", err)
	}

	category := models.CategoryElectronics
	results := make([]models.ProductComparison, 0, len(items))
	seen := make(map[string]struct{})

	for _, item := range items {
		if jsonLDType(item["@type"]) != "Product" {
			continue
		}
		offers, _ := item["offers"].(map[string]interface{})
		if offers == nil {
			continue
		}
		avail := jsonLDString(offers["availability"])
		if !strings.Contains(avail, "InStock") {
			continue
		}

		price, err := strconv.ParseFloat(jsonLDString(offers["price"]), 64)
		name := strings.TrimSpace(jsonLDString(item["name"]))
		if name == "" || err != nil || price <= 0 {
			continue
		}

		id := strings.TrimSpace(jsonLDString(item["sku"]))
		if id == "" {
			id = utils.GenerateUUID()
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		currency := jsonLDString(offers["priceCurrency"])
		if currency == "" {
			currency = "EUR"
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    currency,
			StoreName:   "Paradigit",
			Country:     string(models.CountryNetherlands),
			Category:    &category,
		}

		link := e.absoluteURL(jsonLDString(offers["url"]))
		if link == "" {
			link = e.absoluteURL(jsonLDString(item["url"]))
		}
		if link != "" {
			comparison.StoreURL = &link
		}
		if img := jsonLDString(item["image"]); img != "" {
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Paradigit NL extraction completed", utils.Int("results", len(results)))
	return results, nil
}

func (e *ParadigitNLExtractor) absoluteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return e.GetBaseURL() + raw
	}
	return e.GetBaseURL() + "/" + raw
}

func jsonLDType(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []interface{}:
		if len(t) > 0 {
			if s, ok := t[0].(string); ok {
				return s
			}
		}
	}
	return jsonLDString(v)
}

func jsonLDString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}
