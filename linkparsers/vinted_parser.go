package linkparsers

import (
	"net/url"
	"strings"
)

// VintedPTParser handles Vinted Portugal item pages.
// The PDP is Next.js but the product title, price, image, and description
// are in schema.org Product JSON-LD in the initial HTML.
type VintedPTParser struct {
	ShareHTMLParser
}

func (p *VintedPTParser) ExtractTitle(html string, pageURL *url.URL) string {
	if title := extractStructuredDataTitle(html); title != "" && !strings.Contains(strings.ToLower(title), "vinted") {
		return title
	}
	title := p.ShareHTMLParser.ExtractTitle(html, pageURL)
	if idx := strings.Index(title, " | Vinted"); idx != -1 {
		return strings.TrimSpace(title[:idx])
	}
	return title
}

func (p *VintedPTParser) ExtractPrice(html string, pageURL *url.URL) string {
	if price := productJSONLDPrice(html); price != "" {
		return price
	}
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *VintedPTParser) ExtractImage(html string, pageURL *url.URL) string {
	if image := extractStructuredDataImage(html); image != "" {
		return image
	}
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *VintedPTParser) ExtractDescription(html string, pageURL *url.URL) string {
	if desc := extractWithRegex(`"description":"([^"]+)"`, html); desc != "" {
		return desc
	}
	return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}

func (p *VintedPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

func (p *VintedPTParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	return &ParsedProductData{
		Title:       filterTitle(p.ExtractTitle(html, pageURL)),
		Price:       parsePrice(p.ExtractPrice(html, pageURL)),
		Currency:    p.ExtractCurrency(html, pageURL),
		ImageURL:    p.ExtractImage(html, pageURL),
		Description: p.ExtractDescription(html, pageURL),
	}
}
