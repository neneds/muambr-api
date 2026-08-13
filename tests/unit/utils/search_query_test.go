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
