package extractors

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
)

// MercadoLivreParser implements HTMLParser interface for MercadoLivre Brazil
// Following Single Responsibility Principle - only handles MercadoLivre parsing logic
type MercadoLivreParser struct {
	*BaseHTMLParser
}

// NewMercadoLivreParser creates a new MercadoLivre-specific parser
func NewMercadoLivreParser() *MercadoLivreParser {
	return &MercadoLivreParser{
		BaseHTMLParser: NewBaseHTMLParser("MercadoLivre"),
	}
}

// GetProductSelectors returns CSS/regex selectors for finding product containers
func (p *MercadoLivreParser) GetProductSelectors() []string {
	return []string{
		// Main product search results
		`<div[^>]*class="[^"]*ui-search-result[^"]*"[^>]*>(.*?)</div>(?:\s*</div>)*`,
		`<li[^>]*class="[^"]*ui-search-layout__item[^"]*"[^>]*>(.*?)</li>`,
		`<div[^>]*class="[^"]*poly-card[^"]*"[^>]*>(.*?)</div>`,
		`<article[^>]*data-testid="result"[^>]*>(.*?)</article>`,
		// Fallback patterns
		`<div[^>]*class="[^"]*product[^"]*"[^>]*>(.*?)</div>`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *MercadoLivreParser) GetNameSelectors() []string {
	return []string{
		`<h2[^>]*class="[^"]*ui-search-item__title[^"]*"[^>]*>([^<]+)</h2>`,
		`<h2[^>]*class="[^"]*poly-component__title[^"]*"[^>]*>([^<]+)</h2>`,
		`<a[^>]*class="[^"]*ui-search-link[^"]*"[^>]*>([^<]+)</a>`,
		`<h2[^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</h2>`,
		`<a[^>]*class="[^"]*item__title[^"]*"[^>]*>([^<]+)</a>`,
		// More flexible patterns
		`<h[1-6][^>]*>[^<]*title[^<]*>([^<]+)</h[1-6]>`,
		`title="([^"]+)"`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *MercadoLivreParser) GetPriceSelectors() []string {
	return []string{
		// Andes Design System (current MercadoLivre design)
		`<span[^>]*class="[^"]*andes-money-amount__fraction[^"]*"[^>]*>([^<]+)</span>`,
		`<span[^>]*class="[^"]*price-tag-fraction[^"]*"[^>]*>([^<]+)</span>`,
		// With currency context
		`<span[^>]*class="[^"]*andes-money-amount__currency-symbol[^"]*"[^>]*>R\$</span>\s*<span[^>]*class="[^"]*andes-money-amount__fraction[^"]*"[^>]*>([^<]+)</span>`,
		// Broader price patterns
		`<span[^>]*class="[^"]*price[^"]*"[^>]*>R?\$?\s*([0-9.,]+)</span>`,
		`<span[^>]*class="[^"]*money-amount[^"]*"[^>]*>([^<]+)</span>`,
		// JSON-LD fallback will be handled separately
		`"price":\s*"?([0-9.,]+)"?`,
		`"priceAmount":\s*([0-9.,]+)`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *MercadoLivreParser) GetURLSelectors() []string {
	return []string{
		`<a[^>]*href="([^"]*MLB[^"]*)"[^>]*>`,
		`<a[^>]*class="[^"]*ui-search-link[^"]*"[^>]*href="([^"]+)"[^>]*>`,
		`<a[^>]*href="([^"]+)"[^>]*class="[^"]*ui-search-link[^"]*">`,
		`href="([^"]*produto[^"]*)"`,
		`href="([^"]*mercado[^"]*)"`,
	}
}

// ParseProductName extracts the product name from HTML fragment
func (p *MercadoLivreParser) ParseProductName(html string) string {
	selectors := p.GetNameSelectors()
	
	for i, selector := range selectors {
		utils.Debug("🏷️ Trying name pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if name := p.extractWithRegex(selector, html); name != "" {
			// Clean up the name
			name = strings.TrimSpace(name)
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
			
			// Validate name quality
			if len(name) > 5 && !strings.Contains(strings.ToLower(name), "mercado") {
				utils.Debug("✅ Found product name", 
					utils.String("name", name),
					utils.Int("pattern", i+1))
				return name
			}
		}
	}
	
	utils.Debug("❌ No product name found in HTML fragment")
	return ""
}

// ParsePrice extracts price and currency from HTML fragment
func (p *MercadoLivreParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON-LD if present
	if jsonProducts, err := p.extractJSONLD(html); err == nil {
		for _, product := range jsonProducts {
			if productType, ok := product["@type"].(string); ok && productType == "Product" {
				if offers, ok := product["offers"].(map[string]interface{}); ok {
					if priceStr, ok := offers["price"].(string); ok {
						if price, currency, err := p.parsePrice(priceStr, "BRL"); err == nil {
							utils.Debug("💰 Extracted price from JSON-LD", 
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
		utils.Debug("💰 Trying price pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if priceText := p.extractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.parsePrice(priceText, "BRL"); err == nil {
				utils.Debug("✅ Found price", 
					utils.Float64("price", price),
					utils.String("currency", currency),
					utils.Int("pattern", i+1))
				return price, currency, nil
			}
		}
	}
	
	return 0, "BRL", fmt.Errorf("no valid price found")
}

// ParseURL extracts the product URL from HTML fragment
func (p *MercadoLivreParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for i, selector := range selectors {
		utils.Debug("🔗 Trying URL pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return "https://www.mercadolivre.com.br" + urlStr
			} else {
				return "https://www.mercadolivre.com.br/" + urlStr
			}
		}
	}
	
	return baseURL // Fallback to base URL
}

// ParseStore extracts the store name from HTML fragment
func (p *MercadoLivreParser) ParseStore(html string) string {
	// Look for seller information
	storeSelectors := []string{
		`<span[^>]*class="[^"]*seller[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*store[^"]*"[^>]*>([^<]+)</div>`,
		`por\s+([^<\n]+)`,
	}
	
	for _, selector := range storeSelectors {
		if store := p.extractWithRegex(selector, html); store != "" {
			return strings.TrimSpace(store)
		}
	}
	
	return "MercadoLivre" // Default store name
}

// MercadoLivreExtractorV2 is the new pure Go implementation
type MercadoLivreExtractorV2 struct {
	*BaseGoExtractor
}

// NewMercadoLivreExtractorV2 creates a new pure Go MercadoLivre extractor
func NewMercadoLivreExtractorV2() *MercadoLivreExtractorV2 {
	parser := NewMercadoLivreParser()
	baseExtractor := NewBaseGoExtractor(
		"https://lista.mercadolivre.com.br",
		models.CountryBrazil,
		"mercadolivre_v2",
		parser,
	)
	
	return &MercadoLivreExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL overrides the base implementation for MercadoLivre's specific URL format
func (e *MercadoLivreExtractorV2) BuildSearchURL(productName string) (string, error) {
	// MercadoLivre uses a different URL format: /product-name
	encodedProduct := url.PathEscape(strings.ReplaceAll(strings.ToLower(productName), " ", "-"))
	searchURL := fmt.Sprintf("%s/%s", e.GetBaseURL(), encodedProduct)
	
	utils.Info("🔗 Built MercadoLivre search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisonsFromHTML overrides base implementation for MercadoLivre-specific logic
func (e *MercadoLivreExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing MercadoLivre HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// First try JSON-LD structured data (more reliable)
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

// extractFromJSONLD tries to extract products from structured data
func (e *MercadoLivreExtractorV2) extractFromJSONLD(html string) []models.ProductComparison {
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
func (e *MercadoLivreExtractorV2) parseJSONProduct(product map[string]interface{}) *models.ProductComparison {
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
	
	price, currency, err := e.BaseHTMLParser.parsePrice(priceStr, "BRL")
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
		StoreName:   "MercadoLivre",
		StoreURL:    storeURL,
		Country:     string(models.CountryBrazil),
	}
}

