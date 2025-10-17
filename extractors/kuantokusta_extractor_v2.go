package extractors

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
	"github.com/PuerkitoBio/goquery"
)

// KuantoKustaParser implements HTMLParser interface for KuantoKusta Portugal
// Following Single Responsibility Principle - only handles KuantoKusta parsing logic
type KuantoKustaParser struct {
	*BaseHTMLParser
}

// NewKuantoKustaParser creates a new KuantoKusta-specific parser
func NewKuantoKustaParser() *KuantoKustaParser {
	return &KuantoKustaParser{
		BaseHTMLParser: NewBaseHTMLParser("KuantoKusta"),
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
	
	for i, selector := range selectors {
		utils.Debug("🏷️ Trying KuantoKusta name pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if name := p.extractWithRegex(selector, html); name != "" {
			// Clean up the name
			name = strings.TrimSpace(name)
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
			
			// Validate name quality
			if len(name) > 3 && !strings.Contains(strings.ToLower(name), "kuantokusta") {
				utils.Debug("✅ Found KuantoKusta product name", 
					utils.String("name", name),
					utils.Int("pattern", i+1))
				return name
			}
		}
	}
	
	utils.Debug("❌ No product name found in KuantoKusta HTML fragment")
	return ""
}

// ParsePrice extracts price and currency from HTML fragment
func (p *KuantoKustaParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON-LD if present
	if jsonProducts, err := p.extractJSONLD(html); err == nil {
		for _, product := range jsonProducts {
			if productType, ok := product["@type"].(string); ok && productType == "Product" {
				if offers, ok := product["offers"].(map[string]interface{}); ok {
					if priceStr, ok := offers["price"].(string); ok {
						if price, currency, err := p.parsePrice(priceStr, "EUR"); err == nil {
							utils.Debug("💰 Extracted KuantoKusta price from JSON-LD", 
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
		utils.Debug("💰 Trying KuantoKusta price pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if priceText := p.extractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.parsePrice(priceText, "EUR"); err == nil {
				utils.Debug("✅ Found KuantoKusta price", 
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
func (p *KuantoKustaParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for i, selector := range selectors {
		utils.Debug("🔗 Trying KuantoKusta URL pattern", 
			utils.Int("pattern", i+1),
			utils.String("selector", selector[:min(50, len(selector))]))
			
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
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
		if store := p.extractWithRegex(selector, html); store != "" {
			return strings.TrimSpace(store)
		}
	}
	
	return "KuantoKusta - PT" // Default store name
}

// KuantoKustaExtractorV2 is the new pure Go implementation
type KuantoKustaExtractorV2 struct {
	*BaseGoExtractor
}

// NewKuantoKustaExtractorV2 creates a new pure Go KuantoKusta extractor
func NewKuantoKustaExtractorV2() *KuantoKustaExtractorV2 {
	parser := NewKuantoKustaParser()
	baseExtractor := NewBaseGoExtractor(
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

// GetComparisonsFromHTML overrides base implementation for KuantoKusta-specific logic
func (e *KuantoKustaExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing KuantoKusta HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// First try JSON-LD structured data (more reliable)
	if jsonComparisons := e.extractFromJSONLD(html); len(jsonComparisons) > 0 {
		utils.Info("✅ Extracted KuantoKusta products from JSON-LD", utils.Int("count", len(jsonComparisons)))
		return jsonComparisons, nil
	}
	
	// Use CSS selectors for HTML parsing (based on our analysis)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}
	
	// Extract products using data-test-id selectors (confirmed working)
	productCards := doc.Find("[data-test-id='product-card']")
	utils.Info("🔍 Found KuantoKusta product cards", utils.Int("count", productCards.Length()))
	
	productCards.Each(func(i int, card *goquery.Selection) {
		// Extract product name
		nameElement := card.Find("[data-test-id='product-card-name']")
		productName := strings.TrimSpace(nameElement.Text())
		
		if productName == "" {
			utils.Debug("⚠️ Skipping card with no name", utils.Int("index", i))
			return
		}
		
		// Extract price
		priceElement := card.Find("[data-test-id='product-card-offers-price-min']")
		priceText := strings.TrimSpace(priceElement.Text())
		
		// Clean price text and extract numeric value
		priceText = regexp.MustCompile(`^desde\s*`).ReplaceAllString(priceText, "")
		priceText = regexp.MustCompile(`€.*$`).ReplaceAllString(priceText, "")
		priceText = strings.ReplaceAll(priceText, ",", ".")
		
		price, err := strconv.ParseFloat(priceText, 64)
		if err != nil {
			utils.Debug("⚠️ Failed to parse price", 
				utils.String("priceText", priceText), 
				utils.String("originalText", priceElement.Text()),
				utils.Error(err))
			return
		}
		
		// Extract product URL
		productLink := card.Find("a[href*='/p/']").First()
		productURL, exists := productLink.Attr("href")
		if !exists {
			utils.Debug("⚠️ No product URL found for", utils.String("name", productName))
			productURL = ""
		} else if !strings.HasPrefix(productURL, "http") {
			productURL = "https://www.kuantokusta.pt" + productURL
		}
		
		// Extract seller info (number of shops)
		sellerElement := card.Find("[data-test-id='product-card-seller']")
		sellerText := strings.TrimSpace(sellerElement.Text())
		if sellerText == "" {
			sellerText = "KuantoKusta - PT"
		}
		
		// Extract image URL
		imgElement := card.Find("img").First()
		imageURL, _ := imgElement.Attr("src")
		var imageURLPtr *string
		if imageURL != "" {
			imageURLPtr = &imageURL
		}
		
		// Create product comparison
		var storeURLPtr *string
		if productURL != "" {
			storeURLPtr = &productURL
		}
		
		comparison := models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: productName,
			Price:       price,
			Currency:    "EUR",
			StoreName:   sellerText,
			StoreURL:    storeURLPtr,
			Country:     string(models.CountryPortugal),
			ImageURL:    imageURLPtr,
		}
		
		comparisons = append(comparisons, comparison)
		
		utils.Debug("✅ Extracted KuantoKusta product", 
			utils.String("name", productName),
			utils.Float64("price", price),
			utils.String("seller", sellerText))
	})
	
	utils.Info("✅ Extracted KuantoKusta products from HTML", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractFromJSONLD tries to extract products from structured data
func (e *KuantoKustaExtractorV2) extractFromJSONLD(html string) []models.ProductComparison {
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
func (e *KuantoKustaExtractorV2) parseJSONProduct(product map[string]interface{}) *models.ProductComparison {
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
		StoreName:   "KuantoKusta - PT",
		StoreURL:    storeURL,
		Country:     string(models.CountryPortugal),
	}
}