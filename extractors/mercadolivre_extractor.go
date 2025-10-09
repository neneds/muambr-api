package extractors

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"muambr-api/models"
	"muambr-api/utils"
)

// MercadoLivreExtractor implements the Extractor interface for Mercado Livre marketplace
type MercadoLivreExtractor struct {
	countryCode models.Country
}

// NewMercadoLivreExtractor creates a new Mercado Livre extractor for Brazil
func NewMercadoLivreExtractor() *MercadoLivreExtractor {
	return &MercadoLivreExtractor{
		countryCode: models.CountryBrazil, // Mercado Livre is Brazil-specific
	}
}

// GetCountryCode returns the ISO country code this extractor supports
func (e *MercadoLivreExtractor) GetCountryCode() models.Country {
	return e.countryCode
}

// GetMacroRegion returns the macro region this extractor supports
func (e *MercadoLivreExtractor) GetMacroRegion() models.MacroRegion {
	return e.countryCode.GetMacroRegion()
}

// BaseURL returns the base URL for the extractor's website
func (e *MercadoLivreExtractor) BaseURL() string {
	return "https://lista.mercadolivre.com.br"
}

// GetIdentifier returns a static string identifier for this extractor
func (e *MercadoLivreExtractor) GetIdentifier() string {
	return "mercadolivre"
}

