package extractors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
)

// KuantoKustaExtractor implements the Extractor interface for KuantoKusta price comparison site
type KuantoKustaExtractor struct {
	countryCode models.Country
}

// NewKuantoKustaExtractor creates a new KuantoKusta extractor for Portugal
func NewKuantoKustaExtractor() *KuantoKustaExtractor {
	return &KuantoKustaExtractor{
		countryCode: models.CountryPortugal, // KuantoKusta is Portugal-specific
	}
}

// GetCountryCode returns the ISO country code this extractor supports
func (e *KuantoKustaExtractor) GetCountryCode() models.Country {
	return e.countryCode
}

// GetMacroRegion returns the macro region this extractor supports
func (e *KuantoKustaExtractor) GetMacroRegion() models.MacroRegion {
	return e.countryCode.GetMacroRegion()
}

// BaseURL returns the base URL for the extractor's website
func (e *KuantoKustaExtractor) BaseURL() string {
	return "https://www.kuantokusta.pt"
}

// GetIdentifier returns a static string identifier for this extractor
func (e *KuantoKustaExtractor) GetIdentifier() string {
	return "kuantokusta"
}

// GetComparisons extracts product comparisons from KuantoKusta for the given product name
func (e *KuantoKustaExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	// Build the search URL with query parameters
	searchURL, err := e.buildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Make HTTP request to get the HTML page
	htmlContent, err := e.fetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using Python script
	comparisons, err := e.extractWithPython(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract with Python: %w", err)
	}

	return comparisons, nil
}

// buildSearchURL constructs the search URL with proper query parameters
func (e *KuantoKustaExtractor) buildSearchURL(productName string) (string, error) {
	baseURL := e.BaseURL()
	
	// Build query parameters for KuantoKusta
	params := url.Values{}
	params.Add("q", productName)
	
	// Construct full URL using /search endpoint
	fullURL := fmt.Sprintf("%s/search?%s", baseURL, params.Encode())
	return fullURL, nil
}

// fetchHTML makes an HTTP GET request and returns the HTML content
func (e *KuantoKustaExtractor) fetchHTML(url string) (string, error) {
	utils.Info("Starting HTTP request to KuantoKusta", 
		utils.String("url", url),
		utils.String("extractor", "kuantokusta"),
		utils.String("base_domain", e.BaseURL()))

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		utils.LogError("Failed to create HTTP request for KuantoKusta", 
			utils.String("url", url),
			utils.String("extractor", "kuantokusta"),
			utils.Error(err))
		return "", err
	}

	// Add headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-PT,pt;q=0.8,en-US;q=0.5,en;q=0.3")

	resp, err := client.Do(req)
	if err != nil {
		utils.LogError("HTTP request execution failed for KuantoKusta - possible anti-bot protection", 
			utils.String("url", url),
			utils.String("extractor", "kuantokusta"),
			utils.String("base_domain", e.BaseURL()),
			utils.String("user_agent", req.Header.Get("User-Agent")),
			utils.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Warn("HTTP request returned non-200 status code from KuantoKusta - possible anti-bot protection",
			utils.String("url", url),
			utils.String("extractor", "kuantokusta"),
			utils.String("base_domain", e.BaseURL()),
			utils.Int("status_code", resp.StatusCode),
			utils.String("status", resp.Status),
			utils.String("user_agent", req.Header.Get("User-Agent")))
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	utils.Info("Successfully received HTTP response from KuantoKusta",
		utils.String("url", url),
		utils.String("extractor", "kuantokusta"),
		utils.Int("status_code", resp.StatusCode),
		utils.Any("content_length", resp.ContentLength))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LogError("Failed to read response body from KuantoKusta", 
			utils.String("url", url),
			utils.String("extractor", "kuantokusta"),
			utils.Error(err))
		return "", err
	}

	utils.Info("Successfully fetched HTML content from KuantoKusta",
		utils.String("url", url),
		utils.String("extractor", "kuantokusta"),
		utils.Int("content_size_bytes", len(body)))

	return string(body), nil
}

// extractWithPython calls the Python script to extract product data from HTML
func (e *KuantoKustaExtractor) extractWithPython(htmlContent string) ([]models.ProductComparison, error) {
	// Get the absolute path to the Python script
	scriptPath, err := filepath.Abs("extractors/pythonExtractors/kuantokusta_page.py")
	if err != nil {
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

	// Prepare the Python command
	cmd := exec.Command(pythonPath, "-c", fmt.Sprintf(`
import sys
sys.path.append('%s')
from kuantokusta_page import extract_kuantokusta_products
import json

html_content = '''%s'''
products = extract_kuantokusta_products(html_content)
print(json.dumps(products))
`, filepath.Dir(scriptPath), strings.ReplaceAll(htmlContent, "'", "\\'")))

	// Execute the Python script
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Python script execution failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse the JSON output from Python
	var pythonProducts []map[string]interface{}
	err = json.Unmarshal(out.Bytes(), &pythonProducts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python output: %w", err)
	}

	// Convert to ProductComparison structs
	var comparisons []models.ProductComparison
	for _, product := range pythonProducts {
		// Get currency with fallback to extractor's country currency
		currency := utils.GetString(product["currency"])
		if currency == "" {
			currency = e.countryCode.GetCurrencyCode()
		}
		
		// Parse price as float64
		priceStr := utils.GetString(product["price"])
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			// Skip invalid price entries
			continue
		}

		// Generate unique ID
		id := fmt.Sprintf("kuantokusta_%d", len(comparisons)+1)
		
		// Extract store URL safely
		storeURL := utils.GetString(product["url"])
		var storeURLPtr *string
		if storeURL != "" {
			storeURLPtr = &storeURL
		}
		
		comparison := models.ProductComparison{
			ID:          id,
			ProductName: utils.GetString(product["name"]),
			Price:       price,
			Currency:    currency,
			StoreName:   utils.GetString(product["store"]),
			StoreURL:    storeURLPtr,
			Country:     string(e.countryCode),
		}
		comparisons = append(comparisons, comparison)
	}

	return comparisons, nil
}