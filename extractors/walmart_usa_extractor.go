package extractors

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
)

// WalmartUSAParser implements HTMLParser interface for Walmart USA
// Following Single Responsibility Principle - only handles Walmart USA parsing logic
type WalmartUSAParser struct {
	*BaseHTMLParser
}

// NewWalmartUSAParser creates a new Walmart USA-specific parser
func NewWalmartUSAParser() *WalmartUSAParser {
	return &WalmartUSAParser{
		BaseHTMLParser: NewBaseHTMLParser("walmart_usa"),
	}
}

// GetProductSelectors returns CSS/regex selectors for finding product containers
func (p *WalmartUSAParser) GetProductSelectors() []string {
	return []string{
		// Main product search result containers
		`<div[^>]*data-automation-id="product[^"]*"[^>]*>(.*?)</div>`,
		`<article[^>]*data-testid="item"[^>]*>(.*?)</article>`,
		`<div[^>]*class="[^"]*search-result[^"]*"[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*product-tile[^"]*"[^>]*>(.*?)</div>`,
		// Fallback patterns for product containers
		`<div[^>]*data-item-id="[^"]*"[^>]*>(.*?)</div>`,
		`<article[^>]*class="[^"]*product[^"]*"[^>]*>(.*?)</article>`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *WalmartUSAParser) GetNameSelectors() []string {
	return []string{
		// Primary product title selectors
		`<span[^>]*data-automation-id="product-title"[^>]*>([^<]+)</span>`,
		`<h3[^>]*data-automation-id="product-title"[^>]*>([^<]+)</h3>`,
		`<a[^>]*data-testid="product-title"[^>]*>([^<]+)</a>`,
		// Alternative title patterns
		`<span[^>]*class="[^"]*product-title[^"]*"[^>]*>([^<]+)</span>`,
		`<h[1-6][^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		// Aria-label and alt text fallbacks
		`aria-label="([^"]+)"`,
		`alt="([^"]+)"`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *WalmartUSAParser) GetPriceSelectors() []string {
	return []string{
		// Primary price selectors for current price
		`<span[^>]*data-automation-id="product-price"[^>]*>\$?([0-9,]+\.?[0-9]*)</span>`,
		`<div[^>]*data-testid="price-current"[^>]*>\$?([0-9,]+\.?[0-9]*)</div>`,
		`<span[^>]*class="[^"]*price[^"]*current[^"]*"[^>]*>\$?([0-9,]+\.?[0-9]*)</span>`,
		// General price patterns
		`<span[^>]*class="[^"]*price[^"]*"[^>]*>\$?([0-9,]+\.?[0-9]*)</span>`,
		`<div[^>]*class="[^"]*price[^"]*"[^>]*>\$?([0-9,]+\.?[0-9]*)</div>`,
		// Special Walmart price formats
		`<span[^>]*class="[^"]*visuallyHidden[^"]*"[^>]*>current price \$?([0-9,]+\.?[0-9]*)</span>`,
		`\$([0-9,]+\.?[0-9]*)`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *WalmartUSAParser) GetURLSelectors() []string {
	return []string{
		`<a[^>]*data-automation-id="product-title"[^>]*href="([^"]+)"[^>]*>`,
		`<a[^>]*data-testid="product-title"[^>]*href="([^"]+)"[^>]*>`,
		`<a[^>]*href="([^"]+)"[^>]*data-automation-id="product-title"[^>]*>`,
		`<a[^>]*href="([^"]*ip/[^"]+)"[^>]*>`,
		`href="([^"]*walmart\.com[^"]+)"`,
	}
}

// ParseProductName extracts the product name from HTML fragment
func (p *WalmartUSAParser) ParseProductName(html string) string {
	selectors := p.GetNameSelectors()
	
	for _, selector := range selectors {
		if name := p.extractWithRegex(selector, html); name != "" {
			// Clean up the name
			name = strings.TrimSpace(name)
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
			
			// Validate name quality - avoid generic text
			if len(name) > 5 && !strings.Contains(strings.ToLower(name), "walmart") &&
			   !strings.Contains(strings.ToLower(name), "search") {
				return name
			}
		}
	}
	
	return ""
}

// ParsePrice extracts price and currency from HTML fragment
func (p *WalmartUSAParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON data if present
	if jsonProducts, err := p.extractNextDataProducts(html); err == nil && len(jsonProducts) > 0 {
		for _, product := range jsonProducts {
			if price, currency, err := p.parsePrice(fmt.Sprintf("%.2f", product.Price), "USD"); err == nil {
				return price, currency, nil
			}
		}
	}

	// Fallback to HTML parsing
	selectors := p.GetPriceSelectors()
	
	for _, selector := range selectors {
		if priceText := p.extractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.parsePrice(priceText, "USD"); err == nil {
				return price, currency, nil
			}
		}
	}
	
	return 0, "USD", fmt.Errorf("no valid price found")
}

