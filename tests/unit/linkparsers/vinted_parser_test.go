package linkparsers

import (
	"net/url"
	"testing"

	"muambr-api/linkparsers"
)

func TestVintedPTParser(t *testing.T) {
	parser := &linkparsers.VintedPTParser{}
	pageURL, _ := url.Parse("https://www.vinted.pt/items/10114305097-jbl-charge-6")
	html := `<html><head>
		<title>JBL Charge 6 | Vinted</title>
		<meta property="og:title" content="JBL Charge 6 | Vinted"/>
		<script type="application/ld+json">
		{"@type":"Product","name":"JBL Charge 6","description":"Nova com todos os manuais","image":"https://images1.vinted.net/t/example.webp","brand":{"@type":"Brand","name":"JBL"},"offers":{"@type":"Offer","priceCurrency":"EUR","price":110,"availability":"InStock"},"@context":"https://schema.org"}
		</script>
	</head><body>116,20 €</body></html>`

	data := parser.ParseHTML(html, pageURL)
	if data.Title != "JBL Charge 6" {
		t.Errorf("title: got %q", data.Title)
	}
	if data.Price == nil || *data.Price != 110 {
		t.Errorf("price: got %v, want 110", data.Price)
	}
	if data.Currency != "eur" {
		t.Errorf("currency: got %q", data.Currency)
	}
	if data.ImageURL != "https://images1.vinted.net/t/example.webp" {
		t.Errorf("image: got %q", data.ImageURL)
	}
	if data.Description != "Nova com todos os manuais" {
		t.Errorf("description: got %q", data.Description)
	}

	selected := linkparsers.ParserForURL(pageURL)
	if _, ok := selected.(*linkparsers.VintedPTParser); !ok {
		t.Fatalf("www.vinted.pt selected %T", selected)
	}
}
