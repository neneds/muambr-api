package extractors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"muambr-api/models"
)

// IdealoExtractor implements the Extractor interface for Idealo price comparison site
type IdealoExtractor struct {
	countryCode models.Country
}

// NewIdealoExtractor creates a new Idealo extractor for Spain
func NewIdealoExtractor() *IdealoExtractor {
	return &IdealoExtractor{
		countryCode: models.CountrySpain, // Idealo is Spain-specific
	}
}

// GetCountryCode returns the ISO country code this extractor supports
func (e *IdealoExtractor) GetCountryCode() models.Country {
	return e.countryCode
}

// GetMacroRegion returns the macro region this extractor supports
func (e *IdealoExtractor) GetMacroRegion() models.MacroRegion {
	return e.countryCode.GetMacroRegion()
}

// BaseURL returns the base URL for the extractor's website
func (e *IdealoExtractor) BaseURL() string {
	return "https://www.idealo.es"
}

// GetIdentifier returns a static string identifier for this extractor
func (e *IdealoExtractor) GetIdentifier() string {
	return "idealo"
}

// GetComparisons extracts product comparisons from Idealo for the given product name
func (e *IdealoExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
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
func (e *IdealoExtractor) buildSearchURL(productName string) (string, error) {
	baseURL := e.BaseURL()
	
	// Build query parameters
	params := url.Values{}
	params.Add("q", productName)
	
	// Construct full URL
	fullURL := fmt.Sprintf("%s/preisvergleich/MainSearchProductCategory.html?%s", baseURL, params.Encode())
	return fullURL, nil
}

// fetchHTML makes an HTTP GET request and returns the HTML content
func (e *IdealoExtractor) fetchHTML(url string) (string, error) {
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
func (e *IdealoExtractor) extractWithPython(htmlContent string) ([]models.ProductComparison, error) {
	// Get the absolute path to the Python script
	scriptPath, err := filepath.Abs("extractors/pythonExtractors/idealo_page.py")
	if err != nil {
		return nil, fmt.Errorf("failed to get script path: %w", err)
	}

	// Prepare the Python command
	cmd := exec.Command("python3", "-c", fmt.Sprintf(`
import sys
sys.path.append('%s')
from idealo_page import extract_idealo_products
import json

html_content = '''%s'''
products = extract_idealo_products(html_content)
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
		comparison := models.ProductComparison{
			Name:     getString(product["name"]),
			Price:    getString(product["price"]),
			Store:    getString(product["store"]),
			Currency: getString(product["currency"]),
			URL:      getString(product["url"]),
		}
		comparisons = append(comparisons, comparison)
	}

	return comparisons, nil
}