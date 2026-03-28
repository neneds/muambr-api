package extractors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"
)

// acharPromoDeal represents a deal from the AcharPromo deals page RSC payload
type acharPromoDeal struct {
	ID        int     `json:"id"`
	CreatedAt string  `json:"created_at"`
	Image     string  `json:"image"`
	Source    string  `json:"source"`
	OldPrice  *string `json:"oldPrice"`
	Title     string  `json:"title"`
	Price     string  `json:"price"`
	URL       string  `json:"url"`
}

// acharPromoNoopParser is a minimal HTMLParser implementation for AcharPromo.
// AcharPromo extraction is done via RSC payload parsing, not HTML regex, but
// BaseGoExtractor requires an HTMLParser to be passed in.
type acharPromoNoopParser struct {
	*BaseHTMLParser
}

func newAcharPromoNoopParser() *acharPromoNoopParser {
	return &acharPromoNoopParser{BaseHTMLParser: NewBaseHTMLParser("AcharPromo")}
}

func (p *acharPromoNoopParser) GetProductSelectors() []string { return nil }
func (p *acharPromoNoopParser) GetNameSelectors() []string    { return nil }
func (p *acharPromoNoopParser) GetPriceSelectors() []string   { return nil }
func (p *acharPromoNoopParser) GetURLSelectors() []string     { return nil }
func (p *acharPromoNoopParser) ParseProductName(_ string) string {
	return ""
}
func (p *acharPromoNoopParser) ParsePrice(_ string) (float64, string, error) {
	return 0, "BRL", fmt.Errorf("not implemented")
}
func (p *acharPromoNoopParser) ParseURL(_ string, baseURL string) string {
	return baseURL
}
func (p *acharPromoNoopParser) ParseStore(_ string) string {
	return "AcharPromo Brasil"
}

// AcharPromoExtractorV2 scrapes the AcharPromo /deals page for curated product deals.
// AcharPromo moved to a chat-based AI interface; the old JSON API and search page
// no longer return product data. The /deals page embeds product data in a Next.js
// React Server Component payload which we extract and parse.
type AcharPromoExtractorV2 struct {
	*BaseGoExtractor
}

// NewAcharPromoExtractorV2 creates a new pure Go AcharPromo extractor
func NewAcharPromoExtractorV2() *AcharPromoExtractorV2 {
	// We still need a parser for the base extractor, but we override GetComparisons
	parser := newAcharPromoNoopParser()
	baseExtractor := NewBaseGoExtractor(
		"https://achar.promo",
		models.CountryBrazil,
		"acharpromo_v2",
		parser,
	)

	return &AcharPromoExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// GetComparisons fetches the /deals page and extracts products from the RSC payload,
// then filters by the search term.
func (e *AcharPromoExtractorV2) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("🚀 Starting AcharPromo deals extraction",
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	dealsURL := e.GetBaseURL() + "/deals"

	html, err := e.FetchHTML(dealsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deals page: %w", err)
	}

	comparisons, err := e.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract deals: %w", err)
	}

	// Filter by search term
	filtered := filterBySearchTerm(comparisons, productName)

	utils.Info("✅ Extraction completed",
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("total_deals", len(comparisons)),
		utils.Int("filtered_results", len(filtered)))

	return filtered, nil
}

