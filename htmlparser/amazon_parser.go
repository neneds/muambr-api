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
	// Amazon-specific title patterns in order of preference
	patterns := []string{
		// Product title span (most common)
		`<span[^>]*id="productTitle"[^>]*>([^<]+)</span>`,
		// Meta name="title" 
		`<meta[^>]*name=["']title["'][^>]*content=["'](.*?)["'][^>]*>`,
		// H1 with product title class
		`<h1[^>]*class="[^"]*product[^"]*title[^"]*"[^>]*>([^<]+)</h1>`,
		// Breadcrumb last item (product name)
		`<span[^>]*class="[^"]*breadcrumb[^"]*"[^>]*>.*?<span[^>]*>([^<]+)</span>\s*</span>`,
	}

	for _, pattern := range patterns {
		if title := extractWithRegex(pattern, html); title != "" {
			cleaned := strings.TrimSpace(title)
			// Avoid generic titles like domain names
			if !strings.Contains(strings.ToLower(cleaned), "amazon") && len(cleaned) > 10 {
				return cleaned
			}
		}
	}

	// Try structured data
	if structuredTitle := extractStructuredDataTitle(html); structuredTitle != "" {
		return structuredTitle
	}

	// Fallback to generic extraction
	genericTitle := p.ShareHTMLParser.ExtractTitle(html, pageURL)
	
	// Check if we got a blocked/generic title
	if strings.Contains(strings.ToLower(genericTitle), "amazon.com") || 
		 strings.Contains(strings.ToLower(genericTitle), "www.amazon") {
		// We're likely being blocked - return a more descriptive message
		return "Amazon Product (Anti-bot Protection Active)"
	}
	
	return genericTitle
}

func (p *AmazonParser) ExtractPrice(html string, pageURL *url.URL) string {
	host := strings.ToLower(pageURL.Host)
	isBrazil := strings.Contains(host, ".br") || strings.Contains(host, "amazon.com.br")

	// Amazon uses specific price patterns in a-offscreen spans
	var patterns []string
	if isBrazil {
		// Brazilian-specific patterns (R$ with comma as decimal separator)
		patterns = []string{
			`<span class="a-offscreen">R\$\s*([0-9.,]+)</span>`,     // R$612,25 or R$ 612,25
			`<span class="aok-offscreen">\s*R\$\s*([0-9.,]+)\s*</span>`, // Alternative class
			`"priceAmount":([0-9.]+)`,                               // JSON-LD price data
			`data-asin-price="([0-9.,]+)"`,                          // Data attribute price
			`<span[^>]*class="[^"]*a-price-whole[^"]*"[^>]*>([0-9.,]+)</span>`, // Price whole part
			`<span[^>]*class="[^"]*price[^"]*"[^>]*>R\$\s*([0-9.,]+)</span>`,   // Generic price span
		}
	} else {
		// European patterns (€ with comma/dot variations)
		patterns = []string{
			`<span class="a-offscreen">([0-9.,]+)\s*€</span>`,       // 470,00€ or 470,00 €
			`<span class="aok-offscreen">\s*([0-9.,]+)\s*€\s*</span>`, // Alternative class
			`"priceAmount":([0-9.]+)`,                               // JSON-LD price data
			`data-asin-price="([0-9.,]+)"`,                          // Data attribute price
			`<span[^>]*class="[^"]*a-price-whole[^"]*"[^>]*>([0-9.,]+)</span>`, // Price whole part
		}
	}

	for i, pattern := range patterns {
		utils.Debug("💰 LinkPreviewParser: Trying Amazon price pattern", utils.Int("pattern", i+1))
		if priceMatch := extractWithRegex(pattern, html); priceMatch != "" {
			utils.Debug("💰 LinkPreviewParser: Found Amazon price match", utils.String("match", priceMatch), utils.Int("pattern", i+1))
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