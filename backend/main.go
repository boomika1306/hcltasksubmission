package main

import (
	"log"
	"os"

	"livepoll-backend/config"
	"livepoll-backend/middleware"
	"livepoll-backend/routes"
	"livepoll-backend/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("🚀 Initializing LivePoll Backend Service...")

	// 1. Load application configuration
	cfg := config.LoadConfig()

	// 2. Initialize MongoDB connection
	if err := config.ConnectMongo(); err != nil {
		log.Printf("⚠️ MongoDB connection error: %v", err)
		log.Println("💡 Please verify MongoDB is running or check MONGO_URI in .env")
	}

	// 3. Initialize Redis connection
	if err := config.ConnectRedis(); err != nil {
		log.Printf("⚠️ Redis connection error: %v", err)
		log.Println("💡 Please verify Redis is running or check REDIS_ADDR in .env")
	}

	// 4. Start WebSocket central Hub loop
	go websocket.GlobalHub.Run()
	log.Println("📡 Real-time WebSocket Hub started")

	// 5. Initialize Gin router
	router := gin.Default()

	// Apply CORS
	router.Use(middleware.SetupCORS())

	// Register API and WebSocket routes
	routes.SetupRoutes(router)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf(" LivePoll API Server listening on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
		os.Exit(1)
	}
}
