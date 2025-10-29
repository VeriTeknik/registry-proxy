package middleware

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
)

// APIKeyAuth validates API key for write operations using constant-time comparison
func APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API key from environment
		validAPIKey := os.Getenv("API_KEY")
		log.Printf("[Auth] Validating request to: %s", r.URL.Path)
		log.Printf("[Auth] Valid API key exists: %v", validAPIKey != "")
		log.Printf("[Auth] Valid API key length: %d", len(validAPIKey))
		if len(validAPIKey) > 0 {
			log.Printf("[Auth] Valid API key prefix: %s", validAPIKey[:min(10, len(validAPIKey))])
		}

		if validAPIKey == "" {
			// If no API key is configured, reject all requests
			// Use generic error message to avoid information disclosure
			log.Printf("[Auth] ERROR: No API_KEY configured in environment")
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Get API key from request header
		authHeader := r.Header.Get("Authorization")
		log.Printf("[Auth] Authorization header present: %v", authHeader != "")
		log.Printf("[Auth] Authorization header value: %s", authHeader)

		if authHeader == "" {
			log.Printf("[Auth] ERROR: No Authorization header provided")
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <api_key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("[Auth] ERROR: Invalid Authorization header format, expected 'Bearer <token>'")
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]
		log.Printf("[Auth] Request API key length: %d", len(apiKey))
		if len(apiKey) > 0 {
			log.Printf("[Auth] Request API key prefix: %s", apiKey[:min(10, len(apiKey))])
		}

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validAPIKey)) != 1 {
			log.Printf("[Auth] ERROR: API key mismatch")
			log.Printf("[Auth] Expected length: %d, Got length: %d", len(validAPIKey), len(apiKey))
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		log.Printf("[Auth] SUCCESS: API key validated")
		// API key is valid, proceed
		next.ServeHTTP(w, r)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimitByIP implements simple rate limiting by IP
// This is a basic implementation - for production use a proper rate limiter
func RateLimitByIP(next http.HandlerFunc) http.HandlerFunc {
	// For now, just pass through
	// TODO: Implement proper rate limiting with redis or in-memory cache
	return next
}
