package other

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// zoomBRNoopParser satisfies HTMLParser. ZoomBRExtractor overrides GetComparisons.
type zoomBRNoopParser struct{ *extractors.BaseHTMLParser }

func (p *zoomBRNoopParser) GetProductSelectors() []string       { return nil }
func (p *zoomBRNoopParser) GetNameSelectors() []string          { return nil }
func (p *zoomBRNoopParser) GetPriceSelectors() []string         { return nil }
func (p *zoomBRNoopParser) GetURLSelectors() []string           { return nil }
func (p *zoomBRNoopParser) ParseProductName(html string) string { return "" }
func (p *zoomBRNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *zoomBRNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *zoomBRNoopParser) ParseStore(html string) string               { return "" }

const zoomMaxProductPages = 3

var zoomNextDataPattern = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

// ZoomBRExtractor extracts Brazil store offers from Zoom (zoom.com.br).
//
// Search HTML embeds catalog hits in __NEXT_DATA__. Each hit links to a product
// page whose __NEXT_DATA__ offerList is the store-by-store price summary.
type ZoomBRExtractor struct {
	*extractors.BaseGoExtractor
}

// NewZoomBRExtractor creates a Zoom Brazil extractor.
func NewZoomBRExtractor() *ZoomBRExtractor {
	parser := &zoomBRNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("zoom_br")}
	base := extractors.NewBaseGoExtractor(
		"https://www.zoom.com.br",
		models.CountryBrazil,
		"zoom_br_v1",
		parser,
	)
	return &ZoomBRExtractor{BaseGoExtractor: base}
}

// GetCategory returns the generic category used when the caller does not specify one.
func (e *ZoomBRExtractor) GetCategory() models.ProductCategory {
	return models.CategoryOther
}

// BuildSearchURL constructs the Zoom search page URL.
func (e *ZoomBRExtractor) BuildSearchURL(productName string) (string, error) {
	return e.GetBaseURL() + "/search?q=" + url.QueryEscape(productName), nil
}

// GetComparisons searches Zoom, opens the best matching product pages, and
// returns one comparison per store offer.
func (e *ZoomBRExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("zoom_br: failed to build search URL: %w", err)
	}
	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("zoom_br: failed to fetch search: %w", err)
	}
	hits, err := parseZoomHits(body)
	if err != nil {
		return nil, err
	}
	selected := selectZoomHits(hits, productName, zoomMaxProductPages)
	if len(selected) == 0 {
		return []models.ProductComparison{}, nil
	}

	pages := make([][]models.ProductComparison, len(selected))
	var wg sync.WaitGroup
	for i, hit := range selected {
		wg.Add(1)
		go func(i int, hit zoomHit) {
			defer wg.Done()
			pages[i] = e.comparisonsForHit(hit)
		}(i, hit)
	}
	wg.Wait()

	seen := map[string]struct{}{}
	var out []models.ProductComparison
	for _, page := range pages {
		for _, comparison := range page {
			if _, ok := seen[comparison.ID]; ok {
				continue
			}
			seen[comparison.ID] = struct{}{}
			out = append(out, comparison)
		}
	}
	return out, nil
}

func (e *ZoomBRExtractor) comparisonsForHit(hit zoomHit) []models.ProductComparison {
	pageURL := e.productURL(hit)
	if pageURL == "" {
		return e.hitFallback(hit)
	}
	html, err := e.FetchHTML(pageURL)
	if err != nil {
		utils.Warn("Zoom product page failed, using search price",
			utils.String("url", pageURL),
			utils.Error(err))
		return e.hitFallback(hit)
	}
	offers, err := parseZoomOffers(html)
	if err != nil || len(offers) == 0 {
		return e.hitFallback(hit)
	}
	return e.offersToComparisons(offers, hit.Image)
}

// ParseProductHTML extracts store offers from a Zoom product page.
func (e *ZoomBRExtractor) ParseProductHTML(html string) ([]models.ProductComparison, error) {
	offers, err := parseZoomOffers(html)
	if err != nil {
		return nil, err
	}
	return e.offersToComparisons(offers, ""), nil
}

