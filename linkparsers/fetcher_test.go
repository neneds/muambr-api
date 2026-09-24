package linkparsers

import "testing"

func TestIsAmazonUK(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"www.amazon.co.uk", true},
		{"amazon.co.uk", true},
		{"www.amazon.co.uk:443", true},
		{"smile.amazon.co.uk", true},
		{"amazon.com", false},
		{"amazon.com.br", false},
		{"amazon.de", false},
	}
	for _, tc := range cases {
		if got := isAmazonUK(tc.host); got != tc.want {
			t.Errorf("isAmazonUK(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestAmazonCSRFPattern(t *testing.T) {
	html := `data-a-modal='{"ajaxHeaders":{"anti-csrftoken-a2z":"token-value"}}'`
	escaped := `anti-csrftoken-a2z&quot;:&quot;token-value&quot;`
	for _, sample := range []string{html, escaped} {
		m := amazonCSRFPattern.FindStringSubmatch(sample)
		if len(m) < 2 || m[1] != "token-value" {
			t.Errorf("csrf miss in %q: %v", sample, m)
		}
	}
}
