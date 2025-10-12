package models

import (
	"fmt"
	"muambr-api/localization"
)

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

// GetLanguageCode returns the language code for the country
func (c Country) GetLanguageCode() string {
	switch c {
	case CountryBrazil:
		return "pt" // Portuguese for Brazil
	case CountryUS, CountryUK:
		return "en" // English for US and UK
	case CountryPortugal:
		return "pt" // Portuguese for Portugal
	case CountrySpain:
		return "es" // Spanish for Spain
	case CountryGermany:
		return "en" // English for Germany (we don't have German yet)
	default:
		return "en" // Default to English
	}
}

// GetCountriesInMacroRegion returns all countries that belong to the specified macro region
func GetCountriesInMacroRegion(region MacroRegion) []Country {
	var countries []Country
	allCountries := []Country{CountryBrazil, CountryUS, CountryPortugal, CountrySpain, CountryUK, CountryGermany}
	
	for _, country := range allCountries {
		if country.GetMacroRegion() == region {
			countries = append(countries, country)
		}
	}
	
	return countries
}

// ParseCountryFromISO parses an ISO country code string into a Country enum
func ParseCountryFromISO(isoCode string) (Country, error) {
	switch Country(isoCode) {
	case CountryBrazil, CountryUS, CountryPortugal, CountrySpain, CountryUK, CountryGermany:
		return Country(isoCode), nil
	default:
		return "", fmt.Errorf(localization.TP("api.errors.unsupported_country_iso", map[string]string{
			"code": isoCode,
		}))
	}
}

// ConvertedPrice represents a price converted to a different currency
type ConvertedPrice struct {
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

// ProductComparison represents a single product offer matching Swift client expectations
type ProductComparison struct {
	ID             string          `json:"id"`
	ProductName    string          `json:"productName"`
	Price          float64         `json:"price"`
	Currency       string          `json:"currency"`
	ConvertedPrice *ConvertedPrice `json:"convertedPrice,omitempty"`
	StoreName      string          `json:"storeName"`
	StoreURL       *string         `json:"storeURL,omitempty"`
	Description    *string         `json:"description,omitempty"`
	Country        string          `json:"country"`
	Condition      *string         `json:"condition,omitempty"`
	ImageURL       *string         `json:"imageURL,omitempty"`
	LastUpdated    *string         `json:"lastUpdated,omitempty"`
}

// CountrySection represents a group of product comparisons from a specific country
type CountrySection struct {
	Country      string               `json:"country"`
	CountryName  string               `json:"countryName"`
	Comparisons  []ProductComparison  `json:"comparisons"`
	ResultsCount int                  `json:"resultsCount"`
}

// ProductComparisonResponse represents the API response format expected by Swift client
type ProductComparisonResponse struct {
	Success      bool             `json:"success"`
	Message      *string          `json:"message,omitempty"`
	Sections     []CountrySection `json:"sections"`
	TotalResults int              `json:"totalResults"`
}


