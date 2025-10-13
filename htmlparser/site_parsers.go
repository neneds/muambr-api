package htmlparser

import (
	"muambr-api/utils"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// AmazonParser handles Amazon-specific parsing
type AmazonParser struct {
	ShareHTMLParser
}

func (p *AmazonParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try meta name="title" first for Amazon
	pattern := `<meta[^>]*name=["']title["'][^>]*content=["'](.*?)["'][^>]*>`
	if metaTitle := extractWithRegex(pattern, html); metaTitle != "" {
		return metaTitle
	}

	// Try structured data
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" {
		return structuredTitle
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *AmazonParser) ExtractPrice(html string, pageURL *url.URL) string {
	host := strings.ToLower(pageURL.Host)
	isBrazil := strings.Contains(host, ".br") || strings.Contains(host, "amazon.com.br")

	// Amazon uses specific price patterns in a-offscreen spans
	var patterns []string
	if isBrazil {
		// Brazilian-specific patterns (R$ with comma as decimal separator)
		patterns = []string{
			`<span class="a-offscreen">R\$([0-9.,]+)</span>`,  // R$612,25
			`<span class="a-offscreen">R\$ ([0-9.,]+)</span>`, // R$ 612,25
			`<span class="aok-offscreen">\s*R\$\s*([0-9.,]+)\s*</span>`,
		}
	} else {
		// European patterns (€ with comma/dot variations)
		patterns = []string{
			`<span class="a-offscreen">([0-9.,]+)€</span>`,     // 470,00€
			`<span class="a-offscreen">([0-9.,]+) €</span>`,    // 470,00 €  
			`<span class="aok-offscreen">\s*([0-9.,]+)\s*€\s*</span>`,
		}
	}

	for _, pattern := range patterns {
		if priceMatch := extractWithRegex(pattern, html); priceMatch != "" {
			cleanedPrice := strings.TrimSpace(priceMatch)
			
			if isBrazil {
				// For Brazilian prices, add R$ prefix and convert comma decimal separator to dot
				if strings.Contains(cleanedPrice, ",") {
					if lastCommaIdx := strings.LastIndex(cleanedPrice, ","); lastCommaIdx != -1 {
						afterComma := cleanedPrice[lastCommaIdx+1:]
						// Check if this looks like a decimal (2 digits after comma)
						if len(afterComma) == 2 {
							return "R$" + cleanedPrice[:lastCommaIdx] + "." + afterComma
						}
					}
				}
				return "R$" + cleanedPrice
			} else {
				// For European prices, add € suffix and keep comma as decimal separator
				return cleanedPrice + "€"
			}
		}
	}

	// Fallback to generic
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *AmazonParser) ExtractImage(html string, pageURL *url.URL) string {
	// Amazon uses data-old-hires for high-resolution images
	patterns := []string{
		`data-old-hires="([^"]+)"`,                             // Primary pattern for high-res images
		`'colorImages':\s*\{\s*'initial':\s*\[{"hiRes":"([^"]+)"`, // JavaScript color images data
		`id="landingImage"[^>]*src="([^"]+)"`,                  // Landing image src attribute
	}

	for _, pattern := range patterns {
		if imageURL := extractWithRegex(pattern, html); imageURL != "" {
			// Ensure URL is properly formatted
			if strings.HasPrefix(imageURL, "http") {
				return imageURL
			}
		}
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractImage(html, pageURL)
}

func (p *AmazonParser) ExtractDescription(html string, pageURL *url.URL) string {
	// Amazon product descriptions are in feature-bullets section after "a-size-base-plus a-text-bold"
	patterns := []string{
		// Extract bullet points from feature-bullets section
		`<h1 class="a-size-base-plus a-text-bold">[^<]*</h1>\s*<ul[^>]*>([\s\S]*?)</ul>`,
		// Fallback patterns
		`<div id="productDescription"[^>]*>[\s\S]*?<p>[\s]*<span>([^<]+)</span>`,
		`<div id="productDescription"[^>]*>[\s\S]*?<span>([^<]+)</span>`,
		`productDescription_feature_div[\s\S]*?<p>[\s\S]*?<span>([\s\S]*?)</span>`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile("(?i)" + pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			rawDescription := matches[1]
			
			// For the first pattern (bullet points), extract list items
			if strings.Contains(pattern, "feature-bullets") || strings.Contains(rawDescription, "a-list-item") {
				listItemPattern := `<span class="a-list-item">\s*([^<]+)\s*</span>`
				listRe := regexp.MustCompile(listItemPattern)
				listMatches := listRe.FindAllStringSubmatch(rawDescription, -1)
				
				var descriptions []string
				for _, listMatch := range listMatches {
					if len(listMatch) > 1 {
						item := strings.TrimSpace(listMatch[1])
						if len(item) > 10 { // Skip very short items
							descriptions = append(descriptions, item)
						}
					}
				}
				
				if len(descriptions) > 0 {
					description := strings.Join(descriptions, ". ")
					if len(description) > 300 {
						description = description[:300] + "..."
					}
					return description
				}
			}
			
			// For other patterns, clean up the description
			description := strings.TrimSpace(rawDescription)
			// Remove HTML tags
			description = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(description, "")
			// Clean up whitespace
			description = regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")
			
			if len(description) > 300 {
				description = description[:300] + "..."
			}
			
			if len(description) > 10 { // Only return if meaningful content
				return description
			}
		}
	}

	// Fallback to generic extraction
	return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}

func (p *AmazonParser) ExtractCurrency(html string, pageURL *url.URL) string {
	host := strings.ToLower(pageURL.Host)
	if strings.Contains(host, ".br") || strings.Contains(host, "amazon.com.br") {
		return "brl"
	}
	return "eur" // Amazon Spain and other European Amazon sites use EUR
}

func (p *AmazonParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	utils.Info("📄 AmazonParser: Starting parseHTMLContent")

	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)

	// Use the same logic as ShareHTMLParser but with our extracted data
	title := filterTitle(rawTitle)
	price := p.parseAmazonPrice(priceString, currency)

	utils.Info("📄 AmazonParser: Extracted data",
		utils.String("title", title),
		utils.String("currency", currency),
		utils.String("imageURL", imageURL))
	
	if price != nil {
		utils.Info("📄 AmazonParser: Extracted price", utils.Float64("price", *price))
	}

	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}

func (p *AmazonParser) parseAmazonPrice(priceString string, currency string) *float64 {
	utils.Debug("🔄 AmazonParser: Starting parsePrice", utils.String("input", priceString), utils.String("currency", currency))
	
	if priceString == "" {
		utils.Debug("❌ AmazonParser: Price string is empty")
		return nil
	}

	// Remove currency symbols
	cleanedString := priceString
	cleanedString = strings.ReplaceAll(cleanedString, "$", "")
	cleanedString = strings.ReplaceAll(cleanedString, "€", "")
	cleanedString = strings.ReplaceAll(cleanedString, "£", "")
	cleanedString = strings.ReplaceAll(cleanedString, "R$", "")
	cleanedString = strings.ReplaceAll(cleanedString, "R", "")
	cleanedString = strings.ReplaceAll(cleanedString, "USD", "")
	cleanedString = strings.ReplaceAll(cleanedString, "EUR", "")
	cleanedString = strings.ReplaceAll(cleanedString, "GBP", "")
	cleanedString = strings.ReplaceAll(cleanedString, "BRL", "")
	cleanedString = strings.TrimSpace(cleanedString)

	utils.Debug("🔄 AmazonParser: Cleaned price string", utils.String("cleaned", cleanedString))

	// Handle different decimal separators based on currency
	if currency == "eur" {
		// European format: use comma as decimal separator
		// Convert comma decimal separator to dot for parsing
		if strings.Contains(cleanedString, ",") {
			// Find the last comma (should be decimal separator)
			lastCommaIdx := strings.LastIndex(cleanedString, ",")
			if lastCommaIdx != -1 {
				afterComma := cleanedString[lastCommaIdx+1:]
				// If it's 2 digits after comma, treat as decimal
				if len(afterComma) == 2 {
					cleanedString = cleanedString[:lastCommaIdx] + "." + afterComma
				} else {
					// Remove comma (thousands separator)
					cleanedString = strings.ReplaceAll(cleanedString, ",", "")
				}
			}
		}
	} else {
		// Brazilian/US format: use dot as decimal separator, comma as thousands separator
		// Remove commas (thousands separators) except if it's the last one with 2 digits after
		if strings.Contains(cleanedString, ",") {
			lastCommaIdx := strings.LastIndex(cleanedString, ",")
			if lastCommaIdx != -1 {
				afterComma := cleanedString[lastCommaIdx+1:]
				// If NOT 2 digits after last comma, remove all commas
				if len(afterComma) != 2 {
					cleanedString = strings.ReplaceAll(cleanedString, ",", "")
				}
			}
		}
	}

	if val, err := strconv.ParseFloat(cleanedString, 64); err == nil {
		utils.Debug("🔄 AmazonParser: Parsed price", utils.Float64("value", val))
		return &val
	}

	utils.Debug("❌ AmazonParser: Failed to parse price")
	return nil
}

// OLXPTParser handles OLX Portugal parsing
type OLXPTParser struct {
	ShareHTMLParser
}

func (p *OLXPTParser) ExtractTitle(html string, pageURL *url.URL) string {
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" {
		return structuredTitle
	}
	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *OLXPTParser) ExtractPrice(html string, pageURL *url.URL) string {
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

func (p *OLXPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// OLXBRParser handles OLX Brazil parsing
type OLXBRParser struct {
	ShareHTMLParser
}

func (p *OLXBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

// FnacPTParser handles Fnac Portugal parsing
type FnacPTParser struct {
	ShareHTMLParser
}

func (p *FnacPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// CashConvertersPTParser handles Cash Converters Portugal parsing
type CashConvertersPTParser struct {
	ShareHTMLParser
}

func (p *CashConvertersPTParser) ExtractTitle(html string, pageURL *url.URL) string {
	// Try structured data first
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" {
		return structuredTitle
	}

	// Try h1 tag
	pattern := `<h1[^>]*>([^<]*)</h1>`
	if h1Title := extractWithRegex(pattern, html); h1Title != "" {
		return h1Title
	}

	return p.ShareHTMLParser.ExtractTitle(html, pageURL)
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

// MagazineLuizaBRParser handles Magazine Luiza Brazil parsing
type MagazineLuizaBRParser struct {
	ShareHTMLParser
}

func (p *MagazineLuizaBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

// MercadoLivreBRParser handles Mercado Livre Brazil parsing
type MercadoLivreBRParser struct {
	ShareHTMLParser
}

func (p *MercadoLivreBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

// ElectroluxBRParser handles Electrolux Brazil parsing
type ElectroluxBRParser struct {
	ShareHTMLParser
}

func (p *ElectroluxBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

// PrimarkParser handles Primark parsing
type PrimarkParser struct {
	ShareHTMLParser
}

func (p *PrimarkParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// PrimorEUParser handles Primor EU parsing
type PrimorEUParser struct {
	ShareHTMLParser
}

func (p *PrimorEUParser) ExtractCurrency(html string, pageURL *url.URL) string {
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

// Parser registry
var siteParserRegistry = map[string]func() Parser{
	"amazon.es":             func() Parser { return &AmazonParser{} },
	"amazon.co.uk":          func() Parser { return &AmazonParser{} },
	"amazon.com":            func() Parser { return &AmazonParser{} },
	"amazon.de":             func() Parser { return &AmazonParser{} },
	"amazon.fr":             func() Parser { return &AmazonParser{} },
	"amazon.com.br":         func() Parser { return &AmazonParser{} },
	"a.co":                  func() Parser { return &AmazonParser{} },
	"cashconverters.pt":     func() Parser { return &CashConvertersPTParser{} },
	"fnac.pt":               func() Parser { return &FnacPTParser{} },
	"olx.pt":                func() Parser { return &OLXPTParser{} },
	"olx.br":                func() Parser { return &OLXBRParser{} },
	"magazineluiza.com.br":  func() Parser { return &MagazineLuizaBRParser{} },
	"mercadolivre.com.br":   func() Parser { return &MercadoLivreBRParser{} },
	"electrolux.com.br":     func() Parser { return &ElectroluxBRParser{} },
	"primark.com":           func() Parser { return &PrimarkParser{} },
	"primor.eu":             func() Parser { return &PrimorEUParser{} },
	"worten.pt":             func() Parser { return &WortenPTParser{} },
	"zara.com":              func() Parser { return &ZaraParser{} },
}

// createParser creates the appropriate parser for the URL
func createParser(pageURL *url.URL) Parser {
	host := strings.ToLower(pageURL.Host)

	// Remove www. prefix for matching
	host = strings.TrimPrefix(host, "www.")

	// Try exact match first
	if parserFactory, ok := siteParserRegistry[host]; ok {
		utils.Info("📍 Found exact match parser", utils.String("host", host))
		return parserFactory()
	}

	// Try partial matches
	for configHost, parserFactory := range siteParserRegistry {
		if strings.Contains(host, configHost) || strings.Contains(configHost, host) {
			utils.Info("📍 Found matching parser", 
				utils.String("configHost", configHost),
				utils.String("host", host))
			return parserFactory()
		}
	}

	// Fallback to generic parser
	utils.Warn("⚠️ No specific parser found, using generic parser", utils.String("host", host))
	return &ShareHTMLParser{}
}
