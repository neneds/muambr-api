package other

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"muambr-api/extractors"
	"muambr-api/models"
	"muambr-api/utils"
)

// acharPromoProduct represents a product from the AcharPromo chat API response
type acharPromoProduct struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Price          string  `json:"price"`
	ExtractedPrice float64 `json:"extracted_price"`
	Image          string  `json:"image"`
	URL            string  `json:"url"`
	Source         string  `json:"source"`
	ProductID      string  `json:"product_id"`
	ProductToken   string  `json:"product_token"`
	IsRecommended  bool    `json:"isRecommended"`
}

// acharPromoToolOutput represents the tool-output-available SSE event payload
type acharPromoToolOutput struct {
	Type       string `json:"type"`
	ToolCallID string `json:"toolCallId"`
	Output     struct {
		Status   string              `json:"status"`
		SearchID string              `json:"searchId"`
		Query    string              `json:"query"`
		Category string              `json:"category"`
		Products []acharPromoProduct `json:"products"`
		Metadata struct {
			TotalProducts int      `json:"totalProducts"`
			TotalShops    int      `json:"totalShops"`
			TopShops      []string `json:"topShops"`
			MinPrice      float64  `json:"minPrice"`
		} `json:"metadata"`
	} `json:"output"`
}

// acharPromoChatRequest represents the POST body for the /api/chat endpoint
type acharPromoChatRequest struct {
	ID       string                  `json:"id"`
	Messages []acharPromoChatMessage `json:"messages"`
	Trigger  string                  `json:"trigger"`
}

// acharPromoChatMessage represents a message in the chat request
type acharPromoChatMessage struct {
	ID       string                      `json:"id"`
	Role     string                      `json:"role"`
	Parts    []acharPromoChatMessagePart `json:"parts"`
	Metadata acharPromoChatMessageMeta   `json:"metadata"`
}

// acharPromoChatMessageMeta holds per-message analytics metadata required by the API
type acharPromoChatMessageMeta struct {
	DistinctID string `json:"distinctId"`
	IsInitial  bool   `json:"isInitial"`
}

// acharPromoChatMessagePart represents a part of a chat message
type acharPromoChatMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// acharPromoNoopParser is a minimal HTMLParser implementation for AcharPromo.
// AcharPromo extraction is done via the /api/chat endpoint, not HTML parsing, but
// BaseGoExtractor requires an HTMLParser to be passed in.
type acharPromoNoopParser struct {
	*extractors.BaseHTMLParser
}

func newAcharPromoNoopParser() *acharPromoNoopParser {
	return &acharPromoNoopParser{BaseHTMLParser: extractors.NewBaseHTMLParser("AcharPromo")}
}

func (p *acharPromoNoopParser) GetProductSelectors() []string { return nil }
func (p *acharPromoNoopParser) GetNameSelectors() []string    { return nil }
func (p *acharPromoNoopParser) GetPriceSelectors() []string   { return nil }
func (p *acharPromoNoopParser) GetURLSelectors() []string     { return nil }
func (p *acharPromoNoopParser) ParseProductName(_ string) string {
	return ""
}
func (p *acharPromoNoopParser) ParsePrice(_ string) (float64, string, error) {
	return 0, "BRL", fmt.Errorf("not implemented")
}
func (p *acharPromoNoopParser) ParseURL(_ string, baseURL string) string {
	return baseURL
}
func (p *acharPromoNoopParser) ParseStore(_ string) string {
	return "AcharPromo Brasil"
}

// AcharPromoExtractorV2 uses the AcharPromo /api/chat endpoint to search for products.
// The chat API uses a Vercel AI SDK streaming format and returns Google Shopping
// results for Brazil via the searchByText tool.
type AcharPromoExtractorV2 struct {
	*extractors.BaseGoExtractor
}

// NewAcharPromoExtractorV2 creates a new pure Go AcharPromo extractor
func NewAcharPromoExtractorV2() *AcharPromoExtractorV2 {
	parser := newAcharPromoNoopParser()
	baseExtractor := extractors.NewBaseGoExtractor(
		"https://achar.promo",
		models.CountryBrazil,
		"acharpromo_v2",
		parser,
	)

	return &AcharPromoExtractorV2{
		BaseGoExtractor: baseExtractor,
	}
}

