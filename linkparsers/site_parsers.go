package linkparsers

import (
	"muambr-api/utils"
	"net/url"
	"strings"
)

// Parser registry
var siteParserRegistry = map[string]func() Parser{
	"amazon.co.uk":          func() Parser { return &AmazonParser{} },
	"amazon.com":            func() Parser { return &AmazonParser{} },
	"amazon.de":             func() Parser { return &AmazonParser{} },
	"amazon.fr":             func() Parser { return &AmazonParser{} },
	"amazon.com.br":         func() Parser { return &AmazonParser{} },
	"a.co":                  func() Parser { return &AmazonParser{} },
	"cashconverters.pt":     func() Parser { return &CashConvertersPTParser{} },
	"electrolux.com.br":     func() Parser { return &ElectroluxBRParser{} },
	"perfumesecompanhia.pt": func() Parser { return &PerfumesECompanhiaParser{} },
	"walmart.com":           func() Parser { return &WalmartParser{} },
}

// createParser creates the appropriate parser for the URL
func createParser(pageURL *url.URL) Parser {
	host := strings.ToLower(pageURL.Host)

	// Remove www. prefix for matching
	host = strings.TrimPrefix(host, "www.")

	// Try exact match first
	if parserFactory, ok := siteParserRegistry[host]; ok {
		utils.Info("📍 Found exact match parser", utils.String("host", host))
		return parserFactory()
	}

	// Subdomains only (e.g. smile.amazon.com). Substring Contains is unsafe:
	// magazineluiza.com.br contains "a.co" and would pick AmazonParser.
	for configHost, parserFactory := range siteParserRegistry {
		if strings.HasSuffix(host, "."+configHost) {
			utils.Info("📍 Found matching parser",
				utils.String("configHost", configHost),
				utils.String("host", host))
			return parserFactory()
		}
	}

	// Fallback to generic parser
	utils.Warn("⚠️ No specific parser found, using generic parser", utils.String("host", host))
	return &ShareHTMLParser{}
}
