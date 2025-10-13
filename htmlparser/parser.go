package htmlparser

import (
	"fmt"
	"muambr-api/utils"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Parser is the interface that all parsers must implement
type Parser interface {
	ParseHTML(html string, pageURL *url.URL) *ParsedProductData
	ExtractTitle(html string, pageURL *url.URL) string
	ExtractPrice(html string, pageURL *url.URL) string
	ExtractImage(html string, pageURL *url.URL) string
	ExtractDescription(html string, pageURL *url.URL) string
	ExtractCurrency(html string, pageURL *url.URL) string
}

// ShareHTMLParser is the base parser with generic extraction methods
type ShareHTMLParser struct{}

// NewShareHTMLParser creates a new base parser
func NewShareHTMLParser() *ShareHTMLParser {
	return &ShareHTMLParser{}
}

// ParserForURL creates the appropriate parser for the given URL
func ParserForURL(pageURL *url.URL) Parser {
	return createParser(pageURL)
}

// ParseHTML is the main entry point for parsing HTML
func ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	parser := createParser(pageURL)
	return parser.ParseHTML(html, pageURL)
}

// ParseHTML parses the HTML and returns structured product data
func (p *ShareHTMLParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	utils.Info("📄 ShareHTMLParser: Starting parseHTMLContent")

	rawTitle := p.ExtractTitle(html, pageURL)
	priceString := p.ExtractPrice(html, pageURL)
	imageURL := p.ExtractImage(html, pageURL)
	description := p.ExtractDescription(html, pageURL)
	currency := p.ExtractCurrency(html, pageURL)

	title := filterTitle(rawTitle)
	price := parsePrice(priceString)

	utils.Info("📄 ShareHTMLParser: Extracted data",
		utils.String("title", title),
		utils.String("currency", currency),
		utils.String("imageURL", imageURL))
	
	if price != nil {
		utils.Info("📄 ShareHTMLParser: Extracted price", utils.Float64("price", *price))
	}

	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}

// ExtractTitle extracts the title from HTML
func (p *ShareHTMLParser) ExtractTitle(html string, pageURL *url.URL) string {
	if title := extractMetaProperty("og:title", html); title != "" {
		return title
	}
	if title := extractMetaProperty("twitter:title", html); title != "" {
		return title
	}
	if title := extractTitleTag(html); title != "" {
		return title
	}
	if pageURL.Host != "" {
		return strings.Title(pageURL.Host)
	}
	return "Product"
}

// ExtractPrice extracts the price from HTML
func (p *ShareHTMLParser) ExtractPrice(html string, pageURL *url.URL) string {
	return extractPriceFromHTML(html)
}

// ExtractImage extracts the image URL from HTML
func (p *ShareHTMLParser) ExtractImage(html string, pageURL *url.URL) string {
	if image := extractMetaProperty("og:image", html); image != "" {
		return image
	}
	if image := extractMetaProperty("twitter:image", html); image != "" {
		return image
	}
	return ""
}

// ExtractDescription extracts the description from HTML
func (p *ShareHTMLParser) ExtractDescription(html string, pageURL *url.URL) string {
	if desc := extractMetaProperty("og:description", html); desc != "" {
		return desc
	}
	if desc := extractMetaProperty("twitter:description", html); desc != "" {
		return desc
	}
	return ""
}

// ExtractCurrency extracts the currency from HTML
func (p *ShareHTMLParser) ExtractCurrency(html string, pageURL *url.URL) string {
	if currency := extractCurrencyFromHTML(html); currency != "" {
		return currency
	}
	return guessCurrencyFromURL(pageURL)
}

// Helper functions

func extractMetaProperty(property, html string) string {
	// Try property attribute first (og:title, og:image, etc.)
	pattern := fmt.Sprintf(`<meta[^>]*(?:property|name)=["\']%s["\'][^>]*content=["\'](.*?)["\'][^>]*>`, regexp.QuoteMeta(property))
	if match := extractWithRegex(pattern, html); match != "" {
		return match
	}

	// Try reversed order (content before property)
	pattern = fmt.Sprintf(`<meta[^>]*content=["\'](.*?)["\'][^>]*(?:property|name)=["\']%s["\'][^>]*>`, regexp.QuoteMeta(property))
	return extractWithRegex(pattern, html)
}

func extractTitleTag(html string) string {
	pattern := `<title[^>]*>(.*?)</title>`
	return strings.TrimSpace(extractWithRegex(pattern, html))
}

