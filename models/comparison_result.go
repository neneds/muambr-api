package models

// ComparisonStatus describes the lifecycle state of a comparison response.
type ComparisonStatus string

const (
	ComparisonStatusComplete      ComparisonStatus = "complete"
	ComparisonStatusPartial       ComparisonStatus = "partial"
	ComparisonStatusEmpty         ComparisonStatus = "empty"
	ComparisonStatusLowConfidence ComparisonStatus = "low_confidence"
)

// APIErrorCode is a machine-readable error code for mobile clients.
type APIErrorCode string

const (
	ErrorCodeProductNotFound          APIErrorCode = "PRODUCT_NOT_FOUND"
	ErrorCodeProductMatchLowConfidence APIErrorCode = "PRODUCT_MATCH_LOW_CONFIDENCE"
	ErrorCodePriceNotFound            APIErrorCode = "PRICE_NOT_FOUND"
	ErrorCodeCurrencyUnknown          APIErrorCode = "CURRENCY_UNKNOWN"
	ErrorCodeCountryUnknown           APIErrorCode = "COUNTRY_UNKNOWN"
	ErrorCodeNoComparisonSources      APIErrorCode = "NO_COMPARISON_SOURCES"
	ErrorCodeProviderTimeout          APIErrorCode = "PROVIDER_TIMEOUT"
	ErrorCodeExchangeRateUnavailable  APIErrorCode = "EXCHANGE_RATE_UNAVAILABLE"
	ErrorCodeComparisonTimeout        APIErrorCode = "COMPARISON_TIMEOUT"
	ErrorCodeInvalidRequest           APIErrorCode = "INVALID_REQUEST"
	ErrorCodeInternalError            APIErrorCode = "INTERNAL_ERROR"
)

// DealScoreLabel is a human-readable interpretation of DealScore.Value.
type DealScoreLabel string

const (
	DealLabelExcellent DealScoreLabel = "Excellent Deal"
	DealLabelGood      DealScoreLabel = "Good Deal"
	DealLabelFair      DealScoreLabel = "Fair Price"
	DealLabelExpensive DealScoreLabel = "Expensive"
	DealLabelPoor      DealScoreLabel = "Poor Deal"
	DealLabelUncertain DealScoreLabel = "Possible Match"
)

// MoneyAmount is a price with optional normalization to the comparison currency.
// normalizedAmount / normalizedCurrency are set only when Currency differs from
// the request currency; otherwise they are omitted so the client does not show
// a duplicate "converted" value.
type MoneyAmount struct {
	Amount             float64  `json:"amount"`
	Currency           string   `json:"currency"`
	Country            string   `json:"country,omitempty"`
	NormalizedAmount   *float64 `json:"normalizedAmount,omitempty"`
	NormalizedCurrency *string  `json:"normalizedCurrency,omitempty"`
	Store              string   `json:"store,omitempty"`
	URL                *string  `json:"url,omitempty"`
	MatchConfidence    *float64 `json:"matchConfidence,omitempty"`
	CapturedAt         *string  `json:"capturedAt,omitempty"`
}

// ComparableAmount is the price in the request (normalization) currency.
// When normalized fields are omitted, Amount is already in that currency.
func (m *MoneyAmount) ComparableAmount() float64 {
	if m == nil {
		return 0
	}
	if m.NormalizedAmount != nil {
		return *m.NormalizedAmount
	}
	return m.Amount
}

// ObservedPriceInput is the price the user actually found (request body).
type ObservedPriceInput struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Country  string  `json:"country,omitempty"`
	Store    string  `json:"store,omitempty"`
}

// ProductIdentityInput describes the product being compared (request body).
type ProductIdentityInput struct {
	Name     string           `json:"name"`
	Brand    string           `json:"brand,omitempty"`
	Model    string           `json:"model,omitempty"`
	Category *ProductCategory `json:"category,omitempty"`
}

// ComparisonSourceInput describes how the product was entered.
type ComparisonSourceInput struct {
	Type  string `json:"type"` // manual | url | camera | barcode
	Store string `json:"store,omitempty"`
	URL   string `json:"url,omitempty"`
}

// CategoryInfo is the category used for provider selection.
type CategoryInfo struct {
	ID         ProductCategory `json:"id"`
	Name       string          `json:"name"`
	Confidence float64         `json:"confidence"`
}

