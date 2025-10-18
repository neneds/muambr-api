package extractors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
	"github.com/google/uuid"
)

// AcharPromoParser implements HTMLParser interface for AcharPromo Brazil
// Following Single Responsibility Principle - only handles AcharPromo parsing logic
type AcharPromoParser struct {
	*BaseHTMLParser
}

// NewAcharPromoParser creates a new AcharPromo-specific parser
func NewAcharPromoParser() *AcharPromoParser {
	return &AcharPromoParser{
		BaseHTMLParser: NewBaseHTMLParser("AcharPromo"),
	}
}

// GetProductSelectors returns CSS/regex selectors for finding product containers
func (p *AcharPromoParser) GetProductSelectors() []string {
	return []string{
		// AcharPromo specific patterns
		`<div[^>]*class="[^"]*product-item[^"]*"[^>]*>(.*?)</div>(?:\s*</div>)*`,
		`<div[^>]*class="[^"]*promo-card[^"]*"[^>]*>(.*?)</div>`,
		`<article[^>]*class="[^"]*product[^"]*"[^>]*>(.*?)</article>`,
		`<li[^>]*class="[^"]*search-result[^"]*"[^>]*>(.*?)</li>`,
		// More generic patterns
		`<div[^>]*data-product[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*card[^"]*"[^>]*>(.*?)</div>`,
	}
}

// GetNameSelectors returns selectors for extracting product names
func (p *AcharPromoParser) GetNameSelectors() []string {
	return []string{
		// AcharPromo specific patterns
		`<h[1-6][^>]*class="[^"]*product-title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		`<a[^>]*class="[^"]*product-link[^"]*"[^>]*title="([^"]+)"[^>]*>`,
		`<span[^>]*class="[^"]*product-name[^"]*"[^>]*>([^<]+)</span>`,
		`<h[1-6][^>]*class="[^"]*promo-title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
		// More flexible patterns
		`title="([^"]+)"`,
		`alt="([^"]+)"`,
		`<a[^>]*href="[^"]*"[^>]*>([^<]+)</a>`,
	}
}

// GetPriceSelectors returns selectors for extracting prices
func (p *AcharPromoParser) GetPriceSelectors() []string {
	return []string{
		// Brazilian price patterns with R$ symbol
		`<span[^>]*class="[^"]*price[^"]*"[^>]*>R\$\s*([0-9.,]+)</span>`,
		`<div[^>]*class="[^"]*price[^"]*"[^>]*>R?\$?\s*([0-9.,]+)</div>`,
		`<span[^>]*class="[^"]*amount[^"]*"[^>]*>([0-9.,]+)</span>`,
		// Currency-specific patterns
		`R\$\s*([0-9.,]+)`,
		`([0-9.,]+)\s*reais`,
		// Generic price patterns
		`<span[^>]*>[^R]*R?\$?\s*([0-9.,]+)[^<]*</span>`,
		`"price":\s*"?([0-9.,]+)"?`,
	}
}

// GetURLSelectors returns selectors for extracting product URLs
func (p *AcharPromoParser) GetURLSelectors() []string {
	return []string{
		`<a[^>]*href="([^"]*achar\.promo[^"]*)"[^>]*>`,
		`<a[^>]*class="[^"]*product-link[^"]*"[^>]*href="([^"]+)"[^>]*>`,
		`<a[^>]*href="([^"]+)"[^>]*class="[^"]*product[^"]*">`,
		`href="([^"]*\/product\/[^"]*)"`,
		`href="([^"]*product[^"]*)"`,
	}
}

// ParseProductName extracts the product name from HTML fragment
func (p *AcharPromoParser) ParseProductName(html string) string {
	selectors := p.GetNameSelectors()
	
	for _, selector := range selectors {
		if name := p.extractWithRegex(selector, html); name != "" {
			// Clean up the name
			name = strings.TrimSpace(name)
			name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
			
			// Validate name quality
			if len(name) > 3 && !strings.Contains(strings.ToLower(name), "achar") {
				return name
			}
		}
	}
	
	return ""
}

