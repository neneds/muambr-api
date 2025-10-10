package extractors_test

import (
	"regexp"
	"strings"
	"testing"
)

// HTMLTestHelper provides utilities for testing HTML content
type HTMLTestHelper struct {
	content string
	t       *testing.T
}

// NewHTMLTestHelper creates a new HTML test helper
func NewHTMLTestHelper(htmlContent string, t *testing.T) *HTMLTestHelper {
	return &HTMLTestHelper{
		content: htmlContent,
		t:       t,
	}
}

// ContainsElement checks if the HTML contains a specific element or text
func (h *HTMLTestHelper) ContainsElement(element string) bool {
	return strings.Contains(strings.ToLower(h.content), strings.ToLower(element))
}

// FindPrices attempts to find price-like patterns in the HTML
func (h *HTMLTestHelper) FindPrices() []string {
	var prices []string
	
	// Brazilian Real pattern (R$ 1.234,56 or R$1234,56)
	brPattern := regexp.MustCompile(`R\$\s*\d{1,3}(?:\.\d{3})*(?:,\d{2})?`)
	prices = append(prices, brPattern.FindAllString(h.content, -1)...)
	
	// Euro pattern (€1.234,56 or 1.234,56 €)
	euroPattern := regexp.MustCompile(`(?:€\s*\d{1,3}(?:\.\d{3})*(?:,\d{2})?|\d{1,3}(?:\.\d{3})*(?:,\d{2})?\s*€)`)
	prices = append(prices, euroPattern.FindAllString(h.content, -1)...)
	
	return prices
}

// FindProductTitles attempts to find product title patterns
func (h *HTMLTestHelper) FindProductTitles() []string {
	var titles []string
	
	// Look for iPad-related titles
	ipadPattern := regexp.MustCompile(`(?i)ipad[^<>]*\d+[^<>]*`)
	titles = append(titles, ipadPattern.FindAllString(h.content, -1)...)
	
	return titles
}

// LogFindings logs what was found in the HTML for debugging
func (h *HTMLTestHelper) LogFindings() {
	prices := h.FindPrices()
	titles := h.FindProductTitles()
	
	h.t.Logf("Found %d potential prices", len(prices))
	if len(prices) > 0 && len(prices) <= 5 {
		for i, price := range prices {
			if i >= 5 { // Limit output
				h.t.Logf("  ... and %d more prices", len(prices)-5)
				break
			}
			h.t.Logf("  Price %d: %s", i+1, price)
		}
	}
	
	h.t.Logf("Found %d potential iPad product titles", len(titles))
	if len(titles) > 0 && len(titles) <= 3 {
		for i, title := range titles {
			if i >= 3 { // Limit output
				h.t.Logf("  ... and %d more titles", len(titles)-3)
				break
			}
			// Truncate long titles
			if len(title) > 80 {
				title = title[:80] + "..."
			}
			h.t.Logf("  Title %d: %s", i+1, title)
		}
	}
}

// AnalyzeStructure provides a comprehensive analysis of the HTML structure
func (h *HTMLTestHelper) AnalyzeStructure() {
	// Count various HTML elements
	divCount := strings.Count(strings.ToLower(h.content), "<div")
	linkCount := strings.Count(strings.ToLower(h.content), "<a ")
	imgCount := strings.Count(strings.ToLower(h.content), "<img")
	
	h.t.Logf("HTML Structure Analysis:")
	h.t.Logf("  Total size: %d bytes", len(h.content))
	h.t.Logf("  Div elements: %d", divCount)
	h.t.Logf("  Links: %d", linkCount)
	h.t.Logf("  Images: %d", imgCount)
	
	// Check for common e-commerce patterns
	patterns := map[string]string{
		"Price containers": `(?i)(price|preco|preço|valor)`,
		"Product containers": `(?i)(product|produto|item)`,
		"Cart/Buy buttons": `(?i)(cart|carrinho|buy|comprar|adicionar)`,
		"Search results": `(?i)(result|resultado|search|busca|pesquisa)`,
	}
	
	for name, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(h.content, -1)
		h.t.Logf("  %s: %d matches", name, len(matches))
	}
}

