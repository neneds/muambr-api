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

// CountryRunMeta tracks extractor outcomes for one comparison country.
type CountryRunMeta struct {
	Attempted int
	Succeeded int
	Failed    int
}

// ExtractionMeta tracks provider execution outcomes for partial-result status.
type ExtractionMeta struct {
	ProvidersAttempted int
	ProvidersSucceeded int
	ProvidersFailed    int
	FailedProviders    []string
	ByCountry          map[string]CountryRunMeta
}

// ComparisonEngineInput is everything needed to build a product-aligned comparison result.
type ComparisonEngineInput struct {
	ProductName         string
	Brand               string
	Model               string
	Category            *models.ProductCategory
	CategoryConfidence  float64
	BaseCountry         models.Country
	CurrentCountry      models.Country
	NormalizedCurrency  string
	Observed            *models.ObservedPriceInput
	Sections            []models.CountrySection
	Comparisons         []models.ProductComparison
	ComparisonCountries []models.ComparisonCountrySpec
	Meta                ExtractionMeta
	EntryMethod         string
	ExchangeRate        *models.ExchangeRateInfo
	Now                 time.Time
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

	comparisonCountries := e.buildCountryComparisons(in, prices, capturedAt)
	bestDeal := e.buildBestDeal(comparisonCountries, bestBase, in.NormalizedCurrency)

	status := e.resolveStatus(len(prices), in.Meta, avgMatch, comparisonCountries)
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
		Success:      true,
		ComparisonID: comparisonID,
		Status:       status,
		Product: models.ProductSummary{
			Name:     in.ProductName,
			Brand:    in.Brand,
			Model:    in.Model,
			Category: in.Category,
		},
		Category: categoryInfo,
		Observed: observed,
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
		ComparisonCountries:     comparisonCountries,
		BestDeal:                bestDeal,
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

func (e *ComparisonEngine) resolveStatus(priceCount int, meta ExtractionMeta, avgMatch float64, countries []models.CountryComparison) models.ComparisonStatus {
	if priceCount == 0 {
		return models.ComparisonStatusEmpty
	}
	if avgMatch > 0 && avgMatch < MatchConfidenceMedium {
		return models.ComparisonStatusLowConfidence
	}
	hasOK := false
	hasGap := false
	for _, c := range countries {
		switch c.Status {
		case models.CountryStatusOK:
			hasOK = true
		case models.CountryStatusNoPrices, models.CountryStatusProviderFailed, models.CountryStatusMatchUnavailable:
			hasGap = true
		}
	}
	if hasOK && hasGap {
		return models.ComparisonStatusPartial
	}
	if meta.ProvidersFailed > 0 && meta.ProvidersSucceeded > 0 {
		return models.ComparisonStatusPartial
	}
	return models.ComparisonStatusComplete
}

func (e *ComparisonEngine) comparisonSpecs(in ComparisonEngineInput) []models.ComparisonCountrySpec {
	if len(in.ComparisonCountries) > 0 {
		return in.ComparisonCountries
	}
	return models.ResolveComparisonSet(models.ComparisonSetInput{
		BaseCountry:     in.BaseCountry,
		ProductLocation: in.CurrentCountry,
	})
}

func (e *ComparisonEngine) buildCountryComparisons(in ComparisonEngineInput, prices []models.PriceOffer, capturedAt string) []models.CountryComparison {
	specs := e.comparisonSpecs(in)
	out := make([]models.CountryComparison, 0, len(specs))
	for _, spec := range specs {
		code := string(spec.Country)
		best := e.bestPriceInCountry(in.Sections, code, in.NormalizedCurrency, capturedAt)
		storeCount := 0
		var confSum float64
		for _, p := range prices {
			if strings.EqualFold(p.Country, code) {
				storeCount++
				confSum += p.MatchConfidence
			}
		}

		status := models.CountryStatusNoPrices
		run := in.Meta.ByCountry[code]
		if best != nil {
			status = models.CountryStatusOK
		} else if run.Failed > 0 && run.Succeeded == 0 {
			status = models.CountryStatusProviderFailed
		}

		var match *float64
		if storeCount > 0 {
			v := round2(confSum / float64(storeCount))
			match = &v
		} else if best != nil && best.MatchConfidence != nil {
			match = best.MatchConfidence
		}

		roles := spec.Roles
		if roles == nil {
			roles = []string{}
		}
		out = append(out, models.CountryComparison{
			Country:         code,
			Roles:           roles,
			Status:          status,
			BestPrice:       best,
			StoreCount:      storeCount,
			MatchConfidence: match,
		})
	}
	return out
}

func (e *ComparisonEngine) buildBestDeal(countries []models.CountryComparison, bestBase *models.MoneyAmount, currency string) *models.BestDeal {
	var winner *models.CountryComparison
	var winnerAmt float64
	for i := range countries {
		c := &countries[i]
		if c.Status != models.CountryStatusOK || c.BestPrice == nil {
			continue
		}
		amt := c.BestPrice.ComparableAmount()
		if amt <= 0 {
			continue
		}
		if winner == nil || amt < winnerAmt {
			winner = c
			winnerAmt = amt
		}
	}
	if winner == nil {
		return nil
	}

	deal := &models.BestDeal{
		Country:         winner.Country,
		NormalizedPrice: round2(winnerAmt),
		Currency:        currency,
		Store:           winner.BestPrice.Store,
	}
	if bestBase != nil {
		baseAmt := bestBase.ComparableAmount()
		if baseAmt > 0 {
			savings := round2(baseAmt - winnerAmt)
			if savings > 0 {
				pct := round2((savings / baseAmt) * 100)
				deal.SavingsVsBase = &savings
				deal.SavingsPercentageVsBase = &pct
			}
		}
	}
	return deal
}

func (e *ComparisonEngine) flattenOffers(sections []models.CountrySection, normalizedCurrency, capturedAt string) []models.PriceOffer {
	var offers []models.PriceOffer
	for _, section := range sections {
		for _, c := range section.Comparisons {
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

			normAmt, normCur := conversionFields(c.Currency, normalizedCurrency, convertedAmountPtr(c))

			offers = append(offers, models.PriceOffer{
				Store:              c.StoreName,
				Country:            c.Country,
				Amount:             c.Price,
				Currency:           c.Currency,
				NormalizedAmount:   normAmt,
				NormalizedCurrency: normCur,
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

	normAmt, normCur := conversionFields(best.Currency, normalizedCurrency, convertedAmountPtr(*best))

	return &models.MoneyAmount{
		Amount:             best.Price,
		Currency:           best.Currency,
		Country:            best.Country,
		NormalizedAmount:   normAmt,
		NormalizedCurrency: normCur,
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

	var converted *float64
	if !strings.EqualFold(observed.Currency, normalizedCurrency) &&
		rateInfo != nil && rateInfo.Rate > 0 &&
		strings.EqualFold(rateInfo.Base, observed.Currency) &&
		strings.EqualFold(rateInfo.Target, normalizedCurrency) {
		v := round2(observed.Amount * rateInfo.Rate)
		converted = &v
	}
	normAmt, normCur := conversionFields(observed.Currency, normalizedCurrency, converted)

	return &models.MoneyAmount{
		Amount:             observed.Amount,
		Currency:           strings.ToUpper(observed.Currency),
		Country:            country,
		NormalizedAmount:   normAmt,
		NormalizedCurrency: normCur,
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
		comparedAmount = observed.ComparableAmount()
	} else if bestCurrent != nil {
		comparedAmount = bestCurrent.ComparableAmount()
		comparedAgainst = "best_current"
	} else {
		return nil
	}

	baseAmount := bestBase.ComparableAmount()
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

// conversionFields returns normalized amount/currency only when the native
// currency differs from the request currency and a converted amount is available.
func conversionFields(nativeCurrency, targetCurrency string, convertedAmount *float64) (*float64, *string) {
	if strings.EqualFold(nativeCurrency, targetCurrency) || convertedAmount == nil {
		return nil, nil
	}
	amt := round2(*convertedAmount)
	cur := strings.ToUpper(targetCurrency)
	return &amt, &cur
}

func convertedAmountPtr(c models.ProductComparison) *float64 {
	if c.ConvertedPrice == nil {
		return nil
	}
	v := c.ConvertedPrice.Price
	return &v
}
