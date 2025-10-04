package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and returns a new Redis client.
// It will panic if it cannot connect to the Redis server.
func NewRedisClient(redisURL string) *redis.Client {
	if redisURL == "" {
		log.Fatal("❌ REDIS_URL environment variable is not set")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("❌ Could not parse Redis URL: %v", err)
	}

	client := redis.NewClient(opts)

	// Ping the Redis server to ensure a connection is established.
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("❌ Could not connect to Redis: %v", err)
	}

	log.Println("✅ Successfully connected to Redis")
	return client
}
