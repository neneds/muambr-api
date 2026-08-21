package other

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// ebuyerUKNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// EbuyerUKExtractor overrides GetComparisons entirely and uses the Algolia search
// JSON API, so the parser methods are never called.
type ebuyerUKNoopParser struct{ *extractors.BaseHTMLParser }

func (p *ebuyerUKNoopParser) GetProductSelectors() []string       { return nil }
func (p *ebuyerUKNoopParser) GetNameSelectors() []string          { return nil }
func (p *ebuyerUKNoopParser) GetPriceSelectors() []string         { return nil }
func (p *ebuyerUKNoopParser) GetURLSelectors() []string           { return nil }
func (p *ebuyerUKNoopParser) ParseProductName(html string) string { return "" }
func (p *ebuyerUKNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *ebuyerUKNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *ebuyerUKNoopParser) ParseStore(html string) string               { return "" }

const ebuyerAlgoliaAppID = "FR8OHZX2IA"
const ebuyerAlgoliaAPIKey = "3073f8b59d4cbce67a141245e4471da5"
const ebuyerAlgoliaIndex = "ebuy_production_search"

// EbuyerUKExtractor extracts generic retail products from Ebuyer (UK).
//
// Search HTML is a client-rendered shell (empty product cards) and the
// storefront is Akamai-blocked from a plain client. Results come from Algolia
// using the public search-only key embedded in the search page.
//
// Endpoint: GET https://{appId}-dsn.algolia.net/1/indexes/ebuy_production_search?query={q}
type EbuyerUKExtractor struct {
	*extractors.BaseGoExtractor
}

// NewEbuyerUKExtractor creates a new Ebuyer UK extractor.
func NewEbuyerUKExtractor() *EbuyerUKExtractor {
	parser := &ebuyerUKNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("ebuyer_uk")}
	base := extractors.NewBaseGoExtractor(
		"https://www.ebuyer.com",
		models.CountryUK,
		"ebuyer_uk_v1",
		parser,
	)
	return &EbuyerUKExtractor{BaseGoExtractor: base}
}

// GetCategory returns the generic/other category for registry filtering.
func (e *EbuyerUKExtractor) GetCategory() models.ProductCategory {
	return models.CategoryOther
}

// BuildSearchURL constructs the Algolia index search URL for Ebuyer UK.
func (e *EbuyerUKExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("query", productName)
	params.Set("hitsPerPage", "20")
	return fmt.Sprintf(
		"https://%s-dsn.algolia.net/1/indexes/%s?%s",
		strings.ToLower(ebuyerAlgoliaAppID), ebuyerAlgoliaIndex, params.Encode(),
	), nil
}

// FetchHTML overrides the base client to send Algolia's required search headers.
func (e *EbuyerUKExtractor) FetchHTML(targetURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("ebuyer_uk: failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Algolia-Application-Id", ebuyerAlgoliaAppID)
	req.Header.Set("X-Algolia-API-Key", ebuyerAlgoliaAPIKey)

	resp, err := utils.CreateAntiBotClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("ebuyer_uk: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ebuyer_uk: failed to read response: %w", err)
	}
	return string(body), nil
}

// GetComparisons fetches products from Algolia and returns comparisons.
func (e *EbuyerUKExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("ebuyer_uk: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("ebuyer_uk: failed to fetch API: %w", err)
	}

	return e.parseAlgoliaResponse(body)
}

type ebuyerAlgoliaResponse struct {
	Hits []ebuyerAlgoliaHit `json:"hits"`
}

type ebuyerAlgoliaHit struct {
	ObjectID        string  `json:"objectID"`
	Name            string  `json:"name"`
	SellingPrice    float64 `json:"sellingPrice"`
	HidePrice       bool    `json:"hidePrice"`
	IsHidden        bool    `json:"isHidden"`
	HideFromSearch  bool    `json:"hideFromSearch"`
	ColourVariantID string  `json:"colourVariantID"`
	ProductID       string  `json:"productId"`
}

func (e *EbuyerUKExtractor) parseAlgoliaResponse(body string) ([]models.ProductComparison, error) {
	var resp ebuyerAlgoliaResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("ebuyer_uk: failed to parse API response: %w", err)
	}

	category := models.CategoryOther
	results := make([]models.ProductComparison, 0, len(resp.Hits))

	for _, h := range resp.Hits {
		if h.HidePrice || h.IsHidden || h.HideFromSearch {
			continue
		}
		name := strings.TrimSpace(h.Name)
		if name == "" || h.SellingPrice <= 0 {
			continue
		}

		id := strings.TrimSpace(h.ColourVariantID)
		if id == "" {
			id = strings.TrimPrefix(strings.TrimSpace(h.ObjectID), "ebuy_")
		}
		if id == "" {
			id = strings.TrimSpace(h.ProductID)
		}
		if id == "" {
			id = utils.GenerateUUID()
		}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       h.SellingPrice,
			Currency:    "GBP",
			StoreName:   "Ebuyer",
			Country:     string(models.CountryUK),
			Category:    &category,
		}

		link := e.GetBaseURL() + "/" + id
		if slug := ebuyerSlug(name); slug != "" {
			link += "-" + slug
		}
		comparison.StoreURL = &link

		results = append(results, comparison)
	}

	utils.Info("Ebuyer UK extraction completed", utils.Int("results", len(results)))
	return results, nil
}

func ebuyerSlug(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
