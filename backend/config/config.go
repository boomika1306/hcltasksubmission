package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the application
type Config struct {
	Port          string
	MongoURI      string
	MongoDB       string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string
	ClientURL     string
}

// AppConfig is the globally accessible configuration instance
var AppConfig *Config

// LoadConfig loads configuration from environment variables or .env file
func LoadConfig() *Config {
	// Try loading .env file (ignore error if not present, e.g., in production)
	if err := godotenv.Load(); err != nil {
		log.Println("ℹ️ No .env file found, reading from system environment variables")
	}

	AppConfig = &Config{
		Port:          getEnv("PORT", "8080"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:       getEnv("MONGO_DB", "livepoll"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", "livepoll_super_secret_jwt_key_guvi_hcl_2026"),
		ClientURL:     getEnv("CLIENT_URL", "http://localhost:5173"),
	}

	return AppConfig
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
