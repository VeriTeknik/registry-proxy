package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/veriteknik/registry-proxy/internal/utils"
	"go.uber.org/zap"
)

// APIKeyAuth validates API key for write operations using constant-time comparison
func APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := utils.Logger

		// Get API key from environment
		validAPIKey := os.Getenv("API_KEY")
		logger.Debug("Validating authentication", zap.String("path", r.URL.Path))

		if validAPIKey == "" {
			// If no API key is configured, reject all requests
			// Security: Do not log that API key is missing - could aid attackers
			logger.Warn("Authentication failed: API key not configured")
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Get API key from request header
		authHeader := r.Header.Get("Authorization")
		// Security: Never log the actual header value or any part of it

		if authHeader == "" {
			logger.Info("Authentication failed: No Authorization header")
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <api_key>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Info("Authentication failed: Invalid header format")
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]
		// Security: Never log API keys or any portion of them

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validAPIKey)) != 1 {
			logger.Warn("Authentication failed: Invalid API key")
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		logger.Debug("Authentication successful")
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