func TestHTMLDataAnalysis(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		site     string
	}{
		{"AcharPromo", "acharpromo_ipad10_search.html", "achar.promo"},
		{"MercadoLivre", "mercadolivre_ipad10_search.html", "mercadolivre.com.br"},
		{"KuantoKusta", "kuantokusta_ipad10_search.html", "kuantokusta.pt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load the HTML content
			htmlContent, err := loadTestData(tc.filename)
			if err != nil {
				t.Skipf("Test data not available for %s: %v", tc.name, err)
				return
			}

			// Create helper and analyze
			helper := NewHTMLTestHelper(htmlContent, t)
			
			t.Logf("=== Analyzing %s HTML ===", tc.name)
			
			// Basic structure analysis
			helper.AnalyzeStructure()
			
			// Look for site-specific content
			if !helper.ContainsElement(tc.site) {
				t.Logf("⚠ Warning: Site identifier '%s' not found in HTML", tc.site)
			} else {
				t.Logf("✓ Site identifier '%s' found in HTML", tc.site)
			}
			
			// Look for iPad search results
			if !helper.ContainsElement("ipad") {
				t.Logf("⚠ Warning: 'iPad' not found in search results")
			} else {
				t.Logf("✓ 'iPad' found in search results")
			}
			
			// Log detailed findings
			helper.LogFindings()
			
			t.Logf("=== End %s Analysis ===", tc.name)
		})
	}
}

func TestHTMLDataUsability(t *testing.T) {
	t.Run("VerifyAllTestDataExists", func(t *testing.T) {
		testFiles := []string{
			"acharpromo_ipad10_search.html",
			"mercadolivre_ipad10_search.html", 
			"kuantokusta_ipad10_search.html",
		}
		
		existingFiles := 0
		for _, filename := range testFiles {
			if _, err := loadTestData(filename); err == nil {
				existingFiles++
				t.Logf("✓ %s exists and is readable", filename)
			} else {
				t.Logf("✗ %s not available: %v", filename, err)
			}
		}
		
		t.Logf("HTML Test Data Summary: %d/%d files available", existingFiles, len(testFiles))
		
		if existingFiles == 0 {
			t.Skip("No HTML test data available - run generate_test_data_v3.sh first")
		}
	})
	
	t.Run("VerifyHTMLQuality", func(t *testing.T) {
		htmlContent, err := loadTestData("acharpromo_ipad10_search.html")
		if err != nil {
			t.Skip("AcharPromo test data not available")
			return
		}
		
		// Quality checks
		helper := NewHTMLTestHelper(htmlContent, t)
		
		// Should contain basic HTML structure
		if !helper.ContainsElement("<!DOCTYPE html>") {
			t.Error("HTML missing DOCTYPE declaration")
		}
		
		if !helper.ContainsElement("<html") {
			t.Error("HTML missing html tag")
		}
		
		if !helper.ContainsElement("<head>") {
			t.Error("HTML missing head section")
		}
		
		if !helper.ContainsElement("<body") {
			t.Error("HTML missing body section")  
		}
		
		// Should not contain obvious error messages
		errorPatterns := []string{
			"404 not found",
			"500 internal server error",
			"access denied",
			"blocked",
			"captcha",
		}
		
		for _, pattern := range errorPatterns {
			if helper.ContainsElement(pattern) {
				t.Logf("⚠ Warning: Potential error pattern found: %s", pattern)
			}
		}
		
		// Should be reasonably sized (not empty, not suspiciously small)
		if len(htmlContent) < 1000 {
			t.Error("HTML content suspiciously small - might be an error page")
		}
		
		if len(htmlContent) > 10000000 { // 10MB
			t.Logf("⚠ Warning: HTML content very large (%d bytes)", len(htmlContent))
		}
		
		t.Logf("✓ HTML quality checks passed for AcharPromo test data")
	})
}