# AGENTS.md — muambr-api Architecture Guide

This file provides AI coding agents with everything needed to implement code that follows the established patterns of this codebase. Read it fully before writing any code.

---

## Project Summary

A pure Go REST API for multi-country **retail** product price comparisons. Go handles HTTP routing, scraping/JSON fetch, HTML parsing, FX conversion, savings, and deal scoring. No external scripting layers.

The mobile client’s primary contract is **`POST /api/v1/product-comparisons`**. There is no legacy GET search.

---

## Design Principles (SOLID, DRY, KISS)

Write the **smallest reusable change**. Prefer extending existing types over new layers.

### KISS — keep it small

- Solve the current use case. Do not add flags, caches, or abstractions “for later”.
- Prefer a public JSON/search API over HTML scraping. Prefer HTML+JSON-LD over brittle CSS.
- If a site is WAF-blocked, **stop**. Do not add retries, cookie jars, or browser automation.
- **Feasibility first:** when a new store, HAR, or hostname is proposed for an extractor or link parser, probe the live search/API (or product page) and report the verdict **before writing any implementation**. Homepage HARs and image-only captures are not enough.
- One file per extractor. One constructor. Override only what `BaseGoExtractor` cannot do.
- JSON-only extractors use the existing **noop parser** pattern. Do not invent a second extractor base until several JSON extractors share non-trivial fetch/parse logic.

### DRY — do not duplicate

- HTTP: `BaseHTTPExtractor.FetchHTML` / `utils.MakeScrapingRequest` — never `http.Get`.
- Grouping, sort, outlier filter: `ComparisonProcessor`.
- Savings, deal score, match confidence, `normalized*` fields: `ComparisonEngine`.
- FX: `ExchangeRateService`. Extractors return **store-native** `price` + ISO currency only.
- Shared parse helpers live in `utils/` or the relevant base type — copy-paste between extractors is a smell.

### SOLID — in this repo

| Principle | Practice |
|-----------|----------|
| **S**ingle responsibility | Extractors fetch + map offers. Handlers validate HTTP. `ComparisonEngine` builds the decision payload. Do not compute savings inside an extractor. |
| **O**pen/closed | New stores = new extractor + registry line. Do not special-case a site in the handler or engine. |
| **L**iskov | Every extractor is interchangeable via `Extractor`. Same `GetComparisons` contract, same `ProductComparison` fields. |
| **I**nterface segregation | `Extractor`, `HTMLParser`, and link `Parser` are separate. JSON extractors still embed `BaseGoExtractor` with a noop `HTMLParser` rather than forcing unused HTML methods onto a new mega-interface. |
| **D**ependency inversion | Registry and handlers depend on `Extractor`, not concrete types. Register in `initializeExtractors` only. |

### Lean reuse checklist

Before adding a type, package, or helper: is there already a base extractor, processor, or util that does this? If the new helper would be used once, inline it.

---

## Directory Map

```
extractors/                 # Extractor interface, registry, base types
  extractor.go
  base_extractor.go         # BaseHTMLParser, BaseGoExtractor, BaseHTTPExtractor
  other/                    # package other — generic / CategoryOther
  beauty/                   # package beauty
  appliances/               # package appliances
handlers/                   # product comparison, link preview, FX
linkparsers/                # link preview HTML parsers — one file per site
localization/               # en.json, pt.json, es.json
models/                     # Country, categories, ProductComparison, comparison result
routes/                     # Gin route registration
tests/
  unit/
  integration/              # gated by INTEGRATION_TESTS=true
  mocks/
utils/                      # logging, anti-bot, FX, ComparisonProcessor, ComparisonEngine
```

---

## Current API

| Method | Path | Role |
|--------|------|------|
| POST | `/api/v1/product-comparisons` | **Primary** comparison (manual, URL, camera, barcode) |
| GET | `/api/v1/linkpreview` | Parse a product URL; `addComparisons=true` also runs comparison |
| GET | `/rates/exchange-rates` | FX table (`updated_at`, `source`) |
| GET | `/health` | Liveness |

### POST `/api/v1/product-comparisons`

Request (JSON):

