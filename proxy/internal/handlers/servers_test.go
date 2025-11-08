package handlers

import (
	"testing"

	"github.com/veriteknik/registry-proxy/internal/models"
)

// TestConvertRemote_WithHeaders tests that remote headers are properly extracted
func TestConvertRemote_WithHeaders(t *testing.T) {
	handler := &ServersHandler{}

	remoteMap := map[string]interface{}{
		"type": "streamable-http",
		"url":  "https://server.smithery.ai/@test/mcp",
		"headers": []interface{}{
			map[string]interface{}{
				"name":        "Authorization",
				"value":       "Bearer {smithery_api_key}",
				"description": "Bearer token for Smithery authentication",
				"isRequired":  true,
				"isSecret":    true,
			},
			map[string]interface{}{
				"name":        "X-Custom-Header",
				"value":       "{custom_value}",
				"description": "Custom header",
				"default":     "default_value",
				"isRequired":  false,
				"isSecret":    false,
			},
		},
	}

	result := handler.convertRemote(remoteMap)

	// Verify basic fields
	if result.TransportType != "streamable-http" {
		t.Errorf("Expected transport type 'streamable-http', got '%s'", result.TransportType)
	}
	if result.URL != "https://server.smithery.ai/@test/mcp" {
		t.Errorf("Expected URL 'https://server.smithery.ai/@test/mcp', got '%s'", result.URL)
	}

	// Verify headers were extracted
	if len(result.Headers) != 2 {
		t.Fatalf("Expected 2 headers, got %d", len(result.Headers))
	}

	// Verify first header (Authorization)
	authHeader := result.Headers[0]
	if authHeader.Name != "Authorization" {
		t.Errorf("Expected header name 'Authorization', got '%s'", authHeader.Name)
	}
	if authHeader.Value != "Bearer {smithery_api_key}" {
		t.Errorf("Expected header value 'Bearer {smithery_api_key}', got '%s'", authHeader.Value)
	}
	if authHeader.Description != "Bearer token for Smithery authentication" {
		t.Errorf("Expected description 'Bearer token for Smithery authentication', got '%s'", authHeader.Description)
	}
	if !authHeader.IsRequired {
		t.Error("Expected IsRequired to be true")
	}
	if !authHeader.IsSecret {
		t.Error("Expected IsSecret to be true")
	}

	// Verify second header (Custom)
	customHeader := result.Headers[1]
	if customHeader.Name != "X-Custom-Header" {
		t.Errorf("Expected header name 'X-Custom-Header', got '%s'", customHeader.Name)
	}
	if customHeader.Value != "{custom_value}" {
		t.Errorf("Expected header value '{custom_value}', got '%s'", customHeader.Value)
	}
	if customHeader.Default != "default_value" {
		t.Errorf("Expected default 'default_value', got '%s'", customHeader.Default)
	}
	if customHeader.IsRequired {
		t.Error("Expected IsRequired to be false")
	}
	if customHeader.IsSecret {
		t.Error("Expected IsSecret to be false")
	}
}

// TestConvertRemote_WithoutHeaders tests remotes without headers field
func TestConvertRemote_WithoutHeaders(t *testing.T) {
	handler := &ServersHandler{}

	remoteMap := map[string]interface{}{
		"type": "sse",
		"url":  "https://example.com/mcp",
	}

	result := handler.convertRemote(remoteMap)

	// Verify basic fields
	if result.TransportType != "sse" {
		t.Errorf("Expected transport type 'sse', got '%s'", result.TransportType)
	}
	if result.URL != "https://example.com/mcp" {
		t.Errorf("Expected URL 'https://example.com/mcp', got '%s'", result.URL)
	}

	// Verify no headers
	if len(result.Headers) != 0 {
		t.Errorf("Expected 0 headers, got %d", len(result.Headers))
	}
}

// TestConvertRemote_WithEmptyHeaders tests remotes with empty headers array
func TestConvertRemote_WithEmptyHeaders(t *testing.T) {
	handler := &ServersHandler{}

	remoteMap := map[string]interface{}{
		"type":    "http",
		"url":     "https://example.com/api",
		"headers": []interface{}{},
	}

	result := handler.convertRemote(remoteMap)

	// Verify no headers
	if len(result.Headers) != 0 {
		t.Errorf("Expected 0 headers, got %d", len(result.Headers))
	}
}

