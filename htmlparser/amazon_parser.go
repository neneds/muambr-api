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
	// Amazon-specific title patterns in order of preference (based on amazon_scraper.go)
	patterns := []string{
		// Product title span (most common)
		`<span[^>]*id="productTitle"[^>]*>([^<]+)</span>`,
		// Alternative product title patterns from scraper
		`div\[data-cy=['"]title-recipe['"]\]\s*a\.a-link-normal[^>]*>([^<]+)</a>`,
		`div\.puisg-col-inner\s*a\.a-link-normal[^>]*>([^<]+)</a>`,
		// Meta name="title" 
		`<meta[^>]*name=["']title["'][^>]*content=["'](.*?)["'][^>]*>`,
		// H1 with product title class
		`<h1[^>]*class="[^"]*product[^"]*title[^"]*"[^>]*>([^<]+)</h1>`,
		// Search results title patterns (from scraper)
		`<h2[^>]*class="[^"]*s-size-mini[^"]*"[^>]*>.*?<a[^>]*>.*?<span[^>]*>([^<]+)</span>`,
		// Breadcrumb last item (product name)
		`<span[^>]*class="[^"]*breadcrumb[^"]*"[^>]*>.*?<span[^>]*>([^<]+)</span>\s*</span>`,
	}

	for i, pattern := range patterns {
		utils.Debug("📄 AmazonParser: Trying title pattern", utils.Int("pattern", i+1))
		if title := extractWithRegex(pattern, html); title != "" {
			cleaned := strings.TrimSpace(title)
			// Filter out unwanted titles (from scraper logic)
			lower := strings.ToLower(cleaned)
			if cleaned != "" &&
				!strings.Contains(lower, "check each product") &&
				!strings.Contains(lower, "let us know") &&
				!strings.Contains(lower, "results") &&
				!strings.Contains(lower, "your search") &&
				!strings.Contains(lower, "amazon.com") &&
				!strings.Contains(lower, "www.amazon") &&
				len(cleaned) > 10 {
				utils.Debug("📄 AmazonParser: Found valid title", utils.String("title", cleaned), utils.Int("pattern", i+1))
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
	
	// Check for Brazilian Amazon domains and short URLs
	isBrazil := strings.Contains(host, ".br") || strings.Contains(host, "amazon.com.br")
	
	// Check for US Amazon domains  
	isUS := strings.Contains(host, "amazon.com") && !strings.Contains(host, ".br")
	
	// Handle Amazon short URLs (a.co, amzn.to) - detect region from HTML content
	isAmazonShortURL := strings.Contains(host, "a.co") || strings.Contains(host, "amzn.to") || strings.Contains(host, "amzn.")
	
	if isAmazonShortURL {
		// For short URLs, detect region from HTML content
		htmlLower := strings.ToLower(html)
		if strings.Contains(html, "R$") || strings.Contains(htmlLower, "reais") || strings.Contains(htmlLower, "brasil") {
			isBrazil = true
			isUS = false
		} else if strings.Contains(html, "$") && (strings.Contains(htmlLower, "usd") || strings.Contains(htmlLower, "dollar")) {
			isUS = true  
			isBrazil = false
		}
		// If no clear indicators, fall back to comprehensive pattern matching
	}

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
	} else if isUS {
		// US patterns ($ with dot as decimal separator)
		patterns = []string{
			`<span class="a-offscreen">\$([0-9.,]+)</span>`,         // $99.99
			`<span class="aok-offscreen">\s*\$([0-9.,]+)\s*</span>`, // Alternative class
			`"priceAmount":([0-9.]+)`,                               // JSON-LD price data
			`data-asin-price="([0-9.,]+)"`,                          // Data attribute price
			`<span[^>]*class="[^"]*a-price-whole[^"]*"[^>]*>([0-9.,]+)</span>`, // Price whole part
			`<span[^>]*class="[^"]*price[^"]*"[^>]*>\$([0-9.,]+)</span>`,       // Generic price span
		}
	} else if isAmazonShortURL {
		// For short URLs, try all currency patterns since we can't determine region from URL
		patterns = []string{
			// Brazilian patterns
			`<span class="a-offscreen">R\$\s*([0-9.,]+)</span>`,     // R$612,25 or R$ 612,25
			`<span class="aok-offscreen">\s*R\$\s*([0-9.,]+)\s*</span>`, 
			// US patterns  
			`<span class="a-offscreen">\$([0-9.,]+)</span>`,         // $99.99
			`<span class="aok-offscreen">\s*\$([0-9.,]+)\s*</span>`,
			// European patterns
			`<span class="a-offscreen">([0-9.,]+)\s*€</span>`,       // 470,00€
			`<span class="aok-offscreen">\s*([0-9.,]+)\s*€\s*</span>`,
			// Generic patterns
			`"priceAmount":([0-9.]+)`,                               // JSON-LD price data
			`data-asin-price="([0-9.,]+)"`,                          // Data attribute price
			`<span[^>]*class="[^"]*a-price-whole[^"]*"[^>]*>([0-9.,]+)</span>`, // Price whole part
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
				// For Brazilian prices, preserve original format and add R$ prefix if needed
				if strings.HasPrefix(cleanedPrice, "R$") {
					return cleanedPrice
				}
				return "R$" + cleanedPrice
			} else if isUS {
				// For US prices, add $ prefix if needed
				if strings.HasPrefix(cleanedPrice, "$") {
					return cleanedPrice
				}
				return "$" + cleanedPrice
			} else if isAmazonShortURL {
				// For short URLs, detect currency from the matched price pattern
				if strings.Contains(priceMatch, "R$") || strings.HasPrefix(cleanedPrice, "R$") {
					return priceMatch // Return as-is since R$ is already there
				} else if strings.Contains(priceMatch, "$") || strings.HasPrefix(cleanedPrice, "$") {
					return priceMatch // Return as-is since $ is already there  
				} else if strings.Contains(priceMatch, "€") || strings.HasSuffix(cleanedPrice, "€") {
					return priceMatch // Return as-is since € is already there
				} else {
					// No currency symbol found, determine from HTML context
					htmlLower := strings.ToLower(html)
					if strings.Contains(html, "R$") || strings.Contains(htmlLower, "reais") {
						return "R$" + cleanedPrice
					} else if strings.Contains(html, "$") && strings.Contains(htmlLower, "usd") {
						return "$" + cleanedPrice
					} else if strings.Contains(html, "€") || strings.Contains(htmlLower, "eur") {
						return cleanedPrice + "€"
					} else {
						return "$" + cleanedPrice // Default to USD
					}
				}
			} else {
				// For European prices, add € suffix and preserve original format
				if strings.HasSuffix(cleanedPrice, "€") {
					return cleanedPrice
				}
				return cleanedPrice + "€"
			}
		}
	}

	// Fallback to generic
	return p.ShareHTMLParser.ExtractPrice(html, pageURL)
}

