package htmlparser

import (
	"fmt"
	"io"
	"muambr-api/utils"
	"net/url"
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

	// Make request with anti-bot protection
	resp, err := utils.MakeAntiBotRequest(urlStr, config)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

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
