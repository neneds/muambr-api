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

	"github.com/google/uuid"
	"muambr-api/models"
	"muambr-api/utils"
)

// MercadoLivreExtractor implements the Extractor interface for Mercado Livre marketplace
type MercadoLivreExtractor struct {
	countryCode models.Country
}

// NewMercadoLivreExtractor creates a new Mercado Livre extractor for Brazil
func NewMercadoLivreExtractor() *MercadoLivreExtractor {
	return &MercadoLivreExtractor{
		countryCode: models.CountryBrazil, // Mercado Livre is Brazil-specific
	}
}

// GetCountryCode returns the ISO country code this extractor supports
func (e *MercadoLivreExtractor) GetCountryCode() models.Country {
	return e.countryCode
}

// GetMacroRegion returns the macro region this extractor supports
func (e *MercadoLivreExtractor) GetMacroRegion() models.MacroRegion {
	return e.countryCode.GetMacroRegion()
}

// BaseURL returns the base URL for the extractor's website
func (e *MercadoLivreExtractor) BaseURL() string {
	return "https://lista.mercadolivre.com.br"
}

// GetIdentifier returns a static string identifier for this extractor
func (e *MercadoLivreExtractor) GetIdentifier() string {
	return "mercadolivre"
}

// GetComparisons extracts product comparisons from Mercado Livre for the given product name
func (e *MercadoLivreExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
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

// buildSearchURL constructs the search URL with proper query parameters for Mercado Livre
func (e *MercadoLivreExtractor) buildSearchURL(productName string) (string, error) {
	baseURL := e.BaseURL()
	
	// Build the URL using Mercado Livre's format: /product-name
	// Replace spaces with hyphens and encode properly
	encodedProduct := url.PathEscape(strings.ReplaceAll(strings.ToLower(productName), " ", "-"))
	
	// Construct full URL using Mercado Livre search format
	fullURL := fmt.Sprintf("%s/%s", baseURL, encodedProduct)
	return fullURL, nil
}

// fetchHTML makes an HTTP GET request and returns the HTML content
func (e *MercadoLivreExtractor) fetchHTML(url string) (string, error) {
	utils.Info("Starting HTTP request to MercadoLivre", 
		utils.String("url", url),
		utils.String("extractor", "mercadolivre"),
		utils.String("base_domain", e.BaseURL()))

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		utils.LogError("Failed to create HTTP request for MercadoLivre", 
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.Error(err))
		return "", err
	}

	// Add headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.8,en-US;q=0.5,en;q=0.3")

	resp, err := client.Do(req)
	if err != nil {
		utils.LogError("HTTP request execution failed for MercadoLivre - possible anti-bot protection", 
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.String("base_domain", e.BaseURL()),
			utils.String("user_agent", req.Header.Get("User-Agent")),
			utils.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Warn("HTTP request returned non-200 status code from MercadoLivre - possible anti-bot protection",
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.String("base_domain", e.BaseURL()),
			utils.Int("status_code", resp.StatusCode),
			utils.String("status", resp.Status),
			utils.String("user_agent", req.Header.Get("User-Agent")))
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	utils.Info("Successfully received HTTP response from MercadoLivre",
		utils.String("url", url),
		utils.String("extractor", "mercadolivre"),
		utils.Int("status_code", resp.StatusCode),
		utils.Any("content_length", resp.ContentLength))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LogError("Failed to read response body from MercadoLivre", 
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.Error(err))
		return "", err
	}

	utils.Info("Successfully fetched HTML content from MercadoLivre",
		utils.String("url", url),
		utils.String("extractor", "mercadolivre"),
		utils.Int("content_size_bytes", len(body)))

	return string(body), nil
}

// extractWithPython calls the Python script to extract product data from HTML
func (e *MercadoLivreExtractor) extractWithPython(htmlContent string) ([]models.ProductComparison, error) {
	utils.Info("Starting Python extraction for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.Int("html_size_bytes", len(htmlContent)))

	// Get the absolute path to the Python script
	scriptPath, err := filepath.Abs("extractors/pythonExtractors/mercadolivre_page.py")
	if err != nil {
		utils.LogError("Failed to get Python script path for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
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

	utils.Info("Using Python interpreter for MercadoLivre extraction",
		utils.String("extractor", "mercadolivre"),
		utils.String("python_path", pythonPath),
		utils.String("script_path", scriptPath))

	// Create a temporary file to store the HTML content to avoid "argument list too long" error
	tempFile, err := os.CreateTemp("", "mercadolivre_html_*.html")
	if err != nil {
		utils.LogError("Failed to create temporary file for MercadoLivre HTML",
			utils.String("extractor", "mercadolivre"),
			utils.Error(err))
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file
	defer tempFile.Close()

	// Write HTML content to temporary file
	if _, err := tempFile.WriteString(htmlContent); err != nil {
		utils.LogError("Failed to write HTML to temporary file for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.String("temp_file", tempFile.Name()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to write HTML to temp file: %w", err)
	}
	tempFile.Close() // Close file so Python can read it

	utils.Info("Created temporary HTML file for MercadoLivre extraction",
		utils.String("extractor", "mercadolivre"),
		utils.String("temp_file", tempFile.Name()))

	// Get current working directory for local packages
	workDir, _ := os.Getwd()
	
	// Prepare the Python command using the temporary file
	cmd := exec.Command(pythonPath, "-c", fmt.Sprintf(`
import sys
sys.path.append('%s')
sys.path.append('%s')
try:
    from mercadolivre_page import extract_mercadolivre_products
    import json
    
    with open('%s', 'r', encoding='utf-8') as f:
        html_content = f.read()
    products = extract_mercadolivre_products(html_content)
    print(json.dumps(products))
except ImportError as e:
    print('{"error": "Missing Python dependencies", "details": "' + str(e) + '"}')
except Exception as e:
    print('{"error": "Python extraction failed", "details": "' + str(e) + '"}')
`, filepath.Dir(scriptPath), filepath.Join(workDir, "python_packages"), tempFile.Name()))

	// Execute the Python script
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	utils.Info("Executing Python extraction script for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.String("temp_file", tempFile.Name()))

	err = cmd.Run()
	if err != nil {
		utils.LogError("Python script execution failed for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.String("python_path", pythonPath),
			utils.String("stderr", stderr.String()),
			utils.String("stdout", out.String()),
			utils.Error(err))
		return nil, fmt.Errorf("Python script execution failed: %w, stderr: %s", err, stderr.String())
	}

	utils.Info("Python script executed successfully for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.String("python_stdout", out.String()),
		utils.String("python_stderr", stderr.String()))

	// Check if Python output contains error messages
	outputStr := out.String()
	if strings.Contains(outputStr, `"error":`) {
		var errorResp map[string]interface{}
		if json.Unmarshal(out.Bytes(), &errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				utils.Warn("Python script returned error for MercadoLivre",
					utils.String("extractor", "mercadolivre"),
					utils.String("error_type", errorMsg),
					utils.String("error_details", utils.GetString(errorResp["details"])))
				return []models.ProductComparison{}, nil // Return empty results instead of failing
			}
		}
	}

	// Parse the JSON output from Python
	var pythonProducts []map[string]interface{}
	err = json.Unmarshal(out.Bytes(), &pythonProducts)
	if err != nil {
		utils.LogError("Failed to parse Python JSON output for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.String("python_output", out.String()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to parse Python output: %w", err)
	}

	utils.Info("Successfully parsed Python output for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.Int("products_found", len(pythonProducts)))

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

		// Generate unique UUID
		id := uuid.New().String()
		
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
