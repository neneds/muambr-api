package htmlparser

import (
	"fmt"
	"io"
	"muambr-api/utils"
	"net/http"
	"net/url"
	"time"
)

// FetchHTML fetches HTML content from a URL using anti-bot measures
func FetchHTML(urlStr string) (string, error) {
	utils.Info("🌐 Fetching HTML from URL", utils.String("url", urlStr))

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Create anti-bot config for the site
	config := utils.DefaultAntiBotConfig(urlStr)

	var resp *http.Response
	var err error
	
	// Retry logic for anti-bot protection
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			utils.Info("🔄 Retrying request with different strategy", 
				utils.Int("attempt", attempt+1),
				utils.String("url", parsedURL.Host))
			// Increase delay for retries
			config.MinDelay = time.Duration(attempt) * 2000 * time.Millisecond
			config.MaxDelay = time.Duration(attempt) * 4000 * time.Millisecond
		}
		
		// Make request with anti-bot protection
		resp, err = utils.MakeAntiBotRequest(urlStr, config)
		if err != nil {
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to fetch URL after %d attempts: %w", maxRetries+1, err)
			}
			continue
		}
		
		// Check response status
		if resp.StatusCode == 200 {
			break // Success!
		} else if resp.StatusCode == 403 || resp.StatusCode == 429 {
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
