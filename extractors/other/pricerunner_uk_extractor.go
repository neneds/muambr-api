package other

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// priceRunnerUKNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// PriceRunnerUKExtractor overrides GetComparisons entirely and uses the instant-search
// JSON API, so the parser methods are never called.
type priceRunnerUKNoopParser struct{ *extractors.BaseHTMLParser }

func (p *priceRunnerUKNoopParser) GetProductSelectors() []string       { return nil }
func (p *priceRunnerUKNoopParser) GetNameSelectors() []string          { return nil }
func (p *priceRunnerUKNoopParser) GetPriceSelectors() []string         { return nil }
func (p *priceRunnerUKNoopParser) GetURLSelectors() []string           { return nil }
func (p *priceRunnerUKNoopParser) ParseProductName(html string) string { return "" }
func (p *priceRunnerUKNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *priceRunnerUKNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *priceRunnerUKNoopParser) ParseStore(html string) string               { return "" }

const priceRunnerUKSuggestPath = "/uk/api/instant-search-edge-rest/public/search/suggest/UK"

// PriceRunnerUKExtractor extracts generic UK offers from PriceRunner.
//
// Storefront HTML is a Klarna SPA; category search needs numeric IDs. Results
// come from the public instant-search JSON API (name + lowestPrice). Do not
// scrape pricerunner.com HTML.
//
// Endpoint: GET https://www.pricerunner.com/uk/api/instant-search-edge-rest/public/search/suggest/UK?q={q}
type PriceRunnerUKExtractor struct {
	*extractors.BaseGoExtractor
}

// NewPriceRunnerUKExtractor creates a new PriceRunner UK extractor.
func NewPriceRunnerUKExtractor() *PriceRunnerUKExtractor {
	parser := &priceRunnerUKNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("pricerunner_uk")}
	base := extractors.NewBaseGoExtractor(
		"https://www.pricerunner.com",
		models.CountryUK,
		"pricerunner_uk_v1",
		parser,
	)
	return &PriceRunnerUKExtractor{BaseGoExtractor: base}
}

// GetCategory returns the generic/other category for registry filtering.
func (e *PriceRunnerUKExtractor) GetCategory() models.ProductCategory {
	return models.CategoryOther
}

// BuildSearchURL constructs the instant-search suggest URL for PriceRunner UK.
func (e *PriceRunnerUKExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("q", productName)
	return e.GetBaseURL() + priceRunnerUKSuggestPath + "?" + params.Encode(), nil
}

// GetComparisons fetches products from instant-search and returns comparisons.
func (e *PriceRunnerUKExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("pricerunner_uk: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("pricerunner_uk: failed to fetch API: %w", err)
	}

	return e.parseSuggestResponse(body)
}

type priceRunnerSuggestResponse struct {
	Products []priceRunnerProduct `json:"products"`
}

type priceRunnerProduct struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	URL         string           `json:"url"`
	OutOfStock  bool             `json:"outOfStock"`
	LowestPrice priceRunnerMoney `json:"lowestPrice"`
	Image       priceRunnerImage `json:"image"`
}

type priceRunnerMoney struct {
	Amount   json.Number `json:"amount"`
	Currency string      `json:"currency"`
}

type priceRunnerImage struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

func (e *PriceRunnerUKExtractor) parseSuggestResponse(body string) ([]models.ProductComparison, error) {
	var resp priceRunnerSuggestResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("pricerunner_uk: failed to parse API response: %w", err)
	}

	category := models.CategoryOther
	results := make([]models.ProductComparison, 0, len(resp.Products))

	for _, p := range resp.Products {
		if p.OutOfStock {
			continue
		}
		name := strings.TrimSpace(p.Name)
		price, err := strconv.ParseFloat(string(p.LowestPrice.Amount), 64)
		if name == "" || err != nil || price <= 0 {
			continue
		}

		currency := strings.TrimSpace(p.LowestPrice.Currency)
		if currency == "" {
			currency = "GBP"
		}

		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    currency,
			StoreName:   "PriceRunner",
			Country:     string(models.CountryUK),
			Category:    &category,
		}

		if link := e.absoluteURL(p.URL); link != "" {
			comparison.StoreURL = &link
		}
		if img := e.imageURL(p.Image); img != "" {
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("PriceRunner UK extraction completed", utils.Int("results", len(results)))
	return results, nil
}

func (e *PriceRunnerUKExtractor) absoluteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return e.GetBaseURL() + raw
	}
	return e.GetBaseURL() + "/" + raw
}

func (e *PriceRunnerUKExtractor) imageURL(img priceRunnerImage) string {
	if strings.TrimSpace(img.URL) != "" {
		return e.absoluteURL(img.URL)
	}
	return e.absoluteURL(img.Path)
}
