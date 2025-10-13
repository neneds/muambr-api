package htmlparser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseAllHTMLFiles parses all HTML files and generates a comprehensive summary
func TestParseAllHTMLFiles(t *testing.T) {
	// HTML files with their expected URLs
	testFiles := map[string]string{
		"amazon_es_ref_lp_17328039031_1_2_7c0a6edd.html":                       "https://amazon.es/product",
		"amazon.com.br.html":                                                    "https://amazon.com.br/product",
		"cashconverters_pt_ipad-_28wi-fi_29-_28a2602_29-6_4b6da721.html":      "https://cashconverters.pt/product",
		"fnac.html":                                                             "https://fnac.pt/product",
		"loja_electrolux.html":                                                  "https://electrolux.com.br/product",
		"magazineluiza_com_br_nass_0f0d181b.html":                              "https://magazineluiza.com.br/product",
		"olx_br.html":                                                           "https://olx.br/product",
		"olx_pt_iphone-16-pro-max-256-gb-IDJ2Y_58a0707c.html":                  "https://olx.pt/product",
		"primark.html":                                                          "https://primark.com/product",
		"primor_eu_calvin-klein-ck-one-colonia-un_c4dbb5af.html":               "https://primor.eu/product",
		"produto_mercadolivre_com_br_MLB-3237298873-mochila-basic-o_3a6fe9e8.html": "https://mercadolivre.com.br/product",
		"worten.html":                                                           "https://worten.pt/product",
		"zara_com_seoul-edt-90-ml--3-04-fl--oz--_940553b0.html":                "https://zara.com/product",
	}

	results := make(map[string]*ParseResult)
	
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("HTML PARSER COMPREHENSIVE TEST RESULTS")
	fmt.Println(strings.Repeat("=", 100))

	for filename, urlStr := range testFiles {
		fmt.Printf("\n📄 Testing: %s\n", filename)
		fmt.Println(strings.Repeat("-", 100))

		// Load HTML
		html, err := loadTestHTMLFromMocks(filename)
		if err != nil {
			t.Logf("❌ Failed to load %s: %v", filename, err)
			results[filename] = &ParseResult{
				Filename: filename,
				URL:      urlStr,
				Error:    err.Error(),
			}
			continue
		}

		// Parse URL
		pageURL, _ := url.Parse(urlStr)
		
		// Get parser type
		parser := ParserForURL(pageURL)
		parserType := getParserType(parser)
		
		// Parse HTML
		data := ParseHTML(html, pageURL)

		// Create result
		result := &ParseResult{
			Filename:    filename,
			URL:         urlStr,
			ParserType:  parserType,
			ProductData: data,
		}
		results[filename] = result

		// Print results
		fmt.Printf("🔧 Parser: %s\n", parserType)
		fmt.Printf("📝 Title: %s\n", result.Status("Title", data.Title))
		fmt.Printf("💰 Price: %s\n", result.Status("Price", data.Price))
		fmt.Printf("💱 Currency: %s\n", result.Status("Currency", data.Currency))
		fmt.Printf("🖼️  Image: %s\n", result.Status("Image", data.ImageURL))
		fmt.Printf("📋 Description: %s\n", result.Status("Description", data.Description))
	}

	// Generate Summary
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("SUMMARY STATISTICS")
	fmt.Println(strings.Repeat("=", 100))

	stats := generateStats(results)
	fmt.Printf("\n📊 Total Files: %d\n", stats.TotalFiles)
	fmt.Printf("✅ Successfully Parsed: %d\n", stats.SuccessCount)
	fmt.Printf("❌ Failed: %d\n", stats.FailedCount)
	fmt.Printf("\n📝 Title Extraction: %d/%d (%.1f%%)\n", 
		stats.TitleCount, stats.TotalFiles, stats.TitlePercent)
	fmt.Printf("💰 Price Extraction: %d/%d (%.1f%%)\n", 
		stats.PriceCount, stats.TotalFiles, stats.PricePercent)
	fmt.Printf("💱 Currency Detection: %d/%d (%.1f%%)\n", 
		stats.CurrencyCount, stats.TotalFiles, stats.CurrencyPercent)
	fmt.Printf("🖼️  Image Extraction: %d/%d (%.1f%%)\n", 
		stats.ImageCount, stats.TotalFiles, stats.ImagePercent)
	fmt.Printf("📋 Description Extraction: %d/%d (%.1f%%)\n", 
		stats.DescriptionCount, stats.TotalFiles, stats.DescriptionPercent)

	// Parser Type Summary
	fmt.Println("\n" + strings.Repeat("-", 100))
	fmt.Println("PARSER TYPE DISTRIBUTION")
	fmt.Println(strings.Repeat("-", 100))
	for parserType, count := range stats.ParserTypes {
		fmt.Printf("  %s: %d files\n", parserType, count)
	}

	// What's Working Well
	fmt.Println("\n" + strings.Repeat("-", 100))
	fmt.Println("✅ WHAT'S WORKING WELL")
	fmt.Println(strings.Repeat("-", 100))
	for _, site := range stats.WorkingWell {
		fmt.Printf("  • %s\n", site)
	}

	// What Needs Improvement
	fmt.Println("\n" + strings.Repeat("-", 100))
	fmt.Println("⚠️  WHAT NEEDS IMPROVEMENT")
	fmt.Println(strings.Repeat("-", 100))
	for _, site := range stats.NeedsImprovement {
		fmt.Printf("  • %s\n", site)
	}

	// Export detailed JSON for analysis
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("DETAILED RESULTS (JSON)")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println(string(jsonData))
}

