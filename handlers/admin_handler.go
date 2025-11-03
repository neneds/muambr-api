package handlers

import (
	"net/http"
	"muambr-api/utils"
	"github.com/gin-gonic/gin"
)

// ExchangeRateHandler handles currency exchange rate endpoints
type ExchangeRateHandler struct {
	exchangeRateService *utils.ExchangeRateService
}

// NewExchangeRateHandler creates a new ExchangeRateHandler
func NewExchangeRateHandler() *ExchangeRateHandler {
	return &ExchangeRateHandler{
		exchangeRateService: utils.NewExchangeRateService(),
	}
}

// ExchangeRate represents a currency exchange rate
type ExchangeRate struct {
	Currency string  `json:"currency"`
	Rate     float64 `json:"rate"`
}

// GetExchangeRates returns exchange rates for a base currency
func (h *ExchangeRateHandler) GetExchangeRates(c *gin.Context) {
	baseCurrency := c.Query("baseCurrency")
	if baseCurrency == "" {
		baseCurrency = "USD" // Default to USD
	}
	
	filteredCurrencies := []string{"USD", "EUR", "BRL", "GBP", "JPY"}

	// Get all rates for the base currency (uses cache if available)
	rates, err := h.exchangeRateService.GetExchangeRates(baseCurrency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	// Filter only the currencies we want to return
	exchangeRatesList := make([]ExchangeRate, 0)

	for _, curr := range filteredCurrencies {
		if rate, exists := rates[curr]; exists {
			exchangeRatesList = append(exchangeRatesList, ExchangeRate{
				Currency: curr,
				Rate:     rate,
			})
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"base_currency": baseCurrency,
		"rates": exchangeRatesList,
		"total_rates": len(exchangeRatesList),
	})
}