package other

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	ID        string                    `json:"id"`
	Messages  []acharPromoChatMessage   `json:"messages"`
	Trigger   string                    `json:"trigger"`
	MessageID string                    `json:"messageId"`
}

// acharPromoChatMessage represents a message in the chat request
type acharPromoChatMessage struct {
	ID      string                      `json:"id"`
	Role    string                      `json:"role"`
	Content string                      `json:"content"`
	Parts   []acharPromoChatMessagePart `json:"parts"`
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
				ID:      utils.GenerateUUID(),
				Role:    "user",
				Content: productName,
				Parts: []acharPromoChatMessagePart{
					{Type: "text", Text: productName},
				},
			},
		},
		Trigger:   "user",
		MessageID: utils.GenerateUUID(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	products, err := e.fetchChatProducts(chatURL, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("chat API request failed: %w", err)
	}

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
	req.Header.Set("User-Agent", utils.GetRandomUserAgent())
	req.Header.Set("Accept", "*/*")

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

	products := parseSSEProducts(string(respBody))
	if len(products) == 0 {
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
		if p.URL != "" {
			u := p.URL
			storeURL = &u
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
// GetCategory returns the product category this extractor is optimised for
func (e *AcharPromoExtractorV2) GetCategory() models.ProductCategory {
	return models.CategoryOther
}
