package other

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// asdaUKNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// AsdaUKExtractor overrides GetComparisons entirely and uses the Algolia search
// JSON API, so the parser methods are never called.
type asdaUKNoopParser struct{ *extractors.BaseHTMLParser }

func (p *asdaUKNoopParser) GetProductSelectors() []string       { return nil }
func (p *asdaUKNoopParser) GetNameSelectors() []string          { return nil }
func (p *asdaUKNoopParser) GetPriceSelectors() []string         { return nil }
func (p *asdaUKNoopParser) GetURLSelectors() []string           { return nil }
func (p *asdaUKNoopParser) ParseProductName(html string) string { return "" }
func (p *asdaUKNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *asdaUKNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *asdaUKNoopParser) ParseStore(html string) string               { return "" }

const asdaAlgoliaAppID = "8I6WSKCCNV"
const asdaAlgoliaAPIKey = "03e4272048dd17f771da37b57ff8a75e"
const asdaAlgoliaIndex = "ASDA_PRODUCTS"

// AsdaUKExtractor extracts generic grocery products from Asda (UK).
//
// The storefront (asda.com / groceries.asda.com) is Cloudflare-blocked, so HTML
// search cannot be used. Results come from Algolia using the public search-only
// key from the storefront autocomplete.
//
// Endpoint: GET https://{appId}-dsn.algolia.net/1/indexes/ASDA_PRODUCTS?query={q}
type AsdaUKExtractor struct {
	*extractors.BaseGoExtractor
}

// NewAsdaUKExtractor creates a new Asda UK extractor.
func NewAsdaUKExtractor() *AsdaUKExtractor {
	parser := &asdaUKNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("asda_uk")}
	base := extractors.NewBaseGoExtractor(
		"https://groceries.asda.com",
		models.CountryUK,
		"asda_uk_v1",
		parser,
	)
	return &AsdaUKExtractor{BaseGoExtractor: base}
}

// GetCategory returns the generic/other category for registry filtering.
func (e *AsdaUKExtractor) GetCategory() models.ProductCategory {
	return models.CategoryOther
}

// BuildSearchURL constructs the Algolia index search URL for Asda UK.
func (e *AsdaUKExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("query", productName)
	params.Set("hitsPerPage", "20")
	return fmt.Sprintf(
		"https://%s-dsn.algolia.net/1/indexes/%s?%s",
		strings.ToLower(asdaAlgoliaAppID), asdaAlgoliaIndex, params.Encode(),
	), nil
}

// FetchHTML overrides the base client to send Algolia's required search headers.
func (e *AsdaUKExtractor) FetchHTML(targetURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("asda_uk: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Algolia-Application-Id", asdaAlgoliaAppID)
	req.Header.Set("X-Algolia-API-Key", asdaAlgoliaAPIKey)

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("asda_uk: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("asda_uk: failed to read response: %w", err)
	}
	return string(body), nil
}

// GetComparisons fetches products from Algolia and returns comparisons.
func (e *AsdaUKExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("asda_uk: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("asda_uk: failed to fetch API: %w", err)
	}

	return e.parseAlgoliaResponse(body)
}

type asdaAlgoliaResponse struct {
	Hits []asdaAlgoliaHit `json:"hits"`
}

type asdaAlgoliaHit struct {
	ObjectID string     `json:"objectID"`
	ID       string     `json:"ID"`
	CIN      string     `json:"CIN"`
	Name     string     `json:"NAME"`
	Status   string     `json:"STATUS"`
	ImageID  string     `json:"IMAGE_ID"`
	Prices   asdaPrices `json:"PRICES"`
}

type asdaPrices struct {
	EN asdaPriceEN `json:"EN"`
}

type asdaPriceEN struct {
	Price float64 `json:"PRICE"`
}

func (e *AsdaUKExtractor) parseAlgoliaResponse(body string) ([]models.ProductComparison, error) {
	var resp asdaAlgoliaResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("asda_uk: failed to parse API response: %w", err)
	}

	category := models.CategoryOther
	results := make([]models.ProductComparison, 0, len(resp.Hits))

	for _, h := range resp.Hits {
		if h.Status != "" && !strings.EqualFold(h.Status, "A") {
			continue
		}
		name := strings.TrimSpace(h.Name)
		price := h.Prices.EN.Price
		if name == "" || price <= 0 {
			continue
		}

		id := strings.TrimSpace(h.CIN)
		if id == "" {
			id = strings.TrimSpace(h.ObjectID)
		}
		if id == "" {
			id = strings.TrimSpace(h.ID)
		}
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "GBP",
			StoreName:   "Asda",
			Country:     string(models.CountryUK),
			Category:    &category,
		}

		link := e.GetBaseURL() + "/product/" + id
		comparison.StoreURL = &link
		if h.ImageID != "" {
			img := "https://ui.assets-asda.com/dm/asda-groceries/" + h.ImageID
			comparison.ImageURL = &img
		}

		results = append(results, comparison)
	}

	utils.Info("Asda UK extraction completed", utils.Int("results", len(results)))
	return results, nil
}
