package beauty

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

// bootsUKNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// BootsUKExtractor overrides GetComparisons entirely and uses the Algolia search
// JSON API, so the parser methods are never called.
type bootsUKNoopParser struct{ *extractors.BaseHTMLParser }

func (p *bootsUKNoopParser) GetProductSelectors() []string       { return nil }
func (p *bootsUKNoopParser) GetNameSelectors() []string          { return nil }
func (p *bootsUKNoopParser) GetPriceSelectors() []string         { return nil }
func (p *bootsUKNoopParser) GetURLSelectors() []string           { return nil }
func (p *bootsUKNoopParser) ParseProductName(html string) string { return "" }
func (p *bootsUKNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *bootsUKNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *bootsUKNoopParser) ParseStore(html string) string               { return "" }

const bootsAlgoliaAppID = "89JDFPR8F6"
const bootsAlgoliaAPIKey = "39ae8989d71b15b23524d9408047414f"
const bootsAlgoliaIndex = "prod_live_products_uk"

// BootsUKExtractor extracts beauty products from Boots (UK).
//
// Storefront /sitesearch redirects to brand CMS pages with no offer JSON.
// Search results come from Algolia using the public search-only key from the
// storefront algoliaConfig. Do not scrape brand landing HTML.
//
// Endpoint: GET https://{appId}-dsn.algolia.net/1/indexes/{index}?query={q}
type BootsUKExtractor struct {
	*extractors.BaseGoExtractor
}

// NewBootsUKExtractor creates a new Boots UK extractor.
func NewBootsUKExtractor() *BootsUKExtractor {
	parser := &bootsUKNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("boots_uk")}
	base := extractors.NewBaseGoExtractor(
		"https://www.boots.com",
		models.CountryUK,
		"boots_uk_v1",
		parser,
	)
	return &BootsUKExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *BootsUKExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL constructs the Algolia index search URL for Boots UK.
func (e *BootsUKExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("query", productName)
	params.Set("hitsPerPage", "20")
	return fmt.Sprintf(
		"https://%s-dsn.algolia.net/1/indexes/%s?%s",
		bootsAlgoliaAppID, bootsAlgoliaIndex, params.Encode(),
	), nil
}

// FetchHTML overrides the base client to send Algolia's required search headers.
func (e *BootsUKExtractor) FetchHTML(targetURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("boots_uk: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Algolia-Application-Id", bootsAlgoliaAppID)
	req.Header.Set("X-Algolia-API-Key", bootsAlgoliaAPIKey)

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("boots_uk: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("boots_uk: failed to read response: %w", err)
	}
	return string(body), nil
}

// GetComparisons fetches products from Algolia and returns comparisons.
func (e *BootsUKExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("boots_uk: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("boots_uk: failed to fetch API: %w", err)
	}

	return e.parseAlgoliaResponse(body)
}

type bootsAlgoliaResponse struct {
	Hits []map[string]any `json:"hits"`
}

var bootsNameKeys = []string{
	"name", "productName", "title", "displayName",
	"longProductName", "shortProductName", "modelName",
}