// ParseURL extracts the product URL from HTML fragment
func (p *WalmartUSAParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for _, selector := range selectors {
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return "https://www.walmart.com" + urlStr
			} else {
				return "https://www.walmart.com/" + urlStr
			}
		}
	}
	
	return baseURL // Fallback to base URL
}

// ParseStore extracts the store name from HTML fragment
func (p *WalmartUSAParser) ParseStore(html string) string {
	// Look for seller information - Walmart products can be sold by third parties
	storeSelectors := []string{
		`<span[^>]*class="[^"]*seller[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*seller-name[^"]*"[^>]*>([^<]+)</div>`,
		`sold by\s+([^<\n]+)`,
		`by\s+([^<\n]+)`,
	}
	
	for _, selector := range storeSelectors {
		if store := p.extractWithRegex(selector, html); store != "" {
			storeName := strings.TrimSpace(store)
			if storeName != "" && !strings.Contains(strings.ToLower(storeName), "walmart") {
				return storeName
			}
		}
	}
	
	return "Walmart" // Default store name
}

// NextDataProduct represents product data from Walmart's __NEXT_DATA__
type NextDataProduct struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
}

// extractNextDataProducts extracts product data from __NEXT_DATA__ JSON
func (p *WalmartUSAParser) extractNextDataProducts(html string) ([]NextDataProduct, error) {
	// Look for __NEXT_DATA__ script tag
	nextDataRegex := regexp.MustCompile(`<script[^>]*id="__NEXT_DATA__"[^>]*>(.*?)</script>`)
	matches := nextDataRegex.FindStringSubmatch(html)
	
	if len(matches) < 2 {
		return nil, fmt.Errorf("no __NEXT_DATA__ found")
	}
	
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__: %w", err)
	}
	
	var products []NextDataProduct
	
	// Navigate through the nested structure to find search results
	// Walmart's structure: props.pageProps.initialData.searchResult.itemStacks
	if props, ok := data["props"].(map[string]interface{}); ok {
		if pageProps, ok := props["pageProps"].(map[string]interface{}); ok {
			if initialData, ok := pageProps["initialData"].(map[string]interface{}); ok {
				if searchResult, ok := initialData["searchResult"].(map[string]interface{}); ok {
					if itemStacks, ok := searchResult["itemStacks"].([]interface{}); ok {
						for _, stack := range itemStacks {
							if stackMap, ok := stack.(map[string]interface{}); ok {
								if items, ok := stackMap["items"].([]interface{}); ok {
									for _, item := range items {
										if product := p.parseNextDataItem(item); product != nil {
											products = append(products, *product)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	return products, nil
}

// parseNextDataItem converts a single item from __NEXT_DATA__ to NextDataProduct
func (p *WalmartUSAParser) parseNextDataItem(item interface{}) *NextDataProduct {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}
	
	id, _ := itemMap["id"].(string)
	name, _ := itemMap["name"].(string)
	
	var price float64
	if priceData, ok := itemMap["price"].(map[string]interface{}); ok {
		if current, ok := priceData["current"].(float64); ok {
			price = current
		}
	}
	
	var url string
	if canonicalUrl, ok := itemMap["canonicalUrl"].(string); ok {
		url = canonicalUrl
	}
	
	if name != "" && price > 0 {
		return &NextDataProduct{
			ID:    id,
			Name:  name,
			Price: price,
			URL:   url,
		}
	}
	
	return nil
}

// WalmartUSAExtractor is the pure Go implementation for Walmart USA
type WalmartUSAExtractor struct {
	*BaseGoExtractor
}

// NewWalmartUSAExtractor creates a new pure Go Walmart USA extractor
func NewWalmartUSAExtractor() *WalmartUSAExtractor {
	parser := NewWalmartUSAParser()
	baseExtractor := NewBaseGoExtractor(
		"https://www.walmart.com",
		models.CountryUS,
		"walmart_usa",
		parser,
	)
	
	return &WalmartUSAExtractor{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL overrides the base implementation for Walmart's specific URL format
func (e *WalmartUSAExtractor) BuildSearchURL(productName string) (string, error) {
	// Walmart uses /search?q=product+name format
	encodedProduct := url.QueryEscape(productName)
	searchURL := fmt.Sprintf("%s/search?q=%s", e.GetBaseURL(), encodedProduct)
	
	utils.Info("🔗 Built Walmart USA search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisonsFromHTML overrides base implementation for Walmart-specific logic
func (e *WalmartUSAExtractor) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing Walmart USA HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// First try __NEXT_DATA__ structured data (more reliable)
	if jsonComparisons := e.extractFromNextData(html); len(jsonComparisons) > 0 {
		utils.Info("✅ Extracted products from __NEXT_DATA__", utils.Int("count", len(jsonComparisons)))
		return jsonComparisons, nil
	}
	
	// Also try JSON-LD if available
	if jsonComparisons := e.extractFromJSONLD(html); len(jsonComparisons) > 0 {
		utils.Info("✅ Extracted products from JSON-LD", utils.Int("count", len(jsonComparisons)))
		return jsonComparisons, nil
	}
	
	// Fallback to HTML parsing using base implementation
	comparisons, err := e.BaseGoExtractor.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, err
	}
	
	utils.Info("✅ Extracted products from HTML", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractFromNextData tries to extract products from __NEXT_DATA__ structured data
func (e *WalmartUSAExtractor) extractFromNextData(html string) []models.ProductComparison {
	var comparisons []models.ProductComparison
	
	// Cast the parser to access custom methods
	walmartParser, ok := e.parser.(*WalmartUSAParser)
	if !ok {
		return comparisons
	}
	
	products, err := walmartParser.extractNextDataProducts(html)
	if err != nil {
		return comparisons
	}
	
	for _, product := range products {
		if comp := e.parseNextDataProduct(product); comp != nil {
			comparisons = append(comparisons, *comp)
		}
	}
	
	return comparisons
}

// extractFromJSONLD tries to extract products from structured data
func (e *WalmartUSAExtractor) extractFromJSONLD(html string) []models.ProductComparison {
	var comparisons []models.ProductComparison
	
	jsonData, err := e.BaseHTMLParser.extractJSONLD(html)
	if err != nil {
		return comparisons
	}
	
	for _, data := range jsonData {
		if comp := e.parseJSONProduct(data); comp != nil {
			comparisons = append(comparisons, *comp)
		}
	}
	
	return comparisons
}

// parseNextDataProduct converts a NextDataProduct to ProductComparison
func (e *WalmartUSAExtractor) parseNextDataProduct(product NextDataProduct) *models.ProductComparison {
	if product.Name == "" || product.Price <= 0 {
		return nil
	}
	
	var storeURL *string
	if product.URL != "" {
		fullURL := product.URL
		if strings.HasPrefix(product.URL, "/") {
			fullURL = "https://www.walmart.com" + product.URL
		}
		storeURL = &fullURL
	}
	
	return &models.ProductComparison{
		ID:          utils.GenerateUUID(),
		ProductName: product.Name,
		Price:       product.Price,
		Currency:    "USD",
		StoreName:   "Walmart",
		StoreURL:    storeURL,
		Country:     string(models.CountryUS),
	}
}

// parseJSONProduct converts a JSON-LD product to ProductComparison
func (e *WalmartUSAExtractor) parseJSONProduct(product map[string]interface{}) *models.ProductComparison {
	productType, ok := product["@type"].(string)
	if !ok || productType != "Product" {
		return nil
	}
	
	name, ok := product["name"].(string)
	if !ok || name == "" {
		return nil
	}
	
	offers, ok := product["offers"].(map[string]interface{})
	if !ok {
		return nil
	}
	
	priceStr, ok := offers["price"].(string)
	if !ok {
		// Try as number
		if priceNum, ok := offers["price"].(float64); ok {
			priceStr = fmt.Sprintf("%.2f", priceNum)
		} else {
			return nil
		}
	}
	
	price, currency, err := e.BaseHTMLParser.parsePrice(priceStr, "USD")
	if err != nil {
		return nil
	}
	
	urlStr, _ := offers["url"].(string)
	var storeURL *string
	if urlStr != "" {
		storeURL = &urlStr
	}
	
	return &models.ProductComparison{
		ID:          utils.GenerateUUID(),
		ProductName: name,
		Price:       price,
		Currency:    currency,
		StoreName:   "Walmart",
		StoreURL:    storeURL,
		Country:     string(models.CountryUS),
	}
}