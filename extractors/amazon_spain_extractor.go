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

// AmazonSpainParser implements HTMLParser interface for Amazon Spain
type AmazonSpainParser struct {
	*BaseHTMLParser
}

// NewAmazonSpainParser creates a new Amazon Spain-specific parser
func NewAmazonSpainParser() *AmazonSpainParser {
	return &AmazonSpainParser{
		BaseHTMLParser: NewBaseHTMLParser("amazon_spain"),
	}
}

// GetProductSelectors returns CSS selectors for finding product containers
func (p *AmazonSpainParser) GetProductSelectors() []string {
	return []string{
		`div[data-asin][data-component-type="s-search-result"]`,
		`div[data-asin].s-result-item`,
		`div.s-result-item`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *AmazonSpainParser) GetNameSelectors() []string {
	return []string{
		`h2.a-size-base-plus span`,
		`h2 a span`,
		`h2 span`,
		`.a-text-normal`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *AmazonSpainParser) GetPriceSelectors() []string {
	return []string{
		`span.a-color-base`,
		`span.a-price-whole`,
		`div[data-cy='price-recipe'] span.a-color-base`,
		`.a-price .a-offscreen`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *AmazonSpainParser) GetURLSelectors() []string {
	return []string{
		`h2 a[href]`,
		`a[href*="/dp/"]`,
		`.s-link-style[href]`,
	}
}

// ParseProductName extracts product name from HTML fragment
func (p *AmazonSpainParser) ParseProductName(html string) string {
	selectors := p.GetNameSelectors()
	for _, selector := range selectors {
		if name := p.extractWithRegex(selector, html); name != "" {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

// ParsePrice extracts price and currency from HTML fragment  
func (p *AmazonSpainParser) ParsePrice(html string) (float64, string, error) {
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
func (p *AmazonSpainParser) ParseURL(html string, baseURL string) string {
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
	return ""
}

// ParseStore extracts the store name from HTML fragment
func (p *AmazonSpainParser) ParseStore(html string) string {
	return "Amazon Spain"
}

// AmazonSpainExtractor implements the Extractor interface for Amazon Spain
type AmazonSpainExtractor struct {
	*BaseGoExtractor
}

// NewAmazonSpainExtractor creates a new Amazon Spain extractor instance
func NewAmazonSpainExtractor() *AmazonSpainExtractor {
	parser := NewAmazonSpainParser()
	baseExtractor := NewBaseGoExtractor(
		"https://www.amazon.es",
		models.CountrySpain,
		"amazon_spain", 
		parser,
	)
	
	return &AmazonSpainExtractor{
		BaseGoExtractor: baseExtractor,
	}
}

// GetCountryCode returns the country code for Spain
func (e *AmazonSpainExtractor) GetCountryCode() models.Country {
	return models.CountrySpain
}

// GetMacroRegion returns the macro region for Spain  
func (e *AmazonSpainExtractor) GetMacroRegion() models.MacroRegion {
	return models.MacroRegionEU
}

// GetIdentifier returns a unique identifier for this extractor
func (e *AmazonSpainExtractor) GetIdentifier() string {
	return "amazon_spain"
}

// BaseURL returns the base URL for Amazon Spain
func (e *AmazonSpainExtractor) BaseURL() string {
	return "https://www.amazon.es"
}

// BuildSearchURL constructs the search URL for Amazon Spain
func (e *AmazonSpainExtractor) BuildSearchURL(productName string) (string, error) {
	if productName == "" {
		return "", fmt.Errorf("product name cannot be empty")
	}

	// Build search parameters
	params := url.Values{}
	params.Set("k", productName)
	params.Set("ref", "sr_pg_1")
	
	searchURL := fmt.Sprintf("%s/s?%s", e.BaseURL(), params.Encode())
	
	utils.Info("🔗 Built Amazon Spain search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisons overrides the base implementation to use Amazon Spain-specific logic
// This prevents the method dispatch issue where base extractor calls b.GetComparisonsFromHTML()
func (e *AmazonSpainExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("🚀 Starting Amazon Spain product extraction", 
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	// Build Amazon Spain search URL
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Fetch HTML using base functionality with gzip support
	html, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using Amazon Spain-specific logic
	comparisons, err := e.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comparisons: %w", err)
	}

	utils.Info("Extraction completed", 
		utils.String("extractor", "amazon_spain"),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// GetComparisonsFromHTML overrides base implementation for Amazon Spain-specific logic
func (e *AmazonSpainExtractor) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing Amazon Spain HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return comparisons, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Amazon Spain uses specific selectors for search results
	productSelector := `div[data-asin][data-component-type="s-search-result"]`
	
	doc.Find(productSelector).Each(func(i int, s *goquery.Selection) {
		// Extract ASIN
		asin, exists := s.Attr("data-asin")
		if !exists || asin == "" {
			return // Skip items without ASIN
		}

		// Extract product title
		titleElement := s.Find("h2.a-size-base-plus span")
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

		// Create comparison object using correct field names
		comparison := models.ProductComparison{
			ID:          fmt.Sprintf("amazon_spain_%s", asin),
			ProductName: title,
			Price:       price,
			Currency:    currency,
			StoreName:   "Amazon Spain",
			StoreURL:    &productURL,
			Country:     string(e.GetCountryCode()),
			ImageURL:    imageURLPtr,
		}
		
		comparisons = append(comparisons, comparison)
	})
	
	utils.Info("Extracted Amazon Spain products", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractPrice extracts price and currency from Amazon product element
func (e *AmazonSpainExtractor) extractPrice(s *goquery.Selection) (float64, string) {
	// Try different price selectors
	priceSelectors := []string{
		"span.a-color-base",                           // Main price span
		"span.a-price-whole",                          // Whole price part
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

	return 0, "EUR"
}

// parsePrice parses price text and extracts numeric value and currency
func (e *AmazonSpainExtractor) parsePrice(priceText string) (float64, string) {
	// Clean the price text
	priceText = strings.TrimSpace(priceText)
	
	// Default currency for Amazon Spain
	currency := "EUR"
	
	// Remove currency symbols and extra spaces
	priceText = strings.ReplaceAll(priceText, "€", "")
	priceText = strings.ReplaceAll(priceText, ",", ".")
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
func (e *AmazonSpainExtractor) extractProductURL(s *goquery.Selection, asin string) string {
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
				return e.BaseURL() + href
			}
		}
	}
	
	// Fallback to constructing URL from ASIN
	if asin != "" {
		return fmt.Sprintf("%s/dp/%s", e.BaseURL(), asin)
	}
	
	return ""
}

// extractImageURL extracts the product image URL from Amazon element
func (e *AmazonSpainExtractor) extractImageURL(s *goquery.Selection) string {
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