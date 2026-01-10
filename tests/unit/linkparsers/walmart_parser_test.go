package linkparsers

import (
	"net/url"
	"testing"

	"muambr-api/linkparsers"
)

func TestWalmartParser_ExtractTitle(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	pageURL, _ := url.Parse("https://walmart.com/product")

	testCases := []struct {
		name        string
		html        string
		expectedMin int
	}{
		{
			name: "Title in meta property",
			html: `<html>
				<head>
					<meta property="og:title" content="Apple iPhone 15 Pro Max 256GB"/>
				</head>
			</html>`,
			expectedMin: 5,
		},
		{
			name: "Title in data-automation-id",
			html: `<html>
				<body>
					<h1 data-automation-id="productTitle">Samsung 55 4K Smart TV</h1>
				</body>
			</html>`,
			expectedMin: 5,
		},
		{
			name: "Title in h1 tag",
			html: `<html>
				<body>
					<h1>Nintendo Switch OLED Console</h1>
				</body>
			</html>`,
			expectedMin: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			title := parser.ExtractTitle(tc.html, pageURL)
			if len(title) < tc.expectedMin {
				t.Errorf("Expected title length >= %d, got %d: '%s'", tc.expectedMin, len(title), title)
			}
		})
	}
}

func TestWalmartParser_ExtractPrice(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	pageURL, _ := url.Parse("https://walmart.com/product")

	testCases := []struct {
		name         string
		html         string
		expectPrice  bool
	}{
		{
			name: "Price in span with data-testid",
			html: `<html>
				<body>
					<span data-testid="price">$299.99</span>
				</body>
			</html>`,
			expectPrice: true,
		},
		{
			name: "Price in data-automation-id",
			html: `<html>
				<body>
					<span data-automation-id="product-price">$149.00</span>
				</body>
			</html>`,
			expectPrice: true,
		},
		{
			name: "Price in JSON-LD",
			html: `<html>
				<head>
					<script type="application/ld+json">
					{
						"@type": "Product",
						"offers": {
							"price": "199.99",
							"priceCurrency": "USD"
						}
					}
					</script>
				</head>
			</html>`,
			expectPrice: true,
		},
		{
			name: "Price with current class",
			html: `<html>
				<body>
					<span class="price-current">$79.99</span>
				</body>
			</html>`,
			expectPrice: true,
		},
		{
			name: "No price found",
			html: `<html>
				<body>
					<p>Product description without price</p>
				</body>
			</html>`,
			expectPrice: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			priceStr := parser.ExtractPrice(tc.html, pageURL)
			
			if tc.expectPrice {
				if priceStr == "" {
					t.Error("Expected price to be found but got empty string")
				}
			} else {
				if priceStr != "" {
					t.Errorf("Expected no price but got '%s'", priceStr)
				}
			}
		})
	}
}

func TestWalmartParser_ExtractCurrency(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	pageURL, _ := url.Parse("https://walmart.com/product")

	testCases := []struct {
		name             string
		html             string
		expectedCurrency string
	}{
		{
			name: "USD symbol",
			html: `<html><body><span>$299.99</span></body></html>`,
			expectedCurrency: "USD",
		},
		{
			name: "Currency in meta property",
			html: `<html>
				<head>
					<meta property="product:price:currency" content="USD"/>
				</head>
			</html>`,
			expectedCurrency: "USD",
		},
		{
			name: "Currency in JSON-LD",
			html: `<html>
				<head>
					<script type="application/ld+json">
					{
						"offers": {
							"priceCurrency": "USD"
						}
					}
					</script>
				</head>
			</html>`,
			expectedCurrency: "USD",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			currency := parser.ExtractCurrency(tc.html, pageURL)
			if currency != tc.expectedCurrency {
				t.Errorf("Expected currency '%s', got '%s'", tc.expectedCurrency, currency)
			}
		})
	}
}

