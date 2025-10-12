package handlers

import (
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

// setupTestRouter creates a test router with the comparison handler
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	comparisonHandler := handlers.NewComparisonHandler()
	v1 := router.Group("/api/v1")
	{
		comparisons := v1.Group("/comparisons")
		{
			comparisons.GET("/search", comparisonHandler.GetComparisons)
		}
	}
	
	return router
}

// TestComparisonHandler_GetComparisons_ValidRequest tests a valid request
func TestComparisonHandler_GetComparisons_ValidRequest(t *testing.T) {
	router := setupTestRouter()
	
	// Test with valid parameters
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone&baseCountry=BR&currentUserCountry=PT&useMacroRegion=true&currency=EUR&limit=5", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	// Should return 200 OK (even if no results from extractors in test environment)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Parse response
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Verify response structure
	assert.True(t, response.Success)
	assert.NotNil(t, response.Sections)
	assert.GreaterOrEqual(t, response.TotalResults, 0)
}

// TestComparisonHandler_GetComparisons_MissingName tests missing product name
func TestComparisonHandler_GetComparisons_MissingName(t *testing.T) {
	router := setupTestRouter()
	
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?baseCountry=BR", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response.Success)
	assert.NotNil(t, response.Message)
	assert.Empty(t, response.Sections)
	assert.Equal(t, 0, response.TotalResults)
}

// TestComparisonHandler_GetComparisons_MissingBaseCountry tests missing base country
func TestComparisonHandler_GetComparisons_MissingBaseCountry(t *testing.T) {
	router := setupTestRouter()
	
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response.Success)
	assert.NotNil(t, response.Message)
}

// TestComparisonHandler_GetComparisons_InvalidCountryCode tests invalid country code
func TestComparisonHandler_GetComparisons_InvalidCountryCode(t *testing.T) {
	router := setupTestRouter()
	
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone&baseCountry=XX", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response.Success)
	assert.NotNil(t, response.Message)
}

// TestComparisonHandler_GetComparisons_DefaultParameters tests default parameter values
func TestComparisonHandler_GetComparisons_DefaultParameters(t *testing.T) {
	router := setupTestRouter()
	
	// Test with only required parameters
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone&baseCountry=BR", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	// Should return 200 OK with defaults applied
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.NotNil(t, response.Sections)
}

// TestComparisonHandler_GetComparisons_LimitParameter tests limit parameter handling
func TestComparisonHandler_GetComparisons_LimitParameter(t *testing.T) {
	testCases := []struct {
		name           string
		limitParam     string
		expectedValid  bool
	}{
		{
			name:          "Valid positive limit",
			limitParam:    "5",
			expectedValid: true,
		},
		{
			name:          "Valid large limit",
			limitParam:    "100",
			expectedValid: true,
		},
		{
			name:          "Invalid negative limit - should use default",
			limitParam:    "-1",
			expectedValid: true, // Should use default value
		},
		{
			name:          "Invalid zero limit - should use default",
			limitParam:    "0",
			expectedValid: true, // Should use default value
		},
		{
			name:          "Invalid non-numeric limit - should use default",
			limitParam:    "abc",
			expectedValid: true, // Should use default value
		},
	}
	
	router := setupTestRouter()
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/comparisons/search?name=iPhone&baseCountry=BR&limit=" + tc.limitParam
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			if tc.expectedValid {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}

// TestComparisonHandler_GetComparisons_MacroRegionParameter tests useMacroRegion parameter
func TestComparisonHandler_GetComparisons_MacroRegionParameter(t *testing.T) {
	testCases := []struct {
		name              string
		macroRegionParam  string
		expectedValid     bool
	}{
		{
			name:             "MacroRegion true",
			macroRegionParam: "true",
			expectedValid:    true,
		},
		{
			name:             "MacroRegion false",
			macroRegionParam: "false",
			expectedValid:    true,
		},
		{
			name:             "MacroRegion TRUE (case insensitive)",
			macroRegionParam: "TRUE",
			expectedValid:    true,
		},
		{
			name:             "MacroRegion invalid value - should default to false",
			macroRegionParam: "maybe",
			expectedValid:    true,
		},
	}
	
	router := setupTestRouter()
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/comparisons/search?name=iPhone&baseCountry=BR&currentUserCountry=PT&useMacroRegion=" + tc.macroRegionParam
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			
			var response models.ProductComparisonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.True(t, response.Success)
		})
	}
}