// GetComparisons calls the AcharPromo /api/chat endpoint with the product name
// and parses the SSE streaming response to extract product comparisons.
func (e *AcharPromoExtractorV2) GetComparisons(productName string) ([]models.ProductComparison, error) {
	utils.Info("Starting AcharPromo chat API extraction",
		utils.String("product", productName),
		utils.String("extractor", e.GetIdentifier()),
		utils.String("country", string(e.GetCountryCode())))

	chatURL := e.GetBaseURL() + "/api/chat"

	reqBody := acharPromoChatRequest{
		ID: utils.GenerateUUID(),
		Messages: []acharPromoChatMessage{
			{
				ID:   utils.GenerateUUID(),
				Role: "user",
				Parts: []acharPromoChatMessagePart{
					{Type: "text", Text: productName},
				},
				Metadata: acharPromoChatMessageMeta{
					DistinctID: utils.GenerateUUID(),
					IsInitial:  true,
				},
			},
		},
		Trigger: "submit-message",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	products, err := e.fetchChatProducts(chatURL, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("chat API request failed: %w", err)
	}

	e.resolveMerchantLinks(products)
	comparisons := e.convertProducts(products)

	utils.Info("Extraction completed",
		utils.String("extractor", e.GetIdentifier()),
		utils.Int("results", len(comparisons)))

	return comparisons, nil
}

// GetComparisonsFromHTML parses an SSE streaming response body (not HTML) for products.
// This allows unit testing with saved SSE response fixtures.
func (e *AcharPromoExtractorV2) GetComparisonsFromHTML(sseBody string) ([]models.ProductComparison, error) {
	products := parseSSEProducts(sseBody)
	if len(products) == 0 {
		return nil, nil
	}
	return e.convertProducts(products), nil
}

// fetchChatProducts makes the POST request to /api/chat and parses the SSE stream.
func (e *AcharPromoExtractorV2) fetchChatProducts(chatURL string, body []byte) ([]acharPromoProduct, error) {
	client := utils.CreateAntiBotClient()
	client.Timeout = 60 * time.Second // chat API can be slow (AI reasoning)

	req, err := http.NewRequest("POST", chatURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://achar.promo")
	req.Header.Set("Referer", "https://achar.promo/")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "ai-sdk/5.0.107 runtime/browser")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from chat API", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	rawResponse := string(respBody)
	products := parseSSEProducts(rawResponse)
	if len(products) == 0 {
		// Log the first 500 bytes of the raw response to aid diagnosis when
		// the SSE format or event types change on the AcharPromo side.
		preview := rawResponse
		if len(preview) > 500 {
			preview = preview[:500]
		}
		utils.Warn("AcharPromo: no products found in SSE response",
			utils.String("responsePreview", preview),
		)
		return nil, fmt.Errorf("no products found in chat API response")
	}

	return products, nil
}

// parseSSEProducts extracts products from the SSE streaming response.
// It scans for "tool-output-available" events containing searchByText results.
func parseSSEProducts(sseBody string) []acharPromoProduct {
	scanner := bufio.NewScanner(strings.NewReader(sseBody))
	// Increase buffer size for large SSE lines
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// Quick check before JSON parsing
		if !strings.Contains(data, "tool-output-available") {
			continue
		}

		var event acharPromoToolOutput
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "tool-output-available" && len(event.Output.Products) > 0 {
			return event.Output.Products
		}
	}

	return nil
}

// convertProducts converts acharPromoProduct items to ProductComparison models.
func (e *AcharPromoExtractorV2) convertProducts(products []acharPromoProduct) []models.ProductComparison {
	var comparisons []models.ProductComparison

	for _, p := range products {
		if p.Title == "" || p.ExtractedPrice <= 0 {
			continue
		}

		storeName := p.Source
		if storeName == "" {
			storeName = "AcharPromo Brasil"
		}

		var storeURL, imageURL *string
		if link := preferredStoreURL(p); link != "" {
			storeURL = &link
		}
		if p.Image != "" {
			img := p.Image
			imageURL = &img
		}

		comparisons = append(comparisons, models.ProductComparison{
			ID:          utils.GenerateUUID(),
			ProductName: strings.TrimSpace(p.Title),
			Price:       p.ExtractedPrice,
			Currency:    "BRL",
			StoreName:   storeName,
			StoreURL:    storeURL,
			ImageURL:    imageURL,
			Country:     string(models.CountryBrazil),
		})
	}

	return comparisons
}

var redirectActionIDPattern = regexp.MustCompile(`(?s)createServerReference\)\("([0-9a-f]+)".{0,200}"getRedirectUrl"`)

func isGoogleHost(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	return host == "google.com" || strings.HasSuffix(host, ".google.com")
}

func isGoogleShoppingURL(raw string) bool {
	if !isGoogleHost(raw) {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Query().Get("ibp") == "oshop"
}

// preferredStoreURL keeps direct merchant links. Google Shopping "oshop" URLs
// open an empty results page, so those fall back to AcharPromo's redirect,
// which resolves product_token into a store link.
func preferredStoreURL(p acharPromoProduct) string {
	if p.URL != "" && !isGoogleShoppingURL(p.URL) {
		return p.URL
	}
	if p.ProductToken != "" {
		return "https://achar.promo/redirect?product_token=" + url.QueryEscape(p.ProductToken)
	}
	if p.ProductID != "" {
		values := url.Values{}
		values.Set("product_id", p.ProductID)
		if p.Source != "" {
			values.Set("source", p.Source)
		}
		return "https://achar.promo/redirect?" + values.Encode()
	}
	return ""
}

func (e *AcharPromoExtractorV2) resolveMerchantLinks(products []acharPromoProduct) {
	indexes := make([]int, 0)
	for i, p := range products {
		if p.ProductToken == "" && p.ProductID == "" {
			continue
		}
		if p.URL != "" && !isGoogleHost(p.URL) {
			continue
		}
		indexes = append(indexes, i)
	}
	if len(indexes) == 0 {
		return
	}

	actionID, err := e.lookupRedirectActionID()
	if err != nil {
		utils.Warn("AcharPromo: could not resolve merchant links", utils.Error(err))
		return
	}

	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, idx := range indexes {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			link, err := e.fetchRedirectURL(actionID, products[idx])
			if err != nil || link == "" || isGoogleHost(link) {
				return
			}
			mu.Lock()
			products[idx].URL = link
			mu.Unlock()
		}(idx)
	}
	wg.Wait()
}

