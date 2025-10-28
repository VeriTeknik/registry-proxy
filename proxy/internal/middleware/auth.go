package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// APIKeyAuth validates API key for write operations using constant-time comparison
func APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API key from environment
		validAPIKey := os.Getenv("API_KEY")
		if validAPIKey == "" {
			// If no API key is configured, reject all requests
			// Use generic error message to avoid information disclosure
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Get API key from request header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <api_key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validAPIKey)) != 1 {
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
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
