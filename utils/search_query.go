package utils

import (
	"strings"
	"unicode"
)

// SanitizeSearchQuery turns a mobile-supplied product name into a store-search query.
// It lowercases the text, keeps letters and digits, replaces other characters with
// spaces, and collapses whitespace. Empty input stays empty.
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

	return strings.TrimSpace(b.String())
}
