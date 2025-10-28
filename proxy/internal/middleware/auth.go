package middleware

import (
	"net/http"
	"os"
	"strings"
)

// APIKeyAuth validates API key for write operations
func APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API key from environment
		validAPIKey := os.Getenv("API_KEY")
		if validAPIKey == "" {
			// If no API key is configured, reject all requests
			http.Error(w, "API authentication not configured", http.StatusServiceUnavailable)
			return
		}

		// Get API key from request header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <api_key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format. Use: Bearer <api_key>", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]
		if apiKey != validAPIKey {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// API key is valid, proceed
		next.ServeHTTP(w, r)
	}
}

// RateLimitByIP implements simple rate limiting by IP
// This is a basic implementation - for production use a proper rate limiter
func RateLimitByIP(next http.HandlerFunc) http.HandlerFunc {
	// For now, just pass through
	// TODO: Implement proper rate limiting with redis or in-memory cache
	return next
}
