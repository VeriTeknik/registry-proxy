package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

// DB holds the database connection pool
type DB struct {
	*sql.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB() (*DB, error) {
	// Get connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Open connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✓ Connected to PostgreSQL database")

	return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// GetServerStats retrieves stats for a server
func (db *DB) GetServerStats(ctx context.Context, serverID string) (rating float64, ratingCount, installCount int, err error) {
	query := `
		SELECT
			COALESCE(rating, 0) as rating,
			COALESCE(rating_count, 0) as rating_count,
			COALESCE(installation_count, 0) as installation_count
		FROM proxy_server_stats
		WHERE server_id = $1
	`

	err = db.QueryRowContext(ctx, query, serverID).Scan(&rating, &ratingCount, &installCount)
	if err == sql.ErrNoRows {
		// No stats yet, return zeros
		return 0, 0, 0, nil
	}
	return rating, ratingCount, installCount, err
}

// UpsertRating inserts or updates a user rating
func (db *DB) UpsertRating(ctx context.Context, serverID, userID string, rating int, comment string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback on error; ignore error if already committed

	// Insert or update rating
	_, err = tx.ExecContext(ctx, `
		INSERT INTO proxy_user_ratings (server_id, user_id, rating, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (server_id, user_id)
		DO UPDATE SET
			rating = $3,
			comment = $4,
			updated_at = NOW()
	`, serverID, userID, rating, comment)
	if err != nil {
		return fmt.Errorf("failed to upsert rating: %w", err)
	}

	// Recalculate and update server stats
	_, err = tx.ExecContext(ctx, `
		INSERT INTO proxy_server_stats (server_id, rating, rating_count, updated_at)
		SELECT
			server_id,
			AVG(rating)::numeric(3,2),
			COUNT(*)::integer,
			NOW()
		FROM proxy_user_ratings
		WHERE server_id = $1
		GROUP BY server_id
		ON CONFLICT (server_id)
		DO UPDATE SET
			rating = EXCLUDED.rating,
			rating_count = EXCLUDED.rating_count,
			updated_at = NOW()
	`, serverID)
	if err != nil {
		return fmt.Errorf("failed to update server stats: %w", err)
	}

	return tx.Commit()
}

// TrackInstallation tracks a server installation
func (db *DB) TrackInstallation(ctx context.Context, serverID, userID, source, version, platform string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback on error; ignore error if already committed

	// Insert installation (or update if exists)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO proxy_user_installations (server_id, user_id, source, version, platform, installed_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (server_id, user_id)
		DO UPDATE SET
			version = $4,
			platform = $5,
			installed_at = NOW()
	`, serverID, userID, source, version, platform)
	if err != nil {
		return fmt.Errorf("failed to track installation: %w", err)
	}

	// Update installation count in server_stats
	_, err = tx.ExecContext(ctx, `
		INSERT INTO proxy_server_stats (server_id, installation_count, updated_at)
		SELECT server_id, COUNT(DISTINCT user_id)::integer, NOW()
		FROM proxy_user_installations
		WHERE server_id = $1
		GROUP BY server_id
		ON CONFLICT (server_id)
		DO UPDATE SET
			installation_count = EXCLUDED.installation_count,
			updated_at = NOW()
	`, serverID)
	if err != nil {
		return fmt.Errorf("failed to update installation count: %w", err)
	}

	return tx.Commit()
}

// Review represents a user review
type Review struct {
	UUID             string    `json:"uuid"`
	ServerSource     string    `json:"server_source"`
	ServerExternalID string    `json:"server_external_id"`
	UserID           string    `json:"user_id"`
	Rating           int       `json:"rating"`
	Comment          string    `json:"comment"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// GetReviews retrieves all reviews for a server
func (db *DB) GetReviews(ctx context.Context, serverID string) ([]Review, error) {
	query := `
		SELECT server_id, user_id, rating, comment, created_at, updated_at
		FROM proxy_user_ratings
		WHERE server_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var serverIDStr, userIDStr string
		var comment sql.NullString

		err := rows.Scan(&serverIDStr, &userIDStr, &r.Rating, &comment, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %w", err)
		}

		// Generate a unique UUID for this review (combination of server_id and user_id)
		r.UUID = fmt.Sprintf("%s:%s", serverIDStr, userIDStr)
		r.ServerSource = "REGISTRY"
		r.ServerExternalID = serverIDStr
		r.UserID = userIDStr
		r.Comment = comment.String

		reviews = append(reviews, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reviews: %w", err)
	}

	return reviews, nil
}

// GetReviewsPaginated retrieves reviews for a server with pagination and sorting
func (db *DB) GetReviewsPaginated(ctx context.Context, serverID string, limit, offset int, sort string) ([]Review, int, error) {
	// Use whitelist for valid sort columns to prevent SQL injection
	validSorts := map[string]string{
		"newest":      "created_at DESC",
		"oldest":      "created_at ASC",
		"rating_high": "rating DESC, created_at DESC",
		"rating_low":  "rating ASC, created_at DESC",
	}

	orderBy, ok := validSorts[sort]
	if !ok {
		orderBy = validSorts["newest"] // safe default
	}

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM proxy_user_ratings WHERE server_id = $1`
	err := db.QueryRowContext(ctx, countQuery, serverID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	// Get paginated reviews - orderBy is now from whitelist, safe to use
	query := `
		SELECT server_id, user_id, rating, comment, created_at, updated_at
		FROM proxy_user_ratings
		WHERE server_id = $1
		ORDER BY ` + orderBy + `
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var serverIDStr, userIDStr string
		var comment sql.NullString

		err := rows.Scan(&serverIDStr, &userIDStr, &r.Rating, &comment, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}

		// Generate a unique UUID for this review (combination of server_id and user_id)
		r.UUID = fmt.Sprintf("%s:%s", serverIDStr, userIDStr)
		r.ServerSource = "REGISTRY"
		r.ServerExternalID = serverIDStr
		r.UserID = userIDStr
		r.Comment = comment.String

		reviews = append(reviews, r)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reviews: %w", err)
	}

	return reviews, totalCount, nil
}

// NewRegistryDB creates a connection to the registry database
func NewRegistryDB() (*DB, error) {
	// Get connection string from environment or use default for docker network
	dbURL := os.Getenv("REGISTRY_DATABASE_URL")
	if dbURL == "" {
		// Default to the PostgreSQL container in the same network
		dbURL = "postgres://mcpregistry:mcpregistry@postgresql:5432/mcp_registry?sslmode=disable"
	}

	// Open connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open registry database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping registry database: %w", err)
	}

	log.Println("✓ Connected to Registry PostgreSQL database")

	return &DB{db}, nil
}

// ServerFilter contains all possible filters for servers
type ServerFilter struct {
	Search           string   `json:"search,omitempty"`
	RegistryTypes    []string `json:"registry_types,omitempty"` // npm, pypi, oci, remote, etc
	Category         string   `json:"category,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	HasTransport     []string `json:"has_transport,omitempty"` // sse, http, stdio
	MinRating        float64  `json:"min_rating,omitempty"`
	MinInstalls      int      `json:"min_installs,omitempty"`
}

// QueryServersEnhanced queries servers with filtering, sorting, and enrichment
// Uses squirrel query builder to prevent SQL injection and improve maintainability
func (db *DB) QueryServersEnhanced(ctx context.Context, filter ServerFilter, sort string, limit, offset int) ([]map[string]interface{}, int, error) {
	// Build the complete query using query builders
	query, args, err := buildMainQuery(filter, sort, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute query
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query servers: %w", err)
	}
	defer rows.Close()

	servers := []map[string]interface{}{}
	var totalCount int

	// Process each row
	for rows.Next() {
		// Scan row into individual fields
		serverName, valueJSON, publishedAt, updatedAt, rating, ratingCount, installCount, total, err := scanServerRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan server: %w", err)
		}

		// Create stats struct
		stats := ServerStats{
			Rating:            rating,
			RatingCount:       ratingCount,
			InstallationCount: installCount,
		}

		// Map row to server with enrichment
		server, err := mapRowToServer(serverName, valueJSON, publishedAt, updatedAt, stats)
		if err != nil {
			return nil, 0, err
		}

		servers = append(servers, server)
		totalCount = total // Will be the same for all rows
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating servers: %w", err)
	}

	return servers, totalCount, nil
}

// calculateQualityScore calculates a quality score for a server
func calculateQualityScore(rating float64, ratingCount, installCount int) float64 {
	// Weighted formula for quality
	// Rating contributes 40%, review count 30%, installs 30%
	ratingScore := rating * 8 // Max 40 (5 * 8)

	// Logarithmic scaling for counts to prevent domination by outliers
	reviewScore := 0.0
	if ratingCount > 0 {
		reviewScore = math.Min(math.Log10(float64(ratingCount)+1)*10, 30) // Max 30
	}

	installScore := 0.0
	if installCount > 0 {
		installScore = math.Min(math.Log10(float64(installCount)+1)*10, 30) // Max 30
	}

	return math.Round((ratingScore+reviewScore+installScore)*10) / 10 // Round to 1 decimal
}

// generateBadges generates achievement badges for a server
func generateBadges(server map[string]interface{}, rating float64, ratingCount, installCount int) []map[string]string {
	badges := []map[string]string{}

	// Top Rated badge
	if rating >= 4.5 && ratingCount >= 10 {
		badges = append(badges, map[string]string{
			"type":  "top_rated",
			"label": "Top Rated",
			"icon":  "⭐",
		})
	}

	// Popular badge
	if installCount >= 100 {
		badges = append(badges, map[string]string{
			"type":  "popular",
			"label": "Popular",
			"icon":  "🔥",
		})
	}

	// Well Reviewed badge
	if ratingCount >= 50 {
		badges = append(badges, map[string]string{
			"type":  "well_reviewed",
			"label": "Well Reviewed",
			"icon":  "💬",
		})
	}

	// Official badge (if from modelcontextprotocol org)
	if packages, ok := server["packages"].([]interface{}); ok && len(packages) > 0 {
		for _, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]interface{}); ok {
				if identifier, ok := pkgMap["identifier"].(string); ok {
					if strings.HasPrefix(identifier, "@modelcontextprotocol/") {
						badges = append(badges, map[string]string{
							"type":  "official",
							"label": "Official",
							"icon":  "✓",
						})
						break
					}
				}
			}
		}
	}

	// New badge (created in last 7 days)
	if publishedAt, ok := server["published_at"].(time.Time); ok {
		if time.Since(publishedAt) < 7*24*time.Hour {
			badges = append(badges, map[string]string{
				"type":  "new",
				"label": "New",
				"icon":  "🆕",
			})
		}
	}

	return badges
}
