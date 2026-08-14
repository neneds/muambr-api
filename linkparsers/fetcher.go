package linkparsers

import (
	"fmt"
	"muambr-api/utils"
	"net/url"
)

// FetchHTML fetches HTML content from a URL.
func FetchHTML(urlStr string) (string, error) {
	utils.Info("Fetching HTML from URL", utils.String("url", urlStr))

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := utils.MakeAntiBotRequest(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	utils.Info("Response received",
		utils.Int("statusCode", resp.StatusCode),
		utils.String("status", resp.Status))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := utils.ReadDecompressedBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(body)
	utils.Info("Successfully fetched HTML",
		utils.Int("bytes", len(html)),
		utils.String("host", parsedURL.Host))

	return html, nil
}

// ParseURL fetches and parses HTML from a URL
func ParseURL(urlStr string) (*ParsedProductData, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	html, err := FetchHTML(urlStr)
	if err != nil {
		return nil, err
	}

	data := ParseHTML(html, parsedURL)
	return data, nil
}