// GetComparisonsFromHTML extracts products from the Next.js RSC payload embedded in the deals page HTML.
func (e *AcharPromoExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing AcharPromo deals HTML", utils.Int("size", len(html)))

	deals := e.extractDealsFromRSCPayload(html)
	if len(deals) == 0 {
		utils.Warn("No deals found in AcharPromo RSC payload")
		return nil, nil
	}

	var comparisons []models.ProductComparison
	for _, deal := range deals {
		comp, err := e.convertDealToComparison(deal)
		if err != nil {
			continue
		}
		comparisons = append(comparisons, comp)
	}

	utils.Info("Extracted AcharPromo deals", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// extractDealsFromRSCPayload parses the __next_f.push RSC payloads to find the
// initialProducts JSON array embedded in the server-rendered deals page.
func (e *AcharPromoExtractorV2) extractDealsFromRSCPayload(html string) []acharPromoDeal {
	// Find the initialProducts array in the RSC payload
	// The data is escaped inside a JS string: \"initialProducts\":[{...}]
	marker := `initialProducts\":`
	idx := strings.Index(html, marker)
	if idx == -1 {
		// Try unescaped variant
		marker = `"initialProducts":`
		idx = strings.Index(html, marker)
		if idx == -1 {
			utils.Info("initialProducts marker not found in HTML")
			return nil
		}
	}

	// Find the opening bracket of the JSON array
	arrStart := strings.Index(html[idx:], "[")
	if arrStart == -1 {
		return nil
	}
	arrStart += idx

	// Find the matching closing bracket
	depth := 0
	end := arrStart
	for j := arrStart; j < len(html); j++ {
		switch html[j] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				end = j + 1
				goto found
			}
		}
	}
	return nil

found:
	raw := html[arrStart:end]
	// Unescape the JSON (it's inside a JS string literal)
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\\`, `\`)

	var deals []acharPromoDeal
	if err := json.Unmarshal([]byte(raw), &deals); err != nil {
		utils.Warn("Failed to parse AcharPromo deals JSON", utils.Error(err))
		return nil
	}

	return deals
}

// convertDealToComparison converts an acharPromoDeal to a ProductComparison.
func (e *AcharPromoExtractorV2) convertDealToComparison(deal acharPromoDeal) (models.ProductComparison, error) {
	if deal.Title == "" || deal.Price == "" {
		return models.ProductComparison{}, fmt.Errorf("missing title or price")
	}

	price, err := parseBrazilianPrice(deal.Price)
	if err != nil || price <= 0 {
		return models.ProductComparison{}, fmt.Errorf("invalid price %q: %w", deal.Price, err)
	}

	storeName := deal.Source
	if storeName == "" {
		storeName = "AcharPromo Brasil"
	}

	var storeURL, imageURL *string
	if deal.URL != "" {
		storeURL = &deal.URL
	}
	if deal.Image != "" {
		imageURL = &deal.Image
	}

	return models.ProductComparison{
		ID:          utils.GenerateUUID(),
		ProductName: strings.TrimSpace(deal.Title),
		Price:       price,
		Currency:    "BRL",
		StoreName:   storeName,
		StoreURL:    storeURL,
		ImageURL:    imageURL,
		Country:     string(models.CountryBrazil),
	}, nil
}

// parseBrazilianPrice parses a Brazilian price string like "1.299,99" or "502" to float64.
func parseBrazilianPrice(priceStr string) (float64, error) {
	s := strings.TrimSpace(priceStr)
	// Remove R$ prefix if present
	s = strings.TrimPrefix(s, "R$")
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, fmt.Errorf("empty price")
	}

	// Brazilian format: dots as thousand separators, comma as decimal
	// "1.299,99" -> "1299.99"
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")

	return strconv.ParseFloat(s, 64)
}

// filterBySearchTerm returns comparisons whose title contains all words from the search term.
func filterBySearchTerm(comparisons []models.ProductComparison, searchTerm string) []models.ProductComparison {
	if searchTerm == "" {
		return comparisons
	}

	words := strings.Fields(strings.ToLower(searchTerm))
	if len(words) == 0 {
		return comparisons
	}

	// Build a regex pattern that matches if any word is present
	var patterns []string
	for _, w := range words {
		patterns = append(patterns, regexp.QuoteMeta(w))
	}
	re, err := regexp.Compile("(?i)(" + strings.Join(patterns, "|") + ")")
	if err != nil {
		return comparisons
	}

	var filtered []models.ProductComparison
	for _, c := range comparisons {
		if re.MatchString(c.ProductName) {
			filtered = append(filtered, c)
		}
	}

	return filtered
}