// GetComparisons extracts product comparisons from Mercado Livre for the given product name
func (e *MercadoLivreExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	// Build the search URL with query parameters
	searchURL, err := e.buildSearchURL(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Make HTTP request to get the HTML page
	htmlContent, err := e.fetchHTML(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	// Extract products using Python script
	comparisons, err := e.extractWithPython(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract with Python: %w", err)
	}

	return comparisons, nil
}

// buildSearchURL constructs the search URL with proper query parameters for Mercado Livre
func (e *MercadoLivreExtractor) buildSearchURL(productName string) (string, error) {
	baseURL := e.BaseURL()
	
	// Build the URL using Mercado Livre's format: /product-name
	// Replace spaces with hyphens and encode properly
	encodedProduct := url.PathEscape(strings.ReplaceAll(strings.ToLower(productName), " ", "-"))
	
dennismerli@192 muambr-goapi % ./muambr-api
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /health                   --> main.main.func2 (4 handlers)
[GIN-debug] GET    /api/v1/comparisons/search --> muambr-api/handlers.(*ComparisonHandler).GetComparisons-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/status --> muambr-api/handlers.(*AdminHandler).GetExchangeRateStatus-fm (4 handlers)
[GIN-debug] DELETE /admin/exchange-rates/cache --> muambr-api/handlers.(*AdminHandler).ClearExchangeRateCache-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/test --> muambr-api/handlers.(*AdminHandler).TestExchangeRateAPI-fm (4 handlers)
2025-10-08T22:51:46.500-0300    INFO    utils/logger.go:123     Starting Product Comparison API server  {"port": "8080"}
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080
^C
dennismerli@192 muambr-goapi % curl -s "http://localhost:8080/api/v1/comparisons/search?name=iphone&baseCountry=BR&curren
cy=BRL" | jq .
dennismerli@192 muambr-goapi % curl -s "http://localhost:8080/api/v1/comparisons/search?name=iphone&baseCountry=BR&curren
cy=BRL"
dennismerli@192 muambr-goapi % ./muambr-api &
[1] 4067
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.              

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /health                   --> main.main.func2 (4 handlers)
[GIN-debug] GET    /api/v1/comparisons/search --> muambr-api/handlers.(*ComparisonHandler).GetComparisons-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/status --> muambr-api/handlers.(*AdminHandler).GetExchangeRateStatus-fm (4 handlers)
[GIN-debug] DELETE /admin/exchange-rates/cache --> muambr-api/handlers.(*AdminHandler).ClearExchangeRateCache-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/test --> muambr-api/handlers.(*AdminHandler).TestExchangeRateAPI-fm (4 handlers)
2025-10-08T22:52:15.655-0300    INFO    utils/logger.go:123     Starting Product Comparison API server  {"port": "8080"}
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080
dennismerli@192 muambr-goapi % sleep 2
dennismerli@192 muambr-goapi % curl -s "http://localhost:8080/api/v1/comparisons/search?name=iphone&baseCountry=BR&curren
cy=BRL" | head -20
2025-10-08T22:52:25.735-0300    INFO    utils/logger.go:123     Starting product comparison search with multiple extractors      {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "extractor_count": 2, "extractor_names": ["mercadolivre", "magalu"]}
2025-10-08T22:52:25.735-0300    INFO    utils/logger.go:123     Starting HTTP request to MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "base_domain": "https://lista.mercadolivre.com.br"}
2025-10-08T22:52:28.293-0300    INFO    utils/logger.go:123     Successfully received HTTP response from MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "status_code": 200, "content_length": -1}
2025-10-08T22:52:28.304-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from MercadoLivre     {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "content_size_bytes": 184250}
2025-10-08T22:52:28.305-0300    INFO    utils/logger.go:123     Starting Python extraction for MercadoLivre     {"extractor": "mercadolivre", "html_size_bytes": 184250}
2025-10-08T22:52:28.305-0300    INFO    utils/logger.go:123     Using Python interpreter for MercadoLivre extraction    {"extractor": "mercadolivre", "python_path": "/Users/dennismerli/Documents/Projects/muambr-goapi/.venv/bin/python", "script_path": "/Users/dennismerli/Documents/Projects/muambr-goapi/extractors/pythonExtractors/mercadolivre_page.py"}
2025-10-08T22:52:28.307-0300    INFO    utils/logger.go:123     Created temporary HTML file for MercadoLivre extraction {"extractor": "mercadolivre", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/mercadolivre_html_3936267661.html"}
2025-10-08T22:52:28.307-0300    INFO    utils/logger.go:123     Executing Python extraction script for MercadoLivre     {"extractor": "mercadolivre", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/mercadolivre_html_3936267661.html"}
2025-10-08T22:52:28.451-0300    INFO    utils/logger.go:123     Python script executed successfully for MercadoLivre    {"extractor": "mercadolivre", "python_stdout": "{\"error\": \"Python extraction failed\", \"details\": \"'utf-8' codec can't decode byte 0x9b in position 0: invalid start byte\"}\n", "python_stderr": ""}
2025-10-08T22:52:28.452-0300    WARN    utils/logger.go:128     Python script returned error for MercadoLivre   {"extractor": "mercadolivre", "error_type": "Python extraction failed", "error_details": "'utf-8' codec can't decode byte 0x9b in position 0: invalid start byte"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/extractors.(*MercadoLivreExtractor).extractWithPython
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/mercadolivre_extractor.go:255
muambr-api/extractors.(*MercadoLivreExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/mercadolivre_extractor.go:68
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:52:28.452-0300    INFO    utils/logger.go:123     Extractor successfully completed product search {"search_term": "iphone", "extractor_name": "mercadolivre", "extractor_country": "BR", "results_count": 0}
2025-10-08T22:52:28.452-0300    INFO    utils/logger.go:123     Starting Magazine Luiza extraction      {"search_term": "iphone", "extractor": "magalu"}
2025-10-08T22:52:28.452-0300    INFO    utils/logger.go:123     Built Magazine Luiza search URL {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu"}
2025-10-08T22:52:28.452-0300    INFO    utils/logger.go:123     Starting HTTP request to Magazine Luiza {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "base_domain": "https://www.magazineluiza.com.br"}
2025-10-08T22:52:30.832-0300    INFO    utils/logger.go:123     Successfully received HTTP response from Magazine Luiza {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "status_code": 200, "content_length": -1}
2025-10-08T22:52:30.958-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from Magazine Luiza   {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "content_size_bytes": 249185}
2025-10-08T22:52:30.958-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from Magazine Luiza   {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "content_size_bytes": 249185}
2025-10-08T22:52:30.958-0300    INFO    utils/logger.go:123     Starting Python extraction for Magazine Luiza   {"extractor": "magalu", "html_size_bytes": 249185}
2025-10-08T22:52:30.958-0300    INFO    utils/logger.go:123     Using Python interpreter for Magazine Luiza extraction  {"extractor": "magalu", "python_path": "python3", "script_path": "extractors/pythonExtractors/magalu_page.py"}
2025-10-08T22:52:30.960-0300    INFO    utils/logger.go:123     Created temporary HTML file for Magazine Luiza extraction{"extractor": "magalu", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/magalu_html_2577777941.html"}
2025-10-08T22:52:30.960-0300    INFO    utils/logger.go:123     Executing Python extraction script for Magazine Luiza   {"extractor": "magalu", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/magalu_html_2577777941.html"}
2025-10-08T22:52:31.000-0300    INFO    utils/logger.go:123     Python script executed successfully for Magazine Luiza  {"extractor": "magalu", "python_stdout": "{\"error\": \"Missing Python dependencies\", \"details\": \"No module named 'bs4'\"}\n", "python_stderr": ""}
2025-10-08T22:52:31.000-0300    WARN    utils/logger.go:128     Python script returned error for Magazine Luiza {"extractor": "magalu", "error_type": "Missing Python dependencies", "error_details": "No module named 'bs4'"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/extractors.(*MagaluExtractor).extractWithPython
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:262
muambr-api/extractors.(*MagaluExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:82
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:52:31.000-0300    INFO    utils/logger.go:123     Successfully extracted products from Magazine Luiza     {"extractor": "magalu", "products_count": 0}
2025-10-08T22:52:31.000-0300    INFO    utils/logger.go:123     Extractor successfully completed product search {"search_term": "iphone", "extractor_name": "magalu", "extractor_country": "BR", "results_count": 0}
2025-10-08T22:52:31.000-0300    INFO    utils/logger.go:123     Product comparison search completed     {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "total_results": 0, "extractors_attempted": 2}
[GIN] 2025/10/08 - 22:52:31 | 200 |  5.265932375s |             ::1 | GET      "/api/v1/comparisons/search?name=iphone&baseCountry=BR&currency=BRL"
{"success":true,"message":"No comparisons found for this product","comparisons":[],"totalResults":0}%                    
dennismerli@192 muambr-goapi % curl -s -H "Accept-Encoding: gzip, deflate, br" "https://lista.mercadolivre.com.br/iphone"
 | head -5
<html>
  <head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <link rel="stylesheet" type="text/css" href="https://http2.mlstatic.com/ui/navigation/2.3.5/mercadolibre/navigation.css">
  <style>
    .ui-empty-state {
dennismerli@192 muambr-goapi % curl -I -H "Accept-Encoding: gzip, deflate, br" "https://lista.mercadolivre.com.br/iphone"

HTTP/2 403 
server: CloudFront
date: Thu, 09 Oct 2025 01:54:09 GMT
content-length: 2586
rps: w403
content-type: application/json
x-cache: Error from cloudfront
via: 1.1 f93176b9e77c36fc40dcf506c762d5b0.cloudfront.net (CloudFront)
x-amz-cf-pop: GRU3-P12
alt-svc: h3=":443"; ma=86400
x-amz-cf-id: SceQPNcaSMF3OdNCp7wtqLS13rcJlXiyTtld_kE4JIU0Jl64tGiQwA==

dennismerli@192 muambr-goapi % go build -o muambr-api
dennismerli@192 muambr-goapi % pkill -f muambr-api
[1]  + terminated  ./muambr-api                                                                                          
dennismerli@192 muambr-goapi % ./muambr-api &
[1] 4972
dennismerli@192 muambr-goapi % [GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /health                   --> main.main.func2 (4 handlers)
[GIN-debug] GET    /api/v1/comparisons/search --> muambr-api/handlers.(*ComparisonHandler).GetComparisons-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/status --> muambr-api/handlers.(*AdminHandler).GetExchangeRateStatus-fm (4 handlers)
[GIN-debug] DELETE /admin/exchange-rates/cache --> muambr-api/handlers.(*AdminHandler).ClearExchangeRateCache-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/test --> muambr-api/handlers.(*AdminHandler).TestExchangeRateAPI-fm (4 handlers)
2025-10-08T22:54:56.428-0300    INFO    utils/logger.go:123     Starting Product Comparison API server  {"port": "8080"}
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080

dennismerli@192 muambr-goapi % sleep 3 && curl -s "http://localhost:8080/api/v1/comparisons/search?name=iphone&baseCountr
y=BR&currency=BRL" > /dev/null
2025-10-08T22:55:03.480-0300    INFO    utils/logger.go:123     Starting product comparison search with multiple extractors      {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "extractor_count": 2, "extractor_names": ["mercadolivre", "magalu"]}
2025-10-08T22:55:03.480-0300    INFO    utils/logger.go:123     Starting HTTP request to MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "base_domain": "https://lista.mercadolivre.com.br"}
2025-10-08T22:55:04.965-0300    WARN    utils/logger.go:128     HTTP request returned non-200 status code from MercadoLivre - possible anti-bot protection       {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "base_domain": "https://lista.mercadolivre.com.br", "status_code": 404, "status": "404 Not Found"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/extractors.(*MercadoLivreExtractor).fetchHTML
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/mercadolivre_extractor.go:112
muambr-api/extractors.(*MercadoLivreExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/mercadolivre_extractor.go:62
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:55:04.967-0300    WARN    utils/logger.go:128     Extractor failed during product search - continuing with remaining extractors    {"search_term": "iphone", "extractor_name": "mercadolivre", "extractor_country": "BR", "base_country": "BR", "target_currency": "BRL", "error": "failed to fetch HTML: HTTP request failed with status: 404"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:94
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:55:04.967-0300    INFO    utils/logger.go:123     Starting Magazine Luiza extraction      {"search_term": "iphone", "extractor": "magalu"}
2025-10-08T22:55:04.967-0300    INFO    utils/logger.go:123     Built Magazine Luiza search URL {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu"}
2025-10-08T22:55:04.967-0300    INFO    utils/logger.go:123     Starting HTTP request to Magazine Luiza {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "base_domain": "https://www.magazineluiza.com.br"}
^C
dennismerli@192 muambr-goapi % 2025-10-08T22:55:06.407-0300     ERROR   utils/logger.go:133     HTTP request failed for Magazine Luiza   {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "error": "Get \"https://validate.perfdrive.com/ca4df1c7abf7ea2cc50ab30bdf7ed2bb/?ssa=4c507842-5817-4489-b945-91d79d54f55a&ssb=54894292090&ssc=https%3A%2F%2Fwww.magazineluiza.com.br%2Fbusca%2Fiphone%2F&ssi=754ddc5e-bi37-47de-a767-68bf7488bb6c&ssk=support@shieldsquare.com&ssm=29268872052545225106457533412934&ssn=fae39314a9f1338848fd2bc4257d88225a2a797c2828-d36a-4994-bf04b7&sso=9c5396c9-55a3f9eb9f6b11f47f59e65bec9abd6350ee303e38e0d5c0&ssp=08956448861759998459175998064095044&ssq=09151527490583266228574905770808008841092&ssr=MTg5LjYyLjQ2LjE3MQ==&sst=Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0&ssv=&ssw=&ssx=W10=\": http2: invalid Connection request header: [\"keep-alive\" \"keep-alive\"]"}
muambr-api/utils.(*ZapLogger).Error
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:133
muambr-api/utils.LogError
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:200
muambr-api/extractors.(*MagaluExtractor).fetchHTML
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:110
muambr-api/extractors.(*MagaluExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:67
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:55:06.408-0300    ERROR   utils/logger.go:133     Failed to fetch HTML from Magazine Luiza        {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "error": "Get \"https://validate.perfdrive.com/ca4df1c7abf7ea2cc50ab30bdf7ed2bb/?ssa=4c507842-5817-4489-b945-91d79d54f55a&ssb=54894292090&ssc=https%3A%2F%2Fwww.magazineluiza.com.br%2Fbusca%2Fiphone%2F&ssi=754ddc5e-bi37-47de-a767-68bf7488bb6c&ssk=support@shieldsquare.com&ssm=29268872052545225106457533412934&ssn=fae39314a9f1338848fd2bc4257d88225a2a797c2828-d36a-4994-bf04b7&sso=9c5396c9-55a3f9eb9f6b11f47f59e65bec9abd6350ee303e38e0d5c0&ssp=08956448861759998459175998064095044&ssq=09151527490583266228574905770808008841092&ssr=MTg5LjYyLjQ2LjE3MQ==&sst=Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0&ssv=&ssw=&ssx=W10=\": http2: invalid Connection request header: [\"keep-alive\" \"keep-alive\"]"}
muambr-api/utils.(*ZapLogger).Error
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:133
muambr-api/utils.LogError
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:200
muambr-api/extractors.(*MagaluExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:69
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:55:06.408-0300    WARN    utils/logger.go:128     Extractor failed during product search - continuing with remaining extractors    {"search_term": "iphone", "extractor_name": "magalu", "extractor_country": "BR", "base_country": "BR", "target_currency": "BRL", "error": "Get \"https://validate.perfdrive.com/ca4df1c7abf7ea2cc50ab30bdf7ed2bb/?ssa=4c507842-5817-4489-b945-91d79d54f55a&ssb=54894292090&ssc=https%3A%2F%2Fwww.magazineluiza.com.br%2Fbusca%2Fiphone%2F&ssi=754ddc5e-bi37-47de-a767-68bf7488bb6c&ssk=support@shieldsquare.com&ssm=29268872052545225106457533412934&ssn=fae39314a9f1338848fd2bc4257d88225a2a797c2828-d36a-4994-bf04b7&sso=9c5396c9-55a3f9eb9f6b11f47f59e65bec9abd6350ee303e38e0d5c0&ssp=08956448861759998459175998064095044&ssq=09151527490583266228574905770808008841092&ssr=MTg5LjYyLjQ2LjE3MQ==&sst=Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0&ssv=&ssw=&ssx=W10=\": http2: invalid Connection request header: [\"keep-alive\" \"keep-alive\"]"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:94
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:55:06.408-0300    INFO    utils/logger.go:123     Product comparison search completed     {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "total_results": 0, "extractors_attempted": 2}
[GIN] 2025/10/08 - 22:55:06 | 200 |     2.928175s |             ::1 | GET      "/api/v1/comparisons/search?name=iphone&baseCountry=BR&currency=BRL"
jobs
[1]  + running    ./muambr-api
dennismerli@192 muambr-goapi % curl -I "https://lista.mercadolivre.com.br/iphone"
HTTP/2 403 
server: CloudFront
date: Thu, 09 Oct 2025 01:55:34 GMT
content-length: 2586
rps: w403
content-type: application/json
x-cache: Error from cloudfront
via: 1.1 e33a1931f45ba5c88d5bbeba53cbd6d6.cloudfront.net (CloudFront)
x-amz-cf-pop: GRU3-P12
alt-svc: h3=":443"; ma=86400
x-amz-cf-id: 7mj9UJfAKYTAfWL6EeOEU_v-wLCWkh8FX01MJsmIse0qT2j3F3MdMQ==

dennismerli@192 muambr-goapi % curl -I "https://lista.mercadolibre.com.ar/iphone"
curl: (6) Could not resolve host: lista.mercadolibre.com.ar
dennismerli@192 muambr-goapi % curl -I "https://www.mercadolivre.com.br/iphone"
HTTP/2 403 
server: CloudFront
date: Thu, 09 Oct 2025 01:55:51 GMT
content-length: 2586
rps: w403
content-type: application/json
x-cache: Error from cloudfront
via: 1.1 e650545b544c95ecfe4fff710d96d26e.cloudfront.net (CloudFront)
x-amz-cf-pop: GRU3-P12
alt-svc: h3=":443"; ma=86400
x-amz-cf-id: UUAqK6RLLb_cG5yASeHisZPOvxz6qvPSghRFMciSPLnQkmLHtnTtfg==

dennismerli@192 muambr-goapi % go build -o muambr-api
dennismerli@192 muambr-goapi % go build -o muambr-api
# muambr-api/extractors
extractors/mercadolivre_extractor.go:10:2: imported and not used: "net/url"
dennismerli@192 muambr-goapi % go build -o muambr-api
dennismerli@192 muambr-goapi % pkill -f muambr-api
[1]  + terminated  ./muambr-api                                                                                          
dennismerli@192 muambr-goapi % ./muambr-api &
[1] 6280
dennismerli@192 muambr-goapi % [GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /health                   --> main.main.func2 (4 handlers)
[GIN-debug] GET    /api/v1/comparisons/search --> muambr-api/handlers.(*ComparisonHandler).GetComparisons-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/status --> muambr-api/handlers.(*AdminHandler).GetExchangeRateStatus-fm (4 handlers)
[GIN-debug] DELETE /admin/exchange-rates/cache --> muambr-api/handlers.(*AdminHandler).ClearExchangeRateCache-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/test --> muambr-api/handlers.(*AdminHandler).TestExchangeRateAPI-fm (4 handlers)
2025-10-08T22:58:23.562-0300    INFO    utils/logger.go:123     Starting Product Comparison API server  {"port": "8080"}
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080

dennismerli@192 muambr-goapi % sleep 3 && curl -s "http://localhost:8080/api/v1/comparisons/search?name=iphone&baseCountr
y=BR&currency=BRL" > /dev/null
2025-10-08T22:58:32.653-0300    INFO    utils/logger.go:123     Starting product comparison search with multiple extractors      {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "extractor_count": 2, "extractor_names": ["mercadolivre", "magalu"]}
2025-10-08T22:58:32.654-0300    INFO    utils/logger.go:123     Starting HTTP request to MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "base_domain": "https://lista.mercadolivre.com.br"}
2025-10-08T22:58:35.164-0300    INFO    utils/logger.go:123     Successfully received HTTP response from MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "status_code": 200, "content_length": -1, "content_type": "text/html; charset=utf-8", "content_encoding": "br"}
2025-10-08T22:58:35.390-0300    INFO    utils/logger.go:123     Response body first bytes inspection for MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "first_bytes_hex": "9b6df7211928bb0f4284"}
2025-10-08T22:58:35.390-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from MercadoLivre     {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "content_size_bytes": 185477}
2025-10-08T22:58:35.390-0300    INFO    utils/logger.go:123     Starting Python extraction for MercadoLivre     {"extractor": "mercadolivre", "html_size_bytes": 185477}
2025-10-08T22:58:35.391-0300    INFO    utils/logger.go:123     Using Python interpreter for MercadoLivre extraction    {"extractor": "mercadolivre", "python_path": "/Users/dennismerli/Documents/Projects/muambr-goapi/.venv/bin/python", "script_path": "/Users/dennismerli/Documents/Projects/muambr-goapi/extractors/pythonExtractors/mercadolivre_page.py"}
2025-10-08T22:58:35.393-0300    INFO    utils/logger.go:123     Created temporary HTML file for MercadoLivre extraction {"extractor": "mercadolivre", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/mercadolivre_html_981360828.html"}
2025-10-08T22:58:35.393-0300    INFO    utils/logger.go:123     Executing Python extraction script for MercadoLivre     {"extractor": "mercadolivre", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/mercadolivre_html_981360828.html"}
2025-10-08T22:58:35.530-0300    INFO    utils/logger.go:123     Python script executed successfully for MercadoLivre    {"extractor": "mercadolivre", "python_stdout": "{\"error\": \"Python extraction failed\", \"details\": \"'utf-8' codec can't decode byte 0x9b in position 0: invalid start byte\"}\n", "python_stderr": ""}
2025-10-08T22:58:35.530-0300    WARN    utils/logger.go:128     Python script returned error for MercadoLivre   {"extractor": "mercadolivre", "error_type": "Python extraction failed", "error_details": "'utf-8' codec can't decode byte 0x9b in position 0: invalid start byte"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/extractors.(*MercadoLivreExtractor).extractWithPython
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/mercadolivre_extractor.go:313
muambr-api/extractors.(*MercadoLivreExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/mercadolivre_extractor.go:68
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:58:35.530-0300    INFO    utils/logger.go:123     Extractor successfully completed product search {"search_term": "iphone", "extractor_name": "mercadolivre", "extractor_country": "BR", "results_count": 0}
2025-10-08T22:58:35.530-0300    INFO    utils/logger.go:123     Starting Magazine Luiza extraction      {"search_term": "iphone", "extractor": "magalu"}
2025-10-08T22:58:35.530-0300    INFO    utils/logger.go:123     Built Magazine Luiza search URL {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu"}
2025-10-08T22:58:35.530-0300    INFO    utils/logger.go:123     Starting HTTP request to Magazine Luiza {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "base_domain": "https://www.magazineluiza.com.br"}
^C
dennismerli@192 muambr-goapi % echo "" && echo "Latest logs:" && sleep 1

Latest logs:
dennismerli@192 muambr-goapi % 2025-10-08T22:58:38.858-0300     INFO    utils/logger.go:123     Successfully received HTTP response from Magazine Luiza  {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "status_code": 200, "content_length": -1}
2025-10-08T22:58:39.044-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from Magazine Luiza   {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "content_size_bytes": 248536}
2025-10-08T22:58:39.044-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from Magazine Luiza   {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "content_size_bytes": 248536}
2025-10-08T22:58:39.044-0300    INFO    utils/logger.go:123     Starting Python extraction for Magazine Luiza   {"extractor": "magalu", "html_size_bytes": 248536}
2025-10-08T22:58:39.044-0300    INFO    utils/logger.go:123     Using Python interpreter for Magazine Luiza extraction  {"extractor": "magalu", "python_path": "python3", "script_path": "extractors/pythonExtractors/magalu_page.py"}
2025-10-08T22:58:39.047-0300    INFO    utils/logger.go:123     Created temporary HTML file for Magazine Luiza extraction{"extractor": "magalu", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/magalu_html_2112534718.html"}
2025-10-08T22:58:39.048-0300    INFO    utils/logger.go:123     Executing Python extraction script for Magazine Luiza   {"extractor": "magalu", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/magalu_html_2112534718.html"}
2025-10-08T22:58:39.087-0300    INFO    utils/logger.go:123     Python script executed successfully for Magazine Luiza  {"extractor": "magalu", "python_stdout": "{\"error\": \"Missing Python dependencies\", \"details\": \"No module named 'bs4'\"}\n", "python_stderr": ""}
2025-10-08T22:58:39.088-0300    WARN    utils/logger.go:128     Python script returned error for Magazine Luiza {"extractor": "magalu", "error_type": "Missing Python dependencies", "error_details": "No module named 'bs4'"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/extractors.(*MagaluExtractor).extractWithPython
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:262
muambr-api/extractors.(*MagaluExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:82
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T22:58:39.088-0300    INFO    utils/logger.go:123     Successfully extracted products from Magazine Luiza     {"extractor": "magalu", "products_count": 0}
2025-10-08T22:58:39.088-0300    INFO    utils/logger.go:123     Extractor successfully completed product search {"search_term": "iphone", "extractor_name": "magalu", "extractor_country": "BR", "results_count": 0}
2025-10-08T22:58:39.088-0300    INFO    utils/logger.go:123     Product comparison search completed     {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "total_results": 0, "extractors_attempted": 2}
[GIN] 2025/10/08 - 22:58:39 | 200 |    6.4346025s |             ::1 | GET      "/api/v1/comparisons/search?name=iphone&baseCountry=BR&currency=BRL"

dennismerli@192 muambr-goapi % go get github.com/andybalholm/brotli
go: downloading github.com/andybalholm/brotli v1.2.0
github.com/andybalholm/brotli imports
        github.com/andybalholm/brotli/matchfinder imports
        slices: package slices is not in GOROOT (/usr/local/go/src/slices)
dennismerli@192 muambr-goapi % go build -o muambr-api
extractors/mercadolivre_extractor.go:16:2: no required module provides package github.com/andybalholm/brotli; to add it:
        go get github.com/andybalholm/brotli
dennismerli@192 muambr-goapi % go mod tidy && go get github.com/andybalholm/brotli
go: downloading go.uber.org/goleak v1.3.0
go: finding module for package github.com/andybalholm/brotli
go: found github.com/andybalholm/brotli in github.com/andybalholm/brotli v1.2.0
go: downloading github.com/xyproto/randomstring v1.0.5
github.com/andybalholm/brotli imports
        github.com/andybalholm/brotli/matchfinder imports
        slices: package slices is not in GOROOT (/usr/local/go/src/slices)
dennismerli@192 muambr-goapi % go version
go version go1.19 darwin/arm64
dennismerli@192 muambr-goapi % go build -o muambr-api
dennismerli@192 muambr-goapi % pkill -f muambr-api && sleep 2 && cd /Users/dennismerli/Documents/Projects/muambr-goapi &&
 ./muambr-api &
[1]  + terminated  ./muambr-api
[1] 7468
dennismerli@192 muambr-goapi % [GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /health                   --> main.main.func2 (4 handlers)
[GIN-debug] GET    /api/v1/comparisons/search --> muambr-api/handlers.(*ComparisonHandler).GetComparisons-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/status --> muambr-api/handlers.(*AdminHandler).GetExchangeRateStatus-fm (4 handlers)
[GIN-debug] DELETE /admin/exchange-rates/cache --> muambr-api/handlers.(*AdminHandler).ClearExchangeRateCache-fm (4 handlers)
[GIN-debug] GET    /admin/exchange-rates/test --> muambr-api/handlers.(*AdminHandler).TestExchangeRateAPI-fm (4 handlers)
2025-10-08T23:00:43.957-0300    INFO    utils/logger.go:123     Starting Product Comparison API server  {"port": "8080"}
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080

dennismerli@192 muambr-goapi % sleep 3 && curl -s "http://localhost:8080/api/v1/comparisons/search?name=iphone&baseCountr
y=BR&currency=BRL" > /dev/null
2025-10-08T23:00:53.205-0300    INFO    utils/logger.go:123     Starting product comparison search with multiple extractors      {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "extractor_count": 2, "extractor_names": ["mercadolivre", "magalu"]}
2025-10-08T23:00:53.205-0300    INFO    utils/logger.go:123     Starting HTTP request to MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "base_domain": "https://lista.mercadolivre.com.br"}
2025-10-08T23:00:55.649-0300    INFO    utils/logger.go:123     Successfully received HTTP response from MercadoLivre   {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "status_code": 200, "content_length": -1, "content_type": "text/html; charset=utf-8", "content_encoding": "gzip"}
2025-10-08T23:00:55.983-0300    INFO    utils/logger.go:123     Detected gzip compressed content from MercadoLivre, decompressing        {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "compressed_size_bytes": 305350}
2025-10-08T23:00:55.993-0300    INFO    utils/logger.go:123     Successfully decompressed gzip content from MercadoLivre{"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "compressed_size_bytes": 305350, "decompressed_size_bytes": 2211148}
2025-10-08T23:00:55.993-0300    INFO    utils/logger.go:123     Successfully fetched HTML content from MercadoLivre     {"url": "https://lista.mercadolivre.com.br/iphone", "extractor": "mercadolivre", "content_size_bytes": 2211148}
2025-10-08T23:00:55.993-0300    INFO    utils/logger.go:123     Starting Python extraction for MercadoLivre     {"extractor": "mercadolivre", "html_size_bytes": 2211148}
2025-10-08T23:00:55.994-0300    INFO    utils/logger.go:123     Using Python interpreter for MercadoLivre extraction    {"extractor": "mercadolivre", "python_path": "/Users/dennismerli/Documents/Projects/muambr-goapi/.venv/bin/python", "script_path": "/Users/dennismerli/Documents/Projects/muambr-goapi/extractors/pythonExtractors/mercadolivre_page.py"}
2025-10-08T23:00:55.996-0300    INFO    utils/logger.go:123     Created temporary HTML file for MercadoLivre extraction {"extractor": "mercadolivre", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/mercadolivre_html_2005319942.html"}
2025-10-08T23:00:55.996-0300    INFO    utils/logger.go:123     Executing Python extraction script for MercadoLivre     {"extractor": "mercadolivre", "temp_file": "/var/folders/nm/4x138fx56x19h_466qwsbp6w0000gn/T/mercadolivre_html_2005319942.html"}
2025-10-08T23:00:56.216-0300    INFO    utils/logger.go:123     Python script executed successfully for MercadoLivre    {"extractor": "mercadolivre", "python_stdout": "[{\"name\": \"Apple iPhone 15 (128 Gb) - Verde - Distribuidor Autorizado\", \"price\": \"4553.3\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-15-128-gb-verde-distribuidor-autorizado/p/MLB1027172671\"}, {\"name\": \"Apple iPhone 16 (128 Gb) - Branco - Distribuidor Autorizado\", \"price\": \"5108.9\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-128-gb-branco-distribuidor-autorizado/p/MLB1040287791\"}, {\"name\": \"iPhone 16e (128 Gb) - Branco - Distribuidor Autorizado\", \"price\": \"3776.64\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-16e-128-gb-branco-distribuidor-autorizado/p/MLB1046217031\"}, {\"name\": \"Apple iPhone 15 (256 Gb) - Preto - Distribuidor Autorizado\", \"price\": \"5221.11\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-15-256-gb-preto-distribuidor-autorizado/p/MLB1027172669\"}, {\"name\": \"Apple iPhone 16 (256 Gb) - Preto - Distribuidor Autorizado\", \"price\": \"5999\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-256-gb-preto-distribuidor-autorizado/p/MLB1040287796\"}, {\"name\": \"iPhone 17 Pro 256gb - Azul-profundo - Distribuidor Autorizado\", \"price\": \"11498.88\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-17-pro-256gb-azul-profundo-distribuidor-autorizado/p/MLB1055308781\"}, {\"name\": \"Apple iPhone 14 (128 Gb) - Estelar - Distribuidor Autorizado\", \"price\": \"3887.73\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-14-128-gb-estelar-distribuidor-autorizado/p/MLB1019615378\"}, {\"name\": \"iPhone 17 256\\u00a0gb - S\\u00e1lvia - Distribuidor Autorizado\", \"price\": \"7998.84\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-17-256gb-salvia-distribuidor-autorizado/p/MLB1055308843\"}, {\"name\": \"iPhone 16e (256 Gb) - Preto - Distribuidor Autorizado\", \"price\": \"4699\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-16e-256-gb-preto-distribuidor-autorizado/p/MLB1046216986\"}, {\"name\": \"Apple iPhone 16 Pro (128 Gb) - Tit\\u00e2nio-preto - Distribuidor Autorizado - Distribuidor Autorizado\", \"price\": \"8221.11\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-128-gb-titnio-preto-distribuidor-autorizado-distribuidor-autorizado/p/MLB1053563668\"}, {\"name\": \"Apple iPhone 16 Pro Max (256 Gb) - Tit\\u00e2nio Natural - Distribuidor Autorizado\", \"price\": \"9322.9\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-max-256-gb-titnio-natural-distribuidor-autorizado/p/MLB1040287860\"}, {\"name\": \"Apple iPhone 14 (256 Gb) - Estelar - Distribuidor Autorizado\", \"price\": \"4998.84\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-14-256-gb-estelar-distribuidor-autorizado/p/MLB1019615376\"}, {\"name\": \"Apple iPhone 16 Plus (128 Gb) - Rosa - Distribuidor Autorizado\", \"price\": \"5974.42\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-plus-128-gb-rosa-distribuidor-autorizado/p/MLB1040287818\"}, {\"name\": \"Apple iPhone 16 Pro Max (512 Gb) - Tit\\u00e2nio Natural - Distribuidor Autorizado\", \"price\": \"10499.04\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-max-512-gb-titnio-natural-distribuidor-autorizado/p/MLB1040287864\"}, {\"name\": \"iPhone 17 512\\u00a0gb - Azul-n\\u00e9voa - Distribuidor Autorizado\", \"price\": \"9498.84\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-17-512gb-azul-nevoa-distribuidor-autorizado/p/MLB1055309028\"}, {\"name\": \"Apple iPhone 16 Pro (128 Gb) - Tit\\u00e2nio Branco - Distribuidor Autorizado\", \"price\": \"8221.11\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-128-gb-titnio-branco-distribuidor-autorizado/p/MLB1040287849\"}, {\"name\": \"iPhone 13 Dual Sim 128 Gb Azul (novo Com Caixa Aberta)\", \"price\": \"2749\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-13-dual-sim-128-gb-azul-novo-com-caixa-aberta/p/MLB2016198248\"}, {\"name\": \"iPhone XR 64 Gb Amarelo - Excelente (Recondicionado)\", \"price\": \"1050\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-xr-64-gb-amarelo-excelente-recondicionado/p/MLB2000064763\"}, {\"name\": \"iPhone Air 256\\u00a0gb - Dourado-claro - Somente Esim - Distribuidor Autorizado\", \"price\": \"10499\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-air-256gb-dourado-claro-somente-esim-distribuidor-autorizado/p/MLB1054107096\"}, {\"name\": \"Apple iPhone 12 (128 Gb) - Branco (novo Com Caixa Aberta)\", \"price\": \"2040\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-12-128-gb-branco-novo-com-caixa-aberta/p/MLB2016194594\"}, {\"name\": \"Apple iPhone 16 Preto 512 Gb - Distribuidor Autorizado\", \"price\": \"7599\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-preto-512-gb-distribuidor-autorizado/p/MLB1040287798\"}, {\"name\": \"Apple iPhone 12 (128 Gb) - Branco\", \"price\": \"2054\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-12-128-gb-branco/p/MLB16163652\"}, {\"name\": \"Apple iPhone 11 (128 Gb) - Preto - Bom (Recondicionado)\", \"price\": \"1508\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-11-128-gb-preto-bom-recondicionado/p/MLB2000074283\"}, {\"name\": \"Apple iPhone 12 (128 Gb) - Branco - Excelente (Recondicionado)\", \"price\": \"1871\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-12-128-gb-branco-excelente-recondicionado/p/MLB2000109702\"}, {\"name\": \"Apple iPhone 16 Pro (256 Gb) - Tit\\u00e2nio-preto - Distribuidor Autorizado - Distribuidor Autorizado\", \"price\": \"9999\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-256-gb-titnio-preto-distribuidor-autorizado-distribuidor-autorizado/p/MLB1053430948\"}, {\"name\": \"Apple iPhone 16 Plus (256 Gb) - Preto - Distribuidor Autorizado\", \"price\": \"8940\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-plus-256-gb-preto-distribuidor-autorizado/p/MLB1040287822\"}, {\"name\": \"Apple iPhone 16 Pro (256 Gb) - Tit\\u00e2nio Natural - Distribuidor Autorizado\", \"price\": \"9100.9\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-256-gb-titnio-natural-distribuidor-autorizado/p/MLB1040287832\"}, {\"name\": \"Apple iPhone 16 Pro Max (1 Tb) - Tit\\u00e2nio Branco - Distribuidor Autorizado\", \"price\": \"11998.92\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-max-1-tb-titnio-branco-distribuidor-autorizado/p/MLB1040287853\"}, {\"name\": \"Apple iPhone 15 (512 Gb) - Amarelo - Distribuidor Autorizado\", \"price\": \"6898.92\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-15-512-gb-amarelo-distribuidor-autorizado/p/MLB1027172674\"}, {\"name\": \"iPhone Air 512\\u00a0gb - Dourado-claro - Somente Esim - Distribuidor Autorizado\", \"price\": \"11998.92\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-air-512gb-dourado-claro-somente-esim-distribuidor-autorizado/p/MLB1054107103\"}, {\"name\": \"Apple iPhone 16 Plus (128 Gb) - Ultramarino - Distribuidor Autorizado\", \"price\": \"7643.16\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-plus-128-gb-ultramarino-distribuidor-autorizado/p/MLB1040287825\"}, {\"name\": \"iPhone 17 Pro Max 256gb - Azul-profundo\", \"price\": \"12689.9\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-17-pro-max-256gb-azul-profundo/p/MLB55308632\"}, {\"name\": \"Apple iPhone 16 Pro (512 Gb) - Tit\\u00e2nio-deserto - Distribuidor Autorizado\", \"price\": \"11000\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-512-gb-titnio-deserto-distribuidor-autorizado/p/MLB1040287837\"}, {\"name\": \"Apple iPhone 15 Pro Max (512 Gb) - Tit\\u00e2nio Preto (novo Com Caixa Aberta)\", \"price\": \"7600\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-15-pro-max-512-gb-titnio-preto-novo-com-caixa-aberta/p/MLB2016209814\"}, {\"name\": \"Apple iPhone 16 Pro (128 Gb) - Tit\\u00e2nio Natural - Distribuidor Autorizado - Distribuidor Autorizado\", \"price\": \"8221.11\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-128-gb-titnio-natural-distribuidor-autorizado-distribuidor-autorizado/p/MLB1053430960\"}, {\"name\": \"iPhone 16e (512 Gb) - Branco - Distribuidor Autorizado\", \"price\": \"5899\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-16e-512-gb-branco-distribuidor-autorizado/p/MLB1046217481\"}, {\"name\": \"Apple iPhone 11 (64 Gb) - Branco - Aceit\\u00e1vel (Recondicionado)\", \"price\": \"1340\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-11-64-gb-branco-aceitavel-recondicionado/p/MLB2000137022\"}, {\"name\": \"iPhone 8 64 Gb Cinza-espacial - Bom (Recondicionado)\", \"price\": \"588\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-8-64-gb-cinza-espacial-bom-recondicionado/p/MLB2000126300\"}, {\"name\": \"Apple iPhone 16 Pro (128 Gb) - Tit\\u00e2nio-preto\", \"price\": \"7379\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-128-gb-titnio-preto/p/MLB40287850\"}, {\"name\": \"iPhone Air 1\\u00a0tb - Branco-nuvem - Somente Esim - Distribuidor Autorizado\", \"price\": \"13499\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-air-1tb-branco-nuvem-somente-esim-distribuidor-autorizado/p/MLB1054107091\"}, {\"name\": \"iPhone XR 128 Gb Vermelho - Aceit\\u00e1vel (Recondicionado)\", \"price\": \"1188\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-xr-128-gb-vermelho-aceitavel-recondicionado/p/MLB2000112702\"}, {\"name\": \"iPhone SE (2nd Generation) 64 Gb Preto - Aceit\\u00e1vel (Recondicionado)\", \"price\": \"746\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-se-2nd-generation-64-gb-preto-aceitavel-recondicionado/p/MLB2000030335\"}, {\"name\": \"iPhone SE (3rd Generation) 128 Gb Meia-noite - Excelente (Recondicionado)\", \"price\": \"1349.25\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-se-3rd-generation-128-gb-meia-noite-excelente-recondicionado/p/MLB2000100186\"}, {\"name\": \"Apple iPhone 16 Pro (256 Gb) - Tit\\u00e2nio-deserto\", \"price\": \"8298.2\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-256-gb-titnio-deserto/p/MLB40287840\"}, {\"name\": \"Apple iPhone 16 Pro (128 Gb) - Tit\\u00e2nio-deserto - Distribuidor Autorizado - Distribuidor Autorizado\", \"price\": \"8221.11\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-pro-128-gb-titnio-deserto-distribuidor-autorizado-distribuidor-autorizado/p/MLB1053430958\"}, {\"name\": \"iPhone 12 Mini 128 Gb Azul - Bom (Recondicionado)\", \"price\": \"1956\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-12-mini-128-gb-azul-bom-recondicionado/p/MLB2000064887\"}, {\"name\": \"iPhone SE 64 Gb Cinza-espacial - Excelente (Recondicionado)\", \"price\": \"405\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-se-64-gb-cinza-espacial-excelente-recondicionado/p/MLB2000134038\"}, {\"name\": \"iPhone 8 Plus 128 Gb Dourado - Excelente (Recondicionado)\", \"price\": \"1549\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-8-plus-128-gb-dourado-excelente-recondicionado/p/MLB2000092924\"}, {\"name\": \"Apple iPhone 16 Plus (128 Gb) - Preto\", \"price\": \"6899\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/apple-iphone-16-plus-128-gb-preto/p/MLB40287827\"}, {\"name\": \"iPhone SE 32 Gb Cinza-espacial - Excelente (Recondicionado)\", \"price\": \"379\", \"store\": \"MercadoLivre\", \"currency\": \"BRL\", \"url\": \"https://www.mercadolivre.com.br/iphone-se-32-gb-cinza-espacial-excelente-recondicionado/p/MLB2000098854\"}]\n", "python_stderr": ""}
2025-10-08T23:00:56.217-0300    INFO    utils/logger.go:123     Successfully parsed Python output for MercadoLivre      {"extractor": "mercadolivre", "products_found": 50}
2025-10-08T23:00:56.217-0300    INFO    utils/logger.go:123     Extractor successfully completed product search {"search_term": "iphone", "extractor_name": "mercadolivre", "extractor_country": "BR", "results_count": 50}
2025-10-08T23:00:56.217-0300    INFO    utils/logger.go:123     Starting Magazine Luiza extraction      {"search_term": "iphone", "extractor": "magalu"}
2025-10-08T23:00:56.217-0300    INFO    utils/logger.go:123     Built Magazine Luiza search URL {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu"}
2025-10-08T23:00:56.217-0300    INFO    utils/logger.go:123     Starting HTTP request to Magazine Luiza {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "base_domain": "https://www.magazineluiza.com.br"}
2025-10-08T23:00:57.403-0300    ERROR   utils/logger.go:133     HTTP request failed for Magazine Luiza  {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "error": "Get \"https://validate.perfdrive.com/ca4df1c7abf7ea2cc50ab30bdf7ed2bb/?ssa=dcea2a78-aadc-4e50-bd60-f734a44ccc80&ssb=40124211485&ssc=https%3A%2F%2Fwww.magazineluiza.com.br%2Fbusca%2Fiphone%2F&ssi=4f8097dc-bi37-44dc-a579-e4aa0a7e3737&ssk=support@shieldsquare.com&ssm=45912509119877957104863960219516&ssn=b1070ebba83aaab86a93edb3a9a6535d29908d52df64-b883-4773-bdaace&sso=47c84513-20cae4510d2a5b1e7f06c27aaa70dec18b9e89d11234d423&ssp=71537933171759902184175997394035064&ssq=67140947525742442277775257683753505129164&ssr=MTg5LjYyLjQ2LjE3MQ==&sst=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15&ssv=&ssw=&ssx=W10=\": http2: invalid Connection request header: [\"keep-alive\" \"keep-alive\"]"}
muambr-api/utils.(*ZapLogger).Error
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:133
muambr-api/utils.LogError
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:200
muambr-api/extractors.(*MagaluExtractor).fetchHTML
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:110
muambr-api/extractors.(*MagaluExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:67
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T23:00:57.403-0300    ERROR   utils/logger.go:133     Failed to fetch HTML from Magazine Luiza        {"url": "https://www.magazineluiza.com.br/busca/iphone/", "extractor": "magalu", "error": "Get \"https://validate.perfdrive.com/ca4df1c7abf7ea2cc50ab30bdf7ed2bb/?ssa=dcea2a78-aadc-4e50-bd60-f734a44ccc80&ssb=40124211485&ssc=https%3A%2F%2Fwww.magazineluiza.com.br%2Fbusca%2Fiphone%2F&ssi=4f8097dc-bi37-44dc-a579-e4aa0a7e3737&ssk=support@shieldsquare.com&ssm=45912509119877957104863960219516&ssn=b1070ebba83aaab86a93edb3a9a6535d29908d52df64-b883-4773-bdaace&sso=47c84513-20cae4510d2a5b1e7f06c27aaa70dec18b9e89d11234d423&ssp=71537933171759902184175997394035064&ssq=67140947525742442277775257683753505129164&ssr=MTg5LjYyLjQ2LjE3MQ==&sst=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15&ssv=&ssw=&ssx=W10=\": http2: invalid Connection request header: [\"keep-alive\" \"keep-alive\"]"}
muambr-api/utils.(*ZapLogger).Error
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:133
muambr-api/utils.LogError
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:200
muambr-api/extractors.(*MagaluExtractor).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/extractors/magalu_extractor.go:69
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:91
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T23:00:57.404-0300    WARN    utils/logger.go:128     Extractor failed during product search - continuing with remaining extractors    {"search_term": "iphone", "extractor_name": "magalu", "extractor_country": "BR", "base_country": "BR", "target_currency": "BRL", "error": "Get \"https://validate.perfdrive.com/ca4df1c7abf7ea2cc50ab30bdf7ed2bb/?ssa=dcea2a78-aadc-4e50-bd60-f734a44ccc80&ssb=40124211485&ssc=https%3A%2F%2Fwww.magazineluiza.com.br%2Fbusca%2Fiphone%2F&ssi=4f8097dc-bi37-44dc-a579-e4aa0a7e3737&ssk=support@shieldsquare.com&ssm=45912509119877957104863960219516&ssn=b1070ebba83aaab86a93edb3a9a6535d29908d52df64-b883-4773-bdaace&sso=47c84513-20cae4510d2a5b1e7f06c27aaa70dec18b9e89d11234d423&ssp=71537933171759902184175997394035064&ssq=67140947525742442277775257683753505129164&ssr=MTg5LjYyLjQ2LjE3MQ==&sst=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15&ssv=&ssw=&ssx=W10=\": http2: invalid Connection request header: [\"keep-alive\" \"keep-alive\"]"}
muambr-api/utils.(*ZapLogger).Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:128
muambr-api/utils.Warn
        /Users/dennismerli/Documents/Projects/muambr-goapi/utils/logger.go:196
muambr-api/handlers.(*ExtractorHandler).GetProductComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/extractor_handler.go:94
muambr-api/handlers.(*ComparisonHandler).GetComparisons
        /Users/dennismerli/Documents/Projects/muambr-goapi/handlers/comparison_handler.go:111
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
main.main.func1
        /Users/dennismerli/Documents/Projects/muambr-goapi/main.go:37
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:102
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.LoggerWithConfig.func1
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/logger.go:240
github.com/gin-gonic/gin.(*Context).Next
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/context.go:174
github.com/gin-gonic/gin.(*Engine).handleHTTPRequest
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:620
github.com/gin-gonic/gin.(*Engine).ServeHTTP
        /Users/dennismerli/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/gin.go:576
net/http.serverHandler.ServeHTTP
        /usr/local/go/src/net/http/server.go:2947
net/http.(*conn).serve
        /usr/local/go/src/net/http/server.go:1991
2025-10-08T23:00:57.404-0300    INFO    utils/logger.go:123     Product comparison search completed     {"search_term": "iphone", "base_country": "BR", "current_country": "", "target_currency": "BRL", "total_results": 50, "extractors_attempted": 2}
[GIN] 2025/10/08 - 23:00:57 | 200 |  4.198786292s |             ::1 | GET      "/api/v1/comparisons/search?name=iphone&baseCountry=BR&currency=BRL"
dennismerli@192 muambr-goapi % 	// Construct full URL using MercadoLivre direct path format: lista.mercadolivre.com.br/product-name
	fullURL := fmt.Sprintf("%s/%s", baseURL, formattedProduct)
	return fullURL, nil
}

// fetchHTML makes an HTTP GET request and returns the HTML content
func (e *MercadoLivreExtractor) fetchHTML(url string) (string, error) {
	utils.Info("Starting HTTP request to MercadoLivre", 
		utils.String("url", url),
		utils.String("extractor", "mercadolivre"),
		utils.String("base_domain", e.BaseURL()))

	// Configure anti-bot protection
	config := utils.DefaultAntiBotConfig(e.BaseURL())
	
	// Make request with anti-bot protection
	resp, err := utils.MakeAntiBotRequest(url, config)
	if err != nil {
		utils.LogError("HTTP request execution failed for MercadoLivre - possible anti-bot protection", 
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.String("base_domain", e.BaseURL()),
			utils.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Warn("HTTP request returned non-200 status code from MercadoLivre - possible anti-bot protection",
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.String("base_domain", e.BaseURL()),
			utils.Int("status_code", resp.StatusCode),
			utils.String("status", resp.Status))
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	utils.Info("Successfully received HTTP response from MercadoLivre",
		utils.String("url", url),
		utils.String("extractor", "mercadolivre"),
		utils.Int("status_code", resp.StatusCode),
		utils.Any("content_length", resp.ContentLength),
		utils.String("content_type", resp.Header.Get("Content-Type")),
		utils.String("content_encoding", resp.Header.Get("Content-Encoding")))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LogError("Failed to read response body from MercadoLivre", 
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.Error(err))
		return "", err
	}

	// Check content encoding and decompress accordingly
	var finalContent []byte
	contentEncoding := resp.Header.Get("Content-Encoding")
	
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		// Content is gzip compressed, decompress it
		utils.Info("Detected gzip compressed content from MercadoLivre, decompressing",
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.Int("compressed_size_bytes", len(body)))
		
		gzipReader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			utils.LogError("Failed to create gzip reader for MercadoLivre response", 
				utils.String("url", url),
				utils.String("extractor", "mercadolivre"),
				utils.Error(err))
			return "", err
		}
		defer gzipReader.Close()
		
		finalContent, err = io.ReadAll(gzipReader)
		if err != nil {
			utils.LogError("Failed to decompress gzip content from MercadoLivre", 
				utils.String("url", url),
				utils.String("extractor", "mercadolivre"),
				utils.Error(err))
			return "", err
		}
		
		utils.Info("Successfully decompressed gzip content from MercadoLivre",
			utils.String("url", url),
			utils.String("extractor", "mercadolivre"),
			utils.Int("compressed_size_bytes", len(body)),
			utils.Int("decompressed_size_bytes", len(finalContent)))
	} else {
		// Content is not compressed
		finalContent = body
		
		// Log first few bytes for debugging if needed
		if len(body) > 2 {
			inspectBytes := 10
			if len(body) < inspectBytes {
				inspectBytes = len(body)
			}
			utils.Info("Response body first bytes inspection for MercadoLivre",
				utils.String("url", url),
				utils.String("extractor", "mercadolivre"),
				utils.String("content_encoding", contentEncoding),
				utils.String("first_bytes_hex", fmt.Sprintf("%x", body[:inspectBytes])))
		}
	}

	utils.Info("Successfully fetched HTML content from MercadoLivre",
		utils.String("url", url),
		utils.String("extractor", "mercadolivre"),
		utils.Int("content_size_bytes", len(finalContent)))

	return string(finalContent), nil
}

// extractWithPython calls the Python script to extract product data from HTML
func (e *MercadoLivreExtractor) extractWithPython(htmlContent string) ([]models.ProductComparison, error) {
	utils.Info("Starting Python extraction for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.Int("html_size_bytes", len(htmlContent)))

	// Get the absolute path to the Python script
	scriptPath, err := filepath.Abs("extractors/pythonExtractors/mercadolivre_page.py")
	if err != nil {
		utils.LogError("Failed to get Python script path for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.Error(err))
		return nil, fmt.Errorf("failed to get script path: %w", err)
	}

	// Get Python path from environment or use default
	pythonPath := os.Getenv("PYTHON_PATH")
	if pythonPath == "" {
		// Check if we're in development with a virtual environment
		venvPath := "/Users/dennismerli/Documents/Projects/muambr-goapi/.venv/bin/python"
		if _, err := os.Stat(venvPath); err == nil {
			pythonPath = venvPath
		} else {
			pythonPath = "python3" // Default for production environments like Render
		}
	}

	utils.Info("Using Python interpreter for MercadoLivre extraction",
		utils.String("extractor", "mercadolivre"),
		utils.String("python_path", pythonPath),
		utils.String("script_path", scriptPath))

	// Create a temporary file to store the HTML content to avoid "argument list too long" error
	tempFile, err := os.CreateTemp("", "mercadolivre_html_*.html")
	if err != nil {
		utils.LogError("Failed to create temporary file for MercadoLivre HTML",
			utils.String("extractor", "mercadolivre"),
			utils.Error(err))
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file
	defer tempFile.Close()

	// Write HTML content to temporary file
	if _, err := tempFile.WriteString(htmlContent); err != nil {
		utils.LogError("Failed to write HTML to temporary file for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.String("temp_file", tempFile.Name()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to write HTML to temp file: %w", err)
	}
	tempFile.Close() // Close file so Python can read it

	utils.Info("Created temporary HTML file for MercadoLivre extraction",
		utils.String("extractor", "mercadolivre"),
		utils.String("temp_file", tempFile.Name()))

	// Get current working directory for local packages
	workDir, _ := os.Getwd()
	
	// Prepare the Python command using the temporary file
	cmd := exec.Command(pythonPath, "-c", fmt.Sprintf(`
import sys
sys.path.append('%s')
sys.path.append('%s')
try:
    from mercadolivre_page import extract_mercadolivre_products
    import json
    
    with open('%s', 'r', encoding='utf-8') as f:
        html_content = f.read()
    products = extract_mercadolivre_products(html_content)
    print(json.dumps(products))
except ImportError as e:
    print('{"error": "Missing Python dependencies", "details": "' + str(e) + '"}')
except Exception as e:
    print('{"error": "Python extraction failed", "details": "' + str(e) + '"}')
`, filepath.Dir(scriptPath), filepath.Join(workDir, "python_packages"), tempFile.Name()))

	// Execute the Python script
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	utils.Info("Executing Python extraction script for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.String("temp_file", tempFile.Name()))

	err = cmd.Run()
	if err != nil {
		utils.LogError("Python script execution failed for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.String("python_path", pythonPath),
			utils.String("stderr", stderr.String()),
			utils.String("stdout", out.String()),
			utils.Error(err))
		return nil, fmt.Errorf("Python script execution failed: %w, stderr: %s", err, stderr.String())
	}

	utils.Info("Python script executed successfully for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.String("python_stdout", out.String()),
		utils.String("python_stderr", stderr.String()))

	// Check if Python output contains error messages
	outputStr := out.String()
	if strings.Contains(outputStr, `"error":`) {
		var errorResp map[string]interface{}
		if json.Unmarshal(out.Bytes(), &errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				utils.Warn("Python script returned error for MercadoLivre",
					utils.String("extractor", "mercadolivre"),
					utils.String("error_type", errorMsg),
					utils.String("error_details", utils.GetString(errorResp["details"])))
				return []models.ProductComparison{}, nil // Return empty results instead of failing
			}
		}
	}

	// Parse the JSON output from Python
	var pythonProducts []map[string]interface{}
	err = json.Unmarshal(out.Bytes(), &pythonProducts)
	if err != nil {
		utils.LogError("Failed to parse Python JSON output for MercadoLivre",
			utils.String("extractor", "mercadolivre"),
			utils.String("python_output", out.String()),
			utils.Error(err))
		return nil, fmt.Errorf("failed to parse Python output: %w", err)
	}

	utils.Info("Successfully parsed Python output for MercadoLivre",
		utils.String("extractor", "mercadolivre"),
		utils.Int("products_found", len(pythonProducts)))

	// Convert to ProductComparison structs
	var comparisons []models.ProductComparison
	for _, product := range pythonProducts {
		// Get currency with fallback to extractor's country currency
		currency := utils.GetString(product["currency"])
		if currency == "" {
			currency = e.countryCode.GetCurrencyCode()
		}
		
		// Parse price as float64
		priceStr := utils.GetString(product["price"])
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			// Skip invalid price entries
			continue
		}

		// Generate unique UUID
		id := uuid.New().String()
		
		// Extract store URL safely
		storeURL := utils.GetString(product["url"])
		var storeURLPtr *string
		if storeURL != "" {
			storeURLPtr = &storeURL
		}
		
		comparison := models.ProductComparison{
			ID:          id,
			ProductName: utils.GetString(product["name"]),
			Price:       price,
			Currency:    currency,
			StoreName:   utils.GetString(product["store"]),
			StoreURL:    storeURLPtr,
			Country:     string(e.countryCode),
		}
		comparisons = append(comparisons, comparison)
	}

	return comparisons, nil
}
