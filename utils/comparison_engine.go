package utils

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"muambr-api/models"
)

const (
	// ComparisonCacheTTL is how long a comparison result is considered fresh.
	ComparisonCacheTTL = 15 * time.Minute

	MatchConfidenceHigh   = 0.90
	MatchConfidenceMedium = 0.75
)

// ExtractionMeta tracks provider execution outcomes for partial-result status.
type ExtractionMeta struct {
	ProvidersAttempted int
	ProvidersSucceeded int
	ProvidersFailed    int
	FailedProviders    []string
}

// ComparisonEngineInput is everything needed to build a product-aligned comparison result.
type ComparisonEngineInput struct {
	ProductName      string
	Brand            string
	Model            string
	Category         *models.ProductCategory
	CategoryConfidence float64
	BaseCountry      models.Country
	CurrentCountry   models.Country
	NormalizedCurrency string
	Observed         *models.ObservedPriceInput
	Sections         []models.CountrySection
	Comparisons      []models.ProductComparison
	Meta             ExtractionMeta
	EntryMethod      string
	ExchangeRate     *models.ExchangeRateInfo
	Now              time.Time
}

// ComparisonEngine builds savings, deal scores, and structured comparison responses.
type ComparisonEngine struct {
	processor *ComparisonProcessor
}

// NewComparisonEngine creates a ComparisonEngine.
func NewComparisonEngine() *ComparisonEngine {
	return &ComparisonEngine{
		processor: NewComparisonProcessor(),
	}
}

// BuildResult constructs a ProductComparisonResult from extractor output + observed price.
func (e *ComparisonEngine) BuildResult(in ComparisonEngineInput) models.ProductComparisonResult {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	capturedAt := now.Format(time.RFC3339)
	expiresAt := now.Add(ComparisonCacheTTL).Format(time.RFC3339)
	comparisonID := GenerateUUID()

	// Annotate offers with match confidence
	query := in.ProductName
	for i := range in.Sections {
		for j := range in.Sections[i].Comparisons {
			conf := MatchConfidence(query, in.Sections[i].Comparisons[j].ProductName)
			in.Sections[i].Comparisons[j].MatchConfidence = &conf
			if in.Sections[i].Comparisons[j].LastUpdated == nil {
				ts := capturedAt
				in.Sections[i].Comparisons[j].LastUpdated = &ts
			}
			if in.Sections[i].Comparisons[j].Availability == "" {
				in.Sections[i].Comparisons[j].Availability = "unknown"
			}
		}
	}

	prices := e.flattenOffers(in.Sections, in.NormalizedCurrency, capturedAt)
	bestBase := e.bestPriceInCountry(in.Sections, string(in.BaseCountry), in.NormalizedCurrency, capturedAt)
	bestCurrent := e.bestPriceInCountry(in.Sections, string(in.CurrentCountry), in.NormalizedCurrency, capturedAt)

	var observed *models.MoneyAmount
	if in.Observed != nil && in.Observed.Amount > 0 {
		observed = e.normalizeObserved(in.Observed, in.CurrentCountry, in.NormalizedCurrency, in.ExchangeRate, capturedAt)
	}

	savings := e.calculateSavings(observed, bestBase, bestCurrent, in.NormalizedCurrency)
	avgMatch := averageMatchConfidence(prices)
	dealScore := e.calculateDealScore(savings, avgMatch)

	status := e.resolveStatus(len(prices), in.Meta, avgMatch)
	totalResults := 0
	for _, s := range in.Sections {
		totalResults += s.ResultsCount
	}

	var categoryInfo *models.CategoryInfo
	if in.Category != nil {
		conf := in.CategoryConfidence
		if conf <= 0 {
			conf = 1.0
		}
		categoryInfo = &models.CategoryInfo{
			ID:         *in.Category,
			Name:       in.Category.CategoryDisplayName(),
			Confidence: conf,
		}
	}

	currentCountry := in.CurrentCountry
	if currentCountry == "" {
		currentCountry = in.BaseCountry
	}

	result := models.ProductComparisonResult{
		Success:                 true,
		ComparisonID:            comparisonID,
		Status:                  status,
		Product: models.ProductSummary{
			Name:     in.ProductName,
			Brand:    in.Brand,
			Model:    in.Model,
			Category: in.Category,
		},
		Category:                categoryInfo,
		Observed:                observed,
		BaseCountry: models.CountryInfo{
			Country:  string(in.BaseCountry),
			Currency: in.BaseCountry.GetCurrencyCode(),
			Name:     in.BaseCountry.GetCountryName(),
		},
		CurrentCountry: models.CountryInfo{
			Country:  string(currentCountry),
			Currency: currentCountry.GetCurrencyCode(),
			Name:     currentCountry.GetCountryName(),
		},
		Prices:                  prices,
		Sections:                in.Sections,
		BestCurrentCountryPrice: bestCurrent,
		BestBaseCountryPrice:    bestBase,
		Savings:                 savings,
		DealScore:               dealScore,
		ExchangeRate:            in.ExchangeRate,
		Metadata: models.ComparisonMetadata{
			ProvidersAttempted: in.Meta.ProvidersAttempted,
			ProvidersSucceeded: in.Meta.ProvidersSucceeded,
			ProvidersFailed:    in.Meta.ProvidersFailed,
			FailedProviders:    in.Meta.FailedProviders,
			PriceType:          "retail",
			EntryMethod:        in.EntryMethod,
		},
		CapturedAt:   capturedAt,
		ExpiresAt:    expiresAt,
		TotalResults: totalResults,
	}

	if status == models.ComparisonStatusEmpty {
		msg := "We couldn't find comparable prices."
		code := models.ErrorCodeProductNotFound
		result.Message = &msg
		result.Code = &code
	}

	return result
}

