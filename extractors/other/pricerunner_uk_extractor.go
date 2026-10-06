package other

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
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

const (
	priceRunnerUKSuggestPath = "/uk/api/instant-search-edge-rest/public/search/suggest/UK"
	priceRunnerImageHost     = "https://owp.klarna.com"
)

var priceRunnerStateScript = regexp.MustCompile(`(?s)<script[^>]*type="application/json"[^>]*>(.*?)</script>`)

// PriceRunnerUKExtractor extracts generic UK offers from PriceRunner.
//
// Suggest (/uk/api/.../search/suggest/UK) returns a handful of autocomplete
// products plus FEATURE links such as /cl/1/Mobile-Phones?attr_.... The category
// page embeds the real listing in __DEHYDRATED_QUERY_STATE__ (cl-list-data-query).
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

// GetComparisons fetches suggest results, then the best matching category listing.
func (e *PriceRunnerUKExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("pricerunner_uk: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("pricerunner_uk: failed to fetch API: %w", err)
	}

	suggestProducts, featureURL, err := e.parseSuggestResponse(body, productName)
	if err != nil {
		return nil, err
	}

	if featureURL != "" {
		listingURL := e.absoluteURL(featureURL)
		html, fetchErr := e.FetchHTML(listingURL)
		if fetchErr != nil {
			utils.Warn("PriceRunner category listing failed, using suggest products",
				utils.String("url", listingURL),
				utils.Error(fetchErr))
		} else if listing, parseErr := e.parseCategoryListing(html); parseErr != nil {
			utils.Warn("PriceRunner category listing parse failed, using suggest products",
				utils.String("url", listingURL),
				utils.Error(parseErr))
		} else if len(listing) > 0 {
			utils.Info("PriceRunner UK extraction completed",
				utils.Int("results", len(listing)),
				utils.String("source", "category"))
			return listing, nil
		}
	}

	utils.Info("PriceRunner UK extraction completed",
		utils.Int("results", len(suggestProducts)),
		utils.String("source", "suggest"))
	return suggestProducts, nil
}

// GetComparisonsFromHTML parses either a suggest JSON body or a category page.
func (e *PriceRunnerUKExtractor) GetComparisonsFromHTML(body string) ([]models.ProductComparison, error) {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") {
		products, _, err := e.parseSuggestResponse(trimmed, "")
		return products, err
	}
	return e.parseCategoryListing(body)
}

type priceRunnerSuggestResponse struct {
	Products    []priceRunnerProduct    `json:"products"`
	Suggestions []priceRunnerSuggestion `json:"suggestions"`
}

type priceRunnerSuggestion struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
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

func (e *PriceRunnerUKExtractor) parseSuggestResponse(body, query string) ([]models.ProductComparison, string, error) {
	var resp priceRunnerSuggestResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, "", fmt.Errorf("pricerunner_uk: failed to parse API response: %w", err)
	}

	results := make([]models.ProductComparison, 0, len(resp.Products))
	for _, p := range resp.Products {
		if c, ok := e.productToComparison(p); ok {
			results = append(results, c)
		}
	}
	return results, pickFeatureURL(query, resp.Suggestions), nil
}

func pickFeatureURL(query string, suggestions []priceRunnerSuggestion) string {
	bestURL := ""
	bestScore := 0.0
	for _, s := range suggestions {
		if !strings.EqualFold(s.Type, "FEATURE") || !strings.Contains(s.URL, "/cl/") {
			continue
		}
		score := utils.MatchConfidence(query, s.Name)
		if score > bestScore {
			bestScore = score
			bestURL = s.URL
		}
	}
	return bestURL
}

type priceRunnerPageState struct {
	Dehydrated struct {
		Queries []priceRunnerQuery `json:"queries"`
	} `json:"__DEHYDRATED_QUERY_STATE__"`
}

type priceRunnerQuery struct {
	QueryKey []json.RawMessage `json:"queryKey"`
	State    struct {
		Data json.RawMessage `json:"data"`
	} `json:"state"`
}

type priceRunnerListData struct {
	Pages []struct {
		Products []priceRunnerProduct `json:"products"`
	} `json:"pages"`
}

func (e *PriceRunnerUKExtractor) parseCategoryListing(html string) ([]models.ProductComparison, error) {
	matches := priceRunnerStateScript.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("pricerunner_uk: category page has no JSON state")
	}

	var products []priceRunnerProduct
	for _, match := range matches {
		if !strings.Contains(match[1], "cl-list-data-query") {
			continue
		}
		var state priceRunnerPageState
		if err := json.Unmarshal([]byte(match[1]), &state); err != nil {
			return nil, fmt.Errorf("pricerunner_uk: failed to parse category state: %w", err)
		}
		for _, q := range state.Dehydrated.Queries {
			if len(q.QueryKey) == 0 || string(q.QueryKey[0]) != `"cl-list-data-query"` {
				continue
			}
			var listing priceRunnerListData
			if err := json.Unmarshal(q.State.Data, &listing); err != nil {
				return nil, fmt.Errorf("pricerunner_uk: failed to parse category products: %w", err)
			}
			for _, page := range listing.Pages {
				products = append(products, page.Products...)
			}
		}
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("pricerunner_uk: category listing has no products")
	}

	results := make([]models.ProductComparison, 0, len(products))
	for _, p := range products {
		if c, ok := e.productToComparison(p); ok {
			results = append(results, c)
		}
	}
	return results, nil
}

func (e *PriceRunnerUKExtractor) productToComparison(p priceRunnerProduct) (models.ProductComparison, bool) {
	if p.OutOfStock {
		return models.ProductComparison{}, false
	}
	name := strings.TrimSpace(p.Name)
	price, err := strconv.ParseFloat(string(p.LowestPrice.Amount), 64)
	if name == "" || err != nil || price <= 0 {
		return models.ProductComparison{}, false
	}
	currency := strings.TrimSpace(p.LowestPrice.Currency)
	if currency == "" {
		currency = "GBP"
	}
	id := strings.TrimSpace(p.ID)
	if id == "" {
		id = utils.GenerateUUID()
	}
	category := models.CategoryOther
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
	return comparison, true
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
	path := strings.TrimSpace(img.Path)
	if strings.HasPrefix(path, "/product/") {
		return priceRunnerImageHost + path
	}
	return e.absoluteURL(path)
}
