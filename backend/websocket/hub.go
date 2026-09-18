package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"livepoll-backend/config"
	"livepoll-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin WebSocket connections
	},
}

// Client represents a single active WebSocket connection
type Client struct {
	hub      *Hub
	pollCode string
	conn     *websocket.Conn
	send     chan []byte
}

// Room holds all active WebSocket clients viewing a specific poll
type Room struct {
	pollCode   string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	ctx        context.Context
	cancel     context.CancelFunc
}

// Hub manages all poll rooms and orchestrates Redis subscriptions
type Hub struct {
	rooms      map[string]*Room
	mutex      sync.RWMutex
	register   chan *Client
	unregister chan *Client
}

// GlobalHub is the singleton instance of WebSocket Hub
var GlobalHub = &Hub{
	rooms:      make(map[string]*Room),
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

// Run starts the central WebSocket hub event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			room, exists := h.rooms[client.pollCode]
			if !exists {
				ctx, cancel := context.WithCancel(context.Background())
				room = &Room{
					pollCode:   client.pollCode,
					clients:    make(map[*Client]bool),
					broadcast:  make(chan []byte, 32),
					register:   make(chan *Client),
					unregister: make(chan *Client),
					ctx:        ctx,
					cancel:     cancel,
				}
				h.rooms[client.pollCode] = room
				go room.run()
				go h.subscribeToRedis(room)
			}
			h.mutex.Unlock()
			room.register <- client

		case client := <-h.unregister:
			h.mutex.RLock()
			room, exists := h.rooms[client.pollCode]
			h.mutex.RUnlock()
			if exists {
				room.unregister <- client
			}
		}
	}
}

// run manages client registration and broadcasting for a specific poll room
func (r *Room) run() {
	defer func() {
		r.cancel()
	}()

	for {
		select {
		case client := <-r.register:
			r.clients[client] = true
			log.Printf(" Client connected to poll [%s]. Active viewers in room: %d", r.pollCode, len(r.clients))

		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
				log.Printf(" Client disconnected from poll [%s]. Remaining viewers: %d", r.pollCode, len(r.clients))
			}

			// Clean up room if no clients remain
			if len(r.clients) == 0 {
				GlobalHub.mutex.Lock()
				delete(GlobalHub.rooms, r.pollCode)
				GlobalHub.mutex.Unlock()
				r.cancel()
				log.Printf("🧹 Poll room [%s] closed - all viewers left", r.pollCode)
				return
			}

		case message := <-r.broadcast:
			for client := range r.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(r.clients, client)
				}
			}

		case <-r.ctx.Done():
			return
		}
	}
}

// subscribeToRedis listens to Redis Pub/Sub channel "poll:{code}" and pushes to the Room broadcast channel
func (h *Hub) subscribeToRedis(room *Room) {
	if config.RedisClient == nil {
		log.Printf("⚠️ Redis client unavailable; Pub/Sub disabled for room [%s]", room.pollCode)
		return
	}

	channelName := "poll:" + room.pollCode
	pubsub := config.RedisClient.Subscribe(room.ctx, channelName)
	defer pubsub.Close()

	log.Printf(" Subscribed to Redis channel: %s", channelName)

	ch := pubsub.Channel()
	for {
		select {
		case <-room.ctx.Done():
			log.Printf(" Unsubscribed from Redis channel: %s", channelName)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Forward Redis event directly to room clients
			room.broadcast <- []byte(msg.Payload)
		}
	}
}

// HandleLivePoll upgrades the HTTP request to a WebSocket connection and registers the client
func HandleLivePoll(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Poll code is required"})
		return
	}

	// Verify poll exists in MongoDB
	pollColl := config.GetCollection("polls")
	var poll models.Poll
	err := pollColl.FindOne(c.Request.Context(), bson.M{"code": code}).Decode(&poll)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade WebSocket connection: %v", err)
		return
	}

	client := &Client{
		hub:      GlobalHub,
		pollCode: code,
		conn:     conn,
		send:     make(chan []byte, 64),
	}

	// Register client with global hub
	GlobalHub.register <- client

	// Immediately send initial snapshot of the poll
	initMsg := models.RealTimeMessage{
		Type:      "INIT",
		Poll:      &poll,
		Timestamp: time.Now(),
	}
	if initBytes, err := json.Marshal(initMsg); err == nil {
		client.send <- initBytes
	}

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}
