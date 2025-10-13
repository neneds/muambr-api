package htmlparser

import (
	"net/url"
	"strings"
)

// CashConvertersPTParser handles Cash Converters Portugal parsing
type CashConvertersPTParser struct {
	ShareHTMLParser
}

func (p *CashConvertersPTParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try h1 tag first - it has the cleanest title
	pattern := `<h1[^>]*>([^<]*)</h1>`
	if h1Title := extractWithRegex(pattern, html); h1Title != "" {
		cleanedTitle := p.cleanCashConvertersTitle(h1Title)
		if cleanedTitle != "" {
			return cleanedTitle
		}
	}

	// Try og:title as fallback
	ogPattern := `<meta[^>]*property="og:title"[^>]*content="([^"]*)"[^>]*/?>`
	if ogTitle := extractWithRegex(ogPattern, html); ogTitle != "" {
		cleanedTitle := p.cleanCashConvertersTitle(ogTitle)
		if cleanedTitle != "" {
			return cleanedTitle
		}
	}

	// Try structured data as last resort
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" {
		cleanedTitle := p.cleanCashConvertersTitle(structuredTitle)
		if cleanedTitle != "" {
			return cleanedTitle
		}
	}

	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *CashConvertersPTParser) cleanCashConvertersTitle(title string) string {
	// Remove Cash Converters default text patterns
	cleanedTitle := title
	
	// Remove "ipad apple ipad" duplication - keep only the second part
	if strings.HasPrefix(strings.ToLower(cleanedTitle), "ipad apple ipad") {
		cleanedTitle = strings.TrimPrefix(cleanedTitle, "ipad apple ")
		cleanedTitle = strings.TrimPrefix(cleanedTitle, "Ipad apple ")
	}
	
	// Remove "na Cash Converters Portugal" and everything after it
	if idx := strings.Index(strings.ToLower(cleanedTitle), " na cash converters portugal"); idx != -1 {
		cleanedTitle = cleanedTitle[:idx]
	}
	
	// Remove "de segunda mão" if it appears at the end
	if strings.HasSuffix(strings.ToLower(cleanedTitle), " de segunda mão") {
		cleanedTitle = cleanedTitle[:len(cleanedTitle)-len(" de segunda mão")]
	}
	if strings.HasSuffix(strings.ToLower(cleanedTitle), " de segunda m&atilde;o") {
		cleanedTitle = cleanedTitle[:len(cleanedTitle)-len(" de segunda m&atilde;o")]
	}
	
	// Trim whitespace
	cleanedTitle = strings.TrimSpace(cleanedTitle)
	
	return cleanedTitle
}

func (p *CashConvertersPTParser) ExtractImage(html string, pageURL *url.URL) string {
	// Prioritize structured data
	if structuredImage := extractStructuredDataImage(html); structuredImage != "" {
		return structuredImage
	}
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *CashConvertersPTParser) ExtractPrice(html string, pageURL *url.URL) string {
	// Try structured data price first
	if structuredPrice := extractStructuredDataPrice(html); structuredPrice != "" {
		return structuredPrice
	}
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *CashConvertersPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

func (p *CashConvertersPTParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	// Custom parsing that uses our cleaned title without additional filtering
	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)

	// Don't apply filterTitle since we already cleaned the title in ExtractTitle
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