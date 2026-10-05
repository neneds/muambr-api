package other

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// KuantoKustaParser implements HTMLParser interface for KuantoKusta Portugal
// Following Single Responsibility Principle - only handles KuantoKusta parsing logic
type KuantoKustaParser struct {
	*extractors.BaseHTMLParser
}

// NewKuantoKustaParser creates a new KuantoKusta-specific parser
func NewKuantoKustaParser() *KuantoKustaParser {
	return &KuantoKustaParser{
		BaseHTMLParser: extractors.NewBaseHTMLParser("KuantoKusta"),
	}
}

// GetProductSelectors returns CSS/regex selectors for finding product containers
func (p *KuantoKustaParser) GetProductSelectors() []string {
	return []string{
		// Main product search results (KuantoKusta specific)
		`<div[^>]*class="[^"]*product-item[^"]*"[^>]*>(.*?)</div>(?:\s*</div>)*`,
		`<article[^>]*class="[^"]*product[^"]*"[^>]*>(.*?)</article>`,
		`<li[^>]*class="[^"]*search-result[^"]*"[^>]*>(.*?)</li>`,
		`<div[^>]*class="[^"]*item[^"]*"[^>]*>(.*?)</div>`,
		// More generic patterns
		`<div[^>]*data-product[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*card[^"]*"[^>]*>(.*?)</div>`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *KuantoKustaParser) GetNameSelectors() []string {
	return []string{
		// KuantoKusta specific patterns
		`<h[1-6][^>]*class="[^"]*product-title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		`<a[^>]*class="[^"]*product-link[^"]*"[^>]*title="([^"]+)"[^>]*>`,
		`<span[^>]*class="[^"]*product-name[^"]*"[^>]*>([^<]+)</span>`,
		`<h[1-6][^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		// More flexible patterns
		`title="([^"]+)"`,
		`alt="([^"]+)"`,
		`<a[^>]*href="[^"]*"[^>]*>([^<]+)</a>`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *KuantoKustaParser) GetPriceSelectors() []string {
	return []string{
		// European price patterns with € symbol
		`<span[^>]*class="[^"]*price[^"]*"[^>]*>([0-9.,]+)\s*€</span>`,
		`<div[^>]*class="[^"]*price[^"]*"[^>]*>€?\s*([0-9.,]+)</div>`,
		`<span[^>]*class="[^"]*amount[^"]*"[^>]*>([0-9.,]+)</span>`,
		// Currency-specific patterns
		`€\s*([0-9.,]+)`,
		`([0-9.,]+)\s*€`,
		// Generic price patterns
		`<span[^>]*>[^€]*€?\s*([0-9.,]+)[^<]*</span>`,
		`"price":\s*"?([0-9.,]+)"?`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *KuantoKustaParser) GetURLSelectors() []string {
	return []string{
		`<a[^>]*href="([^"]*kuantokusta[^"]*)"[^>]*>`,
		`<a[^>]*class="[^"]*product-link[^"]*"[^>]*href="([^"]+)"[^>]*>`,
		`<a[^>]*href="([^"]+)"[^>]*class="[^"]*product[^"]*">`,
		`href="([^"]*\/p\/[^"]*)"`, // KuantoKusta product URL pattern
		`href="([^"]*product[^"]*)"`,
	}
}

// ParseProductName extracts the product name from HTML fragment
func (p *KuantoKustaParser) ParseProductName(html string) string {
	selectors := p.GetNameSelectors()

	for _, selector := range selectors {
		if name := p.ExtractWithRegex(selector, html); name != "" {
			// Clean up the name
			name = strings.TrimSpace(name)
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")

			// Validate name quality
			if len(name) > 3 && !strings.Contains(strings.ToLower(name), "kuantokusta") {
				return name
			}
		}
	}

	return ""
}

// ParsePrice extracts price and currency from HTML fragment
func (p *KuantoKustaParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON-LD if present
	if jsonProducts, err := p.ExtractJSONLD(html); err == nil {
		for _, product := range jsonProducts {
			if productType, ok := product["@type"].(string); ok && productType == "Product" {
				if offers, ok := product["offers"].(map[string]interface{}); ok {
					if priceStr, ok := offers["price"].(string); ok {
						if price, currency, err := p.ParsePriceText(priceStr, "EUR"); err == nil {
							return price, currency, nil
						}
					}
				}
			}
		}
	}

	// Fallback to HTML parsing
	selectors := p.GetPriceSelectors()

	for _, selector := range selectors {
		if priceText := p.ExtractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.ParsePriceText(priceText, "EUR"); err == nil {
				return price, currency, nil
			}
		}
	}

	return 0, "EUR", fmt.Errorf("no valid price found")
}

