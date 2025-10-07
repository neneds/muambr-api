package models

import "fmt"

// Country represents the country enum using ISO codes
type Country string

const (
	CountryBrazil   Country = "BR"
	CountryUS       Country = "US"
	CountryPortugal Country = "PT"
	CountrySpain    Country = "ES"
	CountryUK       Country = "GB"
	CountryGermany  Country = "DE"
)

// MacroRegion represents broader regions for countries
type MacroRegion string

const (
	MacroRegionEU    MacroRegion = "EU"
	MacroRegionNA    MacroRegion = "NA"
	MacroRegionLATAM MacroRegion = "LATAM"
	MacroRegionNone  MacroRegion = "None"
)

// GetCurrencyCode returns the currency code for the country
func (c Country) GetCurrencyCode() string {
	switch c {
	case CountryBrazil:
		return "BRL"
	case CountryUS:
		return "USD"
	case CountryPortugal, CountrySpain, CountryGermany:
		return "EUR"
	case CountryUK:
		return "GBP"
	default:
		return "USD"
	}
}

// Get macro regions for countries
func (c Country) GetMacroRegion() MacroRegion {
	switch c {
	case CountryBrazil:
		return MacroRegionLATAM
	case CountryUS:
		return MacroRegionNA
	case CountryPortugal, CountrySpain, CountryGermany:
		return MacroRegionEU
	case CountryUK:
		return MacroRegionEU		
	default:
		return MacroRegionNone
	}
}

// GetCountryName returns the human-readable country name
func (c Country) GetCountryName() string {
	switch c {
	case CountryBrazil:
		return "Brazil"
	case CountryUS:
		return "United States"
	case CountryPortugal:
		return "Portugal"
	case CountrySpain:
		return "Spain"
	case CountryUK:
		return "United Kingdom"
	case CountryGermany:
		return "Germany"
	default:
		return "Unknown"
	}
}

// ParseCountryFromISO parses an ISO country code string into a Country enum
func ParseCountryFromISO(isoCode string) (Country, error) {
	switch Country(isoCode) {
	case CountryBrazil, CountryUS, CountryPortugal, CountrySpain, CountryUK, CountryGermany:
		return Country(isoCode), nil
	default:
		return "", fmt.Errorf("unsupported country ISO code: %s", isoCode)
	}
}

// ConvertedPrice represents a price converted to a different currency
type ConvertedPrice struct {
	Price    string `json:"price"`
	Currency string `json:"currency"`
}

// ProductComparison represents a single product offer from comparison sites
type ProductComparison struct {
	Name           string          `json:"name"`
	Price          string          `json:"price"`
	Store          string          `json:"store"`
	Currency       string          `json:"currency"`
	URL            string          `json:"url"`
	ConvertedPrice *ConvertedPrice `json:"convertedPrice,omitempty"`
}

// ComparisonResponse represents the response for product comparisons
type ComparisonResponse []ProductComparison
