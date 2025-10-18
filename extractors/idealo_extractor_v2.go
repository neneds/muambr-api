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
	
	for _, selector := range selectors {
		if name := p.extractWithRegex(selector, html); name != "" {
			name = strings.TrimSpace(name)
			if len(name) > 3 { // Basic validation
				return name
			}
		}
	}
	
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
		if priceText := p.extractWithRegex(selector, html); priceText != "" {
			if price, currency, err := p.parsePrice(priceText, "EUR"); err == nil {
				return price, currency, nil
			}
		}
	}
	
	return 0, "EUR", fmt.Errorf("no valid price found")
}

// ParseURL extracts the product URL from HTML fragment
func (p *IdealoParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for _, selector := range selectors {
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
	baseExtractor *BaseGoExtractor
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
	
	idealoExtractor := &IdealoExtractorV2{
		baseExtractor: baseExtractor,
	}
	
	utils.Info("🏗️ IdealoExtractorV2 created successfully", 
		utils.String("identifier", idealoExtractor.GetIdentifier()),
		utils.String("country", string(idealoExtractor.GetCountryCode())),
		utils.String("base_url", idealoExtractor.BaseURL()))
	
	return idealoExtractor
}

// GetComparisons overrides the base implementation to use Idealo-specific URL format
func (e *IdealoExtractorV2) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("🚨 IDEALO OVERRIDE METHOD CALLED! 🚨", 
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	// Build Idealo-specific search URL
	searchURL, err := e.buildIdealoSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Fetch HTML using base functionality
	html, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using Idealo-specific logic
	comparisons, err := e.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comparisons: %w", err)
	}

	utils.Info("✅ Extraction completed", 
		utils.String("extractor", "idealo_v2"),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// Interface delegation methods to satisfy Extractor interface
func (e *IdealoExtractorV2) GetCountryCode() models.Country {
	return e.baseExtractor.GetCountryCode()
}

func (e *IdealoExtractorV2) GetMacroRegion() models.MacroRegion {
	return e.baseExtractor.GetMacroRegion()
}

func (e *IdealoExtractorV2) GetIdentifier() string {
	return e.baseExtractor.GetIdentifier()
}

func (e *IdealoExtractorV2) BaseURL() string {
	return e.baseExtractor.BaseURL()
}

// Helper methods that delegate to baseExtractor
func (e *IdealoExtractorV2) GetBaseURL() string {
	return e.baseExtractor.BaseURL()
}

func (e *IdealoExtractorV2) FetchHTML(url string) (string, error) {
	return e.baseExtractor.FetchHTML(url)
}

func (e *IdealoExtractorV2) parsePrice(priceText string, defaultCurrency string) (float64, string, error) {
	return e.baseExtractor.BaseHTMLParser.parsePrice(priceText, defaultCurrency)
}

// BuildSearchURL overrides the base method to use Idealo-specific URL format
func (e *IdealoExtractorV2) BuildSearchURL(productName string) (string, error) {
	utils.Info("🎯 IdealoExtractorV2 BuildSearchURL CALLED", 
		utils.String("product", productName))
	return e.buildIdealoSearchURL(productName)
}

// buildIdealoSearchURL builds the Idealo-specific URL format
func (e *IdealoExtractorV2) buildIdealoSearchURL(productName string) (string, error) {
	// Idealo uses query parameters: /resultados.html?q=productName
	params := url.Values{}
	params.Add("q", productName)
	
	searchURL := fmt.Sprintf("%s/resultados.html?%s", e.baseExtractor.BaseURL(), params.Encode())
	
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
	
	// Extract products using the result items we identified
	resultItems := doc.Find(".sr-resultList__item_m6xdA")
	utils.Info("🔍 Found Idealo result items", utils.Int("count", resultItems.Length()))
	
	resultItems.Each(func(i int, item *goquery.Selection) {
		// Extract product name from title element
		titleElement := item.Find(".sr-productSummary__title_f5flP")
		productName := strings.TrimSpace(titleElement.Text())
		
		// Try image alt text as fallback if no title
		if productName == "" {
			imgElement := item.Find("img")
			if alt, exists := imgElement.Attr("alt"); exists && alt != "" {
				productName = strings.TrimSpace(alt)
			}
		}
		
		if productName == "" {
			return
		}
		
		// Extract price from the price info section
		priceElement := item.Find(".sr-detailedPriceInfo__price_sYVmx")
		
		priceText := strings.TrimSpace(priceElement.Text())
		
		// Clean price text - remove "desde" prefix and extract numeric value
		// European format: thousands with dots, decimals with commas (e.g., 1.006,05 €)
		priceRegex := regexp.MustCompile(`([0-9]+(?:\.[0-9]{3})*(?:,[0-9]{1,2})?)\s*€`)
		priceMatches := priceRegex.FindStringSubmatch(priceText)
		
		if len(priceMatches) < 2 {
			return
		}
		
		// Convert European format to standard format for parsing
		// European: 1.006,05 → Standard: 1006.05
		priceStr := priceMatches[1]
		// Remove thousands separators (dots)
		priceStr = strings.ReplaceAll(priceStr, ".", "")
		// Convert decimal separator (comma to dot)
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return
		}
		
		// Extract product URL from the button/link within the item
		productURL := ""
		linkElement := item.Find("button.sr-resultItemLink__button_k3jEE")
		if linkElement.Length() > 0 {
			// For Idealo, the links are form submits, so we'll use the base URL + product name
			productURL = fmt.Sprintf("https://www.idealo.es/resultados.html?q=%s", url.QueryEscape(productName))
		}
		
		// Try to find an actual href if available
		if productURL == "" {
			linkWithHref := item.Find("a[href]")
			if href, exists := linkWithHref.Attr("href"); exists && href != "" {
				if strings.HasPrefix(href, "http") {
					productURL = href
				} else {
					productURL = "https://www.idealo.es" + href
				}
			}
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
	
	price, currency, err := e.parsePrice(priceStr, "EUR")
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