package htmlparser

import (
	"net/url"
)

// European retailers parsers

// FnacPTParser handles Fnac Portugal parsing
type FnacPTParser struct {
	ShareHTMLParser
}

func (p *FnacPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// PrimarkParser handles Primark parsing
type PrimarkParser struct {
	ShareHTMLParser
}

func (p *PrimarkParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// PrimorEUParser handles Primor EU parsing
type PrimorEUParser struct {
	ShareHTMLParser
}

func (p *PrimorEUParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// WortenPTParser handles Worten Portugal parsing
type WortenPTParser struct {
	ShareHTMLParser
}

func (p *WortenPTParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}

// ZaraParser handles Zara parsing
type ZaraParser struct {
	ShareHTMLParser
}

func (p *ZaraParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "eur"
}