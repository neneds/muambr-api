package extractors

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"muambr-api/models"
	"muambr-api/utils"
)

// AmazonUSParser implements HTMLParser interface for Amazon USA
type AmazonUSParser struct {
	*BaseHTMLParser
}

// NewAmazonUSParser creates a new Amazon USA-specific parser
func NewAmazonUSParser() *AmazonUSParser {
	return &AmazonUSParser{
		BaseHTMLParser: NewBaseHTMLParser("amazon_usa"),
	}
}

// GetProductSelectors returns CSS selectors for finding product containers
func (p *AmazonUSParser) GetProductSelectors() []string {
	return []string{
		`div[data-asin][data-component-type="s-search-result"]`,
		`div[data-asin].s-result-item`,
		`div.s-result-item`,
		`div[data-asin].a-section`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *AmazonUSParser) GetNameSelectors() []string {
	return []string{
		`h2.a-size-medium span`,
		`h2.a-size-base-plus span`,
		`h2 a span`,
		`h2 span`,
		`.a-text-normal`,
		`h2 aria-label`,
		`aria-label="([^"]+)"`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *AmazonUSParser) GetPriceSelectors() []string {
	return []string{
		`span.a-offscreen`,
		`span.a-color-base`,
		`span.a-price-whole`,
		`div[data-cy='price-recipe'] span.a-color-base`,
		`.a-price .a-offscreen`,
		`\$([0-9,]+\.?[0-9]*)`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *AmazonUSParser) GetURLSelectors() []string {
	return []string{
		`h2 a[href]`,
		`a[href*="/dp/"]`,
		`.s-link-style[href]`,
		`a.a-link-normal[href]`,
	}
}

// ParseProductName extracts product name from HTML fragment
func (p *AmazonUSParser) ParseProductName(html string) string {
	// First try to extract from aria-label
	ariaLabelRegex := regexp.MustCompile(`aria-label="([^"]+)"`)
	if matches := ariaLabelRegex.FindStringSubmatch(html); len(matches) > 1 {
		name := strings.TrimSpace(matches[1])
		// Avoid generic labels
		if len(name) > 5 && !strings.Contains(strings.ToLower(name), "rating") && 
		   !strings.Contains(strings.ToLower(name), "button") {
			return name
		}
	}

	// Fall back to CSS selectors
	selectors := p.GetNameSelectors()
	for _, selector := range selectors {
		if name := p.extractWithRegex(selector, html); name != "" {
			name = strings.TrimSpace(name)
			if len(name) > 5 {
				return name
			}
		}
	}
	return ""
}

// ParsePrice extracts price and currency from HTML fragment  
func (p *AmazonUSParser) ParsePrice(html string) (float64, string, error) {
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
func (p *AmazonUSParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	for _, selector := range selectors {
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return baseURL + urlStr
			}
		}
	}
	return baseURL // Fallback to base URL
}

// ParseStore extracts store name from HTML fragment
func (p *AmazonUSParser) ParseStore(html string) string {
	// Amazon marketplace may have third-party sellers
	storeSelectors := []string{
		`<span[^>]*class="[^"]*seller[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*store[^"]*"[^>]*>([^<]+)</div>`,
		`by\s+([^<\n]+)`,
	}
	
	for _, selector := range storeSelectors {
		if store := p.extractWithRegex(selector, html); store != "" {
			storeName := strings.TrimSpace(store)
			if storeName != "" && !strings.Contains(strings.ToLower(storeName), "amazon") {
				return storeName
			}
		}
	}
	
	return "Amazon" // Default store name
}

// AmazonUSExtractor is the implementation for Amazon USA
type AmazonUSExtractor struct {
	*BaseGoExtractor
}

// NewAmazonUSExtractor creates a new Amazon USA extractor
func NewAmazonUSExtractor() *AmazonUSExtractor {
	parser := NewAmazonUSParser()
	baseExtractor := NewBaseGoExtractor(
		"https://www.amazon.com",
		models.CountryUS,
		"amazon_usa",
		parser,
	)
	
	return &AmazonUSExtractor{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL overrides the base implementation for Amazon's specific URL format
func (e *AmazonUSExtractor) BuildSearchURL(productName string) (string, error) {
	// Amazon uses /s?k=product+name format
	encodedProduct := url.QueryEscape(productName)
	searchURL := fmt.Sprintf("%s/s?k=%s", e.GetBaseURL(), encodedProduct)
	
	utils.Info("🔗 Built Amazon USA search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisons overrides the base implementation to use Amazon USA-specific logic
// This prevents the method dispatch issue where base extractor calls b.GetComparisonsFromHTML()
func (e *AmazonUSExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("🚀 Starting Amazon USA product extraction", 
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	// Build Amazon USA search URL
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Fetch HTML using base functionality with gzip support
	html, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using Amazon USA-specific logic
	comparisons, err := e.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comparisons: %w", err)
	}

	utils.Info("Extraction completed", 
		utils.String("extractor", "amazon_usa"),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// GetComparisonsFromHTML overrides base implementation for Amazon US-specific logic
func (e *AmazonUSExtractor) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing Amazon USA HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return comparisons, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Amazon USA uses specific selectors for search results
	productSelector := `div[data-asin][data-component-type="s-search-result"]`
	
	doc.Find(productSelector).Each(func(i int, s *goquery.Selection) {
		// Extract ASIN
		asin, exists := s.Attr("data-asin")
		if !exists || asin == "" {
			return // Skip items without ASIN
		}

		// Extract product title
		titleElement := s.Find("h2.a-size-medium span")
		if titleElement.Length() == 0 {
			titleElement = s.Find("h2.a-size-base-plus span")
		}
		if titleElement.Length() == 0 {
			titleElement = s.Find("h2 a span")
		}
		title := strings.TrimSpace(titleElement.Text())
		if title == "" {
			return // Skip items without title
		}

		// Extract price
		price, currency := e.extractPrice(s)
		if price <= 0 {
			return // Skip items without valid price
		}

		// Extract product URL
		productURL := e.extractProductURL(s, asin)
		if productURL == "" {
			return // Skip items without URL
		}

		// Extract image URL
		imageURL := e.extractImageURL(s)
		var imageURLPtr *string
		if imageURL != "" {
			imageURLPtr = &imageURL
		}

		// Create comparison object
		comparison := models.ProductComparison{
			ID:          fmt.Sprintf("amazon_usa_%s", asin),
			ProductName: title,
			Price:       price,
			Currency:    currency,
			StoreName:   "Amazon",
			StoreURL:    &productURL,
			Country:     string(models.CountryUS),
			ImageURL:    imageURLPtr,
		}
		
		comparisons = append(comparisons, comparison)
	})
	
	utils.Info("Extracted Amazon USA products", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractPrice extracts price and currency from Amazon product element
func (e *AmazonUSExtractor) extractPrice(s *goquery.Selection) (float64, string) {
	// Try different price selectors
	priceSelectors := []string{
		"span.a-price-whole",                          // Whole price part
		"span.a-color-base",                           // Main price span
		"span[class*='price'] span.a-color-base",      // Nested price spans
		"div[data-cy='price-recipe'] span.a-color-base", // Price recipe container
		"span.a-price.a-text-price.a-size-medium.a-color-base", // Full price class
	}

	for _, selector := range priceSelectors {
		priceElement := s.Find(selector)
		if priceElement.Length() > 0 {
			priceText := strings.TrimSpace(priceElement.Text())
			if priceText != "" {
				price, currency := e.parsePrice(priceText)
				if price > 0 {
					return price, currency
				}
			}
		}
	}

	return 0, "USD"
}

// parsePrice parses price text and extracts numeric value and currency
func (e *AmazonUSExtractor) parsePrice(priceText string) (float64, string) {
	// Clean the price text
	priceText = strings.TrimSpace(priceText)
	
	// Default currency for Amazon USA
	currency := "USD"
	
	// Remove currency symbols and extra spaces
	priceText = strings.ReplaceAll(priceText, "$", "")
	priceText = strings.ReplaceAll(priceText, ",", "")
	priceText = strings.TrimSpace(priceText)
	
	// Extract numeric value using regex
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(priceText)
	if len(matches) > 1 {
		if price, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return price, currency
		}
	}
	
	return 0, currency
}

// extractProductURL extracts the product URL from Amazon element
func (e *AmazonUSExtractor) extractProductURL(s *goquery.Selection, asin string) string {
	// Try to find product link
	linkElement := s.Find("h2 a")
	if linkElement.Length() == 0 {
		linkElement = s.Find("a[href*='/dp/']")
	}
	
	if linkElement.Length() > 0 {
		href, exists := linkElement.Attr("href")
		if exists && href != "" {
			// Normalize URL
			if strings.HasPrefix(href, "http") {
				return href
			} else if strings.HasPrefix(href, "/") {
				return "https://www.amazon.com" + href
			}
		}
	}
	
	// Fallback to constructing URL from ASIN
	if asin != "" {
		return fmt.Sprintf("https://www.amazon.com/dp/%s", asin)
	}
	
	return ""
}

// extractImageURL extracts the product image URL from Amazon element
func (e *AmazonUSExtractor) extractImageURL(s *goquery.Selection) string {
	// Try to find product image
	imgElement := s.Find("img.s-image")
	if imgElement.Length() == 0 {
		imgElement = s.Find("img[data-image-latency='s-product-image']")
	}
	if imgElement.Length() == 0 {
		imgElement = s.Find("img[src*='images-amazon']")
	}
	
	if imgElement.Length() > 0 {
		src, exists := imgElement.Attr("src")
		if exists && src != "" {
			return src
		}
	}
	
	return ""
}