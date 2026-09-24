package linkparsers

import (
	"fmt"
	"io"
	"muambr-api/utils"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

const amazonUKPostcode = "SW1A1AA"

var amazonCSRFPattern = regexp.MustCompile(`anti-csrftoken-a2z(?:&quot;|")\s*:\s*(?:&quot;|")([^"&]+)`)

// FetchHTML fetches HTML content from a URL.
func FetchHTML(urlStr string) (string, error) {
	utils.Info("Fetching HTML from URL", utils.String("url", urlStr))

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if isAmazonUK(parsedURL.Host) {
		return fetchAmazonUK(urlStr)
	}

	resp, err := utils.MakeAntiBotRequest(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	utils.Info("Response received",
		utils.Int("statusCode", resp.StatusCode),
		utils.String("status", resp.Status))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := utils.ReadDecompressedBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(body)
	utils.Info("Successfully fetched HTML",
		utils.Int("bytes", len(html)),
		utils.String("host", parsedURL.Host))

	return html, nil
}

// ParseURL fetches and parses HTML from a URL
func ParseURL(urlStr string) (*ParsedProductData, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	html, err := FetchHTML(urlStr)
	if err != nil {
		return nil, err
	}

	data := ParseHTML(html, parsedURL)
	return data, nil
}

func isAmazonUK(host string) bool {
	host = strings.ToLower(host)
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimPrefix(host, "www.")
	return host == "amazon.co.uk" || strings.HasSuffix(host, ".amazon.co.uk")
}

// fetchAmazonUK sets a London delivery postcode before reading the product page.
// A Brazil delivery location hides UK offers ("cannot be shipped").
func fetchAmazonUK(pageURL string) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("amazon uk cookie jar: %w", err)
	}
	client := utils.CreateAntiBotClient()
	client.Jar = jar

	root, _ := url.Parse("https://www.amazon.co.uk/")
	jar.SetCookies(root, []*http.Cookie{
		{Name: "lc-acbuk", Value: "en_GB", Path: "/"},
		{Name: "i18n-prefs", Value: "GBP", Path: "/"},
	})

	html, status, err := amazonDo(client, http.MethodGet, pageURL, "", "", "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", status)
	}

	if token := amazonCSRFPattern.FindStringSubmatch(html); len(token) > 1 {
		if err := setAmazonUKDelivery(client, pageURL, token[1]); err != nil {
			utils.Warn("Amazon UK delivery postcode was not set", utils.Error(err))
		} else if again, againStatus, againErr := amazonDo(client, http.MethodGet, pageURL, "", "", ""); againErr != nil {
			return "", againErr
		} else if againStatus == http.StatusOK {
			html = again
		}
	}

	return html, nil
}

func setAmazonUKDelivery(client *http.Client, pageURL, token string) error {
	form := url.Values{
		"locationType": {"LOCATION_INPUT"},
		"zipCode":      {amazonUKPostcode},
		"storeContext": {"generic"},
		"deviceType":   {"web"},
		"pageType":     {"Detail"},
		"actionSource": {"glow"},
		"almBrandId":   {"undefined"},
	}
	endpoint := "https://www.amazon.co.uk/portal-migration/hz/glow/address-change?actionSource=glow"
	body, status, err := amazonDo(client, http.MethodPost, endpoint, form.Encode(), token, pageURL)
	if err != nil {
		return err
	}
	if status != http.StatusOK || !strings.Contains(body, `"successful":1`) {
		return fmt.Errorf("address change status %d", status)
	}
	utils.Info("Amazon UK delivery set", utils.String("postcode", amazonUKPostcode), utils.String("page", pageURL))
	return nil
}

func amazonDo(client *http.Client, method, rawURL, form, csrf, referer string) (string, int, error) {
	var body io.Reader
	if form != "" {
		body = strings.NewReader(form)
	}
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return "", 0, err
	}
	utils.ApplyAntiBotHeaders(req)
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	if form != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
		req.Header.Set("Origin", "https://www.amazon.co.uk")
		req.Header.Set("Referer", referer)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("anti-csrftoken-a2z", csrf)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()
	payload, err := utils.ReadDecompressedBody(resp)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}
	return string(payload), resp.StatusCode, nil
}
