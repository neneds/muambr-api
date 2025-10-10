package extractors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"muambr-api/models"
	"muambr-api/utils"
)

// AcharPromoExtractor implements the Extractor interface for achar.promo marketplace
type AcharPromoExtractor struct {
	countryCode models.Country
}

// NewAcharPromoExtractor creates a new achar.promo extractor for Brazil
func NewAcharPromoExtractor() *AcharPromoExtractor {
	return &AcharPromoExtractor{
		countryCode: models.CountryBrazil, // achar.promo is Brazil-specific
	}
}

// GetCountryCode returns the ISO country code this extractor supports
func (e *AcharPromoExtractor) GetCountryCode() models.Country {
	return e.countryCode
}

// GetMacroRegion returns the macro region this extractor supports
func (e *AcharPromoExtractor) GetMacroRegion() models.MacroRegion {
	return e.countryCode.GetMacroRegion()
}

// BaseURL returns the base URL for the extractor's website
func (e *AcharPromoExtractor) BaseURL() string {
	return "https://achar.promo"
}

// GetIdentifier returns a static string identifier for this extractor
func (e *AcharPromoExtractor) GetIdentifier() string {
	return "acharpromo"
}

// AcharPromoAPIRequest represents the request payload for the API
type AcharPromoAPIRequest struct {
	Query string `json:"query"`
}

// AcharPromoAPIResponse represents the API response structure
type AcharPromoAPIResponse struct {
	Results []AcharPromoProduct `json:"results"`
}

// AcharPromoProduct represents a product from the API response
type AcharPromoProduct struct {
	Position        int                        `json:"position"`
	ProductID       string                     `json:"product_id"`
	Title           string                     `json:"title"`
	ProductLink     string                     `json:"product_link"`
	Offers          string                     `json:"offers"`
	OffersLink      string                     `json:"offers_link"`
	Price           string                     `json:"price"`
	ExtractedPrice  float64                    `json:"extracted_price"`
	Installment     *AcharPromoInstallment     `json:"installment,omitempty"`
	Rating          float64                    `json:"rating,omitempty"`
	Reviews         int                        `json:"reviews,omitempty"`
	Seller          string                     `json:"seller,omitempty"`
	Thumbnail       string                     `json:"thumbnail,omitempty"`
	Condition       string                     `json:"condition,omitempty"`
	DeliveryReturn  string                     `json:"delivery_return,omitempty"`
	OriginalPrice   string                     `json:"original_price,omitempty"`
	ExtractedOriginalPrice float64            `json:"extracted_original_price,omitempty"`
	ProductToken    string                     `json:"product_token"`
}

// AcharPromoInstallment represents installment information
type AcharPromoInstallment struct {
	DownPayment              string  `json:"down_payment"`
	ExtractedDownPayment     float64 `json:"extracted_down_payment"`
	Months                   string  `json:"months,omitempty"`
	ExtractedMonths          int     `json:"extracted_months,omitempty"`
	CostPerMonth             string  `json:"cost_per_month,omitempty"`
	ExtractedCostPerMonth    float64 `json:"extracted_cost_per_month,omitempty"`
}

// GetComparisons searches for product comparisons on achar.promo using their API
func (e *AcharPromoExtractor) GetComparisons(searchTerm string) ([]models.ProductComparison, error) {
	utils.Info("Starting achar.promo API product extraction",
		utils.String("search_term", searchTerm),
		utils.String("extractor", e.GetIdentifier()))

	// Make API request
	apiResponse, err := e.makeAPIRequest(searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}

	// Convert API response to ProductComparison objects
	comparisons := e.convertAPIResponseToComparisons(apiResponse, searchTerm)

	utils.Info("Successfully extracted products from achar.promo API",
		utils.String("search_term", searchTerm),
		utils.Int("product_count", len(comparisons)))

	return comparisons, nil
}

// makeAPIRequest makes a POST request to the achar.promo API
func (e *AcharPromoExtractor) makeAPIRequest(searchTerm string) (*AcharPromoAPIResponse, error) {
	apiURL := "https://achar.promo/api/text-search"
	
	utils.Info("Making API request to achar.promo",
		utils.String("url", apiURL),
		utils.String("search_term", searchTerm),
		utils.String("extractor", e.GetIdentifier()))

	// Create request payload
	requestPayload := AcharPromoAPIRequest{
		Query: searchTerm,
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers to mimic browser request
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9,pt;q=0.8")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://achar.promo")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", fmt.Sprintf("https://achar.promo/search?q=%s", searchTerm))
	req.Header.Set("sec-ch-ua", `"Microsoft Edge";v="141", "Not?A_Brand";v="8", "Chromium";v="141"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0")
	
	// You could add the cookie here if needed for session persistence:
	// req.Header.Set("cookie", "ph_phc_UslDhFX6xsEB2f2LBFXk1SjxGWhz9QQOfPhD5VnM9in_posthog=...")

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

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	utils.Info("API request successful for achar.promo",
		utils.String("url", apiURL),
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("response_size_bytes", len(body)))

	// Parse JSON response
	var apiResponse AcharPromoAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		utils.LogError("Failed to parse JSON response from achar.promo API",
			utils.String("extractor", e.GetIdentifier()),
			utils.Error(err),
			utils.String("response_preview", string(body[:min(len(body), 200)])))
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	utils.Info("Successfully parsed API response from achar.promo",
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("product_count", len(apiResponse.Results)))

	return &apiResponse, nil
}

// convertAPIResponseToComparisons converts the API response to ProductComparison objects
func (e *AcharPromoExtractor) convertAPIResponseToComparisons(apiResponse *AcharPromoAPIResponse, searchTerm string) []models.ProductComparison {
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
func (e *AcharPromoExtractor) convertProductToComparison(product AcharPromoProduct, searchTerm string) (models.ProductComparison, error) {
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
func (e *AcharPromoExtractor) extractStoreName(product AcharPromoProduct) string {
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

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}