package extractors

import "fmt"

// getString safely extracts a string value from an interface{}
func getString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}