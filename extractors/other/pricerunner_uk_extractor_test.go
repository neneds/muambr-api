package other

import "testing"

func TestPickFeatureURLPrefersMatchingModel(t *testing.T) {
	suggestions := []priceRunnerSuggestion{
		{Name: "IPhones", URL: "/cl/1/Mobile-Phones?man_id=162", Type: "FEATURE"},
		{Name: "Apple iPhone 17", URL: "/cl/1/Mobile-Phones?attr_60501678=100019743", Type: "FEATURE"},
		{Name: "Apple iPhone 16", URL: "/cl/1/Mobile-Phones?attr_60501678=100018036", Type: "FEATURE"},
		{Name: "not a listing", URL: "/search?q=iphone", Type: "QUERY"},
	}

	got := pickFeatureURL("iphone 16", suggestions)
	if got != "/cl/1/Mobile-Phones?attr_60501678=100018036" {
		t.Fatalf("feature url = %q", got)
	}
}
