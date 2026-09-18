package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Option represents a selectable poll choice
type Option struct {
	ID    string `bson:"id" json:"id"`
	Text  string `bson:"text" json:"text"`
	Votes int    `bson:"votes" json:"votes"`
}

// Poll represents a live poll stored in MongoDB
type Poll struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code         string             `bson:"code" json:"code"`
	Question     string             `bson:"question" json:"question"`
	Options      []Option           `bson:"options" json:"options"`
	TotalVotes   int                `bson:"totalVotes" json:"totalVotes"`
	CreatorID    primitive.ObjectID `bson:"creatorId" json:"creatorId"`
	CreatorName  string             `bson:"creatorName" json:"creatorName"`
	CreatorEmail string             `bson:"creatorEmail" json:"creatorEmail"`
	IsClosed     bool               `bson:"isClosed" json:"isClosed"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// CreatePollRequest defines the payload required to create a new poll
type CreatePollRequest struct {
	Question string   `json:"question" binding:"required,min=5,max=300"`
	Options  []string `json:"options" binding:"required,min=2,dive,required,min=1,max=100"`
}

// VoteRequest defines the payload to submit a vote
type VoteRequest struct {
	OptionID string `json:"optionId" binding:"required"`
}

// RealTimeMessage represents the JSON payload published over Redis and pushed via WebSocket
type RealTimeMessage struct {
	Type      string    `json:"type"` // e.g., "INIT", "VOTE_UPDATE", "POLL_CLOSED", "POLL_DELETED"
	Poll      *Poll     `json:"poll,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
