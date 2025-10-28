package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
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
	defer tx.Rollback()

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
			$1,
			AVG(rating)::numeric(3,2),
			COUNT(*)::integer,
			NOW()
		FROM proxy_user_ratings
		WHERE server_id = $1
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
	defer tx.Rollback()

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
		SELECT $1, COUNT(DISTINCT user_id)::integer, NOW()
		FROM proxy_user_installations
		WHERE server_id = $1
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
