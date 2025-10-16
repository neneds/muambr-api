package htmlparser

import (
	"fmt"
	"io"
	"muambr-api/utils"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchHTML fetches HTML content from a URL using enhanced anti-bot measures
func FetchHTML(urlStr string) (string, error) {
	utils.Info("🌐 Fetching HTML from URL", utils.String("url", urlStr))

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Create anti-bot config for the site with enhanced settings for Amazon
	config := utils.DefaultAntiBotConfig(urlStr)
	
	// Enhanced settings for Amazon and similar e-commerce sites
	if isAmazonURL(parsedURL) {
		config.MinDelay = 1000 * time.Millisecond  // Longer delays for Amazon
		config.MaxDelay = 5000 * time.Millisecond
		config.UseReferer = true
		config.RefererURL = "https://www.google.com/"
	}

	var resp *http.Response
	
	// Retry logic for anti-bot protection
	maxRetries := 3 // Increased retries for better success rate
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			utils.Info("🔄 Retrying request with different strategy", 
				utils.Int("attempt", attempt+1),
				utils.String("url", parsedURL.Host))
			// Progressive delay increase for retries
			config.MinDelay = time.Duration(attempt+1) * 2000 * time.Millisecond
			config.MaxDelay = time.Duration(attempt+1) * 6000 * time.Millisecond
			
			// Rotate user agents on retries
			config.UserAgentRotation = true
		}
		
		// Use enhanced scraping request for Amazon and similar sites
		if isAmazonURL(parsedURL) {
			resp, err = utils.MakeScrapingRequest(urlStr, config)
		} else {
			resp, err = utils.MakeAntiBotRequest(urlStr, config)
		}
		if err != nil {
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to fetch URL after %d attempts: %w", maxRetries+1, err)
			}
			utils.Warn("🔄 Request failed, will retry", utils.Error(err), utils.Int("attempt", attempt+1))
			continue
		}
		
		utils.Info("📊 Response received", 
			utils.Int("statusCode", resp.StatusCode),
			utils.String("status", resp.Status),
			utils.Int("attempt", attempt+1))
		
		// Check response status
		if resp.StatusCode == 200 {
			break // Success!
		} else if resp.StatusCode == 403 || resp.StatusCode == 429 || resp.StatusCode == 503 {
			resp.Body.Close()
			if attempt == maxRetries {
				return "", fmt.Errorf("access denied after %d attempts, status code: %d", maxRetries+1, resp.StatusCode)
			}
			utils.Warn("🚫 Access denied, retrying with different approach", 
				utils.Int("statusCode", resp.StatusCode),
				utils.Int("attempt", attempt+1))
			continue
		} else {
			resp.Body.Close()
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(body)
	utils.Info("✅ Successfully fetched HTML", 
		utils.Int("bytes", len(html)),
		utils.String("host", parsedURL.Host))

	return html, nil
}

// isAmazonURL checks if the URL is an Amazon domain
func isAmazonURL(parsedURL *url.URL) bool {
	host := strings.ToLower(parsedURL.Host)
	return strings.Contains(host, "amazon.") || strings.Contains(host, "amzn.")
}

// ParseURL fetches and parses HTML from a URL
func ParseURL(urlStr string) (*ParsedProductData, error) {
	// Parse URL first
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Fetch HTML
	html, err := FetchHTML(urlStr)
	if err != nil {
		return nil, err
	}

	// Parse HTML
	data := ParseHTML(html, parsedURL)
	return data, nil
}

// ScrapeProductTitles scrapes product titles from a search URL (inspired by amazon_scraper.go)
func ScrapeProductTitles(searchURL string, maxResults int) ([]string, error) {
	utils.Info("🔍 Scraping product titles", utils.String("url", searchURL))
	
	parsedURL, err := url.Parse(searchURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Create enhanced config for scraping
	config := utils.DefaultAntiBotConfig(searchURL)
	config.MinDelay = 1000 * time.Millisecond
	config.MaxDelay = 3000 * time.Millisecond
	config.UserAgentRotation = true

	var resp *http.Response
	
	// Make request with enhanced scraping protection
	if isAmazonURL(parsedURL) {
		resp, err = utils.MakeScrapingRequest(searchURL, config)
	} else {
		resp, err = utils.MakeAntiBotRequest(searchURL, config)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	html := string(body)
	
	// Extract titles using patterns optimized for product search results
	var titlePatterns []string
	if isAmazonURL(parsedURL) {
		// Amazon-specific patterns from amazon_scraper.go
		titlePatterns = []string{
			`div\[data-cy=['"]title-recipe['"]\]\s*a\.a-link-normal[^>]*>([^<]+)</a>`,
			`div\.puisg-col-inner\s*a\.a-link-normal[^>]*>([^<]+)</a>`,
			`<h2[^>]*class="[^"]*s-size-mini[^"]*"[^>]*>.*?<a[^>]*>.*?<span[^>]*>([^<]+)</span>`,
		}
	} else {
		// Generic e-commerce patterns
		titlePatterns = []string{
			`<h[1-6][^>]*class="[^"]*product[^"]*title[^"]*"[^>]*>([^<]+)</h[1-6]>`,
			`<a[^>]*class="[^"]*product[^"]*link[^"]*"[^>]*>([^<]+)</a>`,
			`<span[^>]*class="[^"]*product[^"]*name[^"]*"[^>]*>([^<]+)</span>`,
		}
	}

	allTitles := extractMultipleWithRegex(titlePatterns, html)
	
	// Filter and limit results
	filteredTitles := filterProductTitles(allTitles)
	
	if maxResults > 0 && len(filteredTitles) > maxResults {
		filteredTitles = filteredTitles[:maxResults]
	}

	utils.Info("🔍 Scraped product titles", 
		utils.Int("total", len(allTitles)),
		utils.Int("filtered", len(filteredTitles)))

	return filteredTitles, nil
}