```
product.name                 # required unless productURL yields a title
product.brand / model        # optional
product.category             # electronics | beauty | appliances | fashion | other
observedPrice.amount         # recommended — price the user found
observedPrice.currency
currentCountry               # ISO; defaults to baseCountry
baseCountry                  # required ISO
currency                     # normalization target; defaults to baseCountry currency
limit / useMacroRegion
source.type                  # manual | url | camera | barcode
productURL                   # optional — extracts title/price/country
```

Response: `models.ProductComparisonResult` — `comparisonId`, `status`, `observed`, `prices`, `sections`, `bestCurrentCountryPrice`, `bestBaseCountryPrice`, `savings`, `dealScore`, `exchangeRate`, `metadata`, `capturedAt`, `expiresAt`.

`status`: `complete` | `partial` | `empty` | `low_confidence`.

Error `code` values: `INVALID_REQUEST`, `COUNTRY_UNKNOWN`, `CURRENCY_UNKNOWN`, `PRICE_NOT_FOUND`, `PRODUCT_NOT_FOUND`, `NO_COMPARISON_SOURCES`, `EXCHANGE_RATE_UNAVAILABLE`, `INTERNAL_ERROR` (plus reserved timeout codes).

V1 is **retail price only** (`metadata.priceType: "retail"`). No landed cost / tax.

### Price normalization (mobile contract)

On `prices[]`, `observed`, `bestCurrentCountryPrice`, `bestBaseCountryPrice`:

| Native `currency` vs request `currency` | `normalizedAmount` / `normalizedCurrency` |
|-----------------------------------------|-------------------------------------------|
| Same | **omit** (do not copy amount into normalized fields) |
| Different | set to FX conversion into request currency |

Savings/deal score use `MoneyAmount.ComparableAmount()` (normalized if present, else native amount).

---

## Core Interfaces

### `Extractor` (`extractors/extractor.go`)

```go
type Extractor interface {
    GetCountryCode() models.Country
    GetMacroRegion() models.MacroRegion
    GetCategory() models.ProductCategory
    GetIdentifier() string
    BaseURL() string
    GetComparisons(productName string) ([]models.ProductComparison, error)
}
```

### `HTMLParser` (`extractors/base_extractor.go`)

Used by `BaseGoExtractor`. JSON-only extractors implement a noop parser.

```go
type HTMLParser interface {
    ParseProductName(html string) string
    ParsePrice(html string) (float64, string, error)
    ParseURL(html string, baseURL string) string
    ParseStore(html string) string
    GetProductSelectors() []string
    GetNameSelectors() []string
    GetPriceSelectors() []string
    GetURLSelectors() []string
}
```

### `Parser` (`linkparsers/parser.go`)

For `/api/v1/linkpreview`, not store search.

---

## Key Models

Countries: `BR`, `US`, `PT`, `ES`, `GB`, `DE`.  
Macro regions: `EU`, `NA`, `LATAM`, `None`.  
Categories: `electronics`, `beauty`, `appliances`, `fashion`, `other`.

`ProductComparison` is the **extractor output** (store-native offer). The HTTP envelope is `ProductComparisonResult` in `models/comparison_result.go`.

To add a country: constant in `models/models.go`, every `Country` switch (`GetCurrencyCode`, `GetMacroRegion`, `GetCountryName`, `GetLanguageCode`, `ParseCountryFromISO`), and `GetCountriesInMacroRegion`.

---

## Registered Extractors

| Identifier | Package | Country | Category | Source | Notes |
|------------|---------|---------|----------|--------|--------|
| `acharpromo_v2` | other | BR | other | SSE/JSON | Working |
| `kuantokusta_v2` | other | PT | other | HTML/JSON | Working |
| `walmart_usa` | other | US | other | HTML | **WAF** (PerimeterX `/blocked`) — often empty |
| `epocacosmeticos_v1` | beauty | BR | beauty | VTEX JSON | Working (HTTP 206 is valid) |
| `sephora_br_v1` | beauty | BR | beauty | SFCC AJAX JSON | Working |
| `primor_pt_v1` | beauty | PT | beauty | Empathy JSON | Working — do **not** scrape `pt.primor.eu` HTML (AWS WAF) |
| `perfumes_e_companhia_pt_v1` | beauty | PT | beauty | Doofinder JSON | Working — Origin required; do **not** scrape `/pt/pesquisa/` HTML (client-rendered) |
| `carrefour_br_v1` | appliances | BR | appliances | VTEX JSON | Working |

