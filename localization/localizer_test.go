package localization

import (
	"testing"
)

func TestLocalizer(t *testing.T) {
	// Test English (default)
	err := InitLocalizer("en")
	if err != nil {
		t.Fatalf("Failed to initialize localizer: %v", err)
	}

	// Test simple key lookup
	message := T("api.errors.product_name_required")
	expected := "Product name is required"
	if message != expected {
		t.Errorf("Expected '%s', got '%s'", expected, message)
	}

	// Test nested key lookup
	healthMessage := T("api.health.message")
	expectedHealth := "Muambr API is running"
	if healthMessage != expectedHealth {
		t.Errorf("Expected '%s', got '%s'", expectedHealth, healthMessage)
	}

	// Test parameter substitution
	paramMessage := TP("api.errors.unsupported_country_iso", map[string]string{
		"code": "XX",
	})
	expectedParam := "unsupported country ISO code: XX"
	if paramMessage != expectedParam {
		t.Errorf("Expected '%s', got '%s'", expectedParam, paramMessage)
	}

	// Test invalid key (should return key path)
	invalidMessage := T("invalid.key.path")
	if invalidMessage != "invalid.key.path" {
		t.Errorf("Expected key path for invalid key, got '%s'", invalidMessage)
	}
}

func TestLanguageSwitching(t *testing.T) {
	// Initialize with English
	err := InitLocalizer("en")
	if err != nil {
		t.Fatalf("Failed to initialize localizer: %v", err)
	}

	// Test English message with request localizer
	enLocalizer, err := NewLocalizedContext("en")
	if err != nil {
		t.Fatalf("Failed to create English localizer: %v", err)
	}
	enMessage := enLocalizer.Get("api.errors.product_name_required")
	if enMessage != "Product name is required" {
		t.Errorf("English message incorrect: %s", enMessage)
	}

	// Test Portuguese message with request localizer
	ptLocalizer, err := NewLocalizedContext("pt")
	if err != nil {
		t.Fatalf("Failed to create Portuguese localizer: %v", err)
	}
	ptMessage := ptLocalizer.Get("api.errors.product_name_required")
	if ptMessage != "Nome do produto é obrigatório" {
		t.Errorf("Portuguese message incorrect: %s", ptMessage)
	}

	// Test Spanish message with request localizer
	esLocalizer, err := NewLocalizedContext("es")
	if err != nil {
		t.Fatalf("Failed to create Spanish localizer: %v", err)
	}
	esMessage := esLocalizer.Get("api.errors.product_name_required")
	if esMessage != "El nombre del producto es requerido" {
		t.Errorf("Spanish message incorrect: %s", esMessage)
	}
}

func TestCountryNames(t *testing.T) {
	err := InitLocalizer("en")
	if err != nil {
		t.Fatalf("Failed to initialize localizer: %v", err)
	}

	// Test country names in English with request localizer
	enLocalizer, err := NewLocalizedContext("en")
	if err != nil {
		t.Fatalf("Failed to create English localizer: %v", err)
	}
	brazil := enLocalizer.Get("countries.names.brazil")
	if brazil != "Brazil" {
		t.Errorf("Expected 'Brazil', got '%s'", brazil)
	}

	// Test Portuguese with request localizer
	ptLocalizer, err := NewLocalizedContext("pt")
	if err != nil {
		t.Fatalf("Failed to create Portuguese localizer: %v", err)
	}
	brazilPT := ptLocalizer.Get("countries.names.brazil")
	if brazilPT != "Brasil" {
		t.Errorf("Expected 'Brasil', got '%s'", brazilPT)
	}

	// Test macro region in Portuguese
	eu := ptLocalizer.Get("macro_regions.names.EU")
	if eu != "União Europeia" {
		t.Errorf("Expected 'União Europeia', got '%s'", eu)
	}
}