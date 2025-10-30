package db

import (
	"encoding/json"
	"fmt"
	"time"
)

// ServerStats represents aggregated statistics for a server
type ServerStats struct {
	Rating           float64
	RatingCount      int
	InstallationCount int
}

// mapRowToServer converts a database row to a server map with enriched data
func mapRowToServer(
	serverName string,
	valueJSON []byte,
	publishedAt, updatedAt time.Time,
	stats ServerStats,
) (map[string]interface{}, error) {
	// Parse the JSON value
	var value map[string]interface{}
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		return nil, fmt.Errorf("failed to parse server JSON: %w", err)
	}

	// Add basic server fields
	value["id"] = serverName
	value["name"] = serverName
	value["published_at"] = publishedAt
	value["updated_at"] = updatedAt

	// Enrich with stats
	return enrichServerWithStats(value, stats), nil
}

// enrichServerWithStats adds statistics and derived fields to a server map
func enrichServerWithStats(server map[string]interface{}, stats ServerStats) map[string]interface{} {
	// Add stats fields at top level (expected by frontend)
	server["rating"] = stats.Rating
	server["rating_count"] = stats.RatingCount
	server["installation_count"] = stats.InstallationCount

	// Also keep nested stats for backward compatibility
	server["stats"] = map[string]interface{}{
		"rating":         stats.Rating,
		"rating_count":   stats.RatingCount,
		"install_count":  stats.InstallationCount,
	}

	// Calculate quality score
	server["quality_score"] = calculateQualityScore(stats.Rating, stats.RatingCount, stats.InstallationCount)

	// Add badges
	server["badges"] = generateBadges(server, stats.Rating, stats.RatingCount, stats.InstallationCount)

	return server
}

// addBadges is a convenience function that calls generateBadges with extracted stats
func addBadges(server map[string]interface{}) map[string]interface{} {
	rating, _ := server["rating"].(float64)
	ratingCount, _ := server["rating_count"].(int)
	installCount, _ := server["installation_count"].(int)

	server["badges"] = generateBadges(server, rating, ratingCount, installCount)
	return server
}

// scanServerRow scans a database row into individual fields
func scanServerRow(scanner interface {
	Scan(dest ...interface{}) error
}) (
	serverName string,
	valueJSON []byte,
	publishedAt, updatedAt time.Time,
	rating float64,
	ratingCount, installCount int,
	totalCount int,
	err error,
) {
	err = scanner.Scan(
		&serverName,
		&valueJSON,
		&publishedAt,
		&updatedAt,
		&rating,
		&ratingCount,
		&installCount,
		&totalCount,
	)
	return
}
