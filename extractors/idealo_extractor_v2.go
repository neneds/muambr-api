package extractors

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
	"github.com/PuerkitoBio/goquery"
)

// IdealoParser implements HTMLParser interface for Idealo Spain
// Following Single Responsibility Principle - only handles Idealo parsing logic
type IdealoParser struct {
	*BaseHTMLParser
}

// NewIdealoParser creates a new Idealo-specific parser
func NewIdealoParser() *IdealoParser {
	return &IdealoParser{
		BaseHTMLParser: NewBaseHTMLParser("Idealo"),
	}
}

// GetProductSelectors returns CSS/regex selectors for finding product containers
func (p *IdealoParser) GetProductSelectors() []string {
	return []string{
		// Idealo specific product containers
		`<div[^>]*class="[^"]*sr-resultItemTile__link[^"]*"[^>]*>(.*?)</div>`,
		`<a[^>]*class="[^"]*sr-resultItemTile__link_Q8V4n[^"]*"[^>]*>(.*?)</a>`,
		`<div[^>]*class="[^"]*product-tile[^"]*"[^>]*>(.*?)</div>`,
		// More generic patterns
		`<article[^>]*class="[^"]*product[^"]*"[^>]*>(.*?)</article>`,
		`<div[^>]*class="[^"]*item[^"]*"[^>]*>(.*?)</div>`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *IdealoParser) GetNameSelectors() []string {
	return []string{
		// Idealo specific patterns
		`<div[^>]*class="[^"]*sr-productSummary__title_f5flP[^"]*"[^>]*>([^<]+)</div>`,
		`<h[1-6][^>]*class="[^"]*product-title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		`<a[^>]*title="([^"]+)"[^>]*>`,
		`<span[^>]*class="[^"]*product-name[^"]*"[^>]*>([^<]+)</span>`,
		`<h[1-6][^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *IdealoParser) GetPriceSelectors() []string {
	return []string{
		// Idealo specific price patterns
		`<div[^>]*class="[^"]*sr-detailedPriceInfo__price_sYVmx[^"]*"[^>]*>[^<]*?([0-9]+[,.]?[0-9]*)\s*€`,
		`<span[^>]*class="[^"]*sr-detailedPriceInfo__price[^"]*"[^>]*>[^<]*?([0-9]+[,.]?[0-9]*)\s*€`,
		`class="[^"]*price[^"]*"[^>]*>[^<]*?([0-9]+[,.]?[0-9]*)\s*€`,
		// More flexible EUR patterns
		`€[^0-9]*([0-9]+[,.]?[0-9]*)`,
		`([0-9]+[,.]?[0-9]*)\s*€`,
		// Generic price patterns
		`precio[^:]*:\s*([0-9]+[,.]?[0-9]*)`,
		`price[^:]*:\s*([0-9]+[,.]?[0-9]*)`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *IdealoParser) GetURLSelectors() []string {
	return []string{
		// Idealo specific URL patterns
		`<a[^>]*class="[^"]*sr-resultItemTile__link_Q8V4n[^"]*"[^>]*href="([^"]+)"`,
		`<a[^>]*href="(https://www\.idealo\.es/precios/[^"]+)"`,
		`href="(/precios/[^"]+)"`,
		// More generic link patterns
		`<a[^>]*href="([^"]*product[^"]*)"`,
		`<a[^>]*href="([^"]*p/[^"]*)"`,
	}
}

// ParseName extracts product name from HTML fragment
func (p *IdealoParser) ParseName(html string) string {
	selectors := p.GetNameSelectors()
	
	for i, selector := range selectors {
		utils.Debug("📝 Trying Idealo name pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if name := p.extractWithRegex(selector, html); name != "" {
			name = strings.TrimSpace(name)
			if len(name) > 3 { // Basic validation
				utils.Debug("✅ Found Idealo product name", 
					utils.String("name", name),
					utils.Int("pattern", i+1))
				return name
			}
		}
	}
	
	utils.Debug("❌ No product name found in Idealo HTML fragment")
	return ""
}

// ParseProductName implements the HTMLParser interface
func (p *IdealoParser) ParseProductName(html string) string {
	return p.ParseName(html)
}

