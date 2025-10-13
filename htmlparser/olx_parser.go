package htmlparser

import (
	"net/url"
	"strings"
)

// OLXPTParser handles OLX Portugal parsing
type OLXPTParser struct {
	ShareHTMLParser
}

func (p *OLXPTParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try h4 with product title - very specific for OLX
	h4Pattern := `<h4[^>]*class="css-1au435n"[^>]*>([^<]+)</h4>`
	if h4Title := extractWithRegex(h4Pattern, html); h4Title != "" {
		return h4Title
	}
	
	// Try JSON-LD structured data for product name
	jsonLDPattern := `"@type":"Product","name":"([^"]+)"`
	if jsonTitle := extractWithRegex(jsonLDPattern, html); jsonTitle != "" {
		return jsonTitle
	}
	
	// Try page title but clean it
	titlePattern := `<title[^>]*>([^<]+)</title>`
	if pageTitle := extractWithRegex(titlePattern, html); pageTitle != "" {
		// Remove " • OLX.pt" and location info
		cleanTitle := pageTitle
		if idx := strings.Index(cleanTitle, " • OLX"); idx != -1 {
			cleanTitle = cleanTitle[:idx]
		}
		// Remove location at the end (like "Rio Tinto")
		parts := strings.Split(cleanTitle, " ")
		if len(parts) > 3 {
			// Keep first parts, likely the product name
			cleanTitle = strings.Join(parts[:len(parts)-1], " ")
		}
		return strings.TrimSpace(cleanTitle)
	}
	
	// Try h1 tag
	h1Pattern := `<h1[^>]*>([^<]+)</h1>`
	if h1Title := extractWithRegex(h1Pattern, html); h1Title != "" && !strings.Contains(h1Title, "OLX") {
		return h1Title
	}
	
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *OLXPTParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Try OLX Portugal specific price pattern: <h3 class="css-yauxmy">1.050 €</h3>
	pricePattern := `<h3[^>]*class="css-yauxmy"[^>]*>([^<]+)</h3>`
	if price := extractWithRegex(pricePattern, html); price != "" {
		return price
	}

	// Try structured data as fallback
	if structuredPrice := extractStructuredDataPrice(html); structuredPrice != "" {
		return structuredPrice
	}
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *OLXPTParser) ExtractImage(html string, pageURL *url.URL) string {
	if structuredImage := extractStructuredDataImage(html); structuredImage != "" {
		return structuredImage
	}
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *OLXPTParser) ExtractDescription(html string, pageURL *url.URL) string {
	// Try OLX Portugal specific description pattern: <div class="css-19duwlz">...content...</div>
	descPattern := `<div[^>]*class="css-19duwlz"[^>]*>(.*?)</div>`
	if desc := extractWithRegex(descPattern, html); desc != "" {
		return desc
	}

	// Try generic description patterns as fallback
	return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}

func (p *OLXPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

func (p *OLXPTParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	// Custom parsing that uses our specific OLX extraction methods
	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)

	// Use our extracted title without additional filtering
	title := rawTitle
	price := parsePrice(priceString)

	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}

// OLXBRParser handles OLX Brazil parsing
type OLXBRParser struct {
	ShareHTMLParser
}

func (p *OLXBRParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try h1 tag first - most specific
	h1Pattern := `<h1[^>]*>([^<]+)</h1>`
	if h1Title := extractWithRegex(h1Pattern, html); h1Title != "" && !strings.Contains(h1Title, "OLX") {
		return h1Title
	}
	
	// Try og:title meta tag
	ogPattern := `<meta[^>]*property="og:title"[^>]*content="([^"]*)"[^>]*/?>`
	if ogTitle := extractWithRegex(ogPattern, html); ogTitle != "" && !strings.Contains(ogTitle, "OLX") {
		return ogTitle
	}
	
	// Try structured data as fallback (but filter out site names)
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" && !strings.Contains(structuredTitle, "OLX") {
		return structuredTitle
	}
	
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *OLXBRParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Try OLX Brazil specific price pattern: class="olx-text olx-text--title-large olx-text--block">R$ 9.699</span>
	pricePattern := `class="olx-text olx-text--title-large olx-text--block"[^>]*>([^<]+)</span>`
	if price := extractWithRegex(pricePattern, html); price != "" {
		return price
	}

	// Try structured data as fallback
	if structuredPrice := extractStructuredDataPrice(html); structuredPrice != "" {
		return structuredPrice
	}
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *OLXBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

func (p *OLXBRParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	// Custom parsing that uses our specific OLX extraction methods
	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)

	// Use our extracted title without additional filtering
	title := rawTitle
	price := parsePrice(priceString)

	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}