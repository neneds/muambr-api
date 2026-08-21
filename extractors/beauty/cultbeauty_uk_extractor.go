package beauty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// cultBeautyUKNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// CultBeautyUKExtractor overrides GetComparisons entirely and uses the storefront
// GraphQL InstantSearch API, so the parser methods are never called.
type cultBeautyUKNoopParser struct{ *extractors.BaseHTMLParser }

func (p *cultBeautyUKNoopParser) GetProductSelectors() []string       { return nil }
func (p *cultBeautyUKNoopParser) GetNameSelectors() []string          { return nil }
func (p *cultBeautyUKNoopParser) GetPriceSelectors() []string         { return nil }
func (p *cultBeautyUKNoopParser) GetURLSelectors() []string           { return nil }
func (p *cultBeautyUKNoopParser) ParseProductName(html string) string { return "" }
func (p *cultBeautyUKNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *cultBeautyUKNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *cultBeautyUKNoopParser) ParseStore(html string) string               { return "" }

const cultBeautyGraphQLURL = "https://www.cultbeauty.co.uk/api/graphql?operationName=InstantSearchQuery"
const cultBeautyOrigin = "https://www.cultbeauty.co.uk"

// CultBeautyUKExtractor extracts beauty products from Cult Beauty (UK).
//
// Storefront search HTML 302s into a category PLP of client-rendered skeletons.
// Results come from the public InstantSearch GraphQL operation used by the
// search bar. Origin is required (403 without it) — a CORS-style gate, not a WAF.
// Instant search returns a short autocomplete-sized list, not a full PLP.
//
// Endpoint: POST /api/graphql?operationName=InstantSearchQuery
type CultBeautyUKExtractor struct {
	*extractors.BaseGoExtractor
}

// NewCultBeautyUKExtractor creates a new Cult Beauty UK extractor.
func NewCultBeautyUKExtractor() *CultBeautyUKExtractor {
	parser := &cultBeautyUKNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("cultbeauty_uk")}
	base := extractors.NewBaseGoExtractor(
		cultBeautyOrigin,
		models.CountryUK,
		"cultbeauty_uk_v1",
		parser,
	)
	return &CultBeautyUKExtractor{BaseGoExtractor: base}
}

// GetCategory returns the beauty category for registry filtering.
func (e *CultBeautyUKExtractor) GetCategory() models.ProductCategory {
	return models.CategoryBeauty
}

// BuildSearchURL returns the GraphQL InstantSearch endpoint.
func (e *CultBeautyUKExtractor) BuildSearchURL(productName string) (string, error) {
	return cultBeautyGraphQLURL, nil
}

type cultBeautySearchRequest struct {
	Operation string                    `json:"operation"`
	Variables cultBeautySearchVariables `json:"variables"`
}

type cultBeautySearchVariables struct {
	Query               string `json:"query"`
	Currency            string `json:"currency"`
	ShippingDestination string `json:"shippingDestination"`
	ReviewsEnabled      bool   `json:"reviewsEnabled"`
}

// GetComparisons POSTs InstantSearchQuery and maps hits to comparisons.
func (e *CultBeautyUKExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	payload, err := json.Marshal(cultBeautySearchRequest{
		Operation: "InstantSearchQuery",
		Variables: cultBeautySearchVariables{
			Query:               productName,
			Currency:            "GBP",
			ShippingDestination: "GB",
			ReviewsEnabled:      true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cultbeauty_uk: failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cultBeautyGraphQLURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("cultbeauty_uk: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", cultBeautyOrigin)
	req.Header.Set("Referer", cultBeautyOrigin+"/")

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cultbeauty_uk: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cultbeauty_uk: HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cultbeauty_uk: failed to read response: %w", err)
	}

	return e.parseInstantSearchResponse(string(body))
}

type cultBeautySearchResponse struct {
	Data cultBeautySearchData `json:"data"`
}

type cultBeautySearchData struct {
	InstantSearch cultBeautyInstantSearch `json:"instantSearch"`
}

type cultBeautyInstantSearch struct {
	Products []cultBeautyProduct `json:"products"`
}

type cultBeautyProduct struct {
	Title           string            `json:"title"`
	URL             string            `json:"url"`
	SKU             json.Number       `json:"sku"`
	Images          []cultBeautyImage `json:"images"`
	DefaultVariant  cultBeautyVariant `json:"defaultVariant"`
	CheapestVariant cultBeautyVariant `json:"cheapestVariant"`
}

type cultBeautyImage struct {
	Original string `json:"original"`
}

type cultBeautyVariant struct {
	Price cultBeautyVariantPrice `json:"price"`
}

type cultBeautyVariantPrice struct {
	Price cultBeautyMoney `json:"price"`
}

type cultBeautyMoney struct {
	Currency string      `json:"currency"`
	Amount   json.Number `json:"amount"`
}

func (e *CultBeautyUKExtractor) parseInstantSearchResponse(body string) ([]models.ProductComparison, error) {
	var resp cultBeautySearchResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("cultbeauty_uk: failed to parse API response: %w", err)
	}

	category := models.CategoryBeauty
	products := resp.Data.InstantSearch.Products
	results := make([]models.ProductComparison, 0, len(products))

	for _, p := range products {
		name := strings.TrimSpace(p.Title)
		price, currency := cultBeautyOffer(p)
		if name == "" || price <= 0 {
			continue
		}
		if currency == "" {
			currency = "GBP"
		}

		id := strings.TrimSpace(p.SKU.String())
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    currency,
			StoreName:   "Cult Beauty",
			Country:     string(models.CountryUK),
			Category:    &category,
		}

		if p.URL != "" {
			link := p.URL
			if !strings.HasPrefix(link, "http") {
				link = e.GetBaseURL() + link
			}
			comparison.StoreURL = &link
		}
		if len(p.Images) > 0 && p.Images[0].Original != "" {
			img := p.Images[0].Original
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Cult Beauty UK extraction completed", utils.Int("results", len(results)))
	return results, nil
}

func cultBeautyOffer(p cultBeautyProduct) (float64, string) {
	for _, variant := range []cultBeautyVariant{p.CheapestVariant, p.DefaultVariant} {
		amount, err := strconv.ParseFloat(variant.Price.Price.Amount.String(), 64)
		if err != nil || amount <= 0 {
			continue
		}
		currency := strings.TrimSpace(variant.Price.Price.Currency)
		return amount, currency
	}
	return 0, ""
}
