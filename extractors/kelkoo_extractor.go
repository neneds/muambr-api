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
	"strings"

	"muambr-api/models"
)

// KelkooExtractor implements the Extractor interface for Kelkoo price comparison site
type KelkooExtractor struct {
	countryCode models.Country
}

// NewKelkooExtractor creates a new Kelkoo extractor for Spain
func NewKelkooExtractor() *KelkooExtractor {
	return &KelkooExtractor{
		countryCode: models.CountrySpain, // Kelkoo is Spain-specific
	}
}

// GetCountryCode returns the ISO country code this extractor supports
func (e *KelkooExtractor) GetCountryCode() models.Country {
	return e.countryCode
}

// GetMacroRegion returns the macro region this extractor supports
func (e *KelkooExtractor) GetMacroRegion() models.MacroRegion {
	return e.countryCode.GetMacroRegion()
}

// BaseURL returns the base URL for the extractor's website
func (e *KelkooExtractor) BaseURL() string {
	return "https://www.kelkoo.es"
}

// GetIdentifier returns a static string identifier for this extractor
func (e *KelkooExtractor) GetIdentifier() string {
	return "kelkoo"
}

// GetComparisons extracts product comparisons from Kelkoo for the given product name
func (e *KelkooExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
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

// buildSearchURL constructs the search URL with proper query parameters for Kelkoo
func (e *KelkooExtractor) buildSearchURL(productName string) (string, error) {
	baseURL := e.BaseURL()
	
	// Build query parameters for Kelkoo using 'consulta' parameter
	params := url.Values{}
	params.Add("consulta", productName)
	
	// Construct full URL using Kelkoo search format
	fullURL := fmt.Sprintf("%s/buscar?%s", baseURL, params.Encode())
	return fullURL, nil
}

// fetchHTML makes an HTTP GET request and returns the HTML content
func (e *KelkooExtractor) fetchHTML(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.8,en-US;q=0.5,en;q=0.3")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// extractWithPython calls the Python script to extract product data from HTML
func (e *KelkooExtractor) extractWithPython(htmlContent string) ([]models.ProductComparison, error) {
	// Get the absolute path to the Python script
	scriptPath, err := filepath.Abs("extractors/pythonExtractors/kelkoo_page.py")
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
from kelkoo_page import extract_kelkoo_products
import json

html_content = '''%s'''
products = extract_kelkoo_products(html_content)
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
		currency := getString(product["currency"])
		if currency == "" {
			currency = e.countryCode.GetCurrencyCode()
		}
		
		comparison := models.ProductComparison{
			Name:     getString(product["name"]),
			Price:    getString(product["price"]),
			Store:    getString(product["store"]),
			Currency: currency,
			URL:      getString(product["url"]),
		}
		comparisons = append(comparisons, comparison)
	}

	return comparisons, nil
}