// ParseURL extracts the product URL from HTML fragment
func (p *KuantoKustaParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()

	for _, selector := range selectors {
		if urlStr := p.ExtractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return "https://www.kuantokusta.pt" + urlStr
			} else {
				return "https://www.kuantokusta.pt/" + urlStr
			}
		}
	}

	return baseURL // Fallback to base URL
}

// ParseStore extracts the store name from HTML fragment
func (p *KuantoKustaParser) ParseStore(html string) string {
	// Look for seller/store information
	storeSelectors := []string{
		`<span[^>]*class="[^"]*seller[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*store[^"]*"[^>]*>([^<]+)</div>`,
		`<span[^>]*class="[^"]*vendor[^"]*"[^>]*>([^<]+)</span>`,
		`loja:\s*([^<\n]+)`,
		`vendido por\s+([^<\n]+)`,
	}

	for _, selector := range storeSelectors {
		if store := p.ExtractWithRegex(selector, html); store != "" {
			return strings.TrimSpace(store)
		}
	}

	return "KuantoKusta - PT" // Default store name
}

// KuantoKustaExtractorV2 is the new pure Go implementation
type KuantoKustaExtractorV2 struct {
	*extractors.BaseGoExtractor
}

// NewKuantoKustaExtractorV2 creates a new pure Go KuantoKusta extractor
func NewKuantoKustaExtractorV2() *KuantoKustaExtractorV2 {
	parser := NewKuantoKustaParser()
	baseExtractor := extractors.NewBaseGoExtractor(
		"https://www.kuantokusta.pt",
		models.CountryPortugal,
		"kuantokusta_v2",
		parser,
	)

	return &KuantoKustaExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL overrides the base implementation for KuantoKusta's specific URL format
func (e *KuantoKustaExtractorV2) BuildSearchURL(productName string) (string, error) {
	// KuantoKusta uses query parameters: /search?q=productName
	params := url.Values{}
	params.Add("q", productName)

	searchURL := fmt.Sprintf("%s/search?%s", e.GetBaseURL(), params.Encode())

	utils.Info("🔗 Built KuantoKusta search URL",
		utils.String("product", productName),
		utils.String("url", searchURL))

	return searchURL, nil
}

const kuantoKustaAPI = "https://api.kuantokusta.pt"

type kkSearchResponse struct {
	Data []kkProduct `json:"data"`
}

type kkProduct struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	PriceMin float64  `json:"priceMin"`
	URL      string   `json:"url"`
	Images   []string `json:"images"`
}

