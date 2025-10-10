package mocks

import (
	"muambr-api/models"
)

// MockExtractor is a test implementation of the Extractor interface
type MockExtractor struct {
	CountryCode   models.Country
	Identifier    string
	BaseURLValue  string
	ShouldError   bool
	MockResults   []models.ProductComparison
	ErrorMessage  string
}

func (m *MockExtractor) GetCountryCode() models.Country {
	return m.CountryCode
}

func (m *MockExtractor) GetMacroRegion() models.MacroRegion {
	return m.CountryCode.GetMacroRegion()
}

func (m *MockExtractor) GetIdentifier() string {
	return m.Identifier
}

func (m *MockExtractor) BaseURL() string {
	return m.BaseURLValue
}

func (m *MockExtractor) GetComparisons(productName string) ([]models.ProductComparison, error) {
	if m.ShouldError {
		return nil, &ExtractorError{
			Extractor: m.Identifier,
			Message:   m.ErrorMessage,
		}
	}
	
	// Return mock results or empty slice
	if m.MockResults != nil {
		return m.MockResults, nil
	}
	
	return []models.ProductComparison{}, nil
}

// ExtractorError represents an error from an extractor
type ExtractorError struct {
	Extractor string
	Message   string
	Err       error
}

func (e *ExtractorError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// NewMockExtractor creates a new mock extractor with default values
func NewMockExtractor(country models.Country, identifier string) *MockExtractor {
	return &MockExtractor{
		CountryCode:  country,
		Identifier:   identifier,
		BaseURLValue: "https://mock-" + identifier + ".example.com",
		ShouldError:  false,
		MockResults:  nil,
		ErrorMessage: "Mock error for testing",
	}
}

// WithError configures the mock extractor to return an error
func (m *MockExtractor) WithError(message string) *MockExtractor {
	m.ShouldError = true
	m.ErrorMessage = message
	return m
}

// WithResults configures the mock extractor to return specific results
func (m *MockExtractor) WithResults(results []models.ProductComparison) *MockExtractor {
	m.MockResults = results
	m.ShouldError = false
	return m
}