package linkparsers

import (
	"muambr-api/utils"
	"net/url"
	"regexp"
	"strings"
)

// WalmartParser handles Walmart-specific parsing
type WalmartParser struct {
	ShareHTMLParser
}

// ParseHTML overrides the base implementation to ensure Walmart-specific methods are called
func (p *WalmartParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	utils.Info("📄 WalmartParser: Starting parseHTMLContent")

	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)

	title := filterTitle(rawTitle)
	price := parsePrice(priceString)

	utils.Info("📄 WalmartParser: Extracted data",
		utils.String("title", title),
		utils.String("currency", currency),
		utils.String("imageURL", imageURL))
	
	if price != nil {
		utils.Info("📄 WalmartParser: Extracted price", utils.Float64("price", *price))
	}

	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}

func (p *WalmartParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Walmart-specific title patterns in order of preference
	patterns := []string{
		// Product title in h1 tag (most common on Walmart)
		`<h1[^>]*data-automation[^>]*>([^<]+)</h1>`,
		// Alternative h1 patterns
		`<h1[^>]*class="[^"]*heading[^"]*"[^>]*>([^<]+)</h1>`,
		`<h1[^>]*id="main-title"[^>]*>([^<]+)</h1>`,
		// Meta property patterns
		`<meta[^>]*property=["']og:title["'][^>]*content=["'](.*?)["'][^>]*>`,
		`<meta[^>]*name=["']twitter:title["'][^>]*content=["'](.*?)["'][^>]*>`,
		// Structured data patterns
		`"name":\s*"([^"]+)"`,
		// Breadcrumb patterns
		`<span[^>]*class="[^"]*breadcrumb[^"]*"[^>]*>([^<]+)</span>`,
		// Title tag as fallback
		`<title[^>]*>([^<]+)</title>`,
	}

	for _, pattern := range patterns {
		if title := extractWithRegex(pattern, html); title != "" {
			cleaned := strings.TrimSpace(title)
			// Filter out unwanted titles
			lower := strings.ToLower(cleaned)
			if cleaned != "" &&
				!strings.Contains(lower, "walmart.com") &&
				!strings.Contains(lower, "walmart") &&
				!strings.Contains(lower, "search results") &&
				!strings.Contains(lower, "your search") &&
				!strings.Contains(lower, "browse") &&
				len(cleaned) > 3 &&
				len(cleaned) < 200 {
				
				// Clean up common title suffixes
				cleaned = strings.TrimSuffix(cleaned, " - Walmart.com")
				cleaned = strings.TrimSuffix(cleaned, " | Walmart")
				cleaned = strings.TrimSpace(cleaned)
				
				return cleaned
			}
		}
	}

	// Try structured data
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" {
		return structuredTitle
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *WalmartParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Walmart-specific price patterns
	patterns := []string{
		// Current price patterns from actual Walmart HTML
		`Current price is\s*\$([0-9.]+)`,
		`current price[^$]*\$([0-9.]+)`,
		// Standard price span patterns
		`<span[^>]*class="[^"]*current[^"]*price[^"]*"[^>]*>\$([0-9.,]+)</span>`,
		`<span[^>]*data-automation-id="product-price"[^>]*>\$([0-9.,]+)</span>`,
		`<span[^>]*aria-label="current price[^"]*"\s*[^>]*>\$([0-9.,]+)</span>`,
		// Price in data attributes
		`data-price="([0-9.]+)"`,
		// JSON-LD structured data price
		`"price":\s*"?([0-9.]+)"?`,
		`"price":\s*([0-9.]+)`,
		// Generic price patterns with dollar sign
		`\$([0-9]+(?:\.[0-9]{2})?)`,
		// Meta property price
		`<meta[^>]*property=["']product:price:amount["'][^>]*content=["']([0-9.,]+)["'][^>]*>`,
		// Walmart specific price containers
		`<div[^>]*class="[^"]*price[^"]*"[^>]*>[^$]*\$([0-9.,]+)`,
		// Price in spans with specific classes
		`<span[^>]*class="[^"]*price-characteristic[^"]*"[^>]*>\$([0-9.,]+)</span>`,
		// One-time purchase price pattern
		`One-time purchase\s*\$([0-9.]+)`,
		// Now price pattern
		`Now\s*\$([0-9.]+)`,
	}

	for _, pattern := range patterns {
		if priceStr := extractWithRegex(pattern, html); priceStr != "" {
			// Clean up the price string
			priceStr = strings.ReplaceAll(priceStr, ",", "")
			priceStr = strings.TrimSpace(priceStr)
			
			// Validate it's a reasonable price
			if len(priceStr) > 0 && len(priceStr) <= 10 {
				// Check if it's a valid price format
				if matched, _ := regexp.MatchString(`^[0-9]+(\.[0-9]{1,2})?$`, priceStr); matched {
					utils.Info("🏷️ WalmartParser: Found price", utils.String("price", priceStr))
					return priceStr
				}
			}
		}
	}

	// Fallback to generic price extraction
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *WalmartParser) ExtractImage(html string, pageURL *url.URL) string {
	// Walmart-specific image patterns
	patterns := []string{
		// Main product image
		`<img[^>]*data-automation-id="product-image"[^>]*src=["'](https://[^"']+)["'][^>]*>`,
		`<img[^>]*class="[^"]*product[^"]*image[^"]*"[^>]*src=["'](https://[^"']+)["'][^>]*>`,
		// Hero image patterns
		`<img[^>]*class="[^"]*hero[^"]*"[^>]*src=["'](https://[^"']+)["'][^>]*>`,
		// Meta property images
		`<meta[^>]*property=["']og:image["'][^>]*content=["'](https://[^"']+)["'][^>]*>`,
		`<meta[^>]*name=["']twitter:image["'][^>]*content=["'](https://[^"']+)["'][^>]*>`,
		// Walmart images domain
		`src=["'](https://i5\.walmartimages\.com/[^"']+)["']`,
		// JSON-LD image
		`"image":\s*"(https://[^"]+)"`,
		// Any https image as fallback
		`<img[^>]*src=["'](https://[^"']+\.(?:jpg|jpeg|png|webp))["'][^>]*>`,
	}

	for _, pattern := range patterns {
		if imageURL := extractWithRegex(pattern, html); imageURL != "" {
			// Validate the URL
			if strings.HasPrefix(imageURL, "https://") && 
			   (strings.Contains(imageURL, "walmartimages.com") || 
			    strings.Contains(imageURL, ".jpg") || 
			    strings.Contains(imageURL, ".jpeg") || 
			    strings.Contains(imageURL, ".png") || 
			    strings.Contains(imageURL, ".webp")) {
				
				utils.Info("🖼️ WalmartParser: Found image", utils.String("imageURL", imageURL))
				return imageURL
			}
		}
	}

	// Fallback to generic image extraction
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *WalmartParser) ExtractDescription(html string, pageURL *url.URL) string {
	// Walmart-specific description patterns
	patterns := []string{
		// Product description sections
		`<div[^>]*class="[^"]*description[^"]*"[^>]*>([^<]+)</div>`,
		`<div[^>]*data-automation-id="product-description"[^>]*>([^<]+)</div>`,
		// Key features section
		`<div[^>]*class="[^"]*key[^"]*features[^"]*"[^>]*>([^<]+)</div>`,
		// Meta description
		`<meta[^>]*name=["']description["'][^>]*content=["'](.*?)["'][^>]*>`,
		`<meta[^>]*property=["']og:description["'][^>]*content=["'](.*?)["'][^>]*>`,
		// Structured data description
		`"description":\s*"([^"]+)"`,
	}

	for _, pattern := range patterns {
		if description := extractWithRegex(pattern, html); description != "" {
			cleaned := strings.TrimSpace(description)
			if len(cleaned) > 10 && len(cleaned) < 500 {
				utils.Info("📝 WalmartParser: Found description", utils.String("description", cleaned))
				return cleaned
			}
		}
	}

	// Fallback to generic description extraction
	return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}

func (p *WalmartParser) ExtractCurrency(html string, pageURL *url.URL) string {
	// Walmart is primarily US-based, so default to USD
	// Check for any currency indicators in the HTML
	htmlLower := strings.ToLower(html)
	
	if strings.Contains(htmlLower, "cad") || strings.Contains(htmlLower, "canadian") {
		return "CAD"
	}
	
	// Default to USD for Walmart
	return "USD"
}