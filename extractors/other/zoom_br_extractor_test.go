package other

import "testing"

func TestSelectZoomHitsKeepsSearchOrder(t *testing.T) {
	hits := []zoomHit{
		{Name: "Geladeira Electrolux Frost Free Duplex Branca 480L Effici", Price: 3399, URL: "/first"},
		{Name: "Geladeira Electrolux", Price: 1000, URL: "/shorter"},
		{Name: "Capa de celular", Price: 30, URL: "/unrelated"},
	}
	got := selectZoomHits(hits, "geladeira electrolux", 2)
	if len(got) != 2 {
		t.Fatalf("got %d hits", len(got))
	}
	if got[0].URL != "/first" || got[1].URL != "/shorter" {
		t.Fatalf("order = %s, %s", got[0].URL, got[1].URL)
	}
}
