package routes

import (
	"context"
	"net/http"
	"time"

	"livepoll-backend/config"
	"livepoll-backend/handlers"
	"livepoll-backend/middleware"
	"livepoll-backend/websocket"

	"github.com/gin-gonic/gin"
)

// SetupRoutes registers all application endpoints onto the Gin engine
func SetupRoutes(router *gin.Engine) {
	// Root and Health check
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"app":     "LivePoll Backend API",
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	api := router.Group("/api")
	{
		// System Health
		api.GET("/health", func(c *gin.Context) {
			mongoStatus := "connected"
			redisStatus := "connected"

			// Check MongoDB
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if config.MongoClient == nil || config.MongoClient.Ping(ctx, nil) != nil {
				mongoStatus = "disconnected"
			}

			// Check Redis
			if config.RedisClient == nil || config.RedisClient.Ping(ctx).Err() != nil {
				redisStatus = "disconnected"
			}

			c.JSON(http.StatusOK, gin.H{
				"status":    "ok",
				"mongodb":   mongoStatus,
				"redis":     redisStatus,
				"timestamp": time.Now(),
			})
		})

		// Authentication Routes (Public)
		api.POST("/signup", handlers.Signup)
		api.POST("/login", handlers.Login)

		// Public Poll and Voting Routes
		api.GET("/polls/:code", handlers.GetPollByCode)
		api.POST("/polls/:code/vote", handlers.SubmitVote)

		// Real-Time Live WebSocket Endpoint
		api.GET("/polls/:code/live", websocket.HandleLivePoll)

		// Database Explorer & Manual Dashboard endpoints
		api.GET("/admin/db-stats", handlers.GetDBStats)
		api.GET("/admin/collections/:name", handlers.GetCollectionDocuments)

		// Protected Poll Management Routes (Require JWT)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/polls", handlers.CreatePoll)
			protected.GET("/my-polls", handlers.GetMyPolls)
			protected.DELETE("/polls/:id", handlers.DeletePoll)
		}
	}
}
