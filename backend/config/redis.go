package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// ConnectRedis initializes the Redis client and validates connection
func ConnectRedis() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:     AppConfig.RedisAddr,
		Password: AppConfig.RedisPassword,
		DB:       0, // default DB
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis at %s: %w", AppConfig.RedisAddr, err)
	}

	RedisClient = client
	log.Printf(" Connected to Redis at %s", AppConfig.RedisAddr)
	return nil
}

// PublishEvent publishes a message payload to a specified Redis channel
func PublishEvent(ctx context.Context, channel string, message interface{}) error {
	if RedisClient == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return RedisClient.Publish(ctx, channel, message).Err()
}