No `electronics/` or `fashion/` extractors yet. No Spain store extractor (Amazon.es is WAF-blocked). `amazon.es` is **not** in `siteParserRegistry` (buy-box price is not in SSR HTML). `mercadolivre.com.br` is **not** in `siteParserRegistry` (cold fetch 302 → `/gz/account-verification`). `olx.com.br` and `olx.pt` are **not** in `siteParserRegistry` (Cloudflare / CloudFront 403). `fnac.pt` is **not** in `siteParserRegistry` (could not reach). `worten.pt` is **not** in `siteParserRegistry` (Cloudflare 403 challenge). `magazineluiza.com.br` is **not** in `siteParserRegistry` (cold fetch 403). `primark.com` is **not** in `siteParserRegistry` (cold fetch 403 maintenance; PDP is Next.js RSC). `primor.eu` is **not** in `siteParserRegistry` (AWS WAF HTTP 202 — use Empathy search extractor, not HTML preview). `zara.com` is **not** in `siteParserRegistry` (Akamai Bot Manager interstitial). No working US generic extractor (`walmart_usa` is PerimeterX-blocked). Unknown / empty category → `other`, then generic fallback if a requested category has zero providers.

`FetchHTML` treats HTTP **200 and 206** as success (VTEX pagination).

---

## How to Add a New Price Extractor

Extractors live in **category-scoped** packages: `extractors/other/`, `extractors/beauty/`, `extractors/appliances/`, or a new `extractors/<category>/` with matching `package` name.

### Step 0 — Feasibility first (mandatory, before any code)

When a store, HAR, or hostname is proposed: **do not implement yet**. Probe first, then report feasible / blocked / need a better HAR. Only write extractor code after a live probe succeeds.

Confirm the **actual search/API URL** (not the homepage, not product images) returns real product name + price from a plain HTTP client:

```bash
curl -si \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36" \
  -H "Accept: application/json, text/html" \
  "https://www.example.com/api/search?q=test" \
  | head -20
```

**Stop and do not implement** if you see:

| Signal | WAF type |
|--------|----------|
| `HTTP/2 202` + `x-amzn-waf-action: challenge` | AWS WAF |
| `HTTP/1.1 403` + `errors.edgesuite.net` | Akamai |
| `HTTP/1.1 403` + `server: cloudflare` | Cloudflare challenge |
| Empty body + `server: awselb/2.0` | AWS WAF |

`server: cloudflare` with **HTTP 200 and a real JSON/HTML body** is OK (e.g. Empathy, Doofinder). A WAF-blocked extractor will never work in production. Find another site for that country/category.

A homepage HAR with only images/static assets is **not** a passing probe. Capture Network while searching, or find the JSON search endpoint (Empathy, Doofinder, VTEX, SFCC `format=ajax`) and curl that URL. Client-rendered search HTML with no product tiles is a fail — keep looking for JSON.

### Step 1 — Parser (HTML scrapers only)

```go
// extractors/other/mysite_extractor.go
package other

import "muambr-api/extractors"

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
func (p *MySiteParser) ParsePrice(html string) (float64, string, error) { /* price, "EUR", nil */ }
func (p *MySiteParser) ParseURL(html string, baseURL string) string { /* absolute URL */ }
func (p *MySiteParser) ParseStore(html string) string { return "MySite" }
```

JSON-only: copy the noop parser from `epocacosmeticos_extractor.go` / `primor_pt_extractor.go` / `perfumes_e_companhia_pt_extractor.go`.

### Step 2 — Extractor

```go
type MySiteExtractor struct {
    *extractors.BaseGoExtractor
}

func NewMySiteExtractor() *MySiteExtractor {
    parser := NewMySiteParser()
    base := extractors.NewBaseGoExtractor(
        "https://www.mysite.com",
        models.CountryPortugal,
        "mysite_v1",
        parser,
    )
    return &MySiteExtractor{BaseGoExtractor: base}
}

func (e *MySiteExtractor) GetCategory() models.ProductCategory {
    return models.CategoryOther
}

func (e *MySiteExtractor) BuildSearchURL(productName string) (string, error) {
    q := url.QueryEscape(productName)
    return fmt.Sprintf("%s/search?q=%s", e.GetBaseURL(), q), nil
}
```

