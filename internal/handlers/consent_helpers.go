package handlers

import (
	"database/sql"
	"encoding/json"
)

// decodeJSON is a tiny wrapper used by consent.go to lift categories_json
// (stored as TEXT) back into a map for the JSON response.
func decodeJSON(s string, v any) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// nullFloatToInt64 collapses sql.NullFloat64 (sqlc emits this for SUM(CASE ...))
// to a plain int64 so the JSON response doesn't expose {Float64, Valid}.
func nullFloatToInt64(n sql.NullFloat64) int64 {
	if !n.Valid {
		return 0
	}
	return int64(n.Float64)
}
