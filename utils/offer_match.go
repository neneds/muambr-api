package utils

import (
	"strings"
	"unicode"

	"muambr-api/models"
)

var (
	accentFold = strings.NewReplacer(
		"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"n°", "no", "nº", "no",
	)
	// tokenCanon maps spelling variants onto one token before matching.
	tokenCanon = map[string]string{
		"man": "men", "homme": "men", "homem": "men", "masculino": "men",
		"woman": "women", "femme": "women", "feminino": "women",
		"mascara": "mask", "masque": "mask",
		"oleo":          "oil",
		"desodorizante": "deodorant", "desodorante": "deodorant", "deodorante": "deodorant",
		"locao": "lotion", "creme": "cream",
		"shower": "gel", "duche": "gel",
		"conditioner": "condicionador",
		"estojo":      "coffret", "kit": "coffret",
	}
	matchStopwords = map[string]struct{}{
		"eau": {}, "de": {}, "do": {}, "da": {}, "du": {}, "le": {}, "la": {},
		"parfum": {}, "perfume": {}, "edt": {}, "edp": {}, "toilette": {},
		"the": {}, "and": {}, "of": {}, "com": {}, "para": {}, "for": {}, "with": {},
		"pour": {}, "spray": {},
	}
	quantityUnits = map[string]struct{}{
		"ml": {}, "cl": {}, "l": {}, "g": {}, "kg": {}, "mg": {}, "oz": {}, "gb": {}, "tb": {},
	}
	// formGroups are product types. A mask search must not keep the oil.
	formGroups = []map[string]struct{}{
		{"deodorant": {}},
		{"lotion": {}, "cream": {}},
		{"gel": {}},
		{"shampoo": {}},
		{"condicionador": {}},
		{"serum": {}},
		{"oil": {}},
		{"coffret": {}},
		{"stick": {}},
		{"mist": {}},
		{"mask": {}},
	}
	flankerWords = map[string]struct{}{
		"profondo": {}, "absolu": {}, "elixir": {}, "intense": {},
		"amber": {}, "very": {},
	}
)

// annotateComparisons scores each offer and drops titles that are not the searched product.
// The query is tokenized once. Each offer is tokenized once.
func annotateComparisons(query string, sections []models.CountrySection, capturedAt string) []models.CountrySection {
	qTokens := tokenize(query)
	for i := range sections {
		kept := sections[i].Comparisons[:0]
		for _, c := range sections[i].Comparisons {
			pTokens := tokenize(c.ProductName)
			conf := confidence(qTokens, pTokens)
			c.MatchConfidence = &conf
			if c.LastUpdated == nil {
				ts := capturedAt
				c.LastUpdated = &ts
			}
			if c.Availability == "" {
				c.Availability = "unknown"
			}
			if matchesOffer(qTokens, pTokens) {
				kept = append(kept, c)
			}
		}
		sections[i].Comparisons = kept
		sections[i].ResultsCount = len(kept)
	}
	return sections
}

// MatchConfidence returns a 0–1 score based on token overlap between query and product name.
func MatchConfidence(query, productName string) float64 {
	return confidence(tokenize(query), tokenize(productName))
}

func matchesOffer(query, product []string) bool {
	if len(query) == 0 || len(product) == 0 {
		return false
	}
	required := requiredTokens(query)
	if len(required) == 0 {
		return confidence(query, product) >= MatchConfidenceMin
	}
	pSet := tokenSet(product)
	for _, tok := range required {
		if _, ok := pSet[tok]; !ok {
			return false
		}
	}
	return !quantityConflict(query, product) &&
		!formConflict(query, product) &&
		!genderConflict(query, product) &&
		!flankerConflict(query, product) &&
		!bodySprayConflict(query, product)
}