// ParsePrice extracts price and currency from HTML fragment
func (p *AcharPromoParser) ParsePrice(html string) (float64, string, error) {
	// First try to extract from JSON-LD if present
	if jsonProducts, err := p.extractJSONLD(html); err == nil {
		for _, product := range jsonProducts {
			if productType, ok := product["@type"].(string); ok && productType == "Product" {
				if offers, ok := product["offers"].(map[string]interface{}); ok {
					if priceStr, ok := offers["price"].(string); ok {
						if price, currency, err := p.parsePrice(priceStr, "BRL"); err == nil {
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
			if price, currency, err := p.parsePrice(priceText, "BRL"); err == nil {
				return price, currency, nil
			}
		}
	}
	
	return 0, "BRL", fmt.Errorf("no valid price found")
}

// ParseURL extracts the product URL from HTML fragment
func (p *AcharPromoParser) ParseURL(html string, baseURL string) string {
	selectors := p.GetURLSelectors()
	
	for _, selector := range selectors {
		if urlStr := p.extractWithRegex(selector, html); urlStr != "" {
			// Normalize URL
			if strings.HasPrefix(urlStr, "http") {
				return urlStr
			} else if strings.HasPrefix(urlStr, "/") {
				return "https://achar.promo" + urlStr
			} else {
				return "https://achar.promo/" + urlStr
			}
		}
	}
	
	return baseURL // Fallback to base URL
}

// ParseStore extracts the store name from HTML fragment
func (p *AcharPromoParser) ParseStore(html string) string {
	// Look for seller/store information
	storeSelectors := []string{
		`<span[^>]*class="[^"]*seller[^"]*"[^>]*>([^<]+)</span>`,
		`<div[^>]*class="[^"]*store[^"]*"[^>]*>([^<]+)</div>`,
		`<span[^>]*class="[^"]*merchant[^"]*"[^>]*>([^<]+)</span>`,
		`loja:\s*([^<\n]+)`,
		`vendido por\s+([^<\n]+)`,
	}
	
	for _, selector := range storeSelectors {
		if store := p.extractWithRegex(selector, html); store != "" {
			return strings.TrimSpace(store)
		}
	}
	
	return "AcharPromo Brasil" // Default store name
}

// API structures for AcharPromo V2 - shared with original extractor
// Note: These are redeclared to ensure compilation works properly

// AcharPromoAPIRequestV2 represents the API request payload for V2
type AcharPromoAPIRequestV2 struct {
	Query string `json:"query"`
}

// AcharPromoAPIResponseV2 represents the API response structure for V2  
type AcharPromoAPIResponseV2 struct {
	Results []AcharPromoProductV2 `json:"results"`
}

// AcharPromoProductV2 represents a product from the API response for V2
type AcharPromoProductV2 struct {
	Position        int                            `json:"position"`
	ProductID       string                         `json:"product_id"`
	Title           string                         `json:"title"`
	ProductLink     string                         `json:"product_link"`
	Offers          string                         `json:"offers"`
	OffersLink      string                         `json:"offers_link"`
	Price           string                         `json:"price"`
	ExtractedPrice  float64                        `json:"extracted_price"`
	Installment     *AcharPromoInstallmentV2       `json:"installment,omitempty"`
	Rating          float64                        `json:"rating,omitempty"`
	Reviews         int                            `json:"reviews,omitempty"`
	Seller          string                         `json:"seller,omitempty"`
	Thumbnail       string                         `json:"thumbnail,omitempty"`
	Condition       string                         `json:"condition,omitempty"`
	DeliveryReturn  string                         `json:"delivery_return,omitempty"`
	OriginalPrice   string                         `json:"original_price,omitempty"`
	ExtractedOriginalPrice float64                `json:"extracted_original_price,omitempty"`
	ProductToken    string                         `json:"product_token"`
}

// AcharPromoInstallmentV2 represents installment information for V2
type AcharPromoInstallmentV2 struct {
	DownPayment              string  `json:"down_payment"`
	ExtractedDownPayment     float64 `json:"extracted_down_payment"`
	Months                   string  `json:"months,omitempty"`
	ExtractedMonths          int     `json:"extracted_months,omitempty"`
	CostPerMonth             string  `json:"cost_per_month,omitempty"`
	ExtractedCostPerMonth    float64 `json:"extracted_cost_per_month,omitempty"`
}

// AcharPromoExtractorV2 is the new pure Go implementation with API support
type AcharPromoExtractorV2 struct {
	*BaseGoExtractor
	useAPIFirst bool // Flag to determine whether to try API first before HTML scraping
}

// NewAcharPromoExtractorV2 creates a new pure Go AcharPromo extractor
func NewAcharPromoExtractorV2() *AcharPromoExtractorV2 {
	parser := NewAcharPromoParser()
	baseExtractor := NewBaseGoExtractor(
		"https://achar.promo",
		models.CountryBrazil,
		"acharpromo_v2",
		parser,
	)
	
	return &AcharPromoExtractorV2{
		BaseGoExtractor: baseExtractor,
		useAPIFirst:     true, // Prefer API over HTML scraping for better reliability
	}
}

// BuildSearchURL overrides the base implementation for AcharPromo's specific URL format
func (e *AcharPromoExtractorV2) BuildSearchURL(productName string) (string, error) {
	// AcharPromo uses query parameters: /search?q=productName
	params := url.Values{}
	params.Add("q", productName)
	
	searchURL := fmt.Sprintf("%s/search?%s", e.GetBaseURL(), params.Encode())
	
	utils.Info("🔗 Built AcharPromo search URL", 
		utils.String("product", productName),
		utils.String("url", searchURL))
	
	return searchURL, nil
}

// GetComparisonsFromHTML overrides base implementation for AcharPromo-specific logic
func (e *AcharPromoExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing AcharPromo HTML", utils.Int("size", len(html)))
	
	var comparisons []models.ProductComparison
	
	// First try JSON-LD structured data (more reliable)
	if jsonComparisons := e.extractFromJSONLD(html); len(jsonComparisons) > 0 {
		utils.Info("✅ Extracted AcharPromo products from JSON-LD", utils.Int("count", len(jsonComparisons)))
		return jsonComparisons, nil
	}
	
	// Fallback to HTML parsing using base implementation
	comparisons, err := e.BaseGoExtractor.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, err
	}
	
	utils.Info("✅ Extracted AcharPromo products from HTML", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractFromJSONLD tries to extract products from structured data
func (e *AcharPromoExtractorV2) extractFromJSONLD(html string) []models.ProductComparison {
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

// GetComparisons overrides the base implementation to use API first, then HTML fallback
func (e *AcharPromoExtractorV2) GetComparisons(productName string) ([]models.ProductComparison, error) {
	// First try the API if enabled
	if e.useAPIFirst {
		apiComparisons, err := e.getComparisonsFromAPI(productName)
		if err == nil && len(apiComparisons) > 0 {
			utils.Info("✅ AcharPromo V2: Successfully retrieved products from API", 
				utils.String("product", productName),
				utils.Int("count", len(apiComparisons)))
			return apiComparisons, nil
		}
		
		// Log API failure but don't return error yet - fallback to HTML
		utils.Warn("⚠️ AcharPromo V2: API request failed, falling back to HTML scraping",
			utils.String("product", productName),
			utils.Error(err))
	}
	
	// Fallback to HTML scraping using the base implementation
	utils.Info("🔄 AcharPromo V2: Trying HTML scraping fallback", utils.String("product", productName))
	return e.BaseGoExtractor.GetComparisons(productName)
}

// getComparisonsFromAPI retrieves product comparisons using AcharPromo's API
func (e *AcharPromoExtractorV2) getComparisonsFromAPI(searchTerm string) ([]models.ProductComparison, error) {
	apiResponse, err := e.makeAPIRequest(searchTerm)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	
	if apiResponse == nil || len(apiResponse.Results) == 0 {
		return nil, fmt.Errorf("no products found in API response")
	}
	
	comparisons := e.convertAPIResponseToComparisons(apiResponse, searchTerm)
	return comparisons, nil
}

// makeAPIRequest makes a POST request to AcharPromo's text search API
func (e *AcharPromoExtractorV2) makeAPIRequest(searchTerm string) (*AcharPromoAPIResponseV2, error) {
	apiURL := "https://achar.promo/api/text-search"
	
	// Create request payload
	requestPayload := AcharPromoAPIRequestV2{
		Query: searchTerm,
	}
	
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	utils.Info("🚀 Making API request to achar.promo",
		utils.String("url", apiURL),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("search_term", searchTerm))

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers similar to the original implementation
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0")
	
	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		utils.LogError("API request failed for achar.promo",
			utils.String("url", apiURL),
			utils.String("extractor", e.GetIdentifier()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Warn("API request returned non-200 status code",
			utils.String("url", apiURL),
			utils.String("extractor", e.GetIdentifier()),
			utils.Int("status_code", resp.StatusCode),
			utils.String("status", resp.Status))
		return nil, fmt.Errorf("API request failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	// Read response body with proper decompression handling
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	utils.Info("API request successful for achar.promo",
		utils.String("url", apiURL),
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("response_size_bytes", len(body)))

	// Parse JSON response
	var apiResponse AcharPromoAPIResponseV2
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		utils.LogError("Failed to parse JSON response from achar.promo API",
			utils.String("extractor", e.GetIdentifier()),
			utils.Error(err),
			utils.String("response_preview", func() string {
			if len(body) > 200 {
				return string(body[:200])
			}
			return string(body)
		}()))
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	utils.Info("Successfully parsed API response from achar.promo",
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("product_count", len(apiResponse.Results)))

	return &apiResponse, nil
}

// convertAPIResponseToComparisons converts the API response to ProductComparison objects
func (e *AcharPromoExtractorV2) convertAPIResponseToComparisons(apiResponse *AcharPromoAPIResponseV2, searchTerm string) []models.ProductComparison {
	var comparisons []models.ProductComparison
	
	for _, product := range apiResponse.Results {
		comparison, err := e.convertProductToComparison(product, searchTerm)
		if err != nil {
			utils.Warn("Failed to convert API product to comparison for achar.promo",
				utils.String("extractor", e.GetIdentifier()),
				utils.String("product_title", product.Title),
				utils.String("product_id", product.ProductID),
				utils.Error(err))
			continue // Skip invalid products
		}
		comparisons = append(comparisons, comparison)
	}
	
	return comparisons
}

// convertProductToComparison converts an API product to a ProductComparison
func (e *AcharPromoExtractorV2) convertProductToComparison(product AcharPromoProductV2, searchTerm string) (models.ProductComparison, error) {
	// Generate a unique ID for the product
	productID := uuid.New().String()
	
	// Extract store name from seller or product link
	storeName := e.extractStoreName(product)
	
	// Convert string fields to pointers where needed
	var storeURL, imageURL, condition *string
	if product.ProductLink != "" {
		storeURL = &product.ProductLink
	}
	if product.Thumbnail != "" {
		imageURL = &product.Thumbnail
	}
	if product.Condition != "" {
		condition = &product.Condition
	}

	// Create the product comparison
	comparison := models.ProductComparison{
		ID:          productID,
		ProductName: strings.TrimSpace(product.Title),
		Price:       product.ExtractedPrice,
		Currency:    "BRL", // Brazilian Real
		StoreURL:    storeURL,
		StoreName:   storeName,
		ImageURL:    imageURL,
		Country:     string(e.GetCountryCode()),
		Condition:   condition,
	}

	// Validate required fields
	if comparison.ProductName == "" || comparison.Price <= 0 {
		return models.ProductComparison{}, fmt.Errorf("invalid product: missing name or price")
	}

	utils.Debug("Converted API product to comparison",
		utils.String("extractor", e.GetIdentifier()),
		utils.String("product_name", comparison.ProductName),
		utils.Float64("price", comparison.Price),
		utils.String("store_name", comparison.StoreName))

	return comparison, nil
}

// extractStoreName extracts store name from the product seller or product link
func (e *AcharPromoExtractorV2) extractStoreName(product AcharPromoProductV2) string {
	if product.Seller != "" {
		return strings.TrimSpace(product.Seller)
	}
	
	// If no seller info, try to extract from product link domain
	if product.ProductLink != "" {
		// For achar.promo, the links go to Google Shopping, so we use the seller field primarily
		return "Google Shopping"
	}
	
	return "Unknown Store"
}

// parseJSONProduct converts a JSON-LD product to ProductComparison
func (e *AcharPromoExtractorV2) parseJSONProduct(product map[string]interface{}) *models.ProductComparison {
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
		StoreName:   "AcharPromo Brasil",
		StoreURL:    storeURL,
		Country:     string(models.CountryBrazil),
	}
}