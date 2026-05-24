# AGENTS.md — muambr-api Architecture Guide

This file provides AI coding agents with everything needed to implement code that follows the established patterns of this codebase. Read it fully before writing any code.

---

## Project Summary

A pure Go REST API for multi-country product price comparisons. Go handles all HTTP routing, web scraping, HTML parsing, and response processing. No external scripting layers.

---

## Directory Map

```
extractors/            # Extractor interface, registry, and base types
  extractor.go         # Extractor interface + ExtractorRegistry
  base_extractor.go    # BaseHTMLParser, BaseGoExtractor, BaseHTTPExtractor
  other/               # package other — all "general" category extractors
handlers/              # Gin HTTP handlers (comparison, link preview, admin)
linkparsers/           # Link preview HTML parsers — one file per site
localization/          # i18n JSON files (en.json, pt.json, es.json)
models/                # Domain models and enums (Country, MacroRegion, ProductComparison, ProductCategory)
routes/                # Route registration via Gin
tests/
  unit/                # Pure unit tests, no external calls
  integration/         # External-call tests, gated by INTEGRATION_TESTS=true
  mocks/               # Mock extractors and sample HTML fixtures
utils/                 # Shared utilities: logging, anti-bot, comparison processor
```

---

## Core Interfaces

### `Extractor` (extractors/extractor.go)

Every price extractor **must** implement this interface:

```go
type Extractor interface {
    GetCountryCode() models.Country
    GetMacroRegion() models.MacroRegion
    GetCategory() models.ProductCategory  // e.g. models.CategoryOther
    GetIdentifier() string
    BaseURL() string
    GetComparisons(productName string) ([]models.ProductComparison, error)
}
```

### `HTMLParser` (extractors/base_extractor.go)

Responsible for parsing product fields from raw HTML. Used by `BaseGoExtractor`:

```go
type HTMLParser interface {
    ParseProductName(html string) string
    ParsePrice(html string) (float64, string, error) // price, currency, error
    ParseURL(html string, baseURL string) string
    ParseStore(html string) string
    GetProductSelectors() []string
    GetNameSelectors() []string
    GetPriceSelectors() []string
    GetURLSelectors() []string
}
```

### `Parser` (linkparsers/parser.go)

For link preview parsing (not product comparisons):

```go
type Parser interface {
    ParseHTML(html string, pageURL *url.URL) *ParsedProductData
    ExtractTitle(html string, pageURL *url.URL) string
    ExtractPrice(html string, pageURL *url.URL) string
    ExtractImage(html string, pageURL *url.URL) string
    ExtractDescription(html string, pageURL *url.URL) string
    ExtractCurrency(html string, pageURL *url.URL) string
}
```

---

## Key Models (models/models.go)

```go
// Supported countries (ISO codes)
const (
    CountryBrazil   Country = "BR"
    CountryUS       Country = "US"
    CountryPortugal Country = "PT"
    CountrySpain    Country = "ES"
    CountryUK       Country = "GB"
    CountryGermany  Country = "DE"
)

// Macro regions
const (
    MacroRegionEU    MacroRegion = "EU"
    MacroRegionNA    MacroRegion = "NA"
    MacroRegionLATAM MacroRegion = "LATAM"
    MacroRegionNone  MacroRegion = "None"
)

// Product categories
type ProductCategory string
const (
    CategoryElectronics ProductCategory = "electronics"
    CategoryBeauty      ProductCategory = "beauty"
    CategoryAppliances  ProductCategory = "appliances"
    CategoryOther       ProductCategory = "other"
)
// Parse from a query string value (case-insensitive)
func ParseCategoryFromString(s string) (ProductCategory, error) { ... }

// The primary result type returned by all extractors
type ProductComparison struct {
    ID             string           `json:"id"`
    ProductName    string           `json:"productName"`
    Price          float64          `json:"price"`
    Currency       string           `json:"currency"`
    ConvertedPrice *ConvertedPrice  `json:"convertedPrice,omitempty"`
    StoreName      string           `json:"storeName"`
    StoreURL       *string          `json:"storeURL,omitempty"`
    Description    *string          `json:"description,omitempty"`
    Country        string           `json:"country"`
    Condition      *string          `json:"condition,omitempty"`
    ImageURL       *string          `json:"imageURL,omitempty"`
    LastUpdated    *string          `json:"lastUpdated,omitempty"`
    Category       *ProductCategory `json:"category,omitempty"`
}

// API response shape (sections grouped by country)
type CountrySection struct {
    Country      string              `json:"country"`
    CountryName  string              `json:"countryName"`
    Comparisons  []ProductComparison `json:"comparisons"`
    ResultsCount int                 `json:"resultsCount"`
}
```

To add a new country: add a constant to `models/models.go`, update all switch statements in `Country` methods (`GetCurrencyCode`, `GetMacroRegion`, `GetCountryName`, `GetLanguageCode`, `ParseCountryFromISO`), and add the country to the `GetCountriesInMacroRegion` slice.

