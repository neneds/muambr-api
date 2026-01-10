package linkparsers

import (
	htmlesc "html"
	"net/url"
	"strconv"
	"strings"
)

// European retailers parsers

// FnacPTParser handles Fnac Portugal parsing
type FnacPTParser struct {
	ShareHTMLParser
}

func (p *FnacPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// PrimarkParser handles Primark parsing
type PrimarkParser struct {
	ShareHTMLParser
}

func (p *PrimarkParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// PrimorEUParser handles Primor EU parsing (Portugal and Spain)
type PrimorEUParser struct {
	ShareHTMLParser
}

// ParseHTML overrides the base method to handle Primor-specific price parsing
func (p *PrimorEUParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	// Extract all the components using our custom methods
	title := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)
	
	// Filter the title
	filteredTitle := filterTitle(title)
	
	// Parse the price manually for Primor
	var price *float64
	if priceString != "" {
		if val, err := strconv.ParseFloat(priceString, 64); err == nil {
			price = &val
		}
	}
	
	return &ParsedProductData{
		Title:       filteredTitle,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}

func (p *PrimorEUParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try og:title meta tag first (most reliable for Primor)
	if title := extractMetaProperty("og:title", html); title != "" {
		return title
	}

	// Try JSON-LD Product schema name (look specifically for Product schema)
	patterns := []string{
		`"@type":"Product"[^}]*"name":"([^"]+)"`,     // Product schema name
		`"name":"([^"]+)"[^}]*"@type":"Product"`,     // Reversed order
	}
	
	for _, pattern := range patterns {
		if title := extractWithRegex(pattern, html); title != "" {
			return title
		}
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *PrimorEUParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Try product:price:amount meta tag (most reliable for Primor)
	if price := extractMetaProperty("product:price:amount", html); price != "" {
		return price  // Return just the number, currency is handled separately
	}

	// Try structured data JSON-LD offers
	patterns := []string{
		`"price":([0-9]+(?:\.[0-9]+)?)`,     // "price":77.95
		`"price":"([^"]+)"`,                  // "price":"77.95"
	}

	for _, pattern := range patterns {
		if price := extractWithRegex(pattern, html); price != "" {
			return price  // Return just the number
		}
	}

	// Try data layer pricing (analytics data)
	if price := extractWithRegex(`"price":([0-9]+(?:\.[0-9]+)?)`, html); price != "" {
		return price  // Return just the number
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *PrimorEUParser) ExtractImage(html string, pageURL *url.URL) string {
	// Try og:image meta tag (most reliable for Primor)
	if image := extractMetaProperty("og:image", html); image != "" {
		return image
	}

	// Try structured data JSON-LD image
	if image := extractStructuredDataImage(html); image != "" {
		return image
	}

	// Try JSON-LD Product schema images array
	pattern := `"image":\s*\["([^"]+)"\]`
	if image := extractWithRegex(pattern, html); image != "" {
		return image
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *PrimorEUParser) ExtractDescription(html string, pageURL *url.URL) string {
	// Try og:description first
	if desc := extractMetaProperty("og:description", html); desc != "" {
		return desc
	}

	// Try structured data description
	pattern := `"description":"([^"]+)"`
	if desc := extractWithRegex(pattern, html); desc != "" {
		return desc
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}

func (p *PrimorEUParser) ExtractCurrency(html string, pageURL *url.URL) string {
	// Try product:price:currency meta tag
	if currency := extractMetaProperty("product:price:currency", html); currency != "" {
		return strings.ToLower(currency)
	}

	// Try structured data currency
	if strings.Contains(html, `"priceCurrency":"EUR"`) {
		return "eur"
	}

	// Default to EUR as Primor operates in Europe
	return "eur"
}

// WortenPTParser handles Worten Portugal parsing
type WortenPTParser struct {
	ShareHTMLParser
}

func (p *WortenPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// ZaraParser handles Zara parsing
type ZaraParser struct {
	ShareHTMLParser
}

func (p *ZaraParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// PerfumesECompanhiaParser handles Perfumes e Companhia parsing (Portugal and Spain)
type PerfumesECompanhiaParser struct {
	ShareHTMLParser
}



func (p *PerfumesECompanhiaParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try og:title meta tag first (most reliable and complete)
	if title := extractMetaProperty("og:title", html); title != "" {
		// Decode HTML entities
		title = htmlesc.UnescapeString(title)
		
		// Clean up the title by removing the site name suffix
		if idx := strings.Index(title, " | Perfumes e Companhia"); idx != -1 {
			return strings.TrimSpace(title[:idx])
		}
		if idx := strings.Index(title, " - BVLGARI | Perfumes e Companhia"); idx != -1 {
			return strings.TrimSpace(title[:idx])
		}
		return title
	}

	// Try JSON-LD Product schema name (more specific pattern)
	jsonLDPattern := `"@type":"Product"[^}]*?"name":"([^"]+)"`
	if title := extractWithRegex(jsonLDPattern, html); title != "" {
		return title
	}

	// Try alternative JSON-LD pattern
	if title := extractWithRegex(`"@context":"http://schema\.org/"[^}]*?"name":"([^"]+)"`, html); title != "" {
		return title
	}

	// Try data-gtm product name (URL decode it)
	if name := extractWithRegex(`"name":"([^"]+)"`, html); name != "" {
		// URL decode the name
		decoded := strings.ReplaceAll(name, "%20", " ")
		if len(decoded) > 5 { // Make sure it's not just "BVLGARI"
			return decoded
		}
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Try JSON-LD structured data price (most reliable)
	patterns := []string{
		`"offers":\{[^}]*"price":"([^"]+)"`,          // "offers":{"price":"83.02"
		`"price":"([^"]+)"[^}]*"priceCurrency":"EUR"`, // "price":"83.02","priceCurrency":"EUR"
	}

	for _, pattern := range patterns {
		if price := extractWithRegex(pattern, html); price != "" {
			return price
		}
	}

	// Try data-gtm price information
	if price := extractWithRegex(`"price":"([^"]+)"`, html); price != "" {
		return price
	}

	// Try generic price patterns in HTML
	pricePatterns := []string{
		`€\s*([0-9]+(?:\.[0-9]+)?)`,     // €83.02
		`([0-9]+(?:\.[0-9]+)?)\s*€`,     // 83.02€
		`([0-9]+(?:,[0-9]+)?)\s*€`,      // 83,02€
	}

	for _, pattern := range pricePatterns {
		if price := extractWithRegex(pattern, html); price != "" {
			// Convert comma to dot for EUR format
			return strings.ReplaceAll(price, ",", ".")
		}
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractImage(html string, pageURL *url.URL) string {
	// Try JSON-LD structured data images array (first image)
	if image := extractWithRegex(`"image":\s*\["([^"]+)"`, html); image != "" {
		return image
	}

	// Try og:image meta tag
	if image := extractMetaProperty("og:image", html); image != "" {
		return image
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractDescription(html string, pageURL *url.URL) string {
	// Try JSON-LD description
	if desc := extractWithRegex(`"description":"([^"]+)"`, html); desc != "" {
		return desc
	}

	// Try og:description meta tag
	if desc := extractMetaProperty("og:description", html); desc != "" {
		return desc
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractCurrency(html string, pageURL *url.URL) string {
	// Try JSON-LD priceCurrency
	if strings.Contains(html, `"priceCurrency":"EUR"`) {
		return "eur"
	}

	// Default to EUR as Perfumes e Companhia operates in Europe
	return "eur"
}

// ParseHTML overrides the base method to ensure our custom extraction methods are called
func (p *PerfumesECompanhiaParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	// Extract all the components using our custom methods
	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)
	
	// Apply filtering and parsing
	title := filterTitle(rawTitle)
	price := parsePrice(priceString)
	
	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}