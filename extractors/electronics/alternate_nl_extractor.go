package electronics

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// alternateNLNoopParser satisfies the HTMLParser interface required by BaseGoExtractor.
// AlternateNLExtractor overrides GetComparisonsFromHTML, so these methods are never called.
type alternateNLNoopParser struct{ *extractors.BaseHTMLParser }

func (p *alternateNLNoopParser) GetProductSelectors() []string       { return nil }
func (p *alternateNLNoopParser) GetNameSelectors() []string          { return nil }
func (p *alternateNLNoopParser) GetPriceSelectors() []string         { return nil }
func (p *alternateNLNoopParser) GetURLSelectors() []string           { return nil }
func (p *alternateNLNoopParser) ParseProductName(html string) string { return "" }
func (p *alternateNLNoopParser) ParsePrice(html string) (float64, string, error) {
	return 0, "", fmt.Errorf("not implemented")
}
func (p *alternateNLNoopParser) ParseURL(html string, baseURL string) string { return "" }
func (p *alternateNLNoopParser) ParseStore(html string) string               { return "" }

// alternateProductBoxRe matches a listing card: absolute href on the productBox
// anchor, then product-name and euro price inside the same card.
var alternateProductBoxStartRe = regexp.MustCompile(`<a href="(https://www\.alternate\.nl/[^"]+)" class="[^"]*productBox[^"]*"`)
var alternateNameRe = regexp.MustCompile(`product-name font-weight-bold">([\s\S]*?)</div>`)
var alternatePriceRe = regexp.MustCompile(`class="price\s*">€\s*([\d.,]+)`)

var alternateHTMLTagRe = regexp.MustCompile(`<[^>]+>`)
var alternateSpaceRe = regexp.MustCompile(`\s+`)

// AlternateNLExtractor extracts electronics products from Alternate (Netherlands).
//
// Search results are server-rendered on listing.xhtml. The JSF ajax POST is
// filter-only and needs a ViewState, so it is not used.
//
// Endpoint: GET https://www.alternate.nl/listing.xhtml?q={query}
type AlternateNLExtractor struct {
	*extractors.BaseGoExtractor
}

// NewAlternateNLExtractor creates a new Alternate Netherlands extractor.
func NewAlternateNLExtractor() *AlternateNLExtractor {
	parser := &alternateNLNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("alternate_nl")}
	base := extractors.NewBaseGoExtractor(
		"https://www.alternate.nl",
		models.CountryNetherlands,
		"alternate_nl_v1",
		parser,
	)
	return &AlternateNLExtractor{BaseGoExtractor: base}
}

// GetCategory returns the electronics category for registry filtering.
func (e *AlternateNLExtractor) GetCategory() models.ProductCategory {
	return models.CategoryElectronics
}

// BuildSearchURL constructs the Alternate listing search URL.
func (e *AlternateNLExtractor) BuildSearchURL(productName string) (string, error) {
	params := url.Values{}
	params.Set("q", productName)
	return e.GetBaseURL() + "/listing.xhtml?" + params.Encode(), nil
}

// GetComparisons fetches the listing page and extracts product cards.
func (e *AlternateNLExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	searchURL, err := e.BuildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("alternate_nl: failed to build search URL: %w", err)
	}

	body, err := e.FetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("alternate_nl: failed to fetch listing: %w", err)
	}

	return e.GetComparisonsFromHTML(body)
}

// GetComparisonsFromHTML parses productBox cards from Alternate listing HTML.
func (e *AlternateNLExtractor) GetComparisonsFromHTML(body string) ([]models.ProductComparison, error) {
	starts := alternateProductBoxStartRe.FindAllStringSubmatchIndex(body, -1)
	category := models.CategoryElectronics
	results := make([]models.ProductComparison, 0, len(starts))
	seen := make(map[string]struct{})

	for i, loc := range starts {
		if len(loc) < 4 {
			continue
		}
		end := len(body)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		chunk := body[loc[0]:end]
		link := strings.TrimSpace(body[loc[2]:loc[3]])

		nameMatch := alternateNameRe.FindStringSubmatch(chunk)
		priceMatch := alternatePriceRe.FindStringSubmatch(chunk)
		if nameMatch == nil || priceMatch == nil {
			continue
		}
		name := strings.TrimSpace(alternateSpaceRe.ReplaceAllString(alternateHTMLTagRe.ReplaceAllString(nameMatch[1], " "), " "))
		price, _, err := e.ParsePriceText("€"+priceMatch[1], "EUR")
		if name == "" || err != nil || price <= 0 {
			continue
		}

		id := alternateProductID(link)
		if id == "" {
			id = utils.GenerateUUID()
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		comparison := models.ProductComparison{
			ID:          id,
			ProductName: name,
			Price:       price,
			Currency:    "EUR",
			StoreName:   "Alternate",
			Country:     string(models.CountryNetherlands),
			Category:    &category,
		}
		if link != "" {
			comparison.StoreURL = &link
		}
		results = append(results, comparison)
	}

	utils.Info("Alternate NL extraction completed", utils.Int("results", len(results)))
	return results, nil
}

func alternateProductID(link string) string {
	// https://www.alternate.nl/.../html/product/100156407
	if i := strings.LastIndex(link, "/product/"); i >= 0 {
		return strings.Trim(link[i+len("/product/"):], "/")
	}
	return ""
}