---

## How to Add a New Price Extractor

Extractors live in **category-scoped sub-packages** under `extractors/`. All current extractors are in the `other` category and live in `extractors/other/` (`package other`). To add a new extractor in the `other` category, create a file there. For a new category (e.g. `electronics`), create `extractors/electronics/` with `package electronics`.

### Step 1 — Create the parser struct

```go
// extractors/other/mysite_extractor.go
package other

import "muambr-api/extractors"  // needed to reference BaseHTMLParser etc.

type MySiteParser struct {
    *extractors.BaseHTMLParser
}

func NewMySiteParser() *MySiteParser {
    return &MySiteParser{BaseHTMLParser: extractors.NewBaseHTMLParser("mysite")}
}

func (p *MySiteParser) GetProductSelectors() []string { return []string{`div.product-card`} }
func (p *MySiteParser) GetNameSelectors() []string    { return []string{`h2.product-title`} }
func (p *MySiteParser) GetPriceSelectors() []string   { return []string{`span.price`} }
func (p *MySiteParser) GetURLSelectors() []string     { return []string{`a.product-link[href]`} }

func (p *MySiteParser) ParseProductName(html string) string { /* regex or goquery */ }
func (p *MySiteParser) ParsePrice(html string) (float64, string, error) { /* return price, "EUR", nil */ }
func (p *MySiteParser) ParseURL(html string, baseURL string) string { /* return absolute URL */ }
func (p *MySiteParser) ParseStore(html string) string { return "MySite" }
```

### Step 2 — Create the extractor struct

```go
type MySiteExtractor struct {
    *extractors.BaseGoExtractor
}

func NewMySiteExtractor() *MySiteExtractor {
    parser := NewMySiteParser()
    base := extractors.NewBaseGoExtractor(
        "https://www.mysite.com",
        models.CountryPortugal, // or whichever country
        "mysite_v1",
        parser,
    )
    return &MySiteExtractor{BaseGoExtractor: base}
}

// GetCategory returns the category for registry filtering
func (e *MySiteExtractor) GetCategory() models.ProductCategory {
    return models.CategoryOther // change for electronics/beauty/appliances
}

// Override BuildSearchURL for the site's specific search URL pattern
func (e *MySiteExtractor) BuildSearchURL(productName string) (string, error) {
    q := url.QueryEscape(productName)
    return fmt.Sprintf("%s/search?q=%s", e.GetBaseURL(), q), nil
}

// Override GetComparisons only if the default BaseGoExtractor flow doesn't work
func (e *MySiteExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
    searchURL, err := e.BuildSearchURL(productName)
    if err != nil {
        return nil, err
    }
    html, err := e.FetchHTML(searchURL)
    if err != nil {
        return nil, err
    }
    return e.GetComparisonsFromHTML(html)
}
```

If the site provides JSON-LD structured data, implement `extractFromJSONLD` first and fall back to HTML parsing (see `mercadolivre_extractor_v2.go` for the pattern).

If the parser methods are unused (JSON-only extractor), use the noop pattern from `mercadolivre_extractor_v2.go`.

### Step 3 — Register the extractor

In `handlers/extractor_handler.go`, inside `initializeExtractors`. Import the package alias first:

```go
import other "muambr-api/extractors/other"

// inside initializeExtractors:
registry.RegisterExtractor(other.NewMySiteExtractor())
```

### Step 4 — Add tests

- Unit test in `tests/unit/extractors/mysite_test.go`
- Use a saved HTML fixture in `tests/mocks/extractors/sample_responses/`
- Integration test (optional) in `tests/integration/extractors/`

---

## How to Add a New Link Parser

Link parsers are used by the `/api/v1/linkpreview` endpoint.

### Step 1 — Create the parser

```go
// linkparsers/mysite_parser.go
package linkparsers

import "net/url"

type MySiteParser struct {
    ShareHTMLParser
}

func (p *MySiteParser) ExtractTitle(html string, pageURL *url.URL) string {
    // site-specific patterns first, then fallback to generic
    if title := extractWithRegex(`<h1[^>]*id="productTitle"[^>]*>([^<]+)</h1>`, html); title != "" {
        return title
    }
    return p.ShareHTMLParser.ExtractTitle(html, pageURL)
}

func (p *MySiteParser) ExtractPrice(html string, pageURL *url.URL) string {
    return extractWithRegex(`<span[^>]*class="price"[^>]*>([^<]+)</span>`, html)
}

// Implement remaining Parser interface methods, delegating to ShareHTMLParser as fallback
func (p *MySiteParser) ExtractImage(html string, pageURL *url.URL) string {
    return p.ShareHTMLParser.ExtractImage(html, pageURL)
}
func (p *MySiteParser) ExtractDescription(html string, pageURL *url.URL) string {
    return p.ShareHTMLParser.ExtractDescription(html, pageURL)
}
func (p *MySiteParser) ExtractCurrency(html string, pageURL *url.URL) string {
    return p.ShareHTMLParser.ExtractCurrency(html, pageURL)
}
func (p *MySiteParser) ParseHTML(html string, pageURL *url.URL) *ParsedProductData {
    return (&ShareHTMLParser{}).ParseHTML(html, pageURL) // override if needed
}
```

