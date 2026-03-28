package extractors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"muambr-api/models"
	"muambr-api/utils"

	"github.com/PuerkitoBio/goquery"
)

// mercadoLivreNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// MercadoLivreExtractorV2 overrides GetComparisons/GetComparisonsFromHTML so these are unused.
type mercadoLivreNoopParser struct{ *BaseHTMLParser }

func (p *mercadoLivreNoopParser) GetProductSelectors() []string          { return nil }
func (p *mercadoLivreNoopParser) GetNameSelectors() []string             { return nil }
func (p *mercadoLivreNoopParser) GetPriceSelectors() []string            { return nil }
func (p *mercadoLivreNoopParser) GetURLSelectors() []string              { return nil }
func (p *mercadoLivreNoopParser) ParseProductName(html string) string    { return "" }
func (p *mercadoLivreNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *mercadoLivreNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *mercadoLivreNoopParser) ParseStore(html string) string               { return "" }

// MercadoLivreExtractorV2 is a pure Go extractor for MercadoLivre Brazil.
type MercadoLivreExtractorV2 struct {
	*BaseGoExtractor
}

// NewMercadoLivreExtractorV2 creates a new pure Go MercadoLivre extractor.
func NewMercadoLivreExtractorV2() *MercadoLivreExtractorV2 {
	parser := &mercadoLivreNoopParser{BaseHTMLParser: NewBaseHTMLParser("MercadoLivre")}
	baseExtractor := NewBaseGoExtractor(
		"https://lista.mercadolivre.com.br",
		models.CountryBrazil,
		"mercadolivre_v2",
		parser,
	)

	return &MercadoLivreExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// BuildSearchURL builds the MercadoLivre search URL using their path-based format.
func (e *MercadoLivreExtractorV2) BuildSearchURL(productName string) (string, error) {
	slug := strings.ReplaceAll(strings.ToLower(productName), " ", "-")
	searchURL := fmt.Sprintf("%s/%s", e.GetBaseURL(), slug)

	utils.Info("🔗 Built MercadoLivre search URL",
		utils.String("product", productName),
		utils.String("url", searchURL))

	return searchURL, nil
}

// GetComparisons fetches the MercadoLivre search page and extracts products.
func (e *MercadoLivreExtractorV2) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("🚀 Starting MercadoLivre product extraction",
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	html, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	comparisons, err := e.GetComparisonsFromHTML(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract comparisons: %w", err)
	}

	utils.Info("✅ Extraction completed",
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// GetComparisonsFromHTML extracts products from JSON-LD first, then falls back to HTML parsing with goquery.
func (e *MercadoLivreExtractorV2) GetComparisonsFromHTML(html string) ([]models.ProductComparison, error) {
	utils.Info("📄 Parsing MercadoLivre HTML", utils.Int("size", len(html)))

	// Primary: JSON-LD structured data (most reliable)
	if comparisons := e.extractFromJSONLD(html); len(comparisons) > 0 {
		utils.Info("✅ Extracted products from JSON-LD", utils.Int("count", len(comparisons)))
		return comparisons, nil
	}

	// Fallback: parse product cards with goquery
	comparisons, err := e.extractFromHTML(html)
	if err != nil {
		return nil, err
	}

	utils.Info("✅ Extracted products from HTML", utils.Int("count", len(comparisons)))
	return comparisons, nil
}

// ---------- JSON-LD extraction ----------

// extractFromJSONLD extracts products from JSON-LD script blocks.
func (e *MercadoLivreExtractorV2) extractFromJSONLD(html string) []models.ProductComparison {
	var comparisons []models.ProductComparison

	re := regexp.MustCompile(`(?i)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(match[1]), &raw); err != nil {
			continue
		}

		// Handle @graph array (MercadoLivre search pages)
		if graph, ok := raw["@graph"].([]interface{}); ok {
			for _, item := range graph {
				if m, ok := item.(map[string]interface{}); ok {
					if c := e.jsonLDProduct(m); c != nil {
						comparisons = append(comparisons, *c)
					}
				}
			}
		} else if c := e.jsonLDProduct(raw); c != nil {
			comparisons = append(comparisons, *c)
		}
	}

	return comparisons
}

// jsonLDProduct converts a single JSON-LD Product object to a ProductComparison.
func (e *MercadoLivreExtractorV2) jsonLDProduct(data map[string]interface{}) *models.ProductComparison {
	if t, _ := data["@type"].(string); t != "Product" {
		return nil
	}
	name, _ := data["name"].(string)
	if name == "" {
		return nil
	}

	offers, _ := data["offers"].(map[string]interface{})
	if offers == nil {
		return nil
	}

	price := extractJSONPrice(offers)
	if price <= 0 {
		return nil
	}

	currency, _ := offers["priceCurrency"].(string)
	if currency == "" {
		currency = "BRL"
	}

	urlStr, _ := offers["url"].(string)
	imageStr, _ := data["image"].(string)

	var storeURL, imageURL *string
	if urlStr != "" {
		storeURL = &urlStr
	}
	if imageStr != "" {
		imageURL = &imageStr
	}

	return &models.ProductComparison{
		ID:          utils.GenerateUUID(),
		ProductName: strings.TrimSpace(name),
		Price:       price,
		Currency:    currency,
		StoreName:   "MercadoLivre",
		StoreURL:    storeURL,
		ImageURL:    imageURL,
		Country:     string(models.CountryBrazil),
	}
}

// extractJSONPrice reads a price that may be a float64 or a string.
func extractJSONPrice(offers map[string]interface{}) float64 {
	switch v := offers["price"].(type) {
	case float64:
		return v
	case string:
		s := strings.ReplaceAll(v, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	return 0
}

// ---------- HTML (goquery) fallback ----------

// extractFromHTML parses product cards using goquery selectors.
func (e *MercadoLivreExtractorV2) extractFromHTML(html string) ([]models.ProductComparison, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var comparisons []models.ProductComparison

	doc.Find("li.ui-search-layout__item").Each(func(_ int, item *goquery.Selection) {
		// Product name
		name := item.Find("a.poly-component__title").Text()
		if name == "" {
			name = item.Find("h2.poly-component__title-wrapper a").Text()
		}
		name = strings.TrimSpace(name)
		if name == "" || len(name) < 3 {
			return
		}

		// Price (andes-money-amount inside poly-price__current)
		priceText := item.Find(".poly-price__current .andes-money-amount__fraction").First().Text()
		if priceText == "" {
			return
		}
		price := parseMercadoLivrePrice(priceText)
		if price <= 0 {
			return
		}

		// Check for cents
		cents := item.Find(".poly-price__current .andes-money-amount__cents").First().Text()
		if cents != "" {
			if c, err := strconv.ParseFloat("0."+cents, 64); err == nil {
				price += c
			}
		}

		// Product URL
		productURL, _ := item.Find("a.poly-component__title").Attr("href")
		if productURL == "" {
			productURL, _ = item.Find("h2.poly-component__title-wrapper a").Attr("href")
		}

		// Image URL
		imageURL := ""
		if img := item.Find("img.poly-component__picture"); img.Length() > 0 {
			imageURL, _ = img.Attr("src")
		}

		var storeURLPtr, imageURLPtr *string
		if productURL != "" {
			storeURLPtr = &productURL
		}
		if imageURL != "" {
			imageURLPtr = &imageURL
		}

		comparisons = append(comparisons, models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: name,
			Price:       price,
			Currency:    "BRL",
			StoreName:   "MercadoLivre",
			StoreURL:    storeURLPtr,
			ImageURL:    imageURLPtr,
			Country:     string(models.CountryBrazil),
		})
	})

	return comparisons, nil
}

// parseMercadoLivrePrice parses a Brazilian price string like "4.699" to float64.
func parseMercadoLivrePrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")  // remove thousand separators
	s = strings.ReplaceAll(s, ",", ".") // decimal comma -> dot
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

