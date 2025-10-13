package htmlparser

import (
	"net/url"
)

// BrazilianRetailerParser handles Brazilian retailers (Magazine Luiza, Mercado Livre, Electrolux)
type MagazineLuizaBRParser struct {
	ShareHTMLParser
}

func (p *MagazineLuizaBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

// MercadoLivreBRParser handles Mercado Livre Brazil parsing
type MercadoLivreBRParser struct {
	ShareHTMLParser
}

func (p *MercadoLivreBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}

// ElectroluxBRParser handles Electrolux Brazil parsing
type ElectroluxBRParser struct {
	ShareHTMLParser
}

func (p *ElectroluxBRParser) ExtractCurrency(html string, pageURL *url.URL) string {
	return "brl"
}