// CountryInfo is a country + its default currency.
type CountryInfo struct {
	Country  string `json:"country"`
	Currency string `json:"currency"`
	Name     string `json:"name,omitempty"`
}

// SavingsResult is the primary savings calculation vs the base country.
type SavingsResult struct {
	Amount             float64 `json:"amount"`
	Currency           string  `json:"currency"`
	Percentage         float64 `json:"percentage"`
	IsCheaper          bool    `json:"isCheaper"`
	ComparedAgainst    string  `json:"comparedAgainst"` // "observed" | "best_current"
	Explanation        string  `json:"explanation"`
}

// DealScore is a normalized 0–100 score with a label and short explanation.
type DealScore struct {
	Value       int            `json:"value"`
	Label       DealScoreLabel `json:"label"`
	Explanation string         `json:"explanation"`
	IsDefinitive bool          `json:"isDefinitive"` // false when match confidence is low
}

// ExchangeRateInfo exposes the rate used for normalization.
type ExchangeRateInfo struct {
	Base           string  `json:"base"`
	Target         string  `json:"target"`
	Rate           float64 `json:"rate"`
	Source         string  `json:"source"`
	Timestamp      string  `json:"timestamp"`
}

// ComparisonMetadata carries lifecycle and provider diagnostics.
type ComparisonMetadata struct {
	ProvidersAttempted  int      `json:"providersAttempted"`
	ProvidersSucceeded  int      `json:"providersSucceeded"`
	ProvidersFailed     int      `json:"providersFailed"`
	FailedProviders     []string `json:"failedProviders,omitempty"`
	PriceType           string   `json:"priceType"` // "retail" — V1 does not include landed cost
	EntryMethod         string   `json:"entryMethod,omitempty"`
}

// PriceOffer is a single store price in the comparison result.
// normalizedAmount / normalizedCurrency are omitted when Currency already
// matches the request currency.
type PriceOffer struct {
	Store              string   `json:"store"`
	Country            string   `json:"country"`
	Amount             float64  `json:"amount"`
	Currency           string   `json:"currency"`
	NormalizedAmount   *float64 `json:"normalizedAmount,omitempty"`
	NormalizedCurrency *string  `json:"normalizedCurrency,omitempty"`
	URL                *string  `json:"url,omitempty"`
	Availability       string   `json:"availability,omitempty"`
	MatchConfidence    float64  `json:"matchConfidence"`
	CapturedAt         string   `json:"capturedAt"`
	ProductName        string   `json:"productName,omitempty"`
	ImageURL           *string  `json:"imageURL,omitempty"`
}

// ProductComparisonResult is the V1 product-aligned comparison response.
// Used by POST /api/v1/product-comparisons.
type ProductComparisonResult struct {
	Success                 bool              `json:"success"`
	Code                    *APIErrorCode     `json:"code,omitempty"`
	Message                 *string           `json:"message,omitempty"`
	ComparisonID            string            `json:"comparisonId"`
	Status                  ComparisonStatus  `json:"status"`
	Product                 ProductSummary    `json:"product"`
	Category                *CategoryInfo     `json:"category,omitempty"`
	Observed                *MoneyAmount      `json:"observed,omitempty"`
	BaseCountry             CountryInfo       `json:"baseCountry"`
	CurrentCountry          CountryInfo       `json:"currentCountry"`
	Prices                  []PriceOffer      `json:"prices"`
	Sections                []CountrySection  `json:"sections"`
	BestCurrentCountryPrice *MoneyAmount      `json:"bestCurrentCountryPrice,omitempty"`
	BestBaseCountryPrice    *MoneyAmount      `json:"bestBaseCountryPrice,omitempty"`
	Savings                 *SavingsResult    `json:"savings,omitempty"`
	DealScore               *DealScore        `json:"dealScore,omitempty"`
	ExchangeRate            *ExchangeRateInfo `json:"exchangeRate,omitempty"`
	Metadata                ComparisonMetadata `json:"metadata"`
	CapturedAt              string            `json:"capturedAt"`
	ExpiresAt               string            `json:"expiresAt"`
	TotalResults            int               `json:"totalResults"`
}

// ProductSummary is the normalized product identity returned to clients.
type ProductSummary struct {
	Name     string           `json:"name"`
	Brand    string           `json:"brand,omitempty"`
	Model    string           `json:"model,omitempty"`
	Category *ProductCategory `json:"category,omitempty"`
}
