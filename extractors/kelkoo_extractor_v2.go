package extractors

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
)

// KelkooParser implements HTMLParser interface for Kelkoo Spain
// Following Single Responsibility Principle - only handles Kelkoo parsing logic
type KelkooParser struct {
	*BaseHTMLParser
}

// NewKelkooParser creates a new Kelkoo-specific parser
func NewKelkooParser() *KelkooParser {
	return &KelkooParser{
		BaseHTMLParser: NewBaseHTMLParser("Kelkoo"),
	}
}

// GetProductSelectors returns CSS/regex selectors for finding product containers
func (p *KelkooParser) GetProductSelectors() []string {
	return []string{
		// Kelkoo specific patterns
		`<div[^>]*class="[^"]*product-card[^"]*"[^>]*>(.*?)</div>(?:\s*</div>)*`,
		`<article[^>]*class="[^"]*offer[^"]*"[^>]*>(.*?)</article>`,
		`<li[^>]*class="[^"]*search-item[^"]*"[^>]*>(.*?)</li>`,
		`<div[^>]*class="[^"]*result[^"]*"[^>]*>(.*?)</div>`,
		// More generic patterns
		`<div[^>]*data-offer[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*item[^"]*"[^>]*>(.*?)</div>`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *KelkooParser) GetNameSelectors() []string {
	return []string{
		// Kelkoo specific patterns
		`<h[1-6][^>]*class="[^"]*offer-title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		`<a[^>]*class="[^"]*offer-link[^"]*"[^>]*title="([^"]+)"[^>]*>`,
		`<span[^>]*class="[^"]*product-name[^"]*"[^>]*>([^<]+)</span>`,
		`<h[1-6][^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		// More flexible patterns
		`title="([^"]+)"`,
		`alt="([^"]+)"`,
		`<a[^>]*href="[^"]*"[^>]*>([^<]+)</a>`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *KelkooParser) GetPriceSelectors() []string {
	return []string{
		// European price patterns with € symbol
		`<span[^>]*class="[^"]*price[^"]*"[^>]*>([0-9.,]+)\s*€</span>`,
		`<div[^>]*class="[^"]*price[^"]*"[^>]*>€?\s*([0-9.,]+)</div>`,
		`<span[^>]*class="[^"]*amount[^"]*"[^>]*>([0-9.,]+)</span>`,
		// Kelkoo specific patterns
		`<span[^>]*class="[^"]*offer-price[^"]*"[^>]*>([0-9.,]+)\s*€</span>`,
		// Currency-specific patterns
		`€\s*([0-9.,]+)`,
		`([0-9.,]+)\s*€`,
		// Generic price patterns
		`<span[^>]*>[^€]*€?\s*([0-9.,]+)[^<]*</span>`,
		`"price":\s*"?([0-9.,]+)"?`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *KelkooParser) GetURLSelectors() []string {
	return []string{
		`<a[^>]*href="([^"]*kelkoo[^"]*)"[^>]*>`,
		`<a[^>]*class="[^"]*offer-link[^"]*"[^>]*href="([^"]+)"[^>]*>`,
		`<a[^>]*href="([^"]+)"[^>]*class="[^"]*offer[^"]*">`,
		`href="([^"]*\/offer\/[^"]*)"`, // Kelkoo offer URL pattern
		`href="([^"]*product[^"]*)"`,
	}
}

// ParseProductName extracts the product name from HTML fragment
func (p *KelkooParser) ParseProductName(html string) string {
	selectors := p.GetNameSelectors()
	
	for i, selector := range selectors {
		utils.Debug("🏷️ Trying Kelkoo name pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if name := p.extractWithRegex(selector, html); name != "" {
			// Clean up the name
			name = strings.TrimSpace(name)
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
			
			// Validate name quality
			if len(name) > 3 && !strings.Contains(strings.ToLower(name), "kelkoo") {
				utils.Debug("✅ Found Kelkoo product name", 
					utils.String("name", name),
					utils.Int("pattern", i+1))
				return name
			}
		}
	}
	
	utils.Debug("❌ No product name found in Kelkoo HTML fragment")
	return ""
}

// ParsePrice extracts price and currency from HTML fragment
func (p *KelkooParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON-LD if present
	if jsonProducts, err := p.extractJSONLD(html); err == nil {
		for _, product := range jsonProducts {
			if productType, ok := product["@type"].(string); ok && productType == "Product" {
				if offers, ok := product["offers"].(map[string]interface{}); ok {
					if priceStr, ok := offers["price"].(string); ok {
						if price, currency, err := p.parsePrice(priceStr, "EUR"); err == nil {
							utils.Debug("💰 Extracted Kelkoo price from JSON-LD", 
								utils.Float64("price", price),
								utils.String("currency", currency))
							return price, currency, nil
						}
					}
				}
			}
		}
	}

	// Fallback to HTML parsing
	selectors := p.GetPriceSelectors()
	
	for i, selector := range selectors {
		utils.Debug("💰 Trying Kelkoo price pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if priceText := p.extractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.parsePrice(priceText, "EUR"); err == nil {
				utils.Debug("✅ Found Kelkoo price", 
					utils.Float64("price", price),
					utils.String("currency", currency),
					utils.Int("pattern", i+1))
				return price, currency, nil
			}
		}
	}
	
	return 0, "EUR", fmt.Errorf("no valid price found")
}