func (e *AcharPromoExtractorV2) lookupRedirectActionID() (string, error) {
	client := utils.CreateAntiBotClient()
	client.Timeout = 20 * time.Second
	req, err := http.NewRequest(http.MethodGet, e.GetBaseURL()+"/redirect", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	req.Header.Set("Accept", "text/html")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	html := string(body)
	if id := redirectActionIDPattern.FindStringSubmatch(html); len(id) == 2 {
		return id[1], nil
	}

	scriptSrc := regexp.MustCompile(`src="([^"]+\.js[^"]*)"`)
	var actionID string
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, match := range scriptSrc.FindAllStringSubmatch(html, -1) {
		src := match[1]
		if !strings.Contains(src, "/_next/static/chunks/") {
			continue
		}
		if strings.HasPrefix(src, "/") {
			src = e.GetBaseURL() + src
		}
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			mu.Lock()
			done := actionID != ""
			mu.Unlock()
			if done {
				return
			}
			id, err := fetchActionID(client, src)
			if err != nil || id == "" {
				return
			}
			mu.Lock()
			if actionID == "" {
				actionID = id
			}
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	if actionID == "" {
		return "", fmt.Errorf("getRedirectUrl action id not found")
	}
	return actionID, nil
}

func fetchActionID(client *http.Client, scriptURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, scriptURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", utils.DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	match := redirectActionIDPattern.FindSubmatch(body)
	if len(match) != 2 {
		return "", nil
	}
	return string(match[1]), nil
}

type acharPromoRedirectOffer struct {
	AffiliateLink string `json:"affiliateLink"`
	Link          string `json:"link"`
	Merchant      struct {
		Name string `json:"name"`
	} `json:"merchant"`
}

type acharPromoRedirectResult struct {
	Success    bool                      `json:"success"`
	URL        string                    `json:"url"`
	ShowOffers bool                      `json:"showOffers"`
	Offers     []acharPromoRedirectOffer `json:"offers"`
}

func (e *AcharPromoExtractorV2) fetchRedirectURL(actionID string, product acharPromoProduct) (string, error) {
	var productURL any
	if product.URL != "" && !isGoogleHost(product.URL) {
		productURL = product.URL
	}
	var productID any
	if product.ProductID != "" {
		productID = product.ProductID
	}
	var token any
	if product.ProductToken != "" {
		token = product.ProductToken
	}
	payload, err := json.Marshal([]any{productURL, productID, product.IsRecommended, product.Source, token})
	if err != nil {
		return "", err
	}

	client := utils.CreateAntiBotClient()
	client.Timeout = 20 * time.Second
	req, err := http.NewRequest(http.MethodPost, e.GetBaseURL()+"/redirect", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Accept", "text/x-component")
	req.Header.Set("Next-Action", actionID)
	req.Header.Set("Origin", e.GetBaseURL())
	req.Header.Set("Referer", e.GetBaseURL()+"/redirect")
	req.Header.Set("User-Agent", utils.DefaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("redirect action HTTP %d", resp.StatusCode)
	}
	return pickRedirectLink(string(body), product.Source)
}

func pickRedirectLink(body, source string) (string, error) {
	for _, line := range strings.Split(body, "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		payload := strings.TrimSpace(line[idx+1:])
		if !strings.Contains(payload, `"success"`) {
			continue
		}
		var result acharPromoRedirectResult
		if err := json.Unmarshal([]byte(payload), &result); err != nil {
			continue
		}
		if !result.Success {
			return "", fmt.Errorf("redirect action unsuccessful")
		}
		if result.URL != "" && !isGoogleShoppingURL(result.URL) {
			return result.URL, nil
		}
		source = strings.ToLower(strings.TrimSpace(source))
		var fallback string
		for _, offer := range result.Offers {
			link := offer.AffiliateLink
			if link == "" {
				link = offer.Link
			}
			if link == "" || isGoogleShoppingURL(link) {
				continue
			}
			if source != "" && strings.Contains(strings.ToLower(offer.Merchant.Name), source) {
				return link, nil
			}
			if fallback == "" {
				fallback = link
			}
		}
		return fallback, nil
	}
	return "", fmt.Errorf("redirect action returned no link")
}

// GetCategory returns the product category this extractor is optimised for
func (e *AcharPromoExtractorV2) GetCategory() models.ProductCategory {
	return models.CategoryOther
}
