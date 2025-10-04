package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresDB creates and returns a new PostgreSQL connection pool.
// It will gracefully attempt to connect to the database with retries.
func NewPostgresPool(databaseURL string) *pgxpool.Pool {
	if databaseURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is not set")
	}

	var pool *pgxpool.Pool
	var err error

	// Retry connecting to the database a few times in case it's not ready yet.
	// This is useful in containerized environments.
	maxRetries := 5
	retryDelay := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.New(context.Background(), databaseURL)
		if err == nil {
			// Check if we can actually connect
			if connErr := pool.Ping(context.Background()); connErr == nil {
				log.Println("✅ Successfully connected to PostgreSQL database")
				return pool
			} else {
				log.Printf("... failed to ping database: %v", connErr)
				pool.Close()
			}
		}

		log.Printf("... could not connect to database (attempt %d/%d), retrying in %v...", i+1, maxRetries, retryDelay)
		time.Sleep(retryDelay)
	}

	// If we've exhausted all retries, log the final error and exit.
	log.Fatalf("❌ Failed to connect to PostgreSQL after %d attempts: %v", maxRetries, err)
	os.Exit(1)
	return nil
}
