package htmlparser

// ParsedProductData represents the parsed product data from HTML
type ParsedProductData struct {
	Title       string   `json:"title"`
	Price       *float64 `json:"price,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	ImageURL    string   `json:"imageURL,omitempty"`
	Description string   `json:"description,omitempty"`
}