func (e *ZoomBRExtractor) offersToComparisons(offers []zoomOffer, fallbackImage string) []models.ProductComparison {
	category := models.CategoryOther
	out := make([]models.ProductComparison, 0, len(offers))
	for _, offer := range offers {
		if offer.Price <= 0 || strings.TrimSpace(offer.Name) == "" {
			continue
		}
		store := strings.TrimSpace(offer.SellerName)
		if store == "" {
			store = "Zoom"
		}
		comparison := models.ProductComparison{
			ID:          offer.ID,
			ProductName: strings.TrimSpace(offer.Name),
			Price:       offer.Price,
			Currency:    "BRL",
			StoreName:   store,
			Country:     string(models.CountryBrazil),
			Category:    &category,
		}
		if offer.ID != "" {
			link := e.GetBaseURL() + "/lead?oid=" + url.QueryEscape(offer.ID) + "&channel=1"
			comparison.StoreURL = &link
		}
		image := offer.ImageURL
		if image == "" {
			image = fallbackImage
		}
		if image != "" {
			comparison.ImageURL = &image
		}
		out = append(out, comparison)
	}
	return out
}

func (e *ZoomBRExtractor) hitFallback(hit zoomHit) []models.ProductComparison {
	if hit.Price <= 0 || strings.TrimSpace(hit.Name) == "" {
		return nil
	}
	offer := zoomOffer{
		ID:         hit.BestOffer.ID,
		Name:       hit.Name,
		Price:      hit.Price,
		ImageURL:   hit.Image,
		SellerName: hit.BestOffer.MerchantName,
	}
	return e.offersToComparisons([]zoomOffer{offer}, hit.Image)
}

func (e *ZoomBRExtractor) productURL(hit zoomHit) string {
	path := strings.TrimSpace(hit.URL)
	if path == "" && hit.CategorySeoURL != "" && hit.SeoURL != "" {
		path = "/" + hit.CategorySeoURL + "/" + hit.SeoURL
	}
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return e.GetBaseURL() + path
}

type zoomNextData struct {
	Props struct {
		InitialReduxState struct {
			Hits struct {
				Hits []zoomHit `json:"hits"`
			} `json:"hits"`
			Offers struct {
				OfferList []zoomOffer `json:"offerList"`
			} `json:"offers"`
		} `json:"initialReduxState"`
	} `json:"props"`
}

type zoomHit struct {
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	URL            string  `json:"url"`
	SeoURL         string  `json:"seoUrl"`
	CategorySeoURL string  `json:"categorySeoUrl"`
	Image          string  `json:"image"`
	BestOffer      struct {
		ID           string `json:"id"`
		MerchantName string `json:"merchantName"`
	} `json:"bestOffer"`
}

type zoomOffer struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	ImageURL   string  `json:"imageUrl"`
	SellerName string  `json:"sellerName"`
}

func parseZoomHits(html string) ([]zoomHit, error) {
	state, err := parseZoomNextData(html)
	if err != nil {
		return nil, err
	}
	return state.Props.InitialReduxState.Hits.Hits, nil
}

func parseZoomOffers(html string) ([]zoomOffer, error) {
	state, err := parseZoomNextData(html)
	if err != nil {
		return nil, err
	}
	return state.Props.InitialReduxState.Offers.OfferList, nil
}

func parseZoomNextData(html string) (zoomNextData, error) {
	match := zoomNextDataPattern.FindStringSubmatch(html)
	if len(match) < 2 {
		return zoomNextData{}, fmt.Errorf("zoom_br: page has no product data")
	}
	var state zoomNextData
	if err := json.Unmarshal([]byte(match[1]), &state); err != nil {
		return zoomNextData{}, fmt.Errorf("zoom_br: failed to parse product data: %w", err)
	}
	return state, nil
}

func selectZoomHits(hits []zoomHit, query string, limit int) []zoomHit {
	eligible := make([]zoomHit, 0, len(hits))
	weak := make([]zoomHit, 0, len(hits))
	for _, hit := range hits {
		if strings.TrimSpace(hit.Name) == "" || hit.Price <= 0 {
			continue
		}
		if strings.TrimSpace(hit.URL) == "" && (hit.CategorySeoURL == "" || hit.SeoURL == "") {
			continue
		}
		if utils.MatchConfidence(query, hit.Name) >= utils.MatchConfidenceBestPriceMin {
			eligible = append(eligible, hit)
			continue
		}
		weak = append(weak, hit)
	}
	if len(eligible) == 0 {
		eligible = weak
	}
	if limit > len(eligible) {
		limit = len(eligible)
	}
	return eligible[:limit]
}