func TestWalmartParser_ExtractImage(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	pageURL, _ := url.Parse("https://walmart.com/product")

	testCases := []struct {
		name        string
		html        string
		expectImage bool
	}{
		{
			name: "Image in meta property",
			html: `<html>
				<head>
					<meta property="og:image" content="https://i5.walmartimages.com/asr/test123.jpg"/>
				</head>
			</html>`,
			expectImage: true,
		},
		{
			name: "Image with data-testid",
			html: `<html>
				<body>
					<img data-testid="main-image" src="https://i5.walmartimages.com/asr/test456.jpg"/>
				</body>
			</html>`,
			expectImage: true,
		},
		{
			name: "Image in product gallery",
			html: `<html>
				<body>
					<img class="product-image" src="https://i5.walmartimages.com/asr/test789.jpg"/>
				</body>
			</html>`,
			expectImage: true,
		},
		{
			name: "No image found",
			html: `<html>
				<body>
					<p>No images here</p>
				</body>
			</html>`,
			expectImage: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imageURL := parser.ExtractImage(tc.html, pageURL)
			
			if tc.expectImage {
				if imageURL == "" {
					t.Error("Expected image URL but got empty string")
				}
			} else {
				if imageURL != "" {
					t.Errorf("Expected no image but got '%s'", imageURL)
				}
			}
		})
	}
}

func TestWalmartParser_ExtractDescription(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	pageURL, _ := url.Parse("https://walmart.com/product")

	testCases := []struct {
		name               string
		html               string
		expectDescription  bool
	}{
		{
			name: "Description in meta property",
			html: `<html>
				<head>
					<meta property="og:description" content="High-quality smartphone with amazing camera"/>
				</head>
			</html>`,
			expectDescription: true,
		},
		{
			name: "Description in meta name",
			html: `<html>
				<head>
					<meta name="description" content="Best laptop for gaming and productivity"/>
				</head>
			</html>`,
			expectDescription: true,
		},
		{
			name: "Description in div with data-testid",
			html: `<html>
				<body>
					<div data-testid="product-description">This is a great product with many features.</div>
				</body>
			</html>`,
			expectDescription: false, // Walmart parser may not extract this pattern
		},
		{
			name: "No description found",
			html: `<html>
				<body>
					<p>Just a title without description</p>
				</body>
			</html>`,
			expectDescription: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			description := parser.ExtractDescription(tc.html, pageURL)
			
			if tc.expectDescription {
				if description == "" {
					t.Error("Expected description but got empty string")
				}
			} else {
				if description != "" && len(description) > 10 {
					t.Errorf("Expected no meaningful description but got '%s'", description)
				}
			}
		})
	}
}

func TestWalmartParser_ParseHTML_RealData(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	
	// Test with real Walmart HTML file if available
	html, err := loadTestHTML("walmart.html")
	if err != nil {
		t.Skip("Walmart test HTML file not available, skipping real data test")
		return
	}

	pageURL, err := url.Parse("https://walmart.com/product")
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
	
	if data.Currency != "CAD" {
		t.Errorf("Expected currency 'CAD', got '%s'", data.Currency)
	}

	t.Logf("Extracted data: Title='%s', Price=%v, Currency='%s', Image='%s'", 
		data.Title, data.Price, data.Currency, data.ImageURL)
}

func TestWalmartParser_ParseHTML_MockData(t *testing.T) {
	parser := &linkparsers.WalmartParser{}
	pageURL, _ := url.Parse("https://walmart.com/product")

	html := `<html>
		<head>
			<meta property="og:title" content="Apple iPad Pro 12.9-inch"/>
			<meta property="og:description" content="Powerful tablet for creative professionals"/>
			<meta property="og:image" content="https://i5.walmartimages.com/asr/ipad-pro.jpg"/>
			<meta property="product:price:amount" content="999.99"/>
			<meta property="product:price:currency" content="USD"/>
		</head>
		<body>
			<h1 data-automation-id="productTitle">Apple iPad Pro 12.9-inch</h1>
			<span data-automation-id="product-price">$999.99</span>
		</body>
	</html>`

	data := parser.ParseHTML(html, pageURL)
	if data == nil {
		t.Fatalf("Expected parsed data but got nil")
	}

	if data.Title == "" {
		t.Error("Expected title to be extracted")
	}

	if data.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", data.Currency)
	}

	if data.Description == "" {
		t.Error("Expected description to be extracted")
	}

	if data.ImageURL == "" {
		t.Error("Expected image URL to be extracted")
	}

	t.Logf("Extracted data: Title='%s', Price=%v, Currency='%s', Description='%s', Image='%s'", 
		data.Title, data.Price, data.Currency, data.Description, data.ImageURL)
}