func (e *ComparisonEngine) resolveStatus(priceCount int, meta ExtractionMeta, avgMatch float64) models.ComparisonStatus {
	if priceCount == 0 {
		return models.ComparisonStatusEmpty
	}
	if avgMatch > 0 && avgMatch < MatchConfidenceMedium {
		return models.ComparisonStatusLowConfidence
	}
	if meta.ProvidersFailed > 0 && meta.ProvidersSucceeded > 0 {
		return models.ComparisonStatusPartial
	}
	return models.ComparisonStatusComplete
}

func (e *ComparisonEngine) flattenOffers(sections []models.CountrySection, normalizedCurrency, capturedAt string) []models.PriceOffer {
	var offers []models.PriceOffer
	for _, section := range sections {
		for _, c := range section.Comparisons {
			normalized := e.processor.getEffectivePrice(c)

			conf := 0.5
			if c.MatchConfidence != nil {
				conf = *c.MatchConfidence
			}
			ts := capturedAt
			if c.LastUpdated != nil {
				ts = *c.LastUpdated
			}
			availability := c.Availability
			if availability == "" {
				availability = "unknown"
			}

			normCurrency := normalizedCurrency
			if c.ConvertedPrice != nil {
				normCurrency = c.ConvertedPrice.Currency
			}

			offers = append(offers, models.PriceOffer{
				Store:              c.StoreName,
				Country:            c.Country,
				Amount:             c.Price,
				Currency:           c.Currency,
				NormalizedAmount:   normalized,
				NormalizedCurrency: normCurrency,
				URL:                c.StoreURL,
				Availability:       availability,
				MatchConfidence:    conf,
				CapturedAt:         ts,
				ProductName:        c.ProductName,
				ImageURL:           c.ImageURL,
			})
		}
	}
	return offers
}

func (e *ComparisonEngine) bestPriceInCountry(sections []models.CountrySection, country, normalizedCurrency, capturedAt string) *models.MoneyAmount {
	var best *models.ProductComparison
	var bestEffective float64

	for _, section := range sections {
		if !strings.EqualFold(section.Country, country) {
			continue
		}
		for i := range section.Comparisons {
			c := &section.Comparisons[i]
			effective := e.processor.getEffectivePrice(*c)
			if effective <= 0 {
				continue
			}
			if best == nil || effective < bestEffective {
				best = c
				bestEffective = effective
			}
		}
	}

	if best == nil {
		return nil
	}

	conf := 0.5
	if best.MatchConfidence != nil {
		conf = *best.MatchConfidence
	}

	return &models.MoneyAmount{
		Amount:             best.Price,
		Currency:           best.Currency,
		Country:            best.Country,
		NormalizedAmount:   bestEffective,
		NormalizedCurrency: normalizedCurrency,
		Store:              best.StoreName,
		URL:                best.StoreURL,
		MatchConfidence:    &conf,
		CapturedAt:         &capturedAt,
	}
}

func (e *ComparisonEngine) normalizeObserved(
	observed *models.ObservedPriceInput,
	currentCountry models.Country,
	normalizedCurrency string,
	rateInfo *models.ExchangeRateInfo,
	capturedAt string,
) *models.MoneyAmount {
	country := observed.Country
	if country == "" {
		country = string(currentCountry)
	}

	normalized := observed.Amount
	if strings.EqualFold(observed.Currency, normalizedCurrency) {
		normalized = observed.Amount
	} else if rateInfo != nil && rateInfo.Rate > 0 &&
		strings.EqualFold(rateInfo.Base, observed.Currency) &&
		strings.EqualFold(rateInfo.Target, normalizedCurrency) {
		normalized = observed.Amount * rateInfo.Rate
	}

	return &models.MoneyAmount{
		Amount:             observed.Amount,
		Currency:           strings.ToUpper(observed.Currency),
		Country:            country,
		NormalizedAmount:   round2(normalized),
		NormalizedCurrency: normalizedCurrency,
		Store:              observed.Store,
		CapturedAt:         &capturedAt,
	}
}

