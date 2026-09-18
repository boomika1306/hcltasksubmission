package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"livepoll-backend/config"
	"livepoll-backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SubmitVote processes an audience vote, atomically updates MongoDB, and publishes to Redis Pub/Sub
func SubmitVote(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll code is required"})
		return
	}

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vote request: " + err.Error()})
		return
	}

	req.OptionID = strings.TrimSpace(req.OptionID)
	if req.OptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Option ID cannot be empty"})
		return
	}

	pollColl := config.GetCollection("polls")
	if pollColl == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1. Atomically increment the selected option's votes and poll totalVotes in MongoDB
	filter := bson.M{
		"code":       code,
		"options.id": req.OptionID,
		"isClosed":   false,
	}

	update := bson.M{
		"$inc": bson.M{
			"options.$.votes": 1,
			"totalVotes":      1,
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedPoll models.Poll
	err := pollColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedPoll)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Determine specific failure reason for high-quality error message
			var existingPoll models.Poll
			checkErr := pollColl.FindOne(ctx, bson.M{"code": code}).Decode(&existingPoll)
			if checkErr != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Poll does not exist"})
				return
			}
			if existingPoll.IsClosed {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Voting on this poll is closed"})
				return
			}

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option selected for this poll"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record vote: " + err.Error()})
		return
	}

	// 2. Publish real-time vote update to Redis Pub/Sub channel "poll:{code}"
	event := models.RealTimeMessage{
		Type:      "VOTE_UPDATE",
		Poll:      &updatedPoll,
		Timestamp: time.Now(),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf("⚠️ Failed to marshal vote event for Redis: %v", err)
	} else {
		channelName := "poll:" + code
		if err := config.PublishEvent(context.Background(), channelName, string(eventBytes)); err != nil {
			log.Printf("⚠️ Failed to publish vote update to Redis channel [%s]: %v", channelName, err)
		} else {
			log.Printf(" Vote published to Redis channel [%s] (Total votes: %d)", channelName, updatedPoll.TotalVotes)
		}
	}

	// 3. Return response to the voter
	c.JSON(http.StatusOK, gin.H{
		"message": "Vote recorded successfully",
		"poll":    updatedPoll,
	})
}