// GetComparisons calls the public products API. The HTML search page is behind
// Akamai and returns 403 to this client; api.kuantokusta.pt returns the same
// name / priceMin / url fields the page used to embed in __NEXT_DATA__.
func (e *KuantoKustaExtractorV2) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("Starting KuantoKusta product extraction",
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	params := url.Values{}
	params.Set("q", productName)
	params.Set("page", "1")
	params.Set("rows", "24")
	apiURL := kuantoKustaAPI + "/products?" + params.Encode()

	resp, err := utils.MakeAntiBotRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read products response: %w", err)
	}

	comparisons, err := e.GetComparisonsFromHTML(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to extract comparisons: %w", err)
	}

	utils.Info("Extraction completed",
		utils.String("extractor", "kuantokusta_v2"),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// GetComparisonsFromHTML overrides base implementation for KuantoKusta-specific logic
func (e *KuantoKustaExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	trimmed := strings.TrimSpace(html)
	if strings.HasPrefix(trimmed, "{") {
		return parseKuantoKustaProductsJSON(trimmed)
	}

	utils.Info("Parsing KuantoKusta HTML", utils.Int("size", len(html)))

	var comparisons []models.ProductComparison

	// KuantoKusta uses Next.js with JSON data embedded in __NEXT_DATA__ script tag
	if !strings.Contains(html, "__NEXT_DATA__") {
		utils.Info("No __NEXT_DATA__ found in HTML")
		return comparisons, nil
	}

	// Look for the JSON data in the script tag
	jsonStart := strings.Index(html, `<script id="__NEXT_DATA__" type="application/json">`)
	if jsonStart == -1 {
		utils.Warn("__NEXT_DATA__ script tag not found")
		return comparisons, nil
	}

	jsonStart = strings.Index(html[jsonStart:], `>`) + jsonStart + 1
	jsonEnd := strings.Index(html[jsonStart:], `</script>`)
	if jsonEnd == -1 {
		utils.Warn("Malformed __NEXT_DATA__ script tag")
		return comparisons, nil
	}

	jsonData := html[jsonStart : jsonStart+jsonEnd]
	utils.Info("📊 Extracted JSON data", utils.Int("length", len(jsonData)))

	// Parse the JSON data
	var nextData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &nextData); err != nil {
		utils.Info("❌ Failed to parse KuantoKusta JSON data", utils.Error(err))
		return comparisons, nil
	}

	// Navigate to the products data: props.pageProps.basePage.data
	props, ok := nextData["props"].(map[string]interface{})
	if !ok {
		utils.Info("No props found in KuantoKusta JSON")
		return comparisons, nil
	}

	pageProps, ok := props["pageProps"].(map[string]interface{})
	if !ok {
		utils.Info("No pageProps found in KuantoKusta JSON")
		return comparisons, nil
	}

	basePage, ok := pageProps["basePage"].(map[string]interface{})
	if !ok {
		utils.Info("No basePage found in KuantoKusta JSON")
		return comparisons, nil
	}

	dataArray, ok := basePage["data"].([]interface{})
	if !ok {
		utils.Info("No data array found in KuantoKusta JSON")
		return comparisons, nil
	}

	utils.Info("Found KuantoKusta products in JSON data", utils.Int("count", len(dataArray)))

	// Process each product
	for _, item := range dataArray {
		product, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract product name
		name, ok := product["name"].(string)
		if !ok || name == "" {
			continue
		}

		// Extract minimum price
		priceMin, ok := product["priceMin"].(float64)
		if !ok {
			// Try as string or int
			if priceStr, ok := product["priceMin"].(string); ok {
				if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
					priceMin = parsed
				} else {
					continue
				}
			} else if priceInt, ok := product["priceMin"].(int); ok {
				priceMin = float64(priceInt)
			} else {
				continue
			}
		}

		// Extract product URL
		productURL, ok := product["url"].(string)
		if !ok {
			productURL = ""
		}

		// Make URL absolute
		var storeURL *string
		if productURL != "" {
			fullURL := "https://www.kuantokusta.pt" + productURL
			storeURL = &fullURL
		}

		// Extract image URL
		var imageURL *string
		if images, ok := product["images"].([]interface{}); ok && len(images) > 0 {
			if imgStr, ok := images[0].(string); ok && imgStr != "" {
				imageURL = &imgStr
			}
		}

		// Create comparison object
		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: name,
			Price:       priceMin,
			Currency:    "EUR",
			StoreName:   "KuantoKusta - PT",
			StoreURL:    storeURL,
			Country:     string(models.CountryPortugal),
			ImageURL:    imageURL,
		}

		comparisons = append(comparisons, comparison)
	}

	utils.Info("Extracted KuantoKusta products from JSON data", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

func parseKuantoKustaProductsJSON(body string) ([]models.ProductComparison, error) {
	var payload kkSearchResponse
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse KuantoKusta products JSON: %w", err)
	}

	comparisons := make([]models.ProductComparison, 0, len(payload.Data))
	for _, product := range payload.Data {
		name := strings.TrimSpace(product.Name)
		if name == "" || product.PriceMin <= 0 {
			continue
		}

		var storeURL *string
		if product.URL != "" {
			full := product.URL
			if strings.HasPrefix(full, "/") {
				full = "https://www.kuantokusta.pt" + full
			}
			storeURL = &full
		}

		var imageURL *string
		if len(product.Images) > 0 && strings.TrimSpace(product.Images[0]) != "" {
			img := product.Images[0]
			imageURL = &img
		}

		comparisons = append(comparisons, models.ProductComparison{
			ID:          strconv.FormatInt(product.ID, 10),
			ProductName: name,
			Price:       product.PriceMin,
			Currency:    "EUR",
			StoreName:   "KuantoKusta - PT",
			StoreURL:    storeURL,
			Country:     string(models.CountryPortugal),
			ImageURL:    imageURL,
		})
	}
	return comparisons, nil
}

// GetCategory returns the product category this extractor is optimised for
func (e *KuantoKustaExtractorV2) GetCategory() models.ProductCategory {
	return models.CategoryOther
}
