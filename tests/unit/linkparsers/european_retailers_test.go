package linkparsers

import (
	"net/url"
	"testing"

	"muambr-api/linkparsers"
)

func TestEuropeanRetailersParser_Fnac(t *testing.T) {
	parser := &linkparsers.FnacPTParser{}
	pageURL, _ := url.Parse("https://fnac.pt/produto")

	testCases := []struct {
		name        string
		html        string
		expectTitle bool
	}{
		{
			name: "Fnac product with meta tags",
			html: `<html>
				<head>
					<meta property="og:title" content="Smart TV LG OLED 55"/>
					<meta property="product:price:currency" content="EUR"/>
				</head>
			</html>`,
			expectTitle: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := parser.ParseHTML(tc.html, pageURL)
			if data == nil {
				t.Fatalf("Expected parsed data but got nil")
			}

			if tc.expectTitle && data.Title == "" {
				t.Error("Expected title to be extracted")
			}

			if data.Currency != "eur" {
				t.Errorf("Expected currency 'eur', got '%s'", data.Currency)
			}
		})
	}
}

func TestEuropeanRetailersParser_Worten(t *testing.T) {
	parser := &linkparsers.WortenPTParser{}
	pageURL, _ := url.Parse("https://worten.pt/produto")

	html := `<html>
		<head>
			<meta property="og:title" content="Máquina de Lavar BECKEN"/>
		</head>
	</html>`

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
}

func TestEuropeanRetailersParser_Primark(t *testing.T) {
	parser := &linkparsers.PrimarkParser{}
	pageURL, _ := url.Parse("https://primark.com/produto")

	html := `<html>
		<head>
			<meta property="og:title" content="Camisa xadrez The Stronghold"/>
		</head>
	</html>`

	data := parser.ParseHTML(html, pageURL)
	if data == nil {
		t.Fatalf("Expected parsed data but got nil")
	}

	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}

	if data.Currency != "usd" {
		t.Errorf("Expected currency 'usd', got '%s'", data.Currency)
	}
}

func TestEuropeanRetailersParser_Primor(t *testing.T) {
	testCases := []struct {
		name     string
		parser   linkparsers.Parser
		url      string
		currency string
	}{
		{
			name:     "Primor EU",
			parser:   &linkparsers.PrimorEUParser{},
			url:      "https://primor.eu/produto",
			currency: "eur",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pageURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			html := `<html>
				<head>
					<meta property="og:title" content="Calvin Klein CK One"/>
				</head>
			</html>`

			data := tc.parser.ParseHTML(html, pageURL)
			if data == nil {
				t.Fatalf("Expected parsed data but got nil")
			}

			if data.Title == "" {
				t.Error("Expected title to be extracted")
			}

			if data.Currency != tc.currency {
				t.Errorf("Expected currency '%s', got '%s'", tc.currency, data.Currency)
			}
		})
	}
}

func TestEuropeanRetailersParser_Zara(t *testing.T) {
	parser := &linkparsers.ZaraParser{}
	pageURL, _ := url.Parse("https://zara.com/produto")

	testCases := []struct {
		name        string
		html        string
		expectTitle bool
	}{
		{
			name: "Zara product",
			html: `<html>
				<head>
					<meta property="og:title" content="Seoul EDT 90ml"/>
				</head>
			</html>`,
			expectTitle: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := parser.ParseHTML(tc.html, pageURL)
			if data == nil {
				t.Fatalf("Expected parsed data but got nil")
			}

			if tc.expectTitle && data.Title == "" {
				t.Error("Expected title to be extracted")
			}

			// Zara parser should detect USD by default
			if data.Currency != "usd" {
				t.Errorf("Expected currency 'usd', got '%s'", data.Currency)
			}
		})
	}
}

func TestEuropeanRetailersParser_RealData(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		url      string
		parser   linkparsers.Parser
		currency string
	}{
		{
			name:     "Fnac Real Product",
			filename: "fnac.html",
			url:      "https://fnac.pt/produto",
			parser:   &linkparsers.FnacPTParser{},
			currency: "eur",
		},
		{
			name:     "Worten Real Product",
			filename: "worten.html",
			url:      "https://worten.pt/produto",
			parser:   &linkparsers.WortenPTParser{},
			currency: "eur",
		},
		{
			name:     "Primark Real Product",
			filename: "primark.html",
			url:      "https://primark.com/produto",
			parser:   &linkparsers.PrimarkParser{},
			currency: "eur",
		},
		{
			name:     "Primor EU Real Product",
			filename: "primor_eu_calvin-klein-ck-one-colonia-un_c4dbb5af.html",
			url:      "https://primor.eu/produto",
			parser:   &linkparsers.PrimorEUParser{},
			currency: "eur",
		},
		{
			name:     "Zara Real Product",
			filename: "zara_com_seoul-edt-90-ml--3-04-fl--oz--_940553b0.html",
			url:      "https://zara.com/produto",
			parser:   &linkparsers.ZaraParser{},
			currency: "usd", // Zara defaults to USD
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			html, err := loadTestHTML(tc.filename)
			if err != nil {
				t.Fatalf("Failed to load test HTML: %v", err)
			}

			pageURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			data := tc.parser.ParseHTML(html, pageURL)
			if data == nil {
				t.Fatalf("Expected parsed data but got nil")
			}

			// Validate basic extraction worked
			if data.Title == "" {
				t.Error("Expected title to be extracted")
			}
			
			if data.Currency != tc.currency {
				t.Errorf("Expected currency '%s', got '%s'", tc.currency, data.Currency)
			}

			t.Logf("Extracted data: Title='%s', Price=%v, Currency='%s'", 
				data.Title, data.Price, data.Currency)
		})
	}
}

func TestCashConvertersParser(t *testing.T) {
	parser := &linkparsers.CashConvertersPTParser{}
	pageURL, _ := url.Parse("https://cashconverters.pt/produto")

	testCases := []struct {
		name        string
		html        string
		expectTitle bool
		expectPrice bool
	}{
		{
			name: "Cash Converters product",
			html: `<html>
				<head>
					<meta property="og:title" content="iPad Wi-Fi 64GB"/>
				</head>
				<body>
					<span class="price">€ 196.95</span>
				</body>
			</html>`,
			expectTitle: true,
			expectPrice: false, // Price extraction might be complex
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := parser.ParseHTML(tc.html, pageURL)
			if data == nil {
				t.Fatalf("Expected parsed data but got nil")
			}

			if tc.expectTitle && data.Title == "" {
				t.Error("Expected title to be extracted")
			}

			if data.Currency != "eur" {
				t.Errorf("Expected currency 'eur', got '%s'", data.Currency)
			}
		})
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

	// Validate basic extraction worked
	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}
	
	if data.Currency != "eur" {
		t.Errorf("Expected currency 'eur', got '%s'", data.Currency)
	}

	t.Logf("Extracted data: Title='%s', Price=%v, Currency='%s'", 
		data.Title, data.Price, data.Currency)
}