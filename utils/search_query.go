package utils

import (
	"strings"
	"unicode"
)

// searchCountryNoise is location text that camera/OCR titles often append.
// ISO codes are not listed — "no" / "us" / "pt" are real product tokens.
var searchCountryNoise = map[string]struct{}{
	"brazil": {}, "brasil": {},
	"portugal": {},
	"spain":    {}, "espanha": {}, "espana": {},
	"germany": {}, "alemanha": {}, "deutschland": {},
	"netherlands": {}, "nederland": {}, "holland": {}, "holanda": {},
	"england": {}, "britain": {},
	"usa": {},
}

// SanitizeSearchQuery turns a mobile-supplied product name into a store-search query.
// It lowercases the text, keeps letters and digits, replaces other characters with
// spaces, drops consecutive duplicate tokens, and removes country-name noise
// (e.g. a trailing "Brazil" from a camera title). Empty input stays empty.
func SanitizeSearchQuery(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(name))
	prevSpace := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if prevSpace {
			continue
		}
		b.WriteByte(' ')
		prevSpace = true
	}

	tokens := strings.Fields(b.String())
	out := make([]string, 0, len(tokens))
	var prev string
	for _, tok := range tokens {
		if _, skip := searchCountryNoise[tok]; skip {
			continue
		}
		if tok == prev {
			continue
		}
		out = append(out, tok)
		prev = tok
	}
	return strings.Join(out, " ")
}

// SearchQuery is the query sent to store extractors. If sanitizing would
// erase the name, the trimmed original is kept.
func SearchQuery(name string) string {
	if sanitized := SanitizeSearchQuery(name); sanitized != "" {
		return sanitized
	}
	return strings.TrimSpace(name)
}
