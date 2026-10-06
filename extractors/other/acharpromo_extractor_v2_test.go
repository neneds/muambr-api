package other

import "testing"

func TestPickRedirectLinkUsesMerchantURL(t *testing.T) {
	body := "0:{\"a\":\"$@1\"}\n1:{\"success\":true,\"url\":\"https://www.lgimportados.com/produto/918398\"}\n"
	got, err := pickRedirectLink(body, "LG Importados")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://www.lgimportados.com/produto/918398" {
		t.Fatalf("link = %q", got)
	}
}

func TestPickRedirectLinkMatchesOfferSource(t *testing.T) {
	body := `1:{"success":true,"showOffers":true,"offers":[{"affiliateLink":"https://store.example/a","merchant":{"name":"Other"}},{"affiliateLink":"https://store.example/b","merchant":{"name":"iPlace"}}]}`
	got, err := pickRedirectLink(body, "iPlace")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://store.example/b" {
		t.Fatalf("link = %q", got)
	}
}
