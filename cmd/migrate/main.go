package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"    // PostgreSQL driver
	_ "github.com/joho/godotenv/autoload" // Automatically load .env file
	"github.com/pressly/goose/v3"
)

// Using filesystem-based migrations from MIGRATIONS_DIR or default "migrations".

func main() {
	// 1. Get database URL from environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is not set")
	}

	// 2. Open a database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to open database connection: %v", err)
	}
	defer db.Close()

	// 3. Ping the database to ensure connectivity
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	// 4. Configure Goose
	goose.SetDialect("postgres") // Use "postgres" for pgx/v5

	// Select migrations directory (filesystem)
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	if info, err := os.Stat(migrationsDir); err != nil || !info.IsDir() {
		log.Fatalf("❌ Migrations directory not found: %s. Run from the repository root or set MIGRATIONS_DIR. Error: %v", migrationsDir, err)
	}
	log.Printf("✅ Using migrations directory: %s", migrationsDir)

	// 5. Get command and arguments from os.Args
	// Example: 'go run ./cmd/migrate up' -> os.Args will be ["./cmd/migrate/main.go", "up"]
	if len(os.Args) < 2 {
		log.Fatalf("❌ Missing goose command. Usage: go run ./cmd/migrate [up|down|status|...]")
	}
	command := os.Args[1]
	args := os.Args[2:]

	// 6. Run the Goose command
	log.Printf("Running goose command: %s", command)
	if err := goose.RunContext(context.Background(), command, db, migrationsDir, args...); err != nil {
		log.Fatalf("❌ Goose command '%s' failed: %v", command, err)
	}
}