// ParsePrice extracts price and currency from HTML fragment
func (p *IdealoParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON-LD if present
	if jsonProducts, err := p.extractJSONLD(html); err == nil {
		for _, product := range jsonProducts {
			if productType, ok := product["@type"].(string); ok && productType == "Product" {
				if offers, ok := product["offers"].(map[string]interface{}); ok {
					if priceStr, ok := offers["price"].(string); ok {
						if price, currency, err := p.parsePrice(priceStr, "EUR"); err == nil {
							utils.Debug("💰 Extracted Idealo price from JSON-LD", 
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
		utils.Debug("💰 Trying Idealo price pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if priceText := p.extractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.parsePrice(priceText, "EUR"); err == nil {
				utils.Debug("✅ Found Idealo price", 
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
func (p *IdealoParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for i, selector := range selectors {
		utils.Debug("🔗 Trying Idealo URL pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return "https://www.idealo.es" + urlStr
			} else {
				return "https://www.idealo.es/" + urlStr
			}
		}
	}
	
	return baseURL // Fallback to base URL
}

// ParseStore extracts the store name from HTML fragment
func (p *IdealoParser) ParseStore(html string) string {
	// Look for seller/store information
	storeSelectors := []string{
		`<span[^>]*class="[^"]*seller[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*store[^"]*"[^>]*>([^<]+)</div>`,
		`<span[^>]*class="[^"]*vendor[^"]*"[^>]*>([^<]+)</span>`,
		`tienda:\s*([^<\n]+)`,
		`vendido por\s+([^<\n]+)`,
	}
	
	for _, selector := range storeSelectors {
		if store := p.extractWithRegex(selector, html); store != "" {
			return strings.TrimSpace(store)
		}
	}
	
	return "Idealo - ES" // Default store name
}

// IdealoExtractorV2 is the pure Go implementation for Idealo Spain
type IdealoExtractorV2 struct {
	*BaseGoExtractor
}

// NewIdealoExtractorV2 creates a new pure Go Idealo extractor
func NewIdealoExtractorV2() *IdealoExtractorV2 {
	parser := NewIdealoParser()
	baseExtractor := NewBaseGoExtractor(
		"https://www.idealo.es",
		models.CountrySpain,
		"idealo_v2",
		parser,
	)
	
	return &IdealoExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL overrides the base implementation for Idealo's specific URL format
func (e *IdealoExtractorV2) BuildSearchURL(productName string) (string, error) {
	// Idealo uses query parameters: /resultados.html?q=productName
	params := url.Values{}
	params.Add("q", productName)
	
	searchURL := fmt.Sprintf("%s/resultados.html?%s", e.GetBaseURL(), params.Encode())
	
	utils.Info("🔗 Built Idealo search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisonsFromHTML overrides base implementation for Idealo-specific logic
func (e *IdealoExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing Idealo HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// First try JSON-LD structured data (more reliable)
	if jsonComparisons := e.extractFromJSONLD(html); len(jsonComparisons) > 0 {
		utils.Info("✅ Extracted Idealo products from JSON-LD", utils.Int("count", len(jsonComparisons)))
		return jsonComparisons, nil
	}
	
	// Use CSS selectors for HTML parsing (based on our analysis)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}
	
	// Extract products using the CSS classes we identified
	productLinks := doc.Find("a.sr-resultItemTile__link_Q8V4n")
	utils.Info("🔍 Found Idealo product links", utils.Int("count", productLinks.Length()))
	
	productLinks.Each(func(i int, link *goquery.Selection) {
		// Extract product name from title attribute or nested elements
		productName := ""
		
		// Try title attribute first
		if title, exists := link.Attr("title"); exists && title != "" {
			productName = strings.TrimSpace(title)
		}
		
		// Try nested title element
		if productName == "" {
			titleElement := link.Find(".sr-productSummary__title_f5flP")
			productName = strings.TrimSpace(titleElement.Text())
		}
		
		// Try image alt text as fallback
		if productName == "" {
			imgElement := link.Find("img")
			if alt, exists := imgElement.Attr("alt"); exists && alt != "" {
				productName = strings.TrimSpace(alt)
			}
		}
		
		if productName == "" {
			utils.Debug("⚠️ Skipping product with no name", utils.Int("index", i))
			return
		}
		
		// Extract price from the associated price info sections
		// Look for price in the parent container or next sibling elements
		priceElement := link.Parent().Find(".sr-detailedPriceInfo__price_sYVmx")
		if priceElement.Length() == 0 {
			// Try alternative price selectors
			priceElement = link.Parent().Parent().Find(".sr-detailedPriceInfo__price_sYVmx")
		}
		
		priceText := strings.TrimSpace(priceElement.Text())
		
		// Clean price text - remove "desde" prefix and extract numeric value
		priceRegex := regexp.MustCompile(`([0-9]+(?:[,.]?[0-9]*)?)\s*€`)
		priceMatches := priceRegex.FindStringSubmatch(priceText)
		
		if len(priceMatches) < 2 {
			utils.Debug("⚠️ Failed to parse price", 
				utils.String("priceText", priceText), 
				utils.String("productName", productName))
			return
		}
		
		// Convert comma to dot for proper float parsing
		priceStr := strings.ReplaceAll(priceMatches[1], ",", ".")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			utils.Debug("⚠️ Failed to convert price to float", 
				utils.String("priceStr", priceStr), 
				utils.Error(err))
			return
		}
		
		// Extract product URL
		productURL, exists := link.Attr("href")
		if !exists {
			utils.Debug("⚠️ No product URL found for", utils.String("name", productName))
			productURL = ""
		} else if !strings.HasPrefix(productURL, "http") {
			productURL = "https://www.idealo.es" + productURL
		}
		
		// Create comparison object
		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: productName,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Idealo - ES",
			Country:     string(models.CountrySpain),
		}
		
		// Set store URL if available
		if productURL != "" {
			comparison.StoreURL = &productURL
		}
		
		comparisons = append(comparisons, comparison)
		
		utils.Info("✅ Extracted Idealo product", 
			utils.String("name", productName),
			utils.Float64("price", price),
			utils.String("url", productURL))
	})
	
	utils.Info("🎯 Total Idealo products extracted", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractFromJSONLD attempts to extract products from JSON-LD structured data
func (e *IdealoExtractorV2) extractFromJSONLD(html string) []models.ProductComparison {
	var comparisons []models.ProductComparison
	
	// Look for JSON-LD script tags
	jsonLD := regexp.MustCompile(`<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)
	matches := jsonLD.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		var data interface{}
		if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
			continue
		}
		
		// Handle array of products
		if items, ok := data.([]interface{}); ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if comp := e.parseJSONProduct(itemMap); comp != nil {
						comparisons = append(comparisons, *comp)
					}
				}
			}
		} else if itemMap, ok := data.(map[string]interface{}); ok {
			if comp := e.parseJSONProduct(itemMap); comp != nil {
				comparisons = append(comparisons, *comp)
			}
		}
	}
	
	return comparisons
}

// parseJSONProduct converts a JSON-LD product to ProductComparison
func (e *IdealoExtractorV2) parseJSONProduct(product map[string]interface{}) *models.ProductComparison {
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
		StoreName:   "Idealo - ES",
		StoreURL:    storeURL,
		Country:     string(models.CountrySpain),
	}
}