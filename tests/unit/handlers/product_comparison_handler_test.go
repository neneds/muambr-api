package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"muambr-api/handlers"
	"muambr-api/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProductComparisonRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := handlers.NewProductComparisonHandler()
	router.POST("/api/v1/product-comparisons", handler.CreateProductComparison)
	return router
}

func TestCreateProductComparison_MissingProductName(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{"baseCountry":"BR","currentCountry":"US","product":{}}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Code)
	assert.Equal(t, models.ErrorCodeInvalidRequest, *resp.Code)
	require.NotNil(t, resp.Message)
	assert.Equal(t, "Product name is required", *resp.Message)
}

func TestCreateProductComparison_MissingProductName_Portuguese(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{"baseCountry":"BR","currentCountry":"US","product":{}}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "pt-BR")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Message)
	assert.Equal(t, "Nome do produto é obrigatório", *resp.Message)
}

func TestCreateProductComparison_InvalidCountry(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{"baseCountry":"XX","currentCountry":"US","product":{"name":"Sony WH-1000XM6"}}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Code)
	assert.Equal(t, models.ErrorCodeCountryUnknown, *resp.Code)
}

func TestCreateProductComparison_InvalidObservedPrice(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{
		"baseCountry":"BR",
		"currentCountry":"US",
		"product":{"name":"Sony WH-1000XM6","category":"electronics"},
		"observedPrice":{"amount":0,"currency":"USD"}
	}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Code)
	assert.Equal(t, models.ErrorCodePriceNotFound, *resp.Code)
}

func TestCreateProductComparison_ValidRequestShape(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{
		"baseCountry":"BR",
		"currentCountry":"US",
		"product":{"name":"Sony WH-1000XM6","brand":"Sony","model":"WH-1000XM6","category":"electronics"},
		"observedPrice":{"amount":349,"currency":"USD"},
		"source":{"type":"manual","store":"Amazon US"},
		"limit":5
	}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Extractors may return empty in CI; response must still follow the product contract.
	// Category "electronics" falls back to generic ("other") providers when none are registered.
	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.ComparisonID)
	assert.NotEmpty(t, resp.CapturedAt)
	assert.NotEmpty(t, resp.ExpiresAt)
	assert.Equal(t, "BR", resp.BaseCountry.Country)
	assert.Equal(t, "US", resp.CurrentCountry.Country)
	assert.Equal(t, "retail", resp.Metadata.PriceType)
	assert.Equal(t, "Sony WH-1000XM6", resp.Product.Name)
	require.NotNil(t, resp.Observed)
	assert.Equal(t, 349.0, resp.Observed.Amount)
	assert.Equal(t, "USD", resp.Observed.Currency)
	require.NotNil(t, resp.Observed.NormalizedCurrency)
	assert.Equal(t, "BRL", *resp.Observed.NormalizedCurrency)
	require.NotEmpty(t, resp.ComparisonCountries)
	assert.Equal(t, "BR", resp.ComparisonCountries[0].Country)
}

func TestCreateProductComparison_ComparisonCountriesUnknown(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{
		"baseCountry":"BR",
		"product":{"name":"Milka strawberry 250g"},
		"comparisonCountries":["BR","ZZ"]
	}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Code)
	assert.Equal(t, models.ErrorCodeCountryUnknown, *resp.Code)
}

func TestCreateProductComparison_ProductLocationWinsOverCurrentCountry(t *testing.T) {
	router := setupProductComparisonRouter()
	body := `{
		"baseCountry":"BR",
		"currentCountry":"US",
		"productLocation":{"country":"GB","source":"device"},
		"comparisonCountries":["BR","GB"],
		"product":{"name":"Milka strawberry 250g","category":"grocery"}
	}`
	req, _ := http.NewRequest("POST", "/api/v1/product-comparisons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.ProductComparisonResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "GB", resp.CurrentCountry.Country)
	require.Len(t, resp.ComparisonCountries, 2)
	assert.Equal(t, "BR", resp.ComparisonCountries[0].Country)
	assert.Contains(t, resp.ComparisonCountries[0].Roles, "base")
	assert.Equal(t, "GB", resp.ComparisonCountries[1].Country)
	assert.Contains(t, resp.ComparisonCountries[1].Roles, "current")
	require.NotNil(t, resp.Category)
	assert.Equal(t, models.CategoryOther, resp.Category.ID)
}
