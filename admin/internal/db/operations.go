package db

import (
	"context"
	"fmt"
	"time"

	"github.com/pluggedin/registry-admin/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Operations provides database operations
type Operations struct {
	db *MongoDB
}

// NewOperations creates a new operations instance
func NewOperations(db *MongoDB) *Operations {
	return &Operations{db: db}
}

// ListServers retrieves servers with pagination
func (o *Operations) ListServers(ctx context.Context, page, limit int, status, registryName, search string) ([]models.ServerDetail, int64, error) {
	collection := o.db.GetServersCollection()

	// Build filter
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	if registryName != "" {
		filter["packages.registry_name"] = registryName
	}
	if search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find documents with pagination
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var servers []models.ServerDetail
	if err := cursor.All(ctx, &servers); err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

// GetServer retrieves a single server by ID
func (o *Operations) GetServer(ctx context.Context, id string) (*models.ServerDetail, error) {
	collection := o.db.GetServersCollection()

	var server models.ServerDetail
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&server)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("server not found")
		}
		return nil, err
	}

	return &server, nil
}

// ServerExists checks if a server with the given name exists
func (o *Operations) ServerExists(ctx context.Context, name string) (bool, error) {
	collection := o.db.GetServersCollection()
	
	count, err := collection.CountDocuments(ctx, bson.M{"name": name})
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// CreateServer creates a new server
func (o *Operations) CreateServer(ctx context.Context, server *models.ServerDetail) error {
	collection := o.db.GetServersCollection()

	// Check if server already exists
	count, err := collection.CountDocuments(ctx, bson.M{"name": server.Name})
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("server with name %s already exists", server.Name)
	}

	// Generate ID if not provided
	if server.ID == "" {
		server.ID = primitive.NewObjectID().Hex()
	}

	// Set default status if not provided
	if server.Status == "" {
		server.Status = models.ServerStatusActive
	}

	_, err = collection.InsertOne(ctx, server)
	return err
}

// UpdateServer updates an existing server
func (o *Operations) UpdateServer(ctx context.Context, id string, server *models.ServerDetail) error {
	collection := o.db.GetServersCollection()

	// Ensure ID matches
	server.ID = id

	result, err := collection.ReplaceOne(
		ctx,
		bson.M{"id": id},
		server,
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// DeleteServer deletes a server
func (o *Operations) DeleteServer(ctx context.Context, id string) error {
	collection := o.db.GetServersCollection()

	result, err := collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// UpdateStatus updates the status of a server
func (o *Operations) UpdateStatus(ctx context.Context, id string, status models.ServerStatus) error {
	collection := o.db.GetServersCollection()

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": bson.M{"status": status}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// LogAuditEntry creates an audit log entry
func (o *Operations) LogAuditEntry(ctx context.Context, entry *models.AuditLog) error {
	collection := o.db.GetAuditLogCollection()

	entry.ID = primitive.NewObjectID().Hex()
	entry.Timestamp = time.Now()

	_, err := collection.InsertOne(ctx, entry)
	return err
}

// GetAuditLogs retrieves audit logs
func (o *Operations) GetAuditLogs(ctx context.Context, limit int) ([]models.AuditLog, error) {
	collection := o.db.GetAuditLogCollection()

	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}