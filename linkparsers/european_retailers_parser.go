package linkparsers

import (
	htmlesc "html"
	"net/url"
	"strings"
)

// PerfumesECompanhiaParser handles Perfumes e Companhia PDPs (Demandware JSON-LD).
type PerfumesECompanhiaParser struct {
	ShareHTMLParser
}

func (p *PerfumesECompanhiaParser) ExtractTitle(html string, pageURL *url.URL) string {
	if title := extractMetaProperty("og:title", html); title != "" {
		title = htmlesc.UnescapeString(title)
		if idx := strings.Index(title, " | Perfumes e Companhia"); idx != -1 {
			return strings.TrimSpace(title[:idx])
		}
		return title
	}
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractPrice(html string, pageURL *url.URL) string {
	if price := productJSONLDPrice(html); price != "" {
		return price
	}
	if structured := extractStructuredDataPrice(html); structured != "" {
		return structured
	}
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractImage(html string, pageURL *url.URL) string {
	if image := extractWithRegex(`"image":\s*\["([^"]+)"`, html); image != "" {
		return image
	}
	if image := extractMetaProperty("og:image", html); image != "" {
		return image
	}
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *PerfumesECompanhiaParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

func (p *PerfumesECompanhiaParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	return &ParsedProductData{
		Title:       filterTitle(p.ExtractTitle(html, pageURL)),
		Price:       parsePrice(p.ExtractPrice(html, pageURL)),
		Currency:    p.ExtractCurrency(html, pageURL),
		ImageURL:    p.ExtractImage(html, pageURL),
		Description: p.ExtractDescription(html, pageURL),
	}
}