func (p *AmazonParser) ExtractImage(html string, pageURL *url.URL) string {
	// Amazon uses various patterns for images (enhanced with scraper insights)
	patterns := []string{
		`data-old-hires="([^"]+)"`,                             // Primary pattern for high-res images
		`'colorImages':\s*\{\s*'initial':\s*\[{"hiRes":"([^"]+)"`, // JavaScript color images data
		`id="landingImage"[^>]*src="([^"]+)"`,                  // Landing image src attribute
		`data-a-dynamic-image="[^"]*"([^"]*\.jpg[^"]*)"`,       // Dynamic image data
		`<img[^>]*class="[^"]*product[^"]*image[^"]*"[^>]*src="([^"]+)"`, // Product image class
		`<img[^>]*id="[^"]*main[^"]*image[^"]*"[^>]*src="([^"]+)"`, // Main image ID
	}

	utils.Debug("🖼️ AmazonParser: Starting image extraction")
	
	for i, pattern := range patterns {
		utils.Debug("🖼️ AmazonParser: Trying image pattern", utils.Int("pattern", i+1))
		if imageURL := extractWithRegex(pattern, html); imageURL != "" {
			// Clean up the image URL
			cleanedURL := strings.TrimSpace(imageURL)
			
			// Ensure URL is properly formatted
			if strings.HasPrefix(cleanedURL, "http") {
				utils.Debug("🖼️ AmazonParser: Found image URL", utils.String("url", cleanedURL), utils.Int("pattern", i+1))
				return cleanedURL
			} else if strings.HasPrefix(cleanedURL, "//") {
				// Handle protocol-relative URLs
				fullURL := "https:" + cleanedURL
				utils.Debug("🖼️ AmazonParser: Found protocol-relative image URL", utils.String("url", fullURL), utils.Int("pattern", i+1))
				return fullURL
			}
		}
	}

	// Fallback to generic extraction
	utils.Debug("🖼️ AmazonParser: Using generic image extraction")
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
	
	// Brazilian Amazon sites
	if strings.Contains(host, ".br") || strings.Contains(host, "amazon.com.br") {
		return "brl"
	}
	
	// US Amazon sites
	if strings.Contains(host, "amazon.com") && !strings.Contains(host, ".") {
		return "usd"
	}
	
	// UK Amazon sites
	if strings.Contains(host, ".uk") || strings.Contains(host, ".gb") {
		return "gbp"
	}
	
	// Canadian Amazon sites
	if strings.Contains(host, ".ca") {
		return "cad"
	}
	
	// Japanese Amazon sites
	if strings.Contains(host, ".jp") {
		return "jpy"
	}
	
	// European Amazon sites (Spain, Germany, France, Italy, Netherlands)
	if strings.Contains(host, ".es") || strings.Contains(host, ".de") || 
		 strings.Contains(host, ".fr") || strings.Contains(host, ".it") || 
		 strings.Contains(host, ".nl") {
		return "eur"
	}
	
	// Default fallback - check HTML content for currency indicators
	htmlLower := strings.ToLower(html)
	if strings.Contains(html, "R$") || strings.Contains(htmlLower, "reais") {
		return "brl"
	} else if strings.Contains(html, "$") && strings.Contains(htmlLower, "usd") {
		return "usd"
	} else if strings.Contains(html, "£") || strings.Contains(htmlLower, "gbp") {
		return "gbp"
	} else if strings.Contains(html, "€") || strings.Contains(htmlLower, "eur") {
		return "eur"
	}
	
	// Ultimate fallback based on domain patterns
	return "usd"
}

