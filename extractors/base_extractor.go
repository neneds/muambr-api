package extractors

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"muambr-api/models"
	"muambr-api/utils"
	"github.com/google/uuid"
	"github.com/dsnet/compress/brotli"
)

// HTMLParser defines the interface for parsing HTML content from specific sites
// Following Interface Segregation Principle - focused on HTML parsing only
type HTMLParser interface {
	ParseProductName(html string) string
	ParsePrice(html string) (float64, string, error) // price, currency, error
	ParseURL(html string, baseURL string) string
	ParseStore(html string) string
	GetProductSelectors() []string
	GetNameSelectors() []string
	GetPriceSelectors() []string
	GetURLSelectors() []string
}

// HTTPClient defines the interface for making HTTP requests
// Following Interface Segregation Principle - focused on HTTP operations only
type HTTPClient interface {
	FetchHTML(url string) (string, error)
	BuildSearchURL(productName string) (string, error)
	GetBaseURL() string
	GetUserAgent() string
}

// ProductExtractor combines both interfaces following Composition over Inheritance
type ProductExtractor interface {
	HTMLParser
	HTTPClient
}

// BaseHTTPExtractor provides common HTTP functionality for all extractors
// Following Single Responsibility Principle - handles only HTTP operations
type BaseHTTPExtractor struct {
	baseURL     string
	countryCode models.Country
	userAgent   string
	config      *utils.AntiBotConfig
}

// NewBaseHTTPExtractor creates a new base extractor with anti-bot protection
func NewBaseHTTPExtractor(baseURL string, countryCode models.Country) *BaseHTTPExtractor {
	config := utils.DefaultAntiBotConfig(baseURL)
	
	// Customize config based on country and site
	config.UserAgentRotation = true
	config.MinDelay = 1000 * time.Millisecond
	config.MaxDelay = 3000 * time.Millisecond
	
	return &BaseHTTPExtractor{
		baseURL:     baseURL,
		countryCode: countryCode,
		config:      config,
		userAgent:   utils.GetRandomUserAgent(),
	}
}

// readResponseBody reads and decompresses the response body if needed
func readResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	
	contentEncoding := resp.Header.Get("Content-Encoding")
	
	// Handle different compression types
	switch contentEncoding {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			utils.Warn("⚠️ Failed to create gzip reader, falling back to raw read", utils.Error(err))
			// Fall back to reading raw body
			return io.ReadAll(resp.Body)
		}
		defer gzipReader.Close()
		reader = gzipReader
		utils.Debug("📦 Detected gzip compression, decompressing content")
		
	case "br":
		// Brotli compression
		brotliReader, err := brotli.NewReader(resp.Body, nil)
		if err != nil {
			utils.Warn("⚠️ Failed to create brotli reader, falling back to raw read", utils.Error(err))
			// Fall back to reading raw body
			return io.ReadAll(resp.Body)
		}
		defer brotliReader.Close()
		reader = brotliReader
		utils.Debug("📦 Detected brotli compression, decompressing content")
		
	case "deflate":
		// Handle deflate if needed in the future
		utils.Debug("📦 Detected deflate compression, reading as-is (deflate support not implemented)")
		
	default:
		// No compression or unknown compression
		if contentEncoding != "" {
			utils.Debug("📦 Unknown compression type, reading as-is", utils.String("encoding", contentEncoding))
		}
	}
	
	// Read the content (compressed or uncompressed)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	
	utils.Debug("📄 Response body read successfully", 
		utils.Int("bytes", len(body)),
		utils.String("compression", contentEncoding))
	
	return body, nil
}

