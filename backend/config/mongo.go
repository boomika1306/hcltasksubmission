package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
)

// ConnectMongo initializes the MongoDB client and verifies connection
func ConnectMongo() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(AppConfig.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create mongo client: %w", err)
	}

	// Ping database to confirm connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping mongodb: %w", err)
	}

	MongoClient = client
	MongoDB = client.Database(AppConfig.MongoDB)
	log.Printf(" Connected to MongoDB at %s (Database: %s)", AppConfig.MongoURI, AppConfig.MongoDB)

	// Create indexes
	createIndexes()

	return nil
}

// createIndexes ensures essential indexes exist for performance and uniqueness
func createIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Unique index on User email
	userColl := MongoDB.Collection("users")
	_, err := userColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("⚠️ User email index notice: %v", err)
	}

	// 2. Unique index on Poll code
	pollColl := MongoDB.Collection("polls")
	_, err = pollColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("⚠️ Poll code index notice: %v", err)
	}
}

// GetCollection returns a handle to a MongoDB collection
func GetCollection(name string) *mongo.Collection {
	if MongoDB == nil {
		return nil
	}
	return MongoDB.Collection(name)
}
