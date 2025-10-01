package models

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

// GetCountryName returns the full country name
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
		return ""
	}
}

// ParseCountryFromISO converts an ISO code to a Country
func ParseCountryFromISO(isoCode string) (Country, bool) {
	country := Country(isoCode)
	switch country {
	case CountryBrazil, CountryUS, CountryPortugal, CountrySpain, CountryUK, CountryGermany:
		return country, true
	default:
		return "", false
	}
}

// ProductComparison represents a single product offer from comparison sites
type ProductComparison struct {
	Name     string `json:"name"`
	Price    string `json:"price"`
	Store    string `json:"store"`
	Currency string `json:"currency"`
	URL      string `json:"url"`
}

// ComparisonResponse represents the response for product comparisons
type ComparisonResponse []ProductComparison