func (p *AmazonParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
	utils.Info("📄 AmazonParser: Starting parseHTMLContent", utils.String("url", pageURL.String()))

	// Check if we're likely being blocked by examining HTML content
	if isLikelyBlocked(html) {
		utils.Warn("🚫 AmazonParser: Likely blocked by anti-bot protection")
	}

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
	} else {
		utils.Warn("📄 AmazonParser: Failed to extract price", utils.String("priceString", priceString))
	}

	return &ParsedProductData{
		Title:       title,
		Price:       price,
		Currency:    currency,
		ImageURL:    imageURL,
		Description: description,
	}
}

// isLikelyBlocked checks if the HTML content indicates we're being blocked
func isLikelyBlocked(html string) bool {
	blockedIndicators := []string{
		"robot or spider",
		"access denied",
		"captcha",
		"please verify",
		"blocked",
		"not authorized",
		"temporarily unavailable",
		"service unavailable",
	}
	
	lowerHTML := strings.ToLower(html)
	for _, indicator := range blockedIndicators {
		if strings.Contains(lowerHTML, indicator) {
			return true
		}
	}
	
	// Check for very short HTML content (likely an error page)
	return len(html) < 1000
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
	if currency == "eur" || currency == "brl" {
		// European/Brazilian format: comma as decimal separator, dot as thousands separator
		// Examples: 1.250,99 (BRL), 470,00 (EUR)
		if strings.Contains(cleanedString, ",") {
			// Find the last comma (should be decimal separator)
			lastCommaIdx := strings.LastIndex(cleanedString, ",")
			if lastCommaIdx != -1 {
				afterComma := cleanedString[lastCommaIdx+1:]
				// If it's 2 digits after comma, treat as decimal
				if len(afterComma) == 2 {
					// Remove all dots (thousands separators) and convert comma to dot
					beforeComma := strings.ReplaceAll(cleanedString[:lastCommaIdx], ".", "")
					cleanedString = beforeComma + "." + afterComma
				} else {
					// Remove comma (thousands separator)
					cleanedString = strings.ReplaceAll(cleanedString, ",", "")
					// Also remove dots (thousands separator)
					cleanedString = strings.ReplaceAll(cleanedString, ".", "")
				}
			}
		} else if strings.Contains(cleanedString, ".") {
			// No comma, check if dot is thousands separator or decimal
			// If format is X.XXX where XXX is exactly 3 digits, it's thousands separator
			if matches := regexp.MustCompile(`^(\d{1,3})\.(\d{3})$`).FindStringSubmatch(cleanedString); matches != nil {
				// Format like "1.050" - treat as thousands separator
				cleanedString = matches[1] + matches[2]
			}
			// Otherwise, keep dot as decimal separator
		}
	} else {
		// US format: dot as decimal separator, comma as thousands separator
		// Remove commas (thousands separators) except if it's the last one with 2 digits after
		if strings.Contains(cleanedString, ",") {
			lastCommaIdx := strings.LastIndex(cleanedString, ",")
			if lastCommaIdx != -1 {
				afterComma := cleanedString[lastCommaIdx+1:]
				// If 2 digits after last comma, convert comma to dot (decimal)
				if len(afterComma) == 2 {
					cleanedString = cleanedString[:lastCommaIdx] + "." + afterComma
				} else {
					// Remove all commas (thousands separators)
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