func confidence(query, product []string) float64 {
	if len(query) == 0 || len(product) == 0 {
		return 0.5
	}
	qSet := tokenSet(query)
	pSet := tokenSet(product)
	intersection := 0
	for t := range qSet {
		if _, ok := pSet[t]; ok {
			intersection++
		}
	}
	union := len(qSet) + len(pSet) - intersection
	jaccard := 0.0
	if union > 0 {
		jaccard = float64(intersection) / float64(union)
	}
	coverage := float64(intersection) / float64(len(qSet))
	score := 0.4*jaccard + 0.6*coverage
	if score > 1 {
		score = 1
	}
	return round2(score)
}

func requiredTokens(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	seen := map[string]struct{}{}
	for _, t := range tokens {
		if _, ok := seen[t]; ok || isStopword(t) || isQuantityToken(t) || isGenderToken(t) {
			seen[t] = struct{}{}
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func quantityConflict(query, product []string) bool {
	qQty := quantitySet(query)
	pQty := quantitySet(product)
	if len(qQty) == 0 || len(pQty) == 0 {
		return false
	}
	for q := range qQty {
		if _, ok := pQty[q]; ok {
			return false
		}
	}
	return true
}

func quantitySet(tokens []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range tokens {
		if isQuantityToken(t) {
			out[t] = struct{}{}
		}
	}
	return out
}

func formConflict(query, product []string) bool {
	for _, group := range formGroups {
		if hits(group, product) && !hits(group, query) {
			return true
		}
	}
	return false
}

func genderConflict(query, product []string) bool {
	qMen, qWomen := hits(genderMen, query), hits(genderWomen, query)
	pMen, pWomen := hits(genderMen, product), hits(genderWomen, product)
	return (qMen && pWomen && !pMen) || (qWomen && pMen && !pWomen)
}

var (
	genderMen   = map[string]struct{}{"men": {}}
	genderWomen = map[string]struct{}{"women": {}}
)

func flankerConflict(query, product []string) bool {
	qSet := tokenSet(query)
	for _, tok := range product {
		if _, flanker := flankerWords[tok]; flanker {
			if _, ok := qSet[tok]; !ok {
				return true
			}
		}
	}
	return false
}

func bodySprayConflict(query, product []string) bool {
	pSet := tokenSet(product)
	qSet := tokenSet(query)
	if _, ok := pSet["spray"]; !ok {
		return false
	}
	if _, ok := qSet["spray"]; ok {
		return false
	}
	for _, fragrance := range []string{"parfum", "perfume", "edp", "edt", "eau"} {
		if _, ok := pSet[fragrance]; ok {
			return false
		}
	}
	return true
}

func hits(group map[string]struct{}, tokens []string) bool {
	for _, t := range tokens {
		if _, ok := group[t]; ok {
			return true
		}
	}
	return false
}

func tokenSet(tokens []string) map[string]struct{} {
	set := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		set[t] = struct{}{}
	}
	return set
}

func isStopword(tok string) bool {
	_, ok := matchStopwords[tok]
	return ok
}

func isGenderToken(tok string) bool {
	return tok == "men" || tok == "women"
}

func isQuantityToken(tok string) bool {
	i := 0
	for i < len(tok) && tok[i] >= '0' && tok[i] <= '9' {
		i++
	}
	if i == 0 || i == len(tok) {
		return false
	}
	_, ok := quantityUnits[tok[i:]]
	return ok
}

func tokenize(s string) []string {
	s = accentFold.Replace(strings.ToLower(s))
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tok := b.String()
		b.Reset()
		if canon, ok := tokenCanon[tok]; ok {
			tok = canon
		}
		tokens = append(tokens, tok)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return glueQuantities(tokens)
}

func glueQuantities(tokens []string) []string {
	if len(tokens) < 2 {
		return tokens
	}
	out := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		if i+1 < len(tokens) && isDigits(tokens[i]) {
			if _, ok := quantityUnits[tokens[i+1]]; ok {
				out = append(out, tokens[i]+tokens[i+1])
				i++
				continue
			}
		}
		out = append(out, tokens[i])
	}
	return out
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
