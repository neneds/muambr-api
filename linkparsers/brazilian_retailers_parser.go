package linkparsers

import (
	"net/url"
)

// ElectroluxBRParser handles Electrolux Brazil (loja.electrolux.com.br VTEX).
type ElectroluxBRParser struct {
	ShareHTMLParser
}

func (p *ElectroluxBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

func (p *ElectroluxBRParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Product JSON-LD is in SSR HTML. Generic R$ scraping hits filter chips first.
	if price := productJSONLDPrice(html); price != "" {
		return price
	}
	if structured := extractStructuredDataPrice(html); structured != "" {
		return structured
	}
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *ElectroluxBRParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	title := p.ExtractTitle(html, pageURL)
	price := parsePrice(p.ExtractPrice(html, pageURL))
	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    p.ExtractCurrency(html, pageURL),
		ImageURL:    p.ExtractImage(html, pageURL),
		Description: p.ExtractDescription(html, pageURL),
	}
}
