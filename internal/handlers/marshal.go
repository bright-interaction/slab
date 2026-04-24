package handlers

import "encoding/json"

// marshalField converts any value to a JSON string for storage.
func marshalField(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
