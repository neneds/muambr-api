package utils

import (
	"testing"

	"muambr-api/utils"
)

func TestSanitizeSearchQuery(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "already clean", in: "iphone 15", want: "iphone 15"},
		{name: "lowercase", in: "Sony WH-1000XM6", want: "sony wh 1000xm6"},
		{name: "punctuation and store suffix", in: `Apple iPad Pro (2025) 13" 256GB WiFi + 5G Black | Coolblue`, want: "apple ipad pro 2025 13 256gb wifi 5g black coolblue"},
		{name: "ocr noise", in: "  YVES SAINT LAURENT — Libre Berry Crush!!  ", want: "yves saint laurent libre berry crush"},
		{name: "newlines and tabs", in: "iPhone\t15\nPro", want: "iphone 15 pro"},
		{name: "accents kept", in: "Água de Colónia Nº5", want: "água de colónia nº5"},
		{name: "only symbols", in: "!!! ***", want: ""},
		{name: "empty", in: "   ", want: ""},
		{
			name: "olaplex camera title",
			in:   "OLAPLEX Olaplex No. 3 Hair Perfector - 250ml Brazil",
			want: "olaplex no 3 hair perfector 250ml",
		},
		{name: "duplicate brand tokens", in: "Nike Nike Air Max", want: "nike air max"},
		{name: "keeps size tokens", in: "Olaplex No.3 100ml", want: "olaplex no 3 100ml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := utils.SanitizeSearchQuery(tc.in)
			if got != tc.want {
				t.Errorf("SanitizeSearchQuery(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSearchQuery_KeepsOriginalWhenSanitizeEmpties(t *testing.T) {
	if got := utils.SearchQuery("!!! ***"); got != "!!! ***" {
		t.Errorf("SearchQuery(symbols) = %q, want original", got)
	}
	if got := utils.SearchQuery("OLAPLEX Olaplex No. 3 Hair Perfector - 250ml Brazil"); got != "olaplex no 3 hair perfector 250ml" {
		t.Errorf("SearchQuery(olaplex) = %q", got)
	}
}
