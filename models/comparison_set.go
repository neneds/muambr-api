package models

// CountryRole describes why a country is in the comparison set (UI metadata).
type CountryRole string

const (
	CountryRoleBase        CountryRole = "base"
	CountryRoleCurrent     CountryRole = "current"
	CountryRoleJourney     CountryRole = "journey"
	CountryRoleMicroregion CountryRole = "microregion"
)

// ComparisonCountrySpec is one country in the resolved comparison set.
type ComparisonCountrySpec struct {
	Country Country
	Roles   []string
}

// ComparisonSetInput is the data needed to resolve which countries to compare.
type ComparisonSetInput struct {
	BaseCountry       Country
	ProductLocation   Country
	ExplicitCountries []Country
	JourneyCountries  []Country
	ExpandMicroregion bool
}

// ResolveComparisonSet builds the de-duplicated country list and roles.
//
// comparisonCountries (ExplicitCountries) is the source of truth when non-empty.
// Base is always included. Microregion expansion only adds countries.
func ResolveComparisonSet(in ComparisonSetInput) []ComparisonCountrySpec {
	explicit := uniqueCountries(in.ExplicitCountries)
	journey := countrySet(in.JourneyCountries)

	var ordered []Country
	addedByMicroregion := make(map[Country]struct{})

	if len(explicit) > 0 {
		ordered = append(ordered, explicit...)
	} else {
		if in.BaseCountry != "" {
			ordered = append(ordered, in.BaseCountry)
		}
		if in.ProductLocation != "" && in.ProductLocation != in.BaseCountry {
			ordered = append(ordered, in.ProductLocation)
		}
	}

	if in.BaseCountry != "" && !containsCountry(ordered, in.BaseCountry) {
		ordered = append(ordered, in.BaseCountry)
	}

	if in.ExpandMicroregion && in.ProductLocation != "" {
		for _, c := range GetCountriesInMacroRegion(in.ProductLocation.GetMacroRegion()) {
			if containsCountry(ordered, c) {
				continue
			}
			ordered = append(ordered, c)
			addedByMicroregion[c] = struct{}{}
		}
	}

	specs := make([]ComparisonCountrySpec, 0, len(ordered))
	for _, c := range ordered {
		roles := make([]string, 0, 3)
		if c == in.BaseCountry {
			roles = append(roles, string(CountryRoleBase))
		}
		if in.ProductLocation != "" && c == in.ProductLocation {
			roles = append(roles, string(CountryRoleCurrent))
		}
		if _, ok := journey[c]; ok {
			roles = append(roles, string(CountryRoleJourney))
		}
		if _, ok := addedByMicroregion[c]; ok {
			roles = append(roles, string(CountryRoleMicroregion))
		}
		specs = append(specs, ComparisonCountrySpec{Country: c, Roles: roles})
	}
	return specs
}

// Countries returns the country codes from a resolved set, in order.
func CountriesFromSpecs(specs []ComparisonCountrySpec) []Country {
	out := make([]Country, len(specs))
	for i, s := range specs {
		out[i] = s.Country
	}
	return out
}

func uniqueCountries(in []Country) []Country {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[Country]struct{}, len(in))
	out := make([]Country, 0, len(in))
	for _, c := range in {
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

func countrySet(in []Country) map[Country]struct{} {
	out := make(map[Country]struct{}, len(in))
	for _, c := range in {
		if c != "" {
			out[c] = struct{}{}
		}
	}
	return out
}

func containsCountry(list []Country, c Country) bool {
	for _, x := range list {
		if x == c {
			return true
		}
	}
	return false
}
