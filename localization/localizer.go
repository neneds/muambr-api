package localization

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

// Localizer handles string localization
type Localizer struct {
	messages    map[string]map[string]interface{}
	mu          sync.RWMutex
}

// RequestLocalizer is a request-scoped localizer that doesn't change global state
type RequestLocalizer struct {
	globalLocalizer *Localizer
	currentLang     string
}

// Global localizer instance
var globalLocalizer *Localizer
var once sync.Once

// InitLocalizer initializes the global localizer with the specified language
func InitLocalizer(lang string) error {
	var err error
	once.Do(func() {
		globalLocalizer = &Localizer{
			messages:    make(map[string]map[string]interface{}),
		}
		err = globalLocalizer.LoadLanguage(lang)
	})
	return err
}

// GetLocalizer returns the global localizer instance
func GetLocalizer() *Localizer {
	if globalLocalizer == nil {
		// Initialize with English as default if not already initialized
		InitLocalizer("en")
	}
	return globalLocalizer
}

// LoadLanguage loads the specified language file
func (l *Localizer) LoadLanguage(lang string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	filename := fmt.Sprintf("%s.json", lang)
	
	// Try multiple possible paths to find the localization files
	possiblePaths := []string{
		filepath.Join("localization", filename),
		filepath.Join("..", "localization", filename),
		filename,
	}
	
	var data []byte
	var err error
	var filePath string
	
	for _, path := range possiblePaths {
		data, err = ioutil.ReadFile(path)
		if err == nil {
			filePath = path
			break
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to read language file %s (tried paths: %v): %w", filename, possiblePaths, err)
	}

	var messages map[string]interface{}
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to parse language file %s: %w", filePath, err)
	}

	l.messages[lang] = messages
	
	return nil
}

// NewRequestLocalizer creates a request-scoped localizer for the specified language
func NewRequestLocalizer(lang string) (*RequestLocalizer, error) {
	if globalLocalizer == nil {
		return nil, fmt.Errorf("global localizer not initialized")
	}

	// Ensure the language is loaded
	globalLocalizer.mu.RLock()
	_, exists := globalLocalizer.messages[lang]
	globalLocalizer.mu.RUnlock()

	if !exists {
		if err := globalLocalizer.LoadLanguage(lang); err != nil {
			return nil, err
		}
	}

	return &RequestLocalizer{
		globalLocalizer: globalLocalizer,
		currentLang:     lang,
	}, nil
}

// SetLanguage changes the current language (deprecated - use NewRequestLocalizer instead)
func (l *Localizer) SetLanguage(lang string) error {
	// Load language if not already loaded
	if _, exists := l.messages[lang]; !exists {
		if err := l.LoadLanguage(lang); err != nil {
			return err
		}
	}
	
	return nil
}

// Get retrieves a localized string by key path (e.g., "api.errors.product_name_required")
func (rl *RequestLocalizer) Get(keyPath string) string {
	rl.globalLocalizer.mu.RLock()
	defer rl.globalLocalizer.mu.RUnlock()

	return rl.get(keyPath, nil)
}

// GetWithParams retrieves a localized string and replaces parameters
func (rl *RequestLocalizer) GetWithParams(keyPath string, params map[string]string) string {
	rl.globalLocalizer.mu.RLock()
	defer rl.globalLocalizer.mu.RUnlock()

	return rl.get(keyPath, params)
}

// Get retrieves a localized string by key path (global method - deprecated)
func (l *Localizer) Get(keyPath string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.get(keyPath, nil, "en") // Default to English for global access
}

// GetWithParams retrieves a localized string and replaces parameters (global method - deprecated)
func (l *Localizer) GetWithParams(keyPath string, params map[string]string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.get(keyPath, params, "en") // Default to English for global access
}

// get is the internal method for RequestLocalizer to retrieve localized strings
func (rl *RequestLocalizer) get(keyPath string, params map[string]string) string {
	keys := strings.Split(keyPath, ".")
	
	messages, exists := rl.globalLocalizer.messages[rl.currentLang]
	if !exists {
		return keyPath // Return key path if language not found
	}

	var current interface{} = messages
	
	// Navigate through the nested structure
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[key]; exists {
				current = val
			} else {
				return keyPath // Return key path if key not found
			}
		} else {
			return keyPath // Return key path if structure is invalid
		}
	}

	// Convert final value to string
	var result string
	if str, ok := current.(string); ok {
		result = str
	} else {
		return keyPath // Return key path if final value is not a string
	}

	// Replace parameters if provided
	if params != nil {
		for key, value := range params {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	return result
}

// get is the internal method for global Localizer to retrieve localized strings
func (l *Localizer) get(keyPath string, params map[string]string, lang string) string {
	keys := strings.Split(keyPath, ".")
	
	messages, exists := l.messages[lang]
	if !exists {
		return keyPath // Return key path if language not found
	}

	var current interface{} = messages
	
	// Navigate through the nested structure
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[key]; exists {
				current = val
			} else {
				return keyPath // Return key path if key not found
			}
		} else {
			return keyPath // Return key path if structure is invalid
		}
	}

	// Convert final value to string
	var result string
	if str, ok := current.(string); ok {
		result = str
	} else {
		return keyPath // Return key path if final value is not a string
	}

	// Replace parameters if provided
	if params != nil {
		for key, value := range params {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	return result
}

// GetCurrentLanguage returns the current language for the request
func (rl *RequestLocalizer) GetCurrentLanguage() string {
	return rl.currentLang
}

// GetCurrentLanguage returns the current language (global - deprecated)
func (l *Localizer) GetCurrentLanguage() string {
	return "en" // Always return English for global access
}

// Convenience functions for global localizer (deprecated - use request-scoped localizers)

// T is a shorthand for Get (translate) - uses English by default
func T(keyPath string) string {
	return GetLocalizer().Get(keyPath)
}

// TP is a shorthand for GetWithParams (translate with parameters) - uses English by default
func TP(keyPath string, params map[string]string) string {
	return GetLocalizer().GetWithParams(keyPath, params)
}

// SetGlobalLanguage sets the language for the global localizer (deprecated)
func SetGlobalLanguage(lang string) error {
	return GetLocalizer().SetLanguage(lang)
}

// Request-scoped convenience functions

// NewLocalizedContext creates a new request localizer for the given language
func NewLocalizedContext(lang string) (*RequestLocalizer, error) {
	return NewRequestLocalizer(lang)
}

// TR is a shorthand for RequestLocalizer.Get (translate with request scope)
func TR(localizer *RequestLocalizer, keyPath string) string {
	if localizer == nil {
		return T(keyPath) // Fallback to global if localizer is nil
	}
	return localizer.Get(keyPath)
}

// TRP is a shorthand for RequestLocalizer.GetWithParams (translate with parameters and request scope)
func TRP(localizer *RequestLocalizer, keyPath string, params map[string]string) string {
	if localizer == nil {
		return TP(keyPath, params) // Fallback to global if localizer is nil
	}
	return localizer.GetWithParams(keyPath, params)
}