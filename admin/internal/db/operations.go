package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pluggedin/registry-admin/internal/models"
)

// DefaultServerVersion is the default version assigned to servers when not specified
const DefaultServerVersion = "1.0.0"

// Operations provides database operations
type Operations struct {
	db *PostgresDB
}

// NewOperations creates a new operations instance
func NewOperations(db *PostgresDB) *Operations {
	return &Operations{db: db}
}

// ListServers retrieves servers with pagination
// TODO: Consider using a SQL builder library (e.g., squirrel, goqu) for more robust
// dynamic query construction, especially as query complexity grows with additional filters
func (o *Operations) ListServers(ctx context.Context, page, limit int, status, registryName, search string) ([]models.ServerDetail, int64, error) {
	pool := o.db.GetPool()

	// Build WHERE clause
	whereConditions := []string{"is_latest = true"}
	args := []interface{}{}
	argPos := 1

	if status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
		argPos++
	}

	if registryName != "" {
		// TODO(performance): Add a specific GIN index for packages.registry_name queries:
		// CREATE INDEX IF NOT EXISTS idx_servers_packages_registry_name
		// ON servers USING GIN ((value->'packages'));
		// This will improve performance for registry_name filtering on large datasets
		whereConditions = append(whereConditions, fmt.Sprintf("value->'packages' @> $%d::jsonb", argPos))
		args = append(args, fmt.Sprintf(`[{"registry_name":"%s"}]`, registryName))
		argPos++
	}

	if search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(server_name ILIKE $%d OR value->>'description' ILIKE $%d)", argPos, argPos))
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		argPos++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM servers %s", whereClause)
	var total int64
	err := pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count servers: %w", err)
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Query servers with pagination
	query := fmt.Sprintf(`
		SELECT server_name, version, value, status, published_at
		FROM servers
		%s
		ORDER BY server_name
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query servers: %w", err)
	}
	defer rows.Close()

	var servers []models.ServerDetail
	for rows.Next() {
		var serverName, version, status string
		var valueJSON []byte
		var publishedAt time.Time

		if err := rows.Scan(&serverName, &version, &valueJSON, &status, &publishedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan server: %w", err)
		}

		var server models.ServerDetail
		if err := json.Unmarshal(valueJSON, &server); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal server JSON: %w", err)
		}

		// Set additional fields
		server.ID = serverName
		server.Status = models.ServerStatus(status)

		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return servers, total, nil
}

// GetServer retrieves a single server by ID (server_name)
func (o *Operations) GetServer(ctx context.Context, id string) (*models.ServerDetail, error) {
	pool := o.db.GetPool()

	query := `
		SELECT server_name, version, value, status, published_at
		FROM servers
		WHERE server_name = $1 AND is_latest = true
		LIMIT 1
	`

	var serverName, version, status string
	var valueJSON []byte
	var publishedAt time.Time

	err := pool.QueryRow(ctx, query, id).Scan(&serverName, &version, &valueJSON, &status, &publishedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to query server: %w", err)
	}

	var server models.ServerDetail
	if err := json.Unmarshal(valueJSON, &server); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server JSON: %w", err)
	}

	server.ID = serverName
	server.Status = models.ServerStatus(status)

	return &server, nil
}

// ServerExists checks if a server with the given name exists
func (o *Operations) ServerExists(ctx context.Context, name string) (bool, error) {
	pool := o.db.GetPool()

	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers WHERE server_name = $1", name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check server existence: %w", err)
	}

	return count > 0, nil
}

// CreateServer creates a new server
func (o *Operations) CreateServer(ctx context.Context, server *models.ServerDetail) error {
	pool := o.db.GetPool()

	// Check if server already exists
	exists, err := o.ServerExists(ctx, server.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("server with name %s already exists", server.Name)
	}

	// Set default status if not provided
	if server.Status == "" {
		server.Status = models.ServerStatusActive
	}

	// Marshal server to JSON
	valueJSON, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server: %w", err)
	}

	// Use the server name as ID if not provided
	if server.ID == "" {
		server.ID = server.Name
	}

	// Insert server
	query := `
		INSERT INTO servers (server_name, version, value, status, published_at, updated_at, is_latest)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	version := DefaultServerVersion
	if server.VersionDetail.Version != "" {
		version = server.VersionDetail.Version
	}

	_, err = pool.Exec(ctx, query, server.Name, version, valueJSON, server.Status, now, now, true)
	if err != nil {
		return fmt.Errorf("failed to insert server: %w", err)
	}

	return nil
}

// UpdateServer updates an existing server
func (o *Operations) UpdateServer(ctx context.Context, id string, server *models.ServerDetail) error {
	pool := o.db.GetPool()

	// Ensure ID matches
	server.ID = id
	server.Name = id

	// Marshal server to JSON
	valueJSON, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server: %w", err)
	}

	// Update server
	query := `
		UPDATE servers
		SET value = $1, status = $2, updated_at = $3
		WHERE server_name = $4 AND is_latest = true
	`

	result, err := pool.Exec(ctx, query, valueJSON, server.Status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// DeleteServer deletes a server
func (o *Operations) DeleteServer(ctx context.Context, id string) error {
	pool := o.db.GetPool()

	result, err := pool.Exec(ctx, "DELETE FROM servers WHERE server_name = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// UpdateStatus updates the status of a server
func (o *Operations) UpdateStatus(ctx context.Context, id string, status models.ServerStatus) error {
	pool := o.db.GetPool()

	query := `
		UPDATE servers
		SET status = $1, updated_at = $2
		WHERE server_name = $3 AND is_latest = true
	`

	result, err := pool.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// LogAuditEntry logs an audit entry
// Note: This requires an audit_logs table which may not exist yet
func (o *Operations) LogAuditEntry(ctx context.Context, entry *models.AuditLog) error {
	// For now, just log to console since audit table may not exist
	fmt.Printf("[AUDIT] %s - %s by %s\n", entry.Action, entry.ServerID, entry.User)
	return nil
}

// GetAuditLogs retrieves audit logs
// Note: This requires an audit_logs table which may not exist yet
func (o *Operations) GetAuditLogs(ctx context.Context, limit int) ([]models.AuditLog, error) {
	// Return empty for now since audit table may not exist
	return []models.AuditLog{}, nil
}
