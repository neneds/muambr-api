package models

import (
	"fmt"
	"strings"
)

// Country represents the country enum using ISO codes
type Country string

const (
	CountryBrazil      Country = "BR"
	CountryUS          Country = "US"
	CountryPortugal    Country = "PT"
	CountrySpain       Country = "ES"
	CountryUK          Country = "GB"
	CountryGermany     Country = "DE"
	CountryNetherlands Country = "NL"
)

// MacroRegion represents broader regions for countries
type MacroRegion string

const (
	MacroRegionEU    MacroRegion = "EU"
	MacroRegionNA    MacroRegion = "NA"
	MacroRegionLATAM MacroRegion = "LATAM"
	MacroRegionNone  MacroRegion = "None"
)

// ProductCategory represents the category of a product
type ProductCategory string

const (
	CategoryElectronics ProductCategory = "electronics"
	CategoryBeauty      ProductCategory = "beauty"
	CategoryAppliances  ProductCategory = "appliances"
	CategoryFashion     ProductCategory = "fashion"
	CategoryOther       ProductCategory = "other"
)

// CategoryDisplayName returns a human-readable category name.
func (c ProductCategory) CategoryDisplayName() string {
	switch c {
	case CategoryElectronics:
		return "Electronics"
	case CategoryBeauty:
		return "Beauty"
	case CategoryAppliances:
		return "Appliances"
	case CategoryFashion:
		return "Fashion"
	case CategoryOther:
		return "Other"
	default:
		return string(c)
	}
}

// ParseCategoryFromString parses a string into a ProductCategory enum
func ParseCategoryFromString(s string) (ProductCategory, error) {
	switch ProductCategory(s) {
	case CategoryElectronics, CategoryBeauty, CategoryAppliances, CategoryFashion, CategoryOther:
		return ProductCategory(s), nil
	default:
		return "", fmt.Errorf("unsupported product category: %s (supported: electronics, beauty, appliances, fashion, other)", s)
	}
}

// GetCurrencyCode returns the currency code for the country
func (c Country) GetCurrencyCode() string {
	switch c {
	case CountryBrazil:
		return "BRL"
	case CountryUS:
		return "USD"
	case CountryPortugal, CountrySpain, CountryGermany, CountryNetherlands:
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
	case CountryPortugal, CountrySpain, CountryGermany, CountryNetherlands:
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
	case CountryNetherlands:
		return "Netherlands"
	default:
		return "Unknown"
	}
}

// GetCountriesInMacroRegion returns all countries that belong to the specified macro region
func GetCountriesInMacroRegion(region MacroRegion) []Country {
	var countries []Country
	allCountries := []Country{CountryBrazil, CountryUS, CountryPortugal, CountrySpain, CountryUK, CountryGermany, CountryNetherlands}

	for _, country := range allCountries {
		if country.GetMacroRegion() == region {
			countries = append(countries, country)
		}
	}

	return countries
}

// ParseCountryFromISO parses an ISO country code string into a Country enum.
// "UK" is accepted as an alias of ISO 3166-1 "GB".
func ParseCountryFromISO(isoCode string) (Country, error) {
	code := strings.ToUpper(strings.TrimSpace(isoCode))
	if code == "UK" {
		return CountryUK, nil
	}
	switch Country(code) {
	case CountryBrazil, CountryUS, CountryPortugal, CountrySpain, CountryUK, CountryGermany, CountryNetherlands:
		return Country(code), nil
	default:
		return "", fmt.Errorf("unsupported country ISO code: %s", isoCode)
	}
}

// ConvertedPrice represents a price converted to a different currency
type ConvertedPrice struct {
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

// ProductComparison represents a single product offer matching Swift client expectations
type ProductComparison struct {
	ID              string           `json:"id"`
	ProductName     string           `json:"productName"`
	Price           float64          `json:"price"`
	Currency        string           `json:"currency"`
	ConvertedPrice  *ConvertedPrice  `json:"convertedPrice,omitempty"`
	StoreName       string           `json:"storeName"`
	StoreURL        *string          `json:"storeURL,omitempty"`
	Description     *string          `json:"description,omitempty"`
	Country         string           `json:"country"`
	Condition       *string          `json:"condition,omitempty"`
	ImageURL        *string          `json:"imageURL,omitempty"`
	LastUpdated     *string          `json:"lastUpdated,omitempty"`
	Category        *ProductCategory `json:"category,omitempty"`
	MatchConfidence *float64         `json:"matchConfidence,omitempty"`
	Availability    string           `json:"availability,omitempty"`
}

// CountrySection represents a group of product comparisons from a specific country
type CountrySection struct {
	Country      string              `json:"country"`
	CountryName  string              `json:"countryName"`
	Comparisons  []ProductComparison `json:"comparisons"`
	ResultsCount int                 `json:"resultsCount"`
}