Override `GetComparisons` when the default HTML flow does not apply (JSON API, SSE). Prefer JSON-LD first, HTML fallback.

Return store-native `Price` + ISO `Currency`. Do not set `ConvertedPrice` in the extractor.

### Step 3 — Register

In `handlers/extractor_handler.go` `initializeExtractors`:

```go
import beauty "muambr-api/extractors/beauty"

registry.RegisterExtractor(beauty.NewMySiteExtractor())
```

### Step 4 — Tests

| Extractor type | Unit tests? | Integration tests? |
|----------------|-------------|--------------------|
| HTML / CSS / regex / JSON-LD / SSE | Yes — fixture + `GetComparisonsFromHTML` | Yes |
| Trivial JSON unmarshal (VTEX, Empathy, Doofinder, GraphQL) | No | Yes |

Fixtures: `tests/mocks/extractors/sample_responses/`.  
Tests mirror packages: `tests/unit/extractors/<category>/`, `tests/integration/extractors/<category>/`.  
Integration tests use the shared goroutine + timeout helper and skip unless `INTEGRATION_TESTS=true`.

---

## How to Add a New Link Parser

Used by `GET /api/v1/linkpreview`.

### Step 0 — Feasibility first (mandatory, before any code)

When a hostname is proposed: **do not implement yet**. `curl` a real product URL (not the homepage) and confirm title + price are present in the HTML or JSON-LD. If the page is WAF-blocked or the price is injected only after JS, report blocked and stop. Only write a parser after the probe succeeds.

Site-specific extractors first, then `ShareHTMLParser`. Register hostname (no `www.`) in `linkparsers/site_parsers.go` `siteParserRegistry`. Unmatched hosts fall back to `ShareHTMLParser`.

---

## Utilities

```go
utils.Info("message", utils.String("key", value), utils.Int("count", n))
utils.Warn("message", utils.Error(err))
utils.Error("message", utils.Error(err))
```

Do not commit debug-only logs.

- HTTP: `utils.MakeScrapingRequest` or `FetchHTML` — never raw `http.Get`.
- i18n: keys in `localization/en.json` (and `pt.json` / `es.json`).
- `ComparisonProcessor`: group by country, sort, drop prices &lt; 60% of average.
- `ComparisonEngine`: match confidence, best prices, savings, deal score, omit same-currency `normalized*`.

---

## Extractor Selection

1. Always include `baseCountry` extractors.
2. If `currentCountry` differs from `baseCountry`:
   - `useMacroRegion=true` → all extractors in `currentCountry`’s macro region
   - else → `currentCountry` only
3. If `category` is set, keep matching `GetCategory()`; if none, fall back to `other`.
4. `nil` category → all categories for those countries.

---

## Testing Commands

```bash
make test              # unit tests
make test-all          # unit + integration
make test-coverage     # HTML report in coverage/

INTEGRATION_TESTS=true go test ./tests/integration/...
```

Never gate unit tests on `INTEGRATION_TESTS`.

---

## Conventions Checklist

- [ ] Change is lean: no extra abstraction used only once
- [ ] Shared behavior lives in base types / `utils`, not copied
- [ ] Extractors do not compute FX, savings, or deal score
- [ ] File in the correct category package; `package` name matches directory
- [ ] Import `"muambr-api/extractors"` for `BaseHTMLParser` / `BaseGoExtractor`
- [ ] File `<sitename>_extractor[_v2].go`; types `New<Site>Extractor` / `New<Site>Parser`
- [ ] `GetIdentifier()` stable snake_case; `GetCategory()` uses a `models` constant
- [ ] Prices `float64`; currency ISO 4217; `Country` is the ISO constant string
- [ ] Registered in `initializeExtractors`
- [ ] Link parsers registered in `siteParserRegistry`
- [ ] No raw `http.Get`; no debug logs
- [ ] Feasibility probe done **before** any extractor or link-parser code (live search/API or product page; not homepage HAR)
- [ ] New countries updated in **all** `Country` switches in `models/models.go`