func (e *ComparisonEngine) calculateSavings(
	observed *models.MoneyAmount,
	bestBase *models.MoneyAmount,
	bestCurrent *models.MoneyAmount,
	currency string,
) *models.SavingsResult {
	if bestBase == nil {
		return nil
	}

	var comparedAmount float64
	comparedAgainst := "observed"
	if observed != nil {
		comparedAmount = observed.NormalizedAmount
	} else if bestCurrent != nil {
		comparedAmount = bestCurrent.NormalizedAmount
		comparedAgainst = "best_current"
	} else {
		return nil
	}

	baseAmount := bestBase.NormalizedAmount
	if baseAmount <= 0 {
		return nil
	}

	savingsAmount := round2(baseAmount - comparedAmount)
	percentage := round2((savingsAmount / baseAmount) * 100)
	isCheaper := savingsAmount > 0

	explanation := fmt.Sprintf("%.0f%% cheaper than the best comparable price in your base country", math.Abs(percentage))
	if !isCheaper {
		explanation = fmt.Sprintf("%.0f%% more expensive than the best comparable price in your base country", math.Abs(percentage))
	}

	return &models.SavingsResult{
		Amount:          savingsAmount,
		Currency:        currency,
		Percentage:      percentage,
		IsCheaper:       isCheaper,
		ComparedAgainst: comparedAgainst,
		Explanation:     explanation,
	}
}

func (e *ComparisonEngine) calculateDealScore(savings *models.SavingsResult, avgMatch float64) *models.DealScore {
	if savings == nil {
		return nil
	}

	// Map savings percentage into 0–100. 0% savings → 50, +50% → 100, -50% → 0.
	raw := 50.0 + savings.Percentage
	if raw < 0 {
		raw = 0
	}
	if raw > 100 {
		raw = 100
	}

	// Soft-weight by match confidence so uncertain matches don't look definitive.
	if avgMatch > 0 {
		raw = raw*0.7 + (avgMatch * 100 * 0.3)
	}

	value := int(math.Round(raw))
	isDefinitive := avgMatch >= MatchConfidenceMedium || avgMatch == 0

	label := dealLabelForValue(value)
	if !isDefinitive {
		label = models.DealLabelUncertain
	}

	explanation := savings.Explanation
	if !isDefinitive {
		explanation = "Possible match — review before relying on this comparison. " + explanation
	}

	return &models.DealScore{
		Value:        value,
		Label:        label,
		Explanation:  explanation,
		IsDefinitive: isDefinitive,
	}
}

func dealLabelForValue(value int) models.DealScoreLabel {
	switch {
	case value >= 90:
		return models.DealLabelExcellent
	case value >= 75:
		return models.DealLabelGood
	case value >= 50:
		return models.DealLabelFair
	case value >= 25:
		return models.DealLabelExpensive
	default:
		return models.DealLabelPoor
	}
}

// MatchConfidence returns a 0–1 score based on token overlap between query and product name.
func MatchConfidence(query, productName string) float64 {
	qTokens := tokenize(query)
	pTokens := tokenize(productName)
	if len(qTokens) == 0 || len(pTokens) == 0 {
		return 0.5
	}

	qSet := make(map[string]struct{}, len(qTokens))
	for _, t := range qTokens {
		qSet[t] = struct{}{}
	}
	pSet := make(map[string]struct{}, len(pTokens))
	for _, t := range pTokens {
		pSet[t] = struct{}{}
	}

	intersection := 0
	for t := range qSet {
		if _, ok := pSet[t]; ok {
			intersection++
		}
	}

	// Jaccard similarity, biased toward query coverage (how much of the query appears in the product)
	union := len(qSet) + len(pSet) - intersection
	jaccard := 0.0
	if union > 0 {
		jaccard = float64(intersection) / float64(union)
	}
	coverage := float64(intersection) / float64(len(qSet))

	score := 0.4*jaccard + 0.6*coverage
	if score > 1 {
		score = 1
	}
	return round2(score)
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tok := b.String()
		b.Reset()
		// Skip very short noise tokens except digits/model fragments
		if len(tok) < 2 && !isDigitToken(tok) {
			return
		}
		tokens = append(tokens, tok)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return tokens
}

func isDigitToken(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

func averageMatchConfidence(offers []models.PriceOffer) float64 {
	if len(offers) == 0 {
		return 0
	}
	var sum float64
	for _, o := range offers {
		sum += o.MatchConfidence
	}
	return sum / float64(len(offers))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