// FetchHTML implements the HTTPClient interface
func (b *BaseHTTPExtractor) FetchHTML(url string) (string, error) {
	utils.Info("🌐 Fetching HTML content", 
		utils.String("url", url),
		utils.String("country", string(b.countryCode)))

	resp, err := utils.MakeAntiBotRequest(url, b.config)
	if err != nil {
		utils.LogError("❌ Failed to fetch HTML", 
			utils.String("url", url),
			utils.Error(err))
		return "", fmt.Errorf("failed to fetch HTML: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		utils.Warn("⚠️ Non-200 status code", 
			utils.String("url", url),
			utils.Int("status", resp.StatusCode))
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	html := string(body)
	utils.Info("✅ Successfully fetched HTML", 
		utils.String("url", url),
		utils.Int("size", len(html)))

	return html, nil
}

// GetBaseURL implements the HTTPClient interface
func (b *BaseHTTPExtractor) GetBaseURL() string {
	return b.baseURL
}

// GetUserAgent implements the HTTPClient interface
func (b *BaseHTTPExtractor) GetUserAgent() string {
	return b.userAgent
}

// BuildSearchURL implements the HTTPClient interface (default implementation)
func (b *BaseHTTPExtractor) BuildSearchURL(productName string) (string, error) {
	// Default implementation - can be overridden by specific extractors
	encodedProduct := url.QueryEscape(productName)
	return fmt.Sprintf("%s/search?q=%s", b.baseURL, encodedProduct), nil
}

// BaseHTMLParser provides common HTML parsing utilities
// Following Single Responsibility Principle - handles only HTML parsing
type BaseHTMLParser struct {
	siteName string
}

// NewBaseHTMLParser creates a new base HTML parser
func NewBaseHTMLParser(siteName string) *BaseHTMLParser {
	return &BaseHTMLParser{
		siteName: siteName,
	}
}

// extractWithRegex is a utility function for regex-based extraction
func (b *BaseHTMLParser) extractWithRegex(pattern, html string) string {
	re := regexp.MustCompile("(?i)" + pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractMultipleWithRegex extracts multiple matches using regex patterns
func (b *BaseHTMLParser) extractMultipleWithRegex(patterns []string, html string) []string {
	var results []string
	
	for i, pattern := range patterns {
		utils.Debug("🔍 Trying extraction pattern", 
			utils.Int("pattern", i+1),
			utils.String("site", b.siteName))
			
		re := regexp.MustCompile("(?i)" + pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				result := strings.TrimSpace(match[1])
				if result != "" && len(result) > 3 {
					results = append(results, result)
				}
			}
		}
		
		if len(results) > 0 {
			utils.Debug("✅ Found matches with pattern", 
				utils.Int("pattern", i+1),
				utils.Int("count", len(results)),
				utils.String("site", b.siteName))
			break
		}
	}
	
	return results
}

// parsePrice handles price parsing with currency detection
func (b *BaseHTMLParser) parsePrice(priceText string, defaultCurrency string) (float64, string, error) {
	if priceText == "" {
		return 0, defaultCurrency, fmt.Errorf("empty price text")
	}

	// Detect currency from the text
	currency := defaultCurrency
	if strings.Contains(priceText, "€") || strings.Contains(priceText, "EUR") {
		currency = "EUR"
	} else if strings.Contains(priceText, "$") || strings.Contains(priceText, "USD") {
		currency = "USD"
	} else if strings.Contains(priceText, "R$") || strings.Contains(priceText, "BRL") {
		currency = "BRL"
	} else if strings.Contains(priceText, "£") || strings.Contains(priceText, "GBP") {
		currency = "GBP"
	}

	// Clean the price text
	cleanedPrice := priceText
	cleanedPrice = regexp.MustCompile(`[^\d.,]`).ReplaceAllString(cleanedPrice, "")
	
	// Handle different decimal formats
	if currency == "BRL" {
		// Brazilian format: 1.234,56 -> 1234.56
		if strings.Contains(cleanedPrice, ".") && strings.Contains(cleanedPrice, ",") {
			cleanedPrice = strings.ReplaceAll(cleanedPrice, ".", "")
			cleanedPrice = strings.ReplaceAll(cleanedPrice, ",", ".")
		} else if strings.Contains(cleanedPrice, ",") {
			cleanedPrice = strings.ReplaceAll(cleanedPrice, ",", ".")
		}
	} else {
		// European/US format: 1,234.56 -> 1234.56
		if strings.Contains(cleanedPrice, ",") && strings.Contains(cleanedPrice, ".") {
			cleanedPrice = strings.ReplaceAll(cleanedPrice, ",", "")
		}
	}

	price, err := strconv.ParseFloat(cleanedPrice, 64)
	if err != nil {
		utils.Debug("❌ Failed to parse price", 
			utils.String("original", priceText),
			utils.String("cleaned", cleanedPrice),
			utils.Error(err))
		return 0, currency, fmt.Errorf("failed to parse price: %w", err)
	}

	return price, currency, nil
}

// extractJSONLD extracts structured data from JSON-LD scripts
func (b *BaseHTMLParser) extractJSONLD(html string) ([]map[string]interface{}, error) {
	pattern := `<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`
	re := regexp.MustCompile("(?i)" + pattern)
	matches := re.FindAllStringSubmatch(html, -1)
	
	var results []map[string]interface{}
	
	for _, match := range matches {
		if len(match) > 1 {
			var data interface{}
			if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
				continue
			}
			
			// Handle different JSON-LD structures
			switch v := data.(type) {
			case map[string]interface{}:
				results = append(results, v)
			case []interface{}:
				for _, item := range v {
					if itemMap, ok := item.(map[string]interface{}); ok {
						results = append(results, itemMap)
					}
				}
			}
		}
	}
	
	return results, nil
}

// GoExtractor is the main interface that combines everything
// Following Dependency Inversion Principle - depends on abstractions
type GoExtractor interface {
	Extractor // Original interface
	GetComparisonsFromHTML(html string) ([]models.ProductComparison, error)
}

// BaseGoExtractor combines BaseHTTPExtractor and BaseHTMLParser
// Following Composition over Inheritance
type BaseGoExtractor struct {
	*BaseHTTPExtractor
	*BaseHTMLParser
	parser HTMLParser
	countryCode models.Country
	macroRegion models.MacroRegion
	identifier  string
}

// NewBaseGoExtractor creates a new base Go extractor
func NewBaseGoExtractor(baseURL string, countryCode models.Country, identifier string, parser HTMLParser) *BaseGoExtractor {
	return &BaseGoExtractor{
		BaseHTTPExtractor: NewBaseHTTPExtractor(baseURL, countryCode),
		BaseHTMLParser:    NewBaseHTMLParser(identifier),
		parser:            parser,
		countryCode:       countryCode,
		macroRegion:       countryCode.GetMacroRegion(),
		identifier:        identifier,
	}
}

// Implement the original Extractor interface
func (b *BaseGoExtractor) GetCountryCode() models.Country {
	return b.countryCode
}

func (b *BaseGoExtractor) GetMacroRegion() models.MacroRegion {
	return b.macroRegion
}

func (b *BaseGoExtractor) GetIdentifier() string {
	return b.identifier
}

func (b *BaseGoExtractor) BaseURL() string {
	return b.GetBaseURL()
}

// GetComparisons implements the main extraction workflow
func (b *BaseGoExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("🚀 Starting BASE product extraction", 
		utils.String("product", productName),
		utils.String("extractor", b.identifier),
		utils.String("country", string(b.countryCode)))

	// Build search URL
	searchURL, err := b.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Fetch HTML
	html, err := b.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using the specific parser
	comparisons, err := b.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comparisons: %w", err)
	}

	utils.Info("✅ Extraction completed", 
		utils.String("extractor", b.identifier),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// GetComparisonsFromHTML extracts products from HTML using the injected parser
func (b *BaseGoExtractor) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	var comparisons []models.ProductComparison
	
	// Get product selectors from the specific parser
	productSelectors := b.parser.GetProductSelectors()
	
	for _, selector := range productSelectors {
		// Use regex to find product containers
		products := b.extractMultipleWithRegex([]string{selector}, html)
		
		for _, productHTML := range products {
			// Parse individual product
			name := b.parser.ParseProductName(productHTML)
			if name == "" || len(name) < 3 {
				continue
			}

			price, currency, err := b.parser.ParsePrice(productHTML)
			if err != nil {
				utils.Debug("⚠️ Failed to parse price", 
					utils.String("product", name),
					utils.Error(err))
				continue
			}

			productURL := b.parser.ParseURL(productHTML, b.GetBaseURL())
			storeName := b.parser.ParseStore(productHTML)

			// Create comparison object
			var storeURLPtr *string
			if productURL != "" {
				storeURLPtr = &productURL
			}

			comparison := models.ProductComparison{
				ID:          uuid.New().String(),
				ProductName: name,
				Price:       price,
				Currency:    currency,
				StoreName:   storeName,
				StoreURL:    storeURLPtr,
				Country:     string(b.countryCode),
			}

			comparisons = append(comparisons, comparison)
		}
		
		// If we found results with this selector, stop trying others
		if len(comparisons) > 0 {
			break
		}
	}

	return comparisons, nil
}

// min helper function for use across extractors
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}