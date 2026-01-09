package main
import (
	"fmt"
	"os"
	"muambr-api/extractors"
)

func main() {
	html, _ := os.ReadFile("tests/testdata/html/amazonus.html")
	extractor := extractors.NewAmazonUSExtractor()
	comparisons, err := extractor.GetComparisonsFromHTML(string(html))
	if err != nil {
		fmt.Printf("Error: %v
", err)
		return
	}
	fmt.Printf("✅ Amazon US extraction working! Found %d products
", len(comparisons))
	if len(comparisons) > 0 {
		fmt.Printf("Sample: %s - $%.2f
", comparisons[0].ProductName, comparisons[0].Price)
	}
}
