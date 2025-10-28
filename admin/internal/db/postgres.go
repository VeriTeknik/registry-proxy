package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB represents a PostgreSQL connection
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB creates a new PostgreSQL connection
func NewPostgresDB(ctx context.Context) (*PostgresDB, error) {
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://localhost:5432/mcp_registry"
	}

	pool, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &PostgresDB{
		pool: pool,
	}, nil
}

// Close closes the PostgreSQL connection
func (p *PostgresDB) Close() {
	p.pool.Close()
}

// GetPool returns the connection pool
func (p *PostgresDB) GetPool() *pgxpool.Pool {
	return p.pool
}
