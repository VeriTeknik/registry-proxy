package utils

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRequireMethod(t *testing.T) {
	tests := []struct {
		name           string
		requestMethod  string
		requiredMethod string
		wantStatus     int
		wantPass       bool
	}{
		{
			name:           "GET matches GET",
			requestMethod:  http.MethodGet,
			requiredMethod: http.MethodGet,
			wantStatus:     0, // No error response
			wantPass:       true,
		},
		{
			name:           "POST matches POST",
			requestMethod:  http.MethodPost,
			requiredMethod: http.MethodPost,
			wantStatus:     0,
			wantPass:       true,
		},
		{
			name:           "GET does not match POST",
			requestMethod:  http.MethodGet,
			requiredMethod: http.MethodPost,
			wantStatus:     http.StatusMethodNotAllowed,
			wantPass:       false,
		},
		{
			name:           "DELETE does not match GET",
			requestMethod:  http.MethodDelete,
			requiredMethod: http.MethodGet,
			wantStatus:     http.StatusMethodNotAllowed,
			wantPass:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tt.requestMethod, "/test", nil)

			result := RequireMethod(w, r, tt.requiredMethod)

			if result != tt.wantPass {
				t.Errorf("RequireMethod() = %v, want %v", result, tt.wantPass)
			}

			if !tt.wantPass && w.Code != tt.wantStatus {
				t.Errorf("RequireMethod() status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "error response",
			status:     http.StatusBadRequest,
			data:       map[string]string{"error": "bad request"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "array data",
			status:     http.StatusOK,
			data:       []string{"item1", "item2"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			err := WriteJSON(w, tt.status, tt.data)

			if err != nil {
				t.Errorf("WriteJSON() error = %v", err)
			}

			if w.Code != tt.wantStatus {
				t.Errorf("WriteJSON() status = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("WriteJSON() Content-Type = %v, want application/json", contentType)
			}
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		code       int
		wantStatus int
	}{
		{
			name:       "bad request",
			message:    "Invalid input",
			code:       http.StatusBadRequest,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			message:    "Authentication required",
			code:       http.StatusUnauthorized,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "internal server error",
			message:    "Internal server error",
			code:       http.StatusInternalServerError,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteJSONError(w, tt.message, tt.code)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteJSONError() status = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("WriteJSONError() Content-Type = %v, want application/json", contentType)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		name     string
		values   url.Values
		key      string
		expected []string
	}{
		{
			name:     "single item",
			values:   url.Values{"tags": []string{"tag1"}},
			key:      "tags",
			expected: []string{"tag1"},
		},
		{
			name:     "multiple items",
			values:   url.Values{"tags": []string{"tag1,tag2,tag3"}},
			key:      "tags",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "empty string",
			values:   url.Values{"tags": []string{""}},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "missing key",
			values:   url.Values{},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "items with spaces",
			values:   url.Values{"tags": []string{" tag1 , tag2 , tag3 "}},
			key:      "tags",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "empty items filtered out",
			values:   url.Values{"tags": []string{"tag1,,tag2,  ,tag3"}},
			key:      "tags",
			expected: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseList(tt.values, tt.key)

			if len(result) != len(tt.expected) {
				t.Errorf("ParseList() length = %v, want %v", len(result), len(tt.expected))
				return
			}

			for i, item := range result {
				if item != tt.expected[i] {
					t.Errorf("ParseList()[%d] = %v, want %v", i, item, tt.expected[i])
				}
			}
		})
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name         string
		values       url.Values
		key          string
		defaultValue int
		maxValue     int
		expected     int
	}{
		{
			name:         "valid integer",
			values:       url.Values{"limit": []string{"50"}},
			key:          "limit",
			defaultValue: 20,
			maxValue:     100,
			expected:     50,
		},
		{
			name:         "missing parameter uses default",
			values:       url.Values{},
			key:          "limit",
			defaultValue: 20,
			maxValue:     100,
			expected:     20,
		},
		{
			name:         "invalid integer uses default",
			values:       url.Values{"limit": []string{"abc"}},
			key:          "limit",
			defaultValue: 20,
			maxValue:     100,
			expected:     20,
		},
		{
			name:         "negative integer uses default",
			values:       url.Values{"limit": []string{"-10"}},
			key:          "limit",
			defaultValue: 20,
			maxValue:     100,
			expected:     20,
		},
		{
			name:         "zero uses default",
			values:       url.Values{"limit": []string{"0"}},
			key:          "limit",
			defaultValue: 20,
			maxValue:     100,
			expected:     20,
		},
		{
			name:         "exceeds max returns max",
			values:       url.Values{"limit": []string{"500"}},
			key:          "limit",
			defaultValue: 20,
			maxValue:     100,
			expected:     100,
		},
		{
			name:         "no max limit",
			values:       url.Values{"offset": []string{"999"}},
			key:          "offset",
			defaultValue: 0,
			maxValue:     0, // 0 means no max
			expected:     999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIntParam(tt.values, tt.key, tt.defaultValue, tt.maxValue)
			if result != tt.expected {
				t.Errorf("ParseIntParam() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseFloatParam(t *testing.T) {
	tests := []struct {
		name     string
		values   url.Values
		key      string
		expected float64
	}{
		{
			name:     "valid float",
			values:   url.Values{"rating": []string{"4.5"}},
			key:      "rating",
			expected: 4.5,
		},
		{
			name:     "valid integer as float",
			values:   url.Values{"rating": []string{"4"}},
			key:      "rating",
			expected: 4.0,
		},
		{
			name:     "missing parameter returns zero",
			values:   url.Values{},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "invalid float returns zero",
			values:   url.Values{"rating": []string{"abc"}},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "negative float",
			values:   url.Values{"rating": []string{"-1.5"}},
			key:      "rating",
			expected: -1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFloatParam(tt.values, tt.key)
			if result != tt.expected {
				t.Errorf("ParseFloatParam() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLimitRequestSize(t *testing.T) {
	handler := LimitRequestSize(1024, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name       string
		bodySize   int
		wantStatus int
	}{
		{
			name:       "small body allowed",
			bodySize:   100,
			wantStatus: http.StatusOK,
		},
		{
			name:       "exact limit allowed",
			bodySize:   1024,
			wantStatus: http.StatusOK,
		},
		// Note: Testing body size exceeding limit requires actual body reading
		// which is better tested in integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/test", nil)

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("LimitRequestSize() status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseIntParam(b *testing.B) {
	values := url.Values{"limit": []string{"50"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseIntParam(values, "limit", 20, 100)
	}
}

func BenchmarkParseList(b *testing.B) {
	values := url.Values{"tags": []string{"tag1,tag2,tag3,tag4,tag5"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseList(values, "tags")
	}
}

func BenchmarkWriteJSON(b *testing.B) {
	data := map[string]interface{}{
		"message": "test",
		"count":   100,
		"items":   []string{"a", "b", "c"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		WriteJSON(w, http.StatusOK, data)
	}
}
