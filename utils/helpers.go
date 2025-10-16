package utils

import (
	"github.com/google/uuid"
)

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

// SafeGetString safely extracts a string from an interface{} (alternative to existing GetString)
func SafeGetString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// GetFloat64 safely extracts a float64 from an interface{}
func GetFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}