func (e *BootsUKExtractor) parseAlgoliaResponse(body string) ([]models.ProductComparison, error) {
	var resp bootsAlgoliaResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("boots_uk: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	results := make([]models.ProductComparison, 0, len(resp.Hits))

	skippedPrice := 0
	skippedName := 0
	for _, h := range resp.Hits {
		price := bootsAsFloat(h["currentPrice"])
		if price <= 0 {
			price = bootsAsFloat(h["price"])
		}
		if price <= 0 {
			skippedPrice++
			continue
		}

		name := bootsLookupName(h)
		if name == "" {
			skippedName++
			continue
		}

		id := firstNonEmpty(bootsAsString(h["objectID"]), bootsAsString(h["parentProductPartNumber"]))
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "GBP",
			StoreName:   "Boots",
			Country:     string(models.CountryUK),
			Category:    &category,
		}

		if link := bootsLookupURL(h, e.GetBaseURL()); link != "" {
			comparison.StoreURL = &link
		}
		if img := firstNonEmpty(
			bootsAsString(h["thumbnail"]),
			bootsAsString(h["imageURL"]),
			bootsAsString(h["imageUrl"]),
			bootsAsString(h["image"]),
		); img != "" {
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Boots UK extraction completed",
		utils.Int("hits", len(resp.Hits)),
		utils.Int("results", len(results)),
		utils.Int("skipped_price", skippedPrice),
		utils.Int("skipped_name", skippedName))
	return results, nil
}

func bootsLookupName(v any) string {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range bootsNameKeys {
			if s := bootsAsString(bootsMapCI(t, k)); s != "" {
				return s
			}
		}
		if s := bootsFirstNameLike(t); s != "" {
			return s
		}
		if s := bootsLookupName(t["parentProduct"]); s != "" {
			return s
		}
		if s := bootsLookupName(bootsMapCI(t, "productAttributes")); s != "" {
			return s
		}
		if hl, ok := bootsMapCI(t, "_highlightResult").(map[string]any); ok {
			if s := bootsHighlightName(hl); s != "" {
				return s
			}
		}
	case []any:
		for _, item := range t {
			if s := bootsLookupName(item); s != "" {
				return s
			}
		}
	}
	return ""
}

func bootsMapCI(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	lower := strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return nil
}

func bootsFirstNameLike(m map[string]any) string {
	for k, v := range m {
		lk := strings.ToLower(k)
		if !strings.Contains(lk, "name") {
			continue
		}
		if strings.Contains(lk, "category") || strings.Contains(lk, "brand") ||
			strings.Contains(lk, "list") || strings.Contains(lk, "colour") ||
			strings.Contains(lk, "color") || strings.Contains(lk, "campaign") {
			continue
		}
		if s := bootsAsString(v); s != "" {
			return s
		}
	}
	return ""
}

func bootsHighlightName(hl map[string]any) string {
	for _, k := range bootsNameKeys {
		if n, ok := bootsMapCI(hl, k).(map[string]any); ok {
			if s := stripAlgoliaEm(bootsAsString(n["value"])); s != "" {
				return s
			}
		}
	}
	for k, val := range hl {
		lk := strings.ToLower(k)
		if !strings.Contains(lk, "name") || strings.Contains(lk, "category") {
			continue
		}
		if n, ok := val.(map[string]any); ok {
			if s := stripAlgoliaEm(bootsAsString(n["value"])); s != "" {
				return s
			}
		}
	}
	return ""
}

func bootsLookupURL(h map[string]any, baseURL string) string {
	link := firstNonEmpty(
		bootsAsString(h["url"]),
		bootsAsString(h["productUrl"]),
		bootsAsString(h["actionURL"]),
		bootsAsString(h["seoURL"]),
		bootsAsString(h["seoUrl"]),
		bootsAsString(h["pageURL"]),
		bootsAsString(h["slug"]),
	)
	if link == "" {
		return ""
	}
	if strings.HasPrefix(link, "http") {
		return link
	}
	if !strings.HasPrefix(link, "/") {
		link = "/" + link
	}
	return baseURL + link
}

func bootsAsString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return strings.TrimSpace(t.String())
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}

func bootsAsFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0
		}
		return f
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(t, ",", "")), 64)
		if err != nil {
			return 0
		}
		return f
	case map[string]any:
		for _, k := range []string{"value", "amount", "currentPrice", "price"} {
			if f := bootsAsFloat(t[k]); f > 0 {
				return f
			}
		}
		return 0
	default:
		return 0
	}
}

func stripAlgoliaEm(s string) string {
	s = strings.ReplaceAll(s, "<em>", "")
	s = strings.ReplaceAll(s, "</em>", "")
	return strings.TrimSpace(s)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
