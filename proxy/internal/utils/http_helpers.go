package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// RequireMethod checks if the request method matches the expected method.
// Returns true if the method matches, false otherwise (and writes error response).
func RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		WriteJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		return err
	}
	return nil
}

// WriteJSONError writes a standardized JSON error response.
// Never exposes internal error details - use generic messages only.
func WriteJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	response := map[string]interface{}{
		"error": message,
		"code":  code,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

// ParseList parses a comma-separated list from URL query parameters.
// Returns nil if the parameter is empty.
func ParseList(vals url.Values, key string) []string {
	value := strings.TrimSpace(vals.Get(key))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ParseIntParam parses an integer parameter with default and maximum values.
// If the parameter is missing or invalid, returns the default value.
// If the parameter exceeds the maximum, returns the maximum value.
func ParseIntParam(vals url.Values, key string, defaultValue, maxValue int) int {
	str := vals.Get(key)
	if str == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(str)
	if err != nil || value <= 0 {
		return defaultValue
	}

	if maxValue > 0 && value > maxValue {
		return maxValue
	}

	return value
}

// ParseFloatParam parses a float parameter from URL query parameters.
// Returns 0 if the parameter is missing or invalid.
func ParseFloatParam(vals url.Values, key string) float64 {
	str := vals.Get(key)
	if str == "" {
		return 0
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}

	return value
}

// LimitRequestSize wraps an http.Handler to enforce a maximum request body size.
// This prevents DoS attacks via large payloads.
func LimitRequestSize(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}
