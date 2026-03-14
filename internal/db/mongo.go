package db

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func Connect(ctx context.Context) (*MongoDB, error) {
	uri := os.Getenv("MONGODB_URL")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "book_db"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

func (m *MongoDB) Disconnect(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

func (m *MongoDB) Books() *mongo.Collection {
	return m.Database.Collection("books")
}

func (m *MongoDB) Chapters() *mongo.Collection {
	return m.Database.Collection("chapters")
}

func (m *MongoDB) Questions() *mongo.Collection {
	return m.Database.Collection("questions")
}

func (m *MongoDB) UserAnswers() *mongo.Collection {
	return m.Database.Collection("user_answers")
}

func (m *MongoDB) Users() *mongo.Collection {
	return m.Database.Collection("users")
}

func (m *MongoDB) AllowedSources() *mongo.Collection {
	return m.Database.Collection("allowed_sources")
}
