package utils

import "fmt"

// GetString safely extracts a string value from an interface{}
func GetString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}