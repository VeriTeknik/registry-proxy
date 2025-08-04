package db

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB represents a MongoDB connection
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(ctx context.Context) (*MongoDB, error) {
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "registry"
	}

	clientOptions := options.Client().
		ApplyURI(mongoURL).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &MongoDB{
		client:   client,
		database: client.Database(dbName),
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// GetServersCollection returns the servers collection
func (m *MongoDB) GetServersCollection() *mongo.Collection {
	return m.database.Collection("servers_v2")
}

// GetAuditLogCollection returns the audit log collection
func (m *MongoDB) GetAuditLogCollection() *mongo.Collection {
	return m.database.Collection("audit_logs")
}