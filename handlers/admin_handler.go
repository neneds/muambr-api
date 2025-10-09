package handlers

import (
	"net/http"
	"muambr-api/utils"
	"muambr-api/localization"
	"github.com/gin-gonic/gin"
)

// AdminHandler handles administrative endpoints
type AdminHandler struct {
	exchangeRateService *utils.ExchangeRateService
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		exchangeRateService: utils.NewExchangeRateService(),
	}
}

// GetExchangeRateStatus returns the current status of exchange rate cache
func (h *AdminHandler) GetExchangeRateStatus(c *gin.Context) {
	status := h.exchangeRateService.GetCacheStatus()
	
	response := gin.H{
		"cache_status": status,
		"cache_count": len(status),
	}
	
	c.JSON(http.StatusOK, response)
}

// ClearExchangeRateCache clears all cached exchange rates
func (h *AdminHandler) ClearExchangeRateCache(c *gin.Context) {
	h.exchangeRateService.ClearCache()
	
	c.JSON(http.StatusOK, gin.H{
		"message": localization.T("api.success.cache_cleared"),
	})
}

// TestExchangeRateAPI tests the exchange rate API with a specific currency
func (h *AdminHandler) TestExchangeRateAPI(c *gin.Context) {
	currency := c.Query("currency")
	if currency == "" {
		currency = "USD" // Default to USD
	}
	
	rates, err := h.exchangeRateService.GetExchangeRates(currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	// Return a subset of rates for readability
	selectedRates := make(map[string]float64)
	commonCurrencies := []string{"USD", "EUR", "BRL", "GBP", "JPY"}
	
	for _, curr := range commonCurrencies {
		if rate, exists := rates[curr]; exists {
			selectedRates[curr] = rate
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"base_currency": currency,
		"rates": selectedRates,
		"total_rates": len(rates),
	})
}