func extractPriceFromHTML(html string) string {
	utils.Debug("💰 LinkPreviewParser: Starting price extraction")

	patterns := []string{
		`\$[0-9,]+(?:\.[0-9]{2})?`,     // $99.99, $1,999.99
		`€[0-9,]+(?:\.[0-9]{2})?`,      // €99.99
		`£[0-9,]+(?:\.[0-9]{2})?`,      // £99.99
		`R\$[0-9,]+(?:\,[0-9]{2})?`,    // R$99,99
		`[0-9,]+\.[0-9]{2}\s*USD`,      // 99.99 USD
		`[0-9,]+\,[0-9]{2}\s*EUR`,      // 99,99 EUR
	}

	for i, pattern := range patterns {
		utils.Debug("💰 LinkPreviewParser: Trying price pattern", utils.Int("pattern", i+1))
		if match := extractWithRegex(pattern, html); match != "" {
			utils.Info("✅ LinkPreviewParser: Found price with pattern", 
				utils.Int("pattern", i+1),
				utils.String("price", match))
			return match
		}
	}

	utils.Debug("❌ LinkPreviewParser: No price found with any pattern")
	return ""
}

func parsePrice(priceString string) *float64 {
	utils.Debug("🔄 LinkPreviewParser: Starting parsePrice", utils.String("input", priceString))
	if priceString == "" {
		utils.Debug("❌ LinkPreviewParser: Price string is empty")
		return nil
	}

	// Remove currency symbols and common formatting
	cleanedString := priceString
	cleanedString = strings.ReplaceAll(cleanedString, "$", "")
	cleanedString = strings.ReplaceAll(cleanedString, "€", "")
	cleanedString = strings.ReplaceAll(cleanedString, "£", "")
	cleanedString = strings.ReplaceAll(cleanedString, "R$", "")
	cleanedString = strings.ReplaceAll(cleanedString, "USD", "")
	cleanedString = strings.ReplaceAll(cleanedString, "EUR", "")
	cleanedString = strings.ReplaceAll(cleanedString, "GBP", "")
	cleanedString = strings.ReplaceAll(cleanedString, "BRL", "")
	cleanedString = strings.ReplaceAll(cleanedString, ",", "")
	cleanedString = strings.TrimSpace(cleanedString)

	utils.Debug("🔄 LinkPreviewParser: Cleaned price string", utils.String("cleaned", cleanedString))

	if val, err := strconv.ParseFloat(cleanedString, 64); err == nil {
		utils.Debug("🔄 LinkPreviewParser: Parsed price", utils.Float64("value", val))
		return &val
	}

	utils.Debug("❌ LinkPreviewParser: Failed to parse price")
	return nil
}

func extractCurrencyFromHTML(html string) string {
	utils.Debug("💱 LinkPreviewParser: Starting currency extraction from HTML")

	htmlLower := strings.ToLower(html)

	// Check for BRL first (before checking for generic $)
	if strings.Contains(html, "R$") || strings.Contains(htmlLower, "brl") {
		utils.Info("💱 LinkPreviewParser: Found BRL currency")
		return "brl"
	} else if strings.Contains(html, "€") || strings.Contains(htmlLower, "eur") {
		utils.Info("💱 LinkPreviewParser: Found EUR currency")
		return "eur"
	} else if strings.Contains(html, "£") || strings.Contains(htmlLower, "gbp") {
		utils.Info("💱 LinkPreviewParser: Found GBP currency")
		return "gbp"
	} else if strings.Contains(html, "$") && (strings.Contains(htmlLower, "usd") || strings.Contains(htmlLower, "dollar")) {
		utils.Info("💱 LinkPreviewParser: Found USD currency")
		return "usd"
	} else if strings.Contains(html, "¥") || strings.Contains(htmlLower, "jpy") {
		utils.Info("💱 LinkPreviewParser: Found JPY currency")
		return "jpy"
	}

	utils.Debug("💱 LinkPreviewParser: No currency found in HTML")
	return ""
}

func guessCurrencyFromURL(pageURL *url.URL) string {
	utils.Debug("🌐 LinkPreviewParser: Guessing currency from URL", utils.String("url", pageURL.String()))
	
	host := strings.ToLower(pageURL.Host)
	utils.Debug("🌐 LinkPreviewParser: URL host", utils.String("host", host))

	if strings.Contains(host, ".br") || strings.Contains(host, "brazil") {
		utils.Info("🌐 LinkPreviewParser: Detected Brazil domain - returning BRL")
		return "brl"
	} else if strings.Contains(host, ".uk") || strings.Contains(host, ".gb") {
		utils.Info("🌐 LinkPreviewParser: Detected UK domain - returning GBP")
		return "gbp"
	} else if strings.Contains(host, ".de") || strings.Contains(host, ".fr") || strings.Contains(host, ".it") || strings.Contains(host, ".es") || strings.Contains(host, ".pt") {
		utils.Info("🌐 LinkPreviewParser: Detected European domain - returning EUR")
		return "eur"
	} else if strings.Contains(host, ".jp") || strings.Contains(host, "japan") {
		utils.Info("🌐 LinkPreviewParser: Detected Japan domain - returning JPY")
		return "jpy"
	}

	utils.Debug("🌐 LinkPreviewParser: No specific domain match - returning USD as fallback")
	return "usd"
}