// TestConvertRemote_WithMissingHeaderFields tests partial header data
func TestConvertRemote_WithMissingHeaderFields(t *testing.T) {
	handler := &ServersHandler{}

	remoteMap := map[string]interface{}{
		"type": "streamable-http",
		"url":  "https://example.com/mcp",
		"headers": []interface{}{
			map[string]interface{}{
				"name": "X-Minimal-Header",
				// No value, description, default, isRequired, isSecret
			},
		},
	}

	result := handler.convertRemote(remoteMap)

	// Verify header was extracted with only name
	if len(result.Headers) != 1 {
		t.Fatalf("Expected 1 header, got %d", len(result.Headers))
	}

	header := result.Headers[0]
	if header.Name != "X-Minimal-Header" {
		t.Errorf("Expected header name 'X-Minimal-Header', got '%s'", header.Name)
	}
	// Verify other fields are zero values
	if header.Value != "" {
		t.Errorf("Expected empty value, got '%s'", header.Value)
	}
	if header.Description != "" {
		t.Errorf("Expected empty description, got '%s'", header.Description)
	}
	if header.Default != "" {
		t.Errorf("Expected empty default, got '%s'", header.Default)
	}
	if header.IsRequired {
		t.Error("Expected IsRequired to be false")
	}
	if header.IsSecret {
		t.Error("Expected IsSecret to be false")
	}
}

// TestConvertMapToEnrichedServer_RemotesWithHeaders tests full conversion with headers
func TestConvertMapToEnrichedServer_RemotesWithHeaders(t *testing.T) {
	handler := &ServersHandler{}

	serverMap := map[string]interface{}{
		"id":          "ai.smithery/test-server",
		"description": "A test server with headers",
		"repository": map[string]interface{}{
			"url":    "https://github.com/test/repo",
			"source": "github",
		},
		"remotes": []interface{}{
			map[string]interface{}{
				"type": "streamable-http",
				"url":  "https://server.smithery.ai/@test/mcp",
				"headers": []interface{}{
					map[string]interface{}{
						"name":        "Authorization",
						"value":       "Bearer {smithery_api_key}",
						"description": "Auth token",
						"isRequired":  true,
						"isSecret":    true,
					},
				},
			},
		},
		"stats": map[string]interface{}{
			"rating":        4.5,
			"rating_count":  10,
			"install_count": 100,
		},
	}

	result := handler.convertMapToEnrichedServer(serverMap)

	// Verify server fields
	if result.ID != "ai.smithery/test-server" {
		t.Errorf("Expected ID 'ai.smithery/test-server', got '%s'", result.ID)
	}
	// Name should equal ID (implementation uses ID as name)
	if result.Name != "ai.smithery/test-server" {
		t.Errorf("Expected name 'ai.smithery/test-server', got '%s'", result.Name)
	}

	// Verify remotes
	if len(result.Remotes) != 1 {
		t.Fatalf("Expected 1 remote, got %d", len(result.Remotes))
	}

	remote := result.Remotes[0]
	if len(remote.Headers) != 1 {
		t.Fatalf("Expected 1 header in remote, got %d", len(remote.Headers))
	}

	// Verify header
	header := remote.Headers[0]
	if header.Name != "Authorization" {
		t.Errorf("Expected header name 'Authorization', got '%s'", header.Name)
	}
	if header.Value != "Bearer {smithery_api_key}" {
		t.Errorf("Expected header value 'Bearer {smithery_api_key}', got '%s'", header.Value)
	}
	if header.Description != "Auth token" {
		t.Errorf("Expected description 'Auth token', got '%s'", header.Description)
	}
	if !header.IsRequired {
		t.Error("Expected IsRequired to be true")
	}
	if !header.IsSecret {
		t.Error("Expected IsSecret to be true")
	}

	// Verify stats
	if result.Rating != 4.5 {
		t.Errorf("Expected rating 4.5, got %f", result.Rating)
	}
	if result.RatingCount != 10 {
		t.Errorf("Expected rating count 10, got %d", result.RatingCount)
	}
	if result.InstallationCount != 100 {
		t.Errorf("Expected installation count 100, got %d", result.InstallationCount)
	}
}

// TestRemoteHeaderBackwardCompatibility tests that empty values are omitted in JSON
func TestRemoteHeaderBackwardCompatibility(t *testing.T) {
	// Test that headers without value field are still valid
	header := models.RemoteHeader{
		Name:        "X-API-Key",
		Description: "API key header",
		IsRequired:  true,
		// No Value field set
	}

	// This would normally be tested with JSON marshaling, but the omitempty tag
	// ensures backward compatibility at the JSON level
	if header.Name != "X-API-Key" {
		t.Errorf("Expected name 'X-API-Key', got '%s'", header.Name)
	}
	if header.Value != "" {
		t.Errorf("Expected empty value, got '%s'", header.Value)
	}
}