// ParseResult holds the result of parsing a single HTML file
type ParseResult struct {
	Filename    string                `json:"filename"`
	URL         string                `json:"url"`
	ParserType  string                `json:"parserType"`
	ProductData *ParsedProductData    `json:"productData,omitempty"`
	Error       string                `json:"error,omitempty"`
}

// Status returns a formatted status string for a field
func (r *ParseResult) Status(fieldName string, value interface{}) string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return fmt.Sprintf("❌ Not extracted")
		}
		if len(v) > 80 {
			return fmt.Sprintf("✅ %s... (%d chars)", v[:80], len(v))
		}
		return fmt.Sprintf("✅ %s", v)
	case *float64:
		if v == nil {
			return fmt.Sprintf("❌ Not extracted")
		}
		return fmt.Sprintf("✅ %.2f", *v)
	default:
		return fmt.Sprintf("❌ Not extracted")
	}
}

// Stats holds parsing statistics
type Stats struct {
	TotalFiles         int
	SuccessCount       int
	FailedCount        int
	TitleCount         int
	TitlePercent       float64
	PriceCount         int
	PricePercent       float64
	CurrencyCount      int
	CurrencyPercent    float64
	ImageCount         int
	ImagePercent       float64
	DescriptionCount   int
	DescriptionPercent float64
	ParserTypes        map[string]int
	WorkingWell        []string
	NeedsImprovement   []string
}

func generateStats(results map[string]*ParseResult) *Stats {
	stats := &Stats{
		TotalFiles:  len(results),
		ParserTypes: make(map[string]int),
	}

	for filename, result := range results {
		if result.Error != "" {
			stats.FailedCount++
			stats.NeedsImprovement = append(stats.NeedsImprovement, 
				fmt.Sprintf("%s: Failed to load", filename))
			continue
		}

		stats.SuccessCount++
		stats.ParserTypes[result.ParserType]++

		data := result.ProductData
		successCount := 0
		totalFields := 5.0

		// Count successful extractions
		if data.Title != "" {
			stats.TitleCount++
			successCount++
		}
		if data.Price != nil {
			stats.PriceCount++
			successCount++
		}
		if data.Currency != "" {
			stats.CurrencyCount++
			successCount++
		}
		if data.ImageURL != "" {
			stats.ImageCount++
			successCount++
		}
		if data.Description != "" {
			stats.DescriptionCount++
			successCount++
		}

		// Categorize results
		successRate := float64(successCount) / totalFields
		siteName := strings.TrimSuffix(filename, filepath.Ext(filename))
		
		if successRate >= 0.6 { // 60% or more fields extracted
			stats.WorkingWell = append(stats.WorkingWell, 
				fmt.Sprintf("%s: %d/%d fields (%.0f%%)", siteName, successCount, int(totalFields), successRate*100))
		} else {
			missingFields := []string{}
			if data.Title == "" {
				missingFields = append(missingFields, "title")
			}
			if data.Price == nil {
				missingFields = append(missingFields, "price")
			}
			if data.Currency == "" {
				missingFields = append(missingFields, "currency")
			}
			if data.ImageURL == "" {
				missingFields = append(missingFields, "image")
			}
			if data.Description == "" {
				missingFields = append(missingFields, "description")
			}
			
			stats.NeedsImprovement = append(stats.NeedsImprovement,
				fmt.Sprintf("%s: Missing %s", siteName, strings.Join(missingFields, ", ")))
		}
	}

	// Calculate percentages
	if stats.TotalFiles > 0 {
		stats.TitlePercent = float64(stats.TitleCount) / float64(stats.TotalFiles) * 100
		stats.PricePercent = float64(stats.PriceCount) / float64(stats.TotalFiles) * 100
		stats.CurrencyPercent = float64(stats.CurrencyCount) / float64(stats.TotalFiles) * 100
		stats.ImagePercent = float64(stats.ImageCount) / float64(stats.TotalFiles) * 100
		stats.DescriptionPercent = float64(stats.DescriptionCount) / float64(stats.TotalFiles) * 100
	}

	return stats
}

// loadTestHTMLFromMocks loads HTML from the mocks directory
func loadTestHTMLFromMocks(filename string) (string, error) {
	path := filepath.Join("..", "tests", "mocks", "linkparserpages", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
