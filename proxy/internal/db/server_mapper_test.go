package db

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMapRowToServer(t *testing.T) {
	publishedAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	serverJSON := `{
		"description": "Test MCP server",
		"category": "data-analysis",
		"tags": ["ai", "automation"]
	}`

	stats := ServerStats{
		Rating:            4.5,
		RatingCount:       10,
		InstallationCount: 100,
	}

	result, err := mapRowToServer("test-server", []byte(serverJSON), publishedAt, updatedAt, stats)

	if err != nil {
		t.Fatalf("mapRowToServer() error = %v", err)
	}

	// Check basic fields
	if result["id"] != "test-server" {
		t.Errorf("id = %v, want test-server", result["id"])
	}
	if result["name"] != "test-server" {
		t.Errorf("name = %v, want test-server", result["name"])
	}

	// Check stats fields at top level
	if result["rating"] != 4.5 {
		t.Errorf("rating = %v, want 4.5", result["rating"])
	}
	if result["rating_count"] != 10 {
		t.Errorf("rating_count = %v, want 10", result["rating_count"])
	}
	if result["installation_count"] != 100 {
		t.Errorf("installation_count = %v, want 100", result["installation_count"])
	}

	// Check nested stats
	nestedStats, ok := result["stats"].(map[string]interface{})
	if !ok {
		t.Fatal("stats is not a map")
	}
	if nestedStats["rating"] != 4.5 {
		t.Errorf("stats.rating = %v, want 4.5", nestedStats["rating"])
	}

	// Check quality score is calculated
	if result["quality_score"] == nil {
		t.Error("quality_score is missing")
	}

	// Check badges are generated
	if result["badges"] == nil {
		t.Error("badges are missing")
	}

	// Check original JSON fields are preserved
	if result["description"] != "Test MCP server" {
		t.Errorf("description = %v, want Test MCP server", result["description"])
	}
	if result["category"] != "data-analysis" {
		t.Errorf("category = %v, want data-analysis", result["category"])
	}
}

func TestMapRowToServer_InvalidJSON(t *testing.T) {
	publishedAt := time.Now()
	updatedAt := time.Now()
	stats := ServerStats{Rating: 4.0, RatingCount: 5, InstallationCount: 50}

	invalidJSON := `{"invalid": json}`

	_, err := mapRowToServer("test-server", []byte(invalidJSON), publishedAt, updatedAt, stats)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestEnrichServerWithStats(t *testing.T) {
	server := map[string]interface{}{
		"name":        "test-server",
		"description": "Test description",
	}

	stats := ServerStats{
		Rating:            4.2,
		RatingCount:       15,
		InstallationCount: 200,
	}

	result := EnrichServerWithStats(server, stats)

	// Check top-level stats
	if result["rating"] != 4.2 {
		t.Errorf("rating = %v, want 4.2", result["rating"])
	}
	if result["rating_count"] != 15 {
		t.Errorf("rating_count = %v, want 15", result["rating_count"])
	}
	if result["installation_count"] != 200 {
		t.Errorf("installation_count = %v, want 200", result["installation_count"])
	}

	// Check nested stats
	nestedStats, ok := result["stats"].(map[string]interface{})
	if !ok {
		t.Fatal("stats is not a map")
	}
	if nestedStats["rating"] != 4.2 {
		t.Errorf("stats.rating = %v, want 4.2", nestedStats["rating"])
	}
	if nestedStats["rating_count"] != 15 {
		t.Errorf("stats.rating_count = %v, want 15", nestedStats["rating_count"])
	}
	if nestedStats["install_count"] != 200 {
		t.Errorf("stats.install_count = %v, want 200", nestedStats["install_count"])
	}

	// Check quality score
	qualityScore, ok := result["quality_score"].(float64)
	if !ok {
		t.Fatal("quality_score is not a float64")
	}
	if qualityScore <= 0 {
		t.Error("quality_score should be greater than 0")
	}

	// Check badges
	if result["badges"] == nil {
		t.Error("badges are missing")
	}

	// Check original fields are preserved
	if result["name"] != "test-server" {
		t.Error("Original name field not preserved")
	}
	if result["description"] != "Test description" {
		t.Error("Original description field not preserved")
	}
}

func TestEnrichServerWithStats_ZeroStats(t *testing.T) {
	server := map[string]interface{}{
		"name": "new-server",
	}

	stats := ServerStats{
		Rating:            0,
		RatingCount:       0,
		InstallationCount: 0,
	}

	result := EnrichServerWithStats(server, stats)

	// Should still create stats fields even with zero values
	if result["rating"] != float64(0) {
		t.Errorf("rating = %v, want 0", result["rating"])
	}
	if result["rating_count"] != 0 {
		t.Errorf("rating_count = %v, want 0", result["rating_count"])
	}
	if result["installation_count"] != 0 {
		t.Errorf("installation_count = %v, want 0", result["installation_count"])
	}

	// Quality score should exist even for zero stats
	if result["quality_score"] == nil {
		t.Error("quality_score should exist even for zero stats")
	}
}

func TestServerStatsStructure(t *testing.T) {
	// Test that ServerStats can be properly marshaled/unmarshaled
	stats := ServerStats{
		Rating:            3.8,
		RatingCount:       25,
		InstallationCount: 500,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal ServerStats: %v", err)
	}

	var decoded ServerStats
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ServerStats: %v", err)
	}

	if decoded.Rating != stats.Rating {
		t.Errorf("Rating = %v, want %v", decoded.Rating, stats.Rating)
	}
	if decoded.RatingCount != stats.RatingCount {
		t.Errorf("RatingCount = %v, want %v", decoded.RatingCount, stats.RatingCount)
	}
	if decoded.InstallationCount != stats.InstallationCount {
		t.Errorf("InstallationCount = %v, want %v", decoded.InstallationCount, stats.InstallationCount)
	}
}

// Benchmark tests
func BenchmarkMapRowToServer(b *testing.B) {
	serverJSON := `{
		"description": "Test MCP server",
		"category": "data-analysis",
		"tags": ["ai", "automation"],
		"packages": [{"name": "test-package", "registryType": "npm"}]
	}`
	publishedAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()
	stats := ServerStats{Rating: 4.5, RatingCount: 10, InstallationCount: 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mapRowToServer("test-server", []byte(serverJSON), publishedAt, updatedAt, stats)
	}
}

func BenchmarkEnrichServerWithStats(b *testing.B) {
	server := map[string]interface{}{
		"name":        "test-server",
		"description": "Test description",
		"category":    "data",
	}
	stats := ServerStats{Rating: 4.2, RatingCount: 15, InstallationCount: 200}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EnrichServerWithStats(server, stats)
	}
}
