package extractors

import (
	"bytes"
	"compress/gzip"
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

// KuantoKustaExtractor implements the Extractor interface for KuantoKusta price comparison site
type KuantoKustaExtractor struct {
	countryCode models.Country
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

	// Create anti-bot configuration for KuantoKusta
	config := utils.DefaultAntiBotConfig(e.BaseURL())
	
	// Make request using anti-bot utility
	resp, err := utils.MakeAntiBotRequest(url, config)
	if err != nil {
		utils.LogError("HTTP request execution failed for KuantoKusta - possible anti-bot protection", 
			utils.String("url", url),
			utils.String("extractor", "kuantokusta"),
			utils.String("base_domain", e.BaseURL()),
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
			utils.String("status", resp.Status))
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	utils.Info("Successfully received HTTP response from KuantoKusta",
		utils.String("url", url),
		utils.String("extractor", "kuantokusta"),
		utils.Int("status_code", resp.StatusCode),
		utils.Any("content_length", resp.ContentLength))

	// Handle gzip decompression if needed
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			utils.LogError("Failed to create gzip reader for KuantoKusta response", 
				utils.String("url", url),
				utils.String("extractor", "kuantokusta"),
				utils.Error(err))
			return "", err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	body, err := io.ReadAll(reader)
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
	utils.Info("Starting Python extraction for KuantoKusta",
		utils.String("extractor", "kuantokusta"),
		utils.Int("html_size_bytes", len(htmlContent)))

	// Get the absolute path to the Python script
	workingDir, _ := os.Getwd()
	projectRoot := filepath.Dir(workingDir)
	if filepath.Base(workingDir) != "muambr-goapi" {
		// If we're not in the project root, find it
		for filepath.Base(projectRoot) != "muambr-goapi" && projectRoot != "/" {
			projectRoot = filepath.Dir(projectRoot)
		}
		if projectRoot == "/" {
			// Fallback: assume we're in a subdirectory of the project
			projectRoot = workingDir
			for !fileExists(filepath.Join(projectRoot, "go.mod")) && projectRoot != "/" {
				projectRoot = filepath.Dir(projectRoot)
			}
		}
	} else {
		projectRoot = workingDir
	}
	scriptPath := filepath.Join(projectRoot, "extractors", "pythonExtractors", "kuantokusta_page.py")
	if !fileExists(scriptPath) {
		utils.LogError("Python script not found for KuantoKusta",
			utils.String("extractor", "kuantokusta"),
			utils.String("script_path", scriptPath),
			utils.String("project_root", projectRoot))
		return nil, fmt.Errorf("Python script not found at %s", scriptPath)
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

	utils.Info("Using Python interpreter for KuantoKusta extraction",
		utils.String("extractor", "kuantokusta"),
		utils.String("python_path", pythonPath),
		utils.String("script_path", scriptPath))

	// Create a temporary file to store the HTML content to avoid "argument list too long" error
	tempFile, err := os.CreateTemp("", "kuantokusta_html_*.html")
	if err != nil {
		utils.LogError("Failed to create temporary file for KuantoKusta HTML",
			utils.String("extractor", "kuantokusta"),
			utils.Error(err))
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file
	defer tempFile.Close()

	// Write HTML content to temporary file
	if _, err := tempFile.WriteString(htmlContent); err != nil {
		utils.LogError("Failed to write HTML to temporary file for KuantoKusta",
			utils.String("extractor", "kuantokusta"),
			utils.String("temp_file", tempFile.Name()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to write HTML to temp file: %w", err)
	}
	tempFile.Close() // Close file so Python can read it

	utils.Info("Created temporary HTML file for KuantoKusta extraction",
		utils.String("extractor", "kuantokusta"),
		utils.String("temp_file", tempFile.Name()))

	// Get current working directory for local packages
	workDir, _ := os.Getwd()
	
	// Prepare the Python command using the temporary file
	cmd := exec.Command(pythonPath, "-c", fmt.Sprintf(`
import sys
sys.path.append('%s')
sys.path.append('%s')
try:
    from kuantokusta_page import extract_kuantokusta_products
    import json
    
    with open('%s', 'r', encoding='utf-8') as f:
        html_content = f.read()
    products = extract_kuantokusta_products(html_content)
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

	utils.Info("Executing Python extraction script for KuantoKusta",
		utils.String("extractor", "kuantokusta"),
		utils.String("temp_file", tempFile.Name()))

	err = cmd.Run()
	if err != nil {
		utils.LogError("Python script execution failed for KuantoKusta",
			utils.String("extractor", "kuantokusta"),
			utils.String("python_path", pythonPath),
			utils.String("stderr", stderr.String()),
			utils.String("stdout", out.String()),
			utils.Error(err))
		return nil, fmt.Errorf("Python script execution failed: %w, stderr: %s", err, stderr.String())
	}

	utils.Info("Python script executed successfully for KuantoKusta",
		utils.String("extractor", "kuantokusta"),
		utils.String("python_stdout", out.String()),
		utils.String("python_stderr", stderr.String()))

	// Check if Python output contains error messages
	outputStr := out.String()
	if strings.Contains(outputStr, `"error":`) {
		var errorResp map[string]interface{}
		if json.Unmarshal(out.Bytes(), &errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				utils.Warn("Python script returned error for KuantoKusta",
					utils.String("extractor", "kuantokusta"),
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
		utils.LogError("Failed to parse Python JSON output for KuantoKusta",
			utils.String("extractor", "kuantokusta"),
			utils.String("python_output", out.String()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to parse Python output: %w", err)
	}

	utils.Info("Successfully parsed Python output for KuantoKusta",
		utils.String("extractor", "kuantokusta"),
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