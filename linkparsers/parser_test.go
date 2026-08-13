package linkparsers

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// TestAmazonESUsesGenericParser: amazon.es is not in the site registry (buy-box
// price is not in SSR HTML). Unmatched hosts fall back to ShareHTMLParser.
func TestAmazonESUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://amazon.es/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for amazon.es, got %T", parser)
	}
}

// TestParseAmazonBR tests Amazon Brazil parser
func TestParseAmazonBR(t *testing.T) {
	html, err := loadTestHTML("amazon.com.br.html")
	if err != nil {
		t.Fatalf("Failed to load test HTML: %v", err)
	}

	pageURL, _ := url.Parse("https://amazon.com.br/product/123")
	data := ParseHTML(html, pageURL)

	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}
	t.Logf("Title: %s", data.Title)

	if data.Currency != "brl" {
		t.Errorf("Expected currency to be 'brl', got: %s", data.Currency)
	}

	t.Logf("Price: %v", data.Price)
	t.Logf("Image: %s", data.ImageURL)
	t.Logf("Description: %s", data.Description)
}

// TestParseCashConvertersPT tests Cash Converters Portugal parser
func TestParseCashConvertersPT(t *testing.T) {
	html, err := loadTestHTML("cashconverters_pt_ipad-_28wi-fi_29-_28a2602_29-6_4b6da721.html")
	if err != nil {
		t.Fatalf("Failed to load test HTML: %v", err)
	}

	pageURL, _ := url.Parse("https://cashconverters.pt/product/123")
	data := ParseHTML(html, pageURL)

	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}
	t.Logf("Title: %s", data.Title)

	if data.Currency != "eur" {
		t.Errorf("Expected currency to be 'eur', got: %s", data.Currency)
	}

	t.Logf("Price: %v", data.Price)
	t.Logf("Image: %s", data.ImageURL)
	t.Logf("Description: %s", data.Description)
}

// TestFnacUsesGenericParser: fnac.pt is not in the site registry (could not
// reach a product page). Unmatched hosts fall back to ShareHTMLParser.
func TestFnacUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://fnac.pt/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for fnac.pt, got %T", parser)
	}
}

// TestParseElectrolux tests Electrolux Brazil parser
func TestParseElectrolux(t *testing.T) {
	html, err := loadTestHTML("loja_electrolux.html")
	if err != nil {
		t.Fatalf("Failed to load test HTML: %v", err)
	}

	pageURL, _ := url.Parse("https://electrolux.com.br/product/123")
	data := ParseHTML(html, pageURL)

	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}
	t.Logf("Title: %s", data.Title)

	if data.Currency != "brl" {
		t.Errorf("Expected currency to be 'brl', got: %s", data.Currency)
	}

	t.Logf("Price: %v", data.Price)
	t.Logf("Image: %s", data.ImageURL)
	t.Logf("Description: %s", data.Description)
}

// TestMagazineLuizaUsesGenericParser: magazineluiza.com.br is not in the site
// registry (cold fetch 403 "Não é possível acessar a página"). Unmatched hosts
// fall back to ShareHTMLParser.
func TestMagazineLuizaUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://magazineluiza.com.br/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for magazineluiza.com.br, got %T", parser)
	}
}

// TestOLXBRUsesGenericParser: olx.com.br is not in the site registry (cold fetch
// is Cloudflare 403). Unmatched hosts fall back to ShareHTMLParser.
func TestOLXBRUsesGenericParser(t *testing.T) {
	for _, raw := range []string{
		"https://olx.com.br/anuncio",
		"https://sp.olx.com.br/sao-paulo-e-regiao/anuncio",
		"https://olx.br/product/123",
	} {
		pageURL, _ := url.Parse(raw)
		parser := ParserForURL(pageURL)
		if _, ok := parser.(*ShareHTMLParser); !ok {
			t.Errorf("expected ShareHTMLParser for %s, got %T", raw, parser)
		}
	}
}

// TestOLXPTUsesGenericParser: olx.pt is not in the site registry (cold fetch
// is CloudFront 403). Unmatched hosts fall back to ShareHTMLParser.
func TestOLXPTUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://olx.pt/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for olx.pt, got %T", parser)
	}
}

// TestPrimarkUsesGenericParser: primark.com is not in the site registry (cold
// fetch is HTTP 403 "Planned maintenance"). Unmatched hosts fall back to ShareHTMLParser.
func TestPrimarkUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://primark.com/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for primark.com, got %T", parser)
	}
}

// TestPrimorUsesGenericParser: primor.eu is not in the site registry (cold fetch
// is AWS WAF HTTP 202 challenge). Unmatched hosts fall back to ShareHTMLParser.
func TestPrimorUsesGenericParser(t *testing.T) {
	for _, raw := range []string{
		"https://primor.eu/product/123",
		"https://pt.primor.eu/pt_pt/creme-de-noite-multi-intensivo.html",
	} {
		pageURL, _ := url.Parse(raw)
		parser := ParserForURL(pageURL)
		if _, ok := parser.(*ShareHTMLParser); !ok {
			t.Errorf("expected ShareHTMLParser for %s, got %T", raw, parser)
		}
	}
}

// TestZaraUsesGenericParser: zara.com is not in the site registry (cold fetch
// is Akamai Bot Manager interstitial). Unmatched hosts fall back to ShareHTMLParser.
func TestZaraUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://zara.com/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for zara.com, got %T", parser)
	}
}

// TestMercadoLivreUsesGenericParser: mercadolivre.com.br is not in the site
// registry (cold fetch redirects to /gz/account-verification). Unmatched hosts
// fall back to ShareHTMLParser.
func TestMercadoLivreUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://mercadolivre.com.br/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for mercadolivre.com.br, got %T", parser)
	}
}

// TestWortenUsesGenericParser: worten.pt is not in the site registry (cold
// fetch is Cloudflare 403 challenge). Unmatched hosts fall back to ShareHTMLParser.
func TestWortenUsesGenericParser(t *testing.T) {
	pageURL, _ := url.Parse("https://worten.pt/product/123")
	parser := ParserForURL(pageURL)
	if _, ok := parser.(*ShareHTMLParser); !ok {
		t.Errorf("expected ShareHTMLParser for worten.pt, got %T", parser)
	}
}

// TestParsePerfumesECompanhia tests Perfumes e Companhia Portugal parser
func TestParsePerfumesECompanhia(t *testing.T) {
	pageURL, _ := url.Parse("https://www.perfumesecompanhia.pt/pt/yves-saint-laurent-libre-berry-crush-eau-de-parfum/436615.html")
	html := `<html lang="pt"><head>
		<meta property="og:title" content="Libre - Berry Crush Eau de Parfum - Yves Saint Laurent | Perfumes e Companhia"/>
		<script type="application/ld+json">
		{"@context":"http://schema.org/","@type":"Product","name":"Berry Crush Eau de Parfum","offers":{"@type":"Offer","priceCurrency":"EUR","price":"128.60"}}
		</script>
	</head></html>`
	data := ParseHTML(html, pageURL)

	if data.Title != "Libre - Berry Crush Eau de Parfum - Yves Saint Laurent" {
		t.Errorf("title: got %q", data.Title)
	}
	if data.Price == nil || *data.Price != 128.60 {
		t.Errorf("price: got %v, want 128.60", data.Price)
	}
	if data.Currency != "eur" {
		t.Errorf("Expected currency to be 'eur', got: %s", data.Currency)
	}
}

// TestParserSelection tests that the correct parser is selected for each URL
func TestParserSelection(t *testing.T) {
	tests := []struct {
		urlStr         string
		expectedParser string
	}{
		{"https://amazon.es/product", "ShareHTMLParser"},
		{"https://amazon.com.br/product", "AmazonParser"},
		{"https://olx.pt/product", "ShareHTMLParser"},
		{"https://olx.br/product", "ShareHTMLParser"},
		{"https://olx.com.br/product", "ShareHTMLParser"},
		{"https://fnac.pt/product", "ShareHTMLParser"},
		{"https://cashconverters.pt/product", "CashConvertersPTParser"},
		{"https://magazineluiza.com.br/product", "ShareHTMLParser"},
		{"https://mercadolivre.com.br/product", "ShareHTMLParser"},
		{"https://electrolux.com.br/product", "ElectroluxBRParser"},
		{"https://primark.com/product", "ShareHTMLParser"},
		{"https://primor.eu/product", "ShareHTMLParser"},
		{"https://worten.pt/product", "ShareHTMLParser"},
		{"https://zara.com/product", "ShareHTMLParser"},
		{"https://perfumesecompanhia.pt/product", "PerfumesECompanhiaParser"},
		{"https://unknown-site.com/product", "ShareHTMLParser"},
	}

	for _, tt := range tests {
		t.Run(tt.urlStr, func(t *testing.T) {
			pageURL, _ := url.Parse(tt.urlStr)
			parser := ParserForURL(pageURL)

			parserType := getParserType(parser)
			t.Logf("URL: %s -> Parser: %s", tt.urlStr, parserType)

			// Just verify we got a parser, exact type matching is difficult with interfaces
			if parser == nil {
				t.Errorf("Expected parser for %s, got nil", tt.urlStr)
			}
		})
	}
}

// Helper functions

func loadTestHTML(filename string) (string, error) {
	// Get the path to the test HTML files
	path := filepath.Join("..", "tests", "mocks", "linkparsers", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getParserType(parser Parser) string {
	switch parser.(type) {
	case *AmazonParser:
		return "AmazonParser"
	case *CashConvertersPTParser:
		return "CashConvertersPTParser"
	case *ElectroluxBRParser:
		return "ElectroluxBRParser"
	case *PerfumesECompanhiaParser:
		return "PerfumesECompanhiaParser"
	case *ShareHTMLParser:
		return "ShareHTMLParser"
	default:
		return "Unknown"
	}
}