### Step 2 — Register in the parser registry

In `linkparsers/site_parsers.go`, add to `siteParserRegistry`:

```go
"mysite.com": func() Parser { return &MySiteParser{} },
```

The registry uses exact hostname matching (with `www.` stripped). The `createParser` function falls back to `ShareHTMLParser` when no match is found.

---

## Utilities Reference

### Logging (utils/logger.go)

```go
utils.Info("message", utils.String("key", value), utils.Int("count", n))
utils.Warn("message", utils.Error(err))
utils.Error("message", utils.Error(err))
utils.Debug("message")
```

Do **not** commit debug-only log lines. Remove them after troubleshooting.

### HTTP with anti-bot (utils/antibot.go)

Use `utils.MakeScrapingRequest(url, config)` or `BaseHTTPExtractor.FetchHTML(url)` — never raw `http.Get`.

### Localization (localization/localizer.go)

```go
localization.T("api.errors.unsupported_country_iso")
localization.TP("api.errors.unsupported_country_iso", map[string]string{"code": isoCode})
```

All user-facing strings must reference a key present in `localization/en.json` (and ideally `pt.json`, `es.json`).

### ComparisonProcessor (utils/comparison_processor.go)

Handles grouping by country, sorting by price, and filtering price outliers (>60% below average). Handlers call `processor.ProcessComparisons(comparisons, limit)` — extractors should not duplicate this logic.

---

## API Routes

```
GET /api/v1/comparisons/search
    ?name=<product>
    &baseCountry=<ISO>          # required
    &currentUserCountry=<ISO>   # optional
    &currency=<code>            # optional, defaults to baseCountry currency
    &limit=<int>                # optional, defaults to 10
    &useMacroRegion=<bool>      # optional, uses macro region of currentUserCountry
    &category=<value>           # optional: electronics | beauty | appliances | other

GET /api/v1/linkpreview
    ?url=<url>
    &baseCountry=<ISO>
    &addComparisons=<bool>

GET /rates/exchange-rates
    ?baseCurrency=<code>
```

---

## Extractor Selection Logic

1. Always include extractors for `baseCountry`.
2. If `currentUserCountry` is set and differs from `baseCountry`:
   - `useMacroRegion=true` → include all extractors whose macro region matches `currentUserCountry.GetMacroRegion()`
   - `useMacroRegion=false` (default) → include extractors for `currentUserCountry` only.
3. If `category` query param is provided, only extractors whose `GetCategory()` matches are included.
   - Pass `nil` to the registry methods to skip category filtering (get all categories).
   - Registry methods: `GetExtractorsForCountry(country, *models.ProductCategory)` and `GetExtractorsForMacroRegion(region, *models.ProductCategory)`.

---

## Testing Commands

```bash
make test              # Unit tests only
make test-all          # Unit + integration tests
make test-coverage     # HTML coverage report in coverage/

# Run integration tests directly
INTEGRATION_TESTS=true go test ./tests/integration/...
```

Integration tests are skipped unless `INTEGRATION_TESTS=true` is set. Never gate unit tests behind that flag.

---

## Conventions Checklist

- [ ] Extractor file in the correct category sub-package: `extractors/other/`, `extractors/electronics/`, etc.
- [ ] Package declaration matches the directory name (`package other`, `package electronics`)
- [ ] Import `"muambr-api/extractors"` to access `BaseHTMLParser`, `BaseGoExtractor`, etc.
- [ ] Use `extractors.BaseHTMLParser`, `extractors.NewBaseHTMLParser(...)`, `extractors.NewBaseGoExtractor(...)`
- [ ] Extractor file named `<sitename>_extractor[_v2].go`
- [ ] Parser struct named `<SiteName>Parser`, extractor struct named `<SiteName>Extractor`
- [ ] Constructor named `New<SiteName>Extractor()` / `New<SiteName>Parser()`
- [ ] `GetIdentifier()` returns a stable snake_case string (used for logging/registry)
- [ ] `GetCategory()` returns the appropriate `models.ProductCategory` constant
- [ ] All prices returned as `float64`; currency as ISO 4217 code (`"EUR"`, `"USD"`, `"BRL"`)
- [ ] `ProductComparison.Country` is set to the string value of the `models.Country` constant
- [ ] Extractor registered in `initializeExtractors` in `handlers/extractor_handler.go` via the category package alias
- [ ] Link parser registered in `siteParserRegistry` in `linkparsers/site_parsers.go`
- [ ] No raw `http.Get` — always go through anti-bot utilities
- [ ] No debug log lines committed
- [ ] New country constants added to **all** switch statements in `models/models.go`