// ParseURL extracts the product URL from HTML fragment
func (p *KelkooParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for i, selector := range selectors {
		utils.Debug("🔗 Trying Kelkoo URL pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return "https://www.kelkoo.es" + urlStr
			} else {
				return "https://www.kelkoo.es/" + urlStr
			}
		}
	}
	
	return baseURL // Fallback to base URL
}

// ParseStore extracts the store name from HTML fragment
func (p *KelkooParser) ParseStore(html string) string {
	// Look for seller/store information
	storeSelectors := []string{
		`<span[^>]*class="[^"]*merchant[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*store[^"]*"[^>]*>([^<]+)</div>`,
		`<span[^>]*class="[^"]*vendor[^"]*"[^>]*>([^<]+)</span>`,
		`tienda:\s*([^<\n]+)`,
		`vendido por\s+([^<\n]+)`,
		`disponible en\s+([^<\n]+)`,
	}
	
	for _, selector := range storeSelectors {
		if store := p.extractWithRegex(selector, html); store != "" {
			return strings.TrimSpace(store)
		}
	}
	
	return "Kelkoo Partner Store (Available in Spain)" // Default store name
}

// KelkooExtractorV2 is the new pure Go implementation
type KelkooExtractorV2 struct {
	*BaseGoExtractor
}

// NewKelkooExtractorV2 creates a new pure Go Kelkoo extractor
func NewKelkooExtractorV2() *KelkooExtractorV2 {
	parser := NewKelkooParser()
	baseExtractor := NewBaseGoExtractor(
		"https://www.kelkoo.es",
		models.CountrySpain,
		"kelkoo_v2",
		parser,
	)
	
	return &KelkooExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL overrides the base implementation for Kelkoo's specific URL format
func (e *KelkooExtractorV2) BuildSearchURL(productName string) (string, error) {
	// Kelkoo uses query parameters: /buscar?consulta=productName
	params := url.Values{}
	params.Add("consulta", productName)
	
	searchURL := fmt.Sprintf("%s/buscar?%s", e.GetBaseURL(), params.Encode())
	
	utils.Info("🔗 Built Kelkoo search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisonsFromHTML overrides base implementation for Kelkoo-specific logic
func (e *KelkooExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing Kelkoo HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// First try JSON-LD structured data (more reliable)
	if jsonComparisons := e.extractFromJSONLD(html); len(jsonComparisons) > 0 {
		utils.Info("✅ Extracted Kelkoo products from JSON-LD", utils.Int("count", len(jsonComparisons)))
		return jsonComparisons, nil
	}
	
	// Fallback to HTML parsing using base implementation
	comparisons, err := e.BaseGoExtractor.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, err
	}
	
	utils.Info("✅ Extracted Kelkoo products from HTML", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractFromJSONLD tries to extract products from structured data
func (e *KelkooExtractorV2) extractFromJSONLD(html string) []models.ProductComparison {
	var comparisons []models.ProductComparison
	
	jsonData, err := e.BaseHTMLParser.extractJSONLD(html)
	if err != nil {
		return comparisons
	}
	
	for _, data := range jsonData {
		// Handle @graph structure
		if graph, ok := data["@graph"].([]interface{}); ok {
			for _, item := range graph {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if comp := e.parseJSONProduct(itemMap); comp != nil {
						comparisons = append(comparisons, *comp)
					}
				}
			}
		} else if comp := e.parseJSONProduct(data); comp != nil {
			comparisons = append(comparisons, *comp)
		}
	}
	
	return comparisons
}

// parseJSONProduct converts a JSON-LD product to ProductComparison
func (e *KelkooExtractorV2) parseJSONProduct(product map[string]interface{}) *models.ProductComparison {
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
	
	price, currency, err := e.BaseHTMLParser.parsePrice(priceStr, "EUR")
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
		StoreName:   "Kelkoo Partner Store (Available in Spain)",
		StoreURL:    storeURL,
		Country:     string(models.CountrySpain),
	}
}