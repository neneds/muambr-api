package linkparsers

import (
	"net/url"
	"testing"

	"muambr-api/linkparsers"
)

func TestEuropeanRetailersParser_PerfumesECompanhia(t *testing.T) {
	parser := &linkparsers.PerfumesECompanhiaParser{}
	pageURL, _ := url.Parse("https://www.perfumesecompanhia.pt/pt/yves-saint-laurent-libre-berry-crush-eau-de-parfum/436615.html")

	html := `<html lang="pt">
		<head>
			<meta property="og:title" content="Libre - Berry Crush Eau de Parfum - Yves Saint Laurent | Perfumes e Companhia"/>
			<meta property="og:image" content="https://www.perfumesecompanhia.pt/on/demandware.static/-/Sites-master-catalog-pc/default/dw661556b2/images/hi-res/P01/436615_363189.jpg"/>
			<script type="application/ld+json">
			{"@context":"http://schema.org/","@type":"Product","name":"Berry Crush Eau de Parfum","description":"Yves Saint Laurent, Libre Berry Crush, Eau de Parfum, Fragrância Frutada Floral, 90ml","offers":{"url":"https://www.perfumesecompanhia.pt/pt/yves-saint-laurent-libre-berry-crush-eau-de-parfum/436615.html","@type":"Offer","priceCurrency":"EUR","price":"128.60","availability":"http://schema.org/InStock"},"image":["https://www.perfumesecompanhia.pt/dw/image/v2/BGXV_PRD/on/demandware.static/-/Sites-master-catalog-pc/default/dw661556b2/images/hi-res/P01/436615_363189.jpg?sw=677&sh=677&sm=fit"]}
			</script>
		</head>
		<body><span>10,00 €</span></body>
	</html>`

	data := parser.ParseHTML(html, pageURL)
	if data == nil {
		t.Fatal("Expected parsed data but got nil")
	}
	if data.Title != "Libre - Berry Crush Eau de Parfum - Yves Saint Laurent" {
		t.Errorf("title: got %q", data.Title)
	}
	if data.Price == nil || *data.Price != 128.60 {
		t.Errorf("price: got %v, want 128.60", data.Price)
	}
	if data.Currency != "eur" {
		t.Errorf("currency: got %q, want eur", data.Currency)
	}
	if data.ImageURL == "" {
		t.Error("expected JSON-LD or og:image URL")
	}
}

func TestCashConvertersParser(t *testing.T) {
	parser := &linkparsers.CashConvertersPTParser{}
	pageURL, _ := url.Parse("https://www.cashconverters.pt/pt/pt/segunda-mano/PT004_E475610_0.html")

	html := `<html lang="pt">
		<head>
			<meta property="og:title" content="teclado apple magic keyboard ipad pro de segunda m&atilde;o a: 205,45&euro;. Compra na Cash Converters PT004_E475610_0"/>
			<meta property="og:image" content="https://images.cashconverters.es/productslive/teclado/apple-magic-keyboard-ipad-pro_PT004_E475610-0_0.jpg?d=large"/>
			<script type="application/ld+json">
			{"@context":"http://schema.org/","@type":"Product","name":"teclado apple magic keyboard ipad pro","offers":{"@type":"Offer","priceCurrency":"EUR","price":"205.45","availability":"http://schema.org/InStock"}}
			</script>
		</head>
		<body>
			<h1>teclado apple magic keyboard ipad pro</h1>
		</body>
	</html>`

	data := parser.ParseHTML(html, pageURL)
	if data == nil {
		t.Fatal("Expected parsed data but got nil")
	}
	if data.Title != "teclado apple magic keyboard ipad pro" {
		t.Errorf("title: got %q", data.Title)
	}
	if data.Price == nil || *data.Price != 205.45 {
		t.Errorf("price: got %v, want 205.45", data.Price)
	}
	if data.Currency != "eur" {
		t.Errorf("currency: got %q, want eur", data.Currency)
	}
	if data.ImageURL == "" {
		t.Error("expected og:image URL")
	}
}

func TestCashConvertersParser_RealData(t *testing.T) {
	parser := &linkparsers.CashConvertersPTParser{}

	html, err := loadTestHTML("cashconverters_pt_ipad-_28wi-fi_29-_28a2602_29-6_4b6da721.html")
	if err != nil {
		t.Fatalf("Failed to load test HTML: %v", err)
	}

	pageURL, err := url.Parse("https://cashconverters.pt/produto")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	data := parser.ParseHTML(html, pageURL)
	if data == nil {
		t.Fatalf("Expected parsed data but got nil")
	}

	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}

	if data.Currency != "eur" {
		t.Errorf("Expected currency 'eur', got '%s'", data.Currency)
	}

	t.Logf("Extracted data: Title='%s', Price=%v, Currency='%s'",
		data.Title, data.Price, data.Currency)
}
