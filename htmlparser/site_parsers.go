package htmlparser

import (
	"muambr-api/utils"
	"net/url"
	"strings"
)

// Parser registry
var siteParserRegistry = map[string]func() Parser{
	"amazon.es":             func() Parser { return &AmazonParser{} },
	"amazon.co.uk":          func() Parser { return &AmazonParser{} },
	"amazon.com":            func() Parser { return &AmazonParser{} },
	"amazon.de":             func() Parser { return &AmazonParser{} },
	"amazon.fr":             func() Parser { return &AmazonParser{} },
	"amazon.com.br":         func() Parser { return &AmazonParser{} },
	"a.co":                  func() Parser { return &AmazonParser{} },
	"cashconverters.pt":     func() Parser { return &CashConvertersPTParser{} },
	"fnac.pt":               func() Parser { return &FnacPTParser{} },
	"olx.pt":                func() Parser { return &OLXPTParser{} },
	"olx.br":                func() Parser { return &OLXBRParser{} },
	"olx.com.br":            func() Parser { return &OLXBRParser{} },
	"magazineluiza.com.br":  func() Parser { return &MagazineLuizaBRParser{} },
	"mercadolivre.com.br":   func() Parser { return &MercadoLivreBRParser{} },
	"electrolux.com.br":     func() Parser { return &ElectroluxBRParser{} },
	"primark.com":           func() Parser { return &PrimarkParser{} },
	"primor.eu":             func() Parser { return &PrimorEUParser{} },
	"worten.pt":             func() Parser { return &WortenPTParser{} },
	"zara.com":              func() Parser { return &ZaraParser{} },
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

	// Try partial matches
	for configHost, parserFactory := range siteParserRegistry {
		if strings.Contains(host, configHost) || strings.Contains(configHost, host) {
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