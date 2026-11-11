package beauty

import "testing"

func TestDruniESParseAcceptsStringAndNumericIDs(t *testing.T) {
	body := `{
		"results": [
			{"id": "m304", "title": "Lumiere 50ml", "price": 49.95, "link": "https://www.druni.es/lumiere"},
			{"id": 1201, "title": "Other scent", "best_price": 30, "availability": "out_of_stock"},
			{"id": 88, "title": "In stock scent", "sale_price": 12.5}
		]
	}`
	got, err := NewDruniESExtractor().parseDoofinderResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d offers, want 2 (out of stock skipped)", len(got))
	}
	if got[0].ID != "m304" || got[0].Price != 49.95 {
		t.Fatalf("string id offer: %+v", got[0])
	}
	if got[1].ID != "88" || got[1].Price != 12.5 {
		t.Fatalf("numeric id offer: %+v", got[1])
	}
}
