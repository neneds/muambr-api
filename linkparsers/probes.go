package linkparsers

import (
	"net/url"
	"sort"
)

// probeURLs are live product pages used by cmd/check-linkparsers.
// a.co is the Amazon shortener and uses AmazonParser; short links are not stable.
var probeURLs = map[string]string{
	"amazon.com":            "https://www.amazon.com/dp/B0D1XD1ZV3",
	"amazon.com.br":         "https://www.amazon.com.br/dp/B0CH3WBKN6",
	"amazon.co.uk":          "https://www.amazon.co.uk/dp/B0D1XD1ZV3",
	"amazon.de":             "https://www.amazon.de/dp/B0D1XD1ZV3",
	"amazon.fr":             "https://www.amazon.fr/dp/B0D1XD1ZV3",
	"cashconverters.pt":     "https://www.cashconverters.pt/pt/pt/segunda-mano/PT004_E475610_0.html",
	"electrolux.com.br":     "https://loja.electrolux.com.br/panela-eletrica-easyline-7-xicaras-13l-rcb50-electrolux/p",
	"perfumesecompanhia.pt": "https://www.perfumesecompanhia.pt/pt/yves-saint-laurent-libre-berry-crush-eau-de-parfum/436615.html",
	"walmart.com":           "https://www.walmart.com/ip/Apple-AirPods-Pro-2nd-Gen/1752657021",
	"vinted.pt":             "https://www.vinted.pt/items/10114305097-jbl-charge-6",
}

// RegisteredHosts returns siteParserRegistry hosts, sorted.
func RegisteredHosts() []string {
	hosts := make([]string, 0, len(siteParserRegistry))
	for host := range siteParserRegistry {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts
}

// ProbeURL returns the product page used to live-check a registered host.
func ProbeURL(host string) (string, bool) {
	u, ok := probeURLs[host]
	return u, ok && u != ""
}

// ParserName returns the concrete parser selected for a hostname.
func ParserName(host string) string {
	pageURL := &url.URL{Scheme: "https", Host: host, Path: "/"}
	return parserName(createParser(pageURL))
}

func parserName(p Parser) string {
	switch p.(type) {
	case *AmazonParser:
		return "AmazonParser"
	case *CashConvertersPTParser:
		return "CashConvertersPTParser"
	case *ElectroluxBRParser:
		return "ElectroluxBRParser"
	case *PerfumesECompanhiaParser:
		return "PerfumesECompanhiaParser"
	case *WalmartParser:
		return "WalmartParser"
	case *VintedPTParser:
		return "VintedPTParser"
	default:
		return "ShareHTMLParser"
	}
}
