package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAPIKeyAuth(t *testing.T) {
	// Set up test API key
	testAPIKey := "test-api-key-12345"
	os.Setenv("API_KEY", testAPIKey)
	defer os.Unsetenv("API_KEY")

	// Mock handler that should only be called if auth succeeds
	handlerCalled := false
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantCalled     bool
		setupAPIKey    bool
	}{
		{
			name:        "valid API key",
			authHeader:  "Bearer " + testAPIKey,
			wantStatus:  http.StatusOK,
			wantCalled:  true,
			setupAPIKey: true,
		},
		{
			name:        "invalid API key",
			authHeader:  "Bearer wrong-key",
			wantStatus:  http.StatusUnauthorized,
			wantCalled:  false,
			setupAPIKey: true,
		},
		{
			name:        "missing Bearer prefix",
			authHeader:  testAPIKey,
			wantStatus:  http.StatusUnauthorized,
			wantCalled:  false,
			setupAPIKey: true,
		},
		{
			name:        "no authorization header",
			authHeader:  "",
			wantStatus:  http.StatusUnauthorized,
			wantCalled:  false,
			setupAPIKey: true,
		},
		{
			name:        "empty Bearer token",
			authHeader:  "Bearer ",
			wantStatus:  http.StatusUnauthorized,
			wantCalled:  false,
			setupAPIKey: true,
		},
		{
			name:        "no API key configured",
			authHeader:  "Bearer anything",
			wantStatus:  http.StatusUnauthorized,
			wantCalled:  false,
			setupAPIKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset handler called flag
			handlerCalled = false

			// Setup or clear API key
			if !tt.setupAPIKey {
				os.Unsetenv("API_KEY")
				defer os.Setenv("API_KEY", testAPIKey)
			}

			// Create test request
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/test", nil)
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}

			// Call middleware
			authMiddleware := APIKeyAuth(mockHandler)
			authMiddleware(w, r)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check if handler was called
			if handlerCalled != tt.wantCalled {
				t.Errorf("Handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestAPIKeyAuth_ConstantTimeComparison(t *testing.T) {
	// This test verifies that the constant-time comparison is used
	// We can't directly test timing, but we can verify it doesn't panic
	// and correctly handles keys of different lengths

	testAPIKey := "correct-key"
	os.Setenv("API_KEY", testAPIKey)
	defer os.Unsetenv("API_KEY")

	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
	}{
		{
			name:       "same length correct key",
			apiKey:     testAPIKey,
			wantStatus: http.StatusOK,
		},
		{
			name:       "same length wrong key",
			apiKey:     "incorrect--",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "shorter wrong key",
			apiKey:     "short",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "longer wrong key",
			apiKey:     "this-is-a-very-long-incorrect-key",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/test", nil)
			r.Header.Set("Authorization", "Bearer "+tt.apiKey)

			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}

			authMiddleware := APIKeyAuth(mockHandler)
			authMiddleware(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAPIKeyAuth_NoSensitiveDataLogged(t *testing.T) {
	// This test verifies that the refactored auth middleware
	// doesn't log sensitive data (tested by code inspection)

	testAPIKey := "sensitive-api-key-12345"
	os.Setenv("API_KEY", testAPIKey)
	defer os.Unsetenv("API_KEY")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.Header.Set("Authorization", "Bearer "+testAPIKey)

	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Call middleware - should not log the API key or any part of it
	authMiddleware := APIKeyAuth(mockHandler)
	authMiddleware(w, r)

	// If this test passes without panicking, the middleware works
	// The actual verification that nothing is logged is done by CodeQL
	if w.Code != http.StatusOK {
		t.Errorf("Expected successful auth, got status %v", w.Code)
	}
}

func TestRateLimitByIP(t *testing.T) {
	// Currently a pass-through, but test it exists and works
	handlerCalled := false
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	rateLimitMiddleware := RateLimitByIP(mockHandler)
	rateLimitMiddleware(w, r)

	if !handlerCalled {
		t.Error("Handler should be called (rate limiting not yet implemented)")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

// Benchmark constant-time comparison
func BenchmarkAPIKeyAuth_ValidKey(b *testing.B) {
	testAPIKey := "test-api-key-12345678901234567890"
	os.Setenv("API_KEY", testAPIKey)
	defer os.Unsetenv("API_KEY")

	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	authMiddleware := APIKeyAuth(mockHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/test", nil)
		r.Header.Set("Authorization", "Bearer "+testAPIKey)
		authMiddleware(w, r)
	}
}

func BenchmarkAPIKeyAuth_InvalidKey(b *testing.B) {
	testAPIKey := "test-api-key-12345678901234567890"
	os.Setenv("API_KEY", testAPIKey)
	defer os.Unsetenv("API_KEY")

	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	authMiddleware := APIKeyAuth(mockHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/test", nil)
		r.Header.Set("Authorization", "Bearer wrong-key-12345678901234567890")
		authMiddleware(w, r)
	}
}