// TestComparisonHandler_GetComparisons_CurrencyParameter tests currency parameter handling
func TestComparisonHandler_GetComparisons_CurrencyParameter(t *testing.T) {
	testCases := []struct {
		name           string
		baseCountry    string
		currency       string
		expectedValid  bool
	}{
		{
			name:          "Explicit EUR currency",
			baseCountry:   "BR",
			currency:      "EUR",
			expectedValid: true,
		},
		{
			name:          "Explicit USD currency",
			baseCountry:   "BR", 
			currency:      "USD",
			expectedValid: true,
		},
		{
			name:          "No currency - should use base country default",
			baseCountry:   "BR", // Should default to BRL
			currency:      "",
			expectedValid: true,
		},
		{
			name:          "Portugal base country - should default to EUR",
			baseCountry:   "PT",
			currency:      "",
			expectedValid: true,
		},
	}
	
	router := setupTestRouter()
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/comparisons/search?name=iPhone&baseCountry=" + tc.baseCountry
			if tc.currency != "" {
				url += "&currency=" + tc.currency
			}
			
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			
			var response models.ProductComparisonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.True(t, response.Success)
		})
	}
}

// TestComparisonHandler_GetComparisons_AllCountries tests all supported countries
func TestComparisonHandler_GetComparisons_AllCountries(t *testing.T) {
	supportedCountries := []string{"BR", "US", "PT", "ES", "GB", "DE"}
	
	router := setupTestRouter()
	
	for _, country := range supportedCountries {
		t.Run("Country_"+country, func(t *testing.T) {
			url := "/api/v1/comparisons/search?name=iPhone&baseCountry=" + country
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			
			var response models.ProductComparisonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.True(t, response.Success)
			assert.NotNil(t, response.Sections)
		})
	}
}

// TestComparisonHandler_ResponseStructure tests the response structure matches the model
func TestComparisonHandler_ResponseStructure(t *testing.T) {
	router := setupTestRouter()
	
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone&baseCountry=BR&currentUserCountry=PT&useMacroRegion=true&currency=EUR&limit=10", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Verify required fields are present
	assert.True(t, response.Success)
	assert.NotNil(t, response.Sections)
	assert.GreaterOrEqual(t, response.TotalResults, 0)
	
	// If there are sections, verify their structure
	for _, section := range response.Sections {
		assert.NotEmpty(t, section.Country)
		assert.NotEmpty(t, section.CountryName)
		assert.NotNil(t, section.Comparisons)
		assert.Equal(t, len(section.Comparisons), section.ResultsCount)
		
		// Verify each comparison has required fields
		for _, comparison := range section.Comparisons {
			assert.NotEmpty(t, comparison.ID)
			assert.NotEmpty(t, comparison.ProductName)
			assert.Greater(t, comparison.Price, 0.0)
			assert.NotEmpty(t, comparison.Currency)
			assert.NotEmpty(t, comparison.StoreName)
			assert.NotEmpty(t, comparison.Country)
		}
	}
}

// TestComparisonHandler_GetComparisons_URLEncoding tests URL encoded parameters
func TestComparisonHandler_GetComparisons_URLEncoding(t *testing.T) {
	router := setupTestRouter()
	
	// Test URL encoded product name (iPhone 16 Pro)
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone%2016%20Pro&baseCountry=BR", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
}

// TestComparisonHandler_GetComparisons_EmptyResults tests handling of empty results
func TestComparisonHandler_GetComparisons_EmptyResults(t *testing.T) {
	router := setupTestRouter()
	
	// Use a very specific product name that's unlikely to return results
	req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=NonExistentProductXYZ123&baseCountry=US", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.ProductComparisonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Even with no results, response should be successful with appropriate message
	assert.True(t, response.Success)
	// US has no extractors configured, so should return empty results
	assert.Equal(t, 0, response.TotalResults)
}

// BenchmarkComparisonHandler_GetComparisons benchmarks the handler performance
func BenchmarkComparisonHandler_GetComparisons(b *testing.B) {
	router := setupTestRouter()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/comparisons/search?name=iPhone&baseCountry=BR&limit=5", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
	}
}