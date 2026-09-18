package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"livepoll-backend/config"
	"livepoll-backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// generateUniqueCode creates a friendly, readable 6-character alphanumeric code
func generateUniqueCode() string {
	const charset = "abcdefhkmnpqrstuvwxyz23456789" // omits easily confused characters (l, 1, o, 0)
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// CreatePoll handles poll creation by authenticated users
func CreatePoll(c *gin.Context) {
	var req models.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed: " + err.Error()})
		return
	}

	req.Question = strings.TrimSpace(req.Question)
	if len(req.Question) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Question must be at least 5 characters long"})
		return
	}

	// Filter and validate options
	var cleanedOptions []models.Option
	seen := make(map[string]bool)

	for i, optText := range req.Options {
		trimmed := strings.TrimSpace(optText)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if seen[lower] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Duplicate option detected: '%s'", trimmed)})
			return
		}
		seen[lower] = true

		cleanedOptions = append(cleanedOptions, models.Option{
			ID:    fmt.Sprintf("opt-%d-%s", i+1, generateUniqueCode()[:3]),
			Text:  trimmed,
			Votes: 0,
		})
	}

	if len(cleanedOptions) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least 2 non-empty, unique options are required"})
		return
	}
	if len(cleanedOptions) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A poll cannot have more than 10 options"})
		return
	}

	// Extract authenticated user information from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context missing"})
		return
	}
	userObjID := userID.(primitive.ObjectID)

	userName, _ := c.Get("userName")
	userEmail, _ := c.Get("userEmail")

	pollColl := config.GetCollection("polls")
	if pollColl == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Generate a unique code
	var code string
	for attempts := 0; attempts < 5; attempts++ {
		candidate := generateUniqueCode()
		count, err := pollColl.CountDocuments(ctx, bson.M{"code": candidate})
		if err == nil && count == 0 {
			code = candidate
			break
		}
	}
	if code == "" {
		code = fmt.Sprintf("poll-%d", time.Now().Unix()%1000000)
	}

	newPoll := models.Poll{
		ID:           primitive.NewObjectID(),
		Code:         code,
		Question:     req.Question,
		Options:      cleanedOptions,
		TotalVotes:   0,
		CreatorID:    userObjID,
		CreatorName:  userName.(string),
		CreatorEmail: userEmail.(string),
		IsClosed:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := pollColl.InsertOne(ctx, newPoll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create poll: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newPoll)
}

// GetPollByCode retrieves a single poll for audience viewing or voting
func GetPollByCode(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll code is required"})
		return
	}

	pollColl := config.GetCollection("polls")
	if pollColl == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var poll models.Poll
	err := pollColl.FindOne(ctx, bson.M{"code": code}).Decode(&poll)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or invalid poll code"})
		return
	}

	c.JSON(http.StatusOK, poll)
}

// GetMyPolls lists all polls created by the authenticated user
func GetMyPolls(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context missing"})
		return
	}
	userObjID := userID.(primitive.ObjectID)

	pollColl := config.GetCollection("polls")
	if pollColl == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	findOptions := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := pollColl.Find(ctx, bson.M{"creatorId": userObjID}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query polls: " + err.Error()})
		return
	}
	defer cursor.Close(ctx)

	polls := make([]models.Poll, 0)
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode polls: " + err.Error()})
		return
	}

	// Calculate user stats
	totalVotesAcrossAll := 0
	for _, p := range polls {
		totalVotesAcrossAll += p.TotalVotes
	}

	c.JSON(http.StatusOK, gin.H{
		"polls":      polls,
		"totalPolls": len(polls),
		"totalVotes": totalVotesAcrossAll,
	})
}

// DeletePoll removes a poll owned by the authenticated user
func DeletePoll(c *gin.Context) {
	pollIDStr := c.Param("id")
	pollObjID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID format"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context missing"})
		return
	}
	userObjID := userID.(primitive.ObjectID)

	pollColl := config.GetCollection("polls")
	if pollColl == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Find the poll first to verify ownership and grab code for Redis notification
	var poll models.Poll
	err = pollColl.FindOne(ctx, bson.M{"_id": pollObjID}).Decode(&poll)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	if poll.CreatorID != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this poll"})
		return
	}

	// Delete from MongoDB
	_, err = pollColl.DeleteOne(ctx, bson.M{"_id": pollObjID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete poll: " + err.Error()})
		return
	}

	// Notify real-time clients on Redis channel that the poll has been deleted
	delEvent := models.RealTimeMessage{
		Type:      "POLL_DELETED",
		Poll:      &poll,
		Timestamp: time.Now(),
	}
	if eventBytes, err := json.Marshal(delEvent); err == nil {
		_ = config.PublishEvent(context.Background(), "poll:"+poll.Code, string(eventBytes))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poll successfully deleted", "id": pollIDStr})
}
