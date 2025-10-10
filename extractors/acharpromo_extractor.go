package extractors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// GetComparisons searches for product comparisons on achar.promo
func (e *AcharPromoExtractor) GetComparisons(searchTerm string) ([]models.ProductComparison, error) {
	utils.Info("Starting achar.promo product extraction",
		utils.String("search_term", searchTerm),
		utils.String("extractor", e.GetIdentifier()))

	// Construct the search URL
	searchURL := e.buildSearchURL(searchTerm)
	
	utils.Info("Built achar.promo search URL",
		utils.String("url", searchURL),
		utils.String("search_term", searchTerm))

	// Fetch HTML content
	htmlContent, err := e.fetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using Python script
	comparisons, err := e.extractProducts(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract products: %w", err)
	}

	utils.Info("Successfully extracted products from achar.promo",
		utils.String("search_term", searchTerm),
		utils.Int("product_count", len(comparisons)))

	return comparisons, nil
}

// buildSearchURL constructs the achar.promo search URL
func (e *AcharPromoExtractor) buildSearchURL(searchTerm string) string {
	params := url.Values{}
	params.Add("q", searchTerm)
	return fmt.Sprintf("%s/search?%s", e.BaseURL(), params.Encode())
}

// fetchHTML makes an HTTP GET request and returns the HTML content
func (e *AcharPromoExtractor) fetchHTML(url string) (string, error) {
	utils.Info("Starting HTTP request to achar.promo", 
		utils.String("url", url),
		utils.String("extractor", "acharpromo"),
		utils.String("base_domain", e.BaseURL()))

	// Create anti-bot configuration for achar.promo
	config := utils.DefaultAntiBotConfig(e.BaseURL())
	
	// Make request using anti-bot utility
	resp, err := utils.MakeAntiBotRequest(url, config)
	if err != nil {
		utils.LogError("HTTP request execution failed for achar.promo - possible anti-bot protection", 
			utils.String("url", url),
			utils.String("extractor", "acharpromo"),
			utils.String("base_domain", e.BaseURL()),
			utils.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		utils.Warn("HTTP request returned non-200 status code from achar.promo - possible anti-bot protection",
			utils.String("url", url),
			utils.String("extractor", "acharpromo"),
			utils.String("base_domain", e.BaseURL()),
			utils.Int("status_code", resp.StatusCode),
			utils.String("status", resp.Status))
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	utils.Info("HTTP request successful for achar.promo",
		utils.String("url", url),
		utils.String("extractor", "acharpromo"),
		utils.String("base_domain", e.BaseURL()),
		utils.Int("status_code", resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LogError("Failed to read response body from achar.promo",
			utils.String("url", url),
			utils.String("extractor", "acharpromo"),
			utils.Error(err))
		return "", err
	}

	return string(body), nil
}

// extractProducts uses the Python script to extract product information from HTML
func (e *AcharPromoExtractor) extractProducts(htmlContent string) ([]models.ProductComparison, error) {
	utils.Info("Starting Python extraction for achar.promo",
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("html_size_bytes", len(htmlContent)))

	// Get the absolute path to the Python script
	scriptPath, err := filepath.Abs("extractors/pythonExtractors/acharpromo_page.py")
	if err != nil {
		utils.LogError("Failed to get Python script path for achar.promo",
			utils.String("extractor", e.GetIdentifier()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to get script path: %w", err)
	}

	// Get Python path from environment or use default
	pythonPath := os.Getenv("PYTHON_PATH")
	if pythonPath == "" {
		// Check if we're in development with a virtual environment
		venvPath := "/Users/dennismerli/Documents/Projects/muambr-goapi/.venv/bin/python"
		if _, err := os.Stat(venvPath); err == nil {
			pythonPath = venvPath
		} else {
			pythonPath = "python3" // Default for production environments like Render
		}
	}

	utils.Info("Using Python interpreter for achar.promo extraction",
		utils.String("extractor", e.GetIdentifier()),
		utils.String("python_path", pythonPath),
		utils.String("script_path", scriptPath))

	// Create temporary file for HTML content
	tempFile, err := e.createTempHTMLFile(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary HTML file: %w", err)
	}
	defer os.Remove(tempFile) // Clean up

	utils.Info("Created temporary HTML file for achar.promo extraction",
		utils.String("extractor", e.GetIdentifier()),
		utils.String("temp_file", tempFile))

	// Execute Python script
	utils.Info("Executing Python extraction script for achar.promo",
		utils.String("extractor", e.GetIdentifier()),
		utils.String("temp_file", tempFile))

	cmd := exec.Command(pythonPath, scriptPath, tempFile)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		utils.Warn("Python script execution failed for achar.promo",
			utils.String("extractor", e.GetIdentifier()),
			utils.String("error", err.Error()),
			utils.String("stderr", stderr.String()))
		return nil, fmt.Errorf("python script failed: %w, stderr: %s", err, stderr.String())
	}

	utils.Info("Python script executed successfully for achar.promo",
		utils.String("extractor", e.GetIdentifier()),
		utils.String("python_stdout", out.String()),
		utils.String("python_stderr", stderr.String()))

	// Parse Python output
	return e.parsePythonOutput(out.String())
}

// createTempHTMLFile creates a temporary file with the HTML content
func (e *AcharPromoExtractor) createTempHTMLFile(htmlContent string) (string, error) {
	tempFile, err := os.CreateTemp("", "acharpromo_html_*.html")
	if err != nil {
		utils.LogError("Failed to create temporary file for achar.promo HTML",
			utils.String("extractor", e.GetIdentifier()),
			utils.Error(err))
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Write HTML content to temporary file
	_, err = tempFile.WriteString(htmlContent)
	if err != nil {
		utils.LogError("Failed to write HTML to temporary file for achar.promo",
			utils.String("extractor", e.GetIdentifier()),
			utils.String("temp_file", tempFile.Name()),
			utils.Error(err))
		return "", fmt.Errorf("failed to write HTML to temp file: %w", err)
	}

	// Ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		utils.LogError("Failed to sync temporary file for achar.promo",
			utils.String("extractor", e.GetIdentifier()),
			utils.String("temp_file", tempFile.Name()),
			utils.Error(err))
		return "", fmt.Errorf("failed to sync temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// parsePythonOutput parses the JSON output from the Python script
func (e *AcharPromoExtractor) parsePythonOutput(output string) ([]models.ProductComparison, error) {
	// The Python script returns {"products": [...], "total": N, "source": "achar.promo"}
	var result struct {
		Products []Product `json:"products"`
		Total    int       `json:"total"`
		Source   string    `json:"source"`
	}
	
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		utils.LogError("Failed to parse Python output for achar.promo",
			utils.String("extractor", e.GetIdentifier()),
			utils.String("raw_output", output),
			utils.Error(err))
		return nil, fmt.Errorf("failed to parse Python output: %w", err)
	}
	
	products := result.Products

	var comparisons []models.ProductComparison
	for _, product := range products {
		comparison, err := e.convertToComparison(product)
		if err != nil {
			utils.Warn("Failed to convert product to comparison for achar.promo",
				utils.String("extractor", e.GetIdentifier()),
				utils.String("product_name", product.Name),
				utils.Error(err))
			continue // Skip invalid products
		}
		comparisons = append(comparisons, comparison)
	}

	return comparisons, nil
}

// Product represents a product from achar.promo
type Product struct {
	Name        string  `json:"name"`
	PriceText   string  `json:"price_text"`
	Price       float64 `json:"price"`
	StoreURL    string  `json:"store_url"`
	ImageURL    string  `json:"image_url"`
	Description string  `json:"description"`
}

// convertToComparison converts a Product to a ProductComparison
func (e *AcharPromoExtractor) convertToComparison(product Product) (models.ProductComparison, error) {
	// Generate a unique ID for the product
	productID := uuid.New().String()

	// Parse the price
	price := product.Price
	if price <= 0 {
		// Try to extract price from price text if the numeric price is not available
		if priceFromText, err := e.extractPriceFromText(product.PriceText); err == nil {
			price = priceFromText
		} else {
			return models.ProductComparison{}, fmt.Errorf("invalid price: %f", price)
		}
	}

	// Convert string fields to pointers where needed
	var storeURL, imageURL, description *string
	if product.StoreURL != "" {
		storeURL = &product.StoreURL
	}
	if product.ImageURL != "" {
		imageURL = &product.ImageURL
	}
	if product.Description != "" {
		description = &product.Description
	}

	return models.ProductComparison{
		ID:          productID,
		ProductName: product.Name,
		Price:       price,
		Currency:    "BRL", // achar.promo uses Brazilian Real
		StoreName:   "achar.promo",
		StoreURL:    storeURL,
		ImageURL:    imageURL,
		Description: description,
		Country:     string(e.GetCountryCode()),
	}, nil
}

// extractPriceFromText extracts price from text like "R$ 1.234,56"
func (e *AcharPromoExtractor) extractPriceFromText(priceText string) (float64, error) {
	if priceText == "" {
		return 0, fmt.Errorf("empty price text")
	}

	// Remove currency symbols and normalize
	cleanPrice := strings.ReplaceAll(priceText, "R$", "")
	cleanPrice = strings.ReplaceAll(cleanPrice, ".", "") // Remove thousands separator
	cleanPrice = strings.ReplaceAll(cleanPrice, ",", ".") // Replace decimal comma with dot
	cleanPrice = strings.TrimSpace(cleanPrice)

	// Parse the cleaned price
	price, err := strconv.ParseFloat(cleanPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price '%s': %w", priceText, err)
	}

	return price, nil
}