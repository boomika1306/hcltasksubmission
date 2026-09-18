package handlers

import (
	"context"
	"net/http"
	"time"

	"livepoll-backend/config"
	"livepoll-backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionSummary holds metadata for a MongoDB collection
type CollectionSummary struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// DBStatsResponse holds comprehensive database diagnostics
type DBStatsResponse struct {
	DatabaseName string              `json:"databaseName"`
	MongoURI     string              `json:"mongoUri"`
	Status       string              `json:"status"`
	PingMs       int64               `json:"pingMs"`
	Collections  []CollectionSummary `json:"collections"`
	RedisStatus  string              `json:"redisStatus"`
	RedisAddr    string              `json:"redisAddr"`
}

// GetDBStats returns MongoDB database stats and collection summaries
func GetDBStats(c *gin.Context) {
	if config.MongoDB == nil {
		c.JSON(http.StatusOK, DBStatsResponse{
			DatabaseName: config.AppConfig.MongoDB,
			MongoURI:     config.AppConfig.MongoURI,
			Status:       "disconnected",
			RedisStatus:  "disconnected",
			RedisAddr:    config.AppConfig.RedisAddr,
		})
		return
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Ping check
	status := "connected"
	if err := config.MongoClient.Ping(ctx, nil); err != nil {
		status = "error: " + err.Error()
	}
	pingMs := time.Since(start).Milliseconds()

	// List collections and counts
	collNames, err := config.MongoDB.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		collNames = []string{"users", "polls"}
	}

	summaries := make([]CollectionSummary, 0)
	for _, name := range collNames {
		count, _ := config.MongoDB.Collection(name).EstimatedDocumentCount(ctx)
		summaries = append(summaries, CollectionSummary{
			Name:  name,
			Count: count,
		})
	}

	// Redis status
	redisStatus := "connected"
	if config.RedisClient == nil || config.RedisClient.Ping(ctx).Err() != nil {
		redisStatus = "disconnected"
	}

	c.JSON(http.StatusOK, DBStatsResponse{
		DatabaseName: config.AppConfig.MongoDB,
		MongoURI:     config.AppConfig.MongoURI,
		Status:       status,
		PingMs:       pingMs,
		Collections:  summaries,
		RedisStatus:  redisStatus,
		RedisAddr:    config.AppConfig.RedisAddr,
	})
}

// GetCollectionDocuments returns documents for manual dashboard inspection
func GetCollectionDocuments(c *gin.Context) {
	name := c.Param("name")
	if name != "users" && name != "polls" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection name. Allowed: users, polls"})
		return
	}

	coll := config.GetCollection(name)
	if coll == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	findOpts := options.Find().SetLimit(50)
	if name == "polls" {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	} else if name == "users" {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	}

	cursor, err := coll.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read documents: " + err.Error()})
		return
	}
	defer cursor.Close(ctx)

	if name == "users" {
		var users []models.User
		if err := cursor.All(ctx, &users); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Password is automatically excluded via json:"-" tag
		c.JSON(http.StatusOK, users)
		return
	}

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, polls)
}