func filterTitle(title string) string {
	utils.Debug("🔤 LinkPreviewParser: Starting title filtering", utils.String("title", title))

	// Decode HTML entities first
	filteredTitle := title
	filteredTitle = strings.ReplaceAll(filteredTitle, "&quot;", "\"")
	filteredTitle = strings.ReplaceAll(filteredTitle, "&amp;", "&")
	filteredTitle = strings.ReplaceAll(filteredTitle, "&lt;", "<")
	filteredTitle = strings.ReplaceAll(filteredTitle, "&gt;", ">")
	filteredTitle = strings.ReplaceAll(filteredTitle, "&#x20;", " ")
	filteredTitle = strings.ReplaceAll(filteredTitle, "&nbsp;", " ")
	filteredTitle = strings.ReplaceAll(filteredTitle, "&apos;", "'")

	utils.Debug("🔤 LinkPreviewParser: After HTML entity decoding", utils.String("title", filteredTitle))

	// Handle Amazon case specifically
	if strings.Contains(filteredTitle, ": Amazon") || strings.Contains(filteredTitle, "Amazon.es") || strings.Contains(filteredTitle, "amazon.") {
		amazonPatterns := []string{": Amazon.es: Electrónica", ": Amazon.es", ": Amazon.", "Amazon.es"}
		for _, pattern := range amazonPatterns {
			if idx := strings.Index(strings.ToLower(filteredTitle), strings.ToLower(pattern)); idx != -1 {
				beforeSeparator := strings.TrimSpace(filteredTitle[:idx])
				if len(beforeSeparator) > 10 {
					filteredTitle = beforeSeparator
					utils.Debug("🔤 LinkPreviewParser: Removed Amazon info", 
						utils.String("pattern", pattern),
						utils.String("result", beforeSeparator))
					break
				}
			}
		}
	}

	// Split by common separators to remove website/store information
	websiteSeparators := []string{": Fnac", ": Worten", ": Zara", ": OLX", "| Magazine Luiza", "| MercadoLivre", " - Primark", " : Magazine Luiza", " Primor", " - Magazine Luiza"}

	for _, separator := range websiteSeparators {
		if idx := strings.Index(filteredTitle, separator); idx != -1 {
			beforeSeparator := strings.TrimSpace(filteredTitle[:idx])
			if len(beforeSeparator) > 5 {
				filteredTitle = beforeSeparator
				utils.Debug("🔤 LinkPreviewParser: Removed website info",
					utils.String("separator", separator),
					utils.String("result", beforeSeparator))
				break
			}
		}
	}

	// Remove unnecessary patterns
	patternsToRemove := []string{
		`\d+V\s*$`,                      // Voltage like "127V", "220V" at end
		`\d+W\s*$`,                      // Wattage like "1450W" at end
		`\d+\s*em\s*\d+\s*$`,            // "3 em 1", "2 em 1" at end
		`\s+\(Reacondicionado\)\s*$`,    // "(Reacondicionado)" at end
		`\s+\(Recondicionado\)\s*$`,     // "(Recondicionado)" at end
		`\s+\(Usado\)\s*$`,              // "(Usado)" at end
		`\s+\(Used\)\s*$`,               // "(Used)" at end
	}

	for _, pattern := range patternsToRemove {
		re := regexp.MustCompile("(?i)" + pattern)
		newTitle := strings.TrimSpace(re.ReplaceAllString(filteredTitle, ""))
		if len(newTitle) > 10 {
			filteredTitle = newTitle
		}
	}

	// Truncate if too long
	maxLength := 80
	if len(filteredTitle) > maxLength {
		if lastSpace := strings.LastIndex(filteredTitle[:maxLength], " "); lastSpace != -1 {
			filteredTitle = filteredTitle[:lastSpace]
		} else {
			filteredTitle = filteredTitle[:maxLength]
		}
		utils.Debug("🔤 LinkPreviewParser: Truncated title", 
			utils.Int("maxLength", maxLength),
			utils.String("result", filteredTitle))
	}

	filteredTitle = strings.TrimSpace(filteredTitle)

	// If filtering made the title too short, return original
	if len(filteredTitle) < 5 {
		utils.Debug("🔤 LinkPreviewParser: Filtered title too short, returning original")
		return title
	}

	utils.Debug("🔤 LinkPreviewParser: Final filtered title", utils.String("title", filteredTitle))
	return filteredTitle
}

func extractWithRegex(pattern, html string) string {
	re := regexp.MustCompile("(?i)" + pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func extractStructuredDataTitle(html string) string {
	pattern := `"name":"([^"]+)"`
	return extractWithRegex(pattern, html)
}

func extractStructuredDataPrice(html string) string {
	patterns := []string{
		`"price":(\d+(?:\.\d+)?)`,  // "price":1050
		`"price":"([^"]+)"`,         // "price":"1050.00"
	}

	for _, pattern := range patterns {
		if price := extractWithRegex(pattern, html); price != "" {
			if strings.Contains(html, "EUR") {
				return price + "€"
			}
			return price
		}
	}
	return ""
}

func extractStructuredDataImage(html string) string {
	pattern := `"image":"([^"]+)"`
	return extractWithRegex(pattern, html)
}
