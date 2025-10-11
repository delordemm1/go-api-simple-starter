package session

import (
	"context"
	"time"

	"github.com/delordemm1/go-api-simple-starter/internal/database"
)

// Config controls session TTLs.
type Config struct {
	// SlidingTTL is the idle timeout. Each valid access extends last_active_at by this duration.
	// Default: 7 days.
	SlidingTTL time.Duration

	// AbsoluteTTL is the maximum lifetime from creation. After this duration the session is invalid
	// regardless of activity. Default: 30 days.
	AbsoluteTTL time.Duration
}

// Provider defines operations for managing opaque sessions.
//
// Session IDs MUST be opaque, random, and prefixed with a type, e.g. "auth:".
type Provider interface {
	// CreateAuthSession creates a new auth session for the given user and returns the session ID,
	// e.g. "auth:..." with a base64url-encoded random token part.
	// Optional userAgent and ip can be recorded for auditing.
	CreateAuthSession(ctx context.Context, userID string, userAgent string, ip string) (sessionID string, err error)

	// GetAndExtend validates the given session ID (including TTL checks) and extends the sliding TTL.
	// It returns the associated user ID on success.
	GetAndExtend(ctx context.Context, sessionID string) (userID string, err error)

	// Delete deletes a session by its session ID. It should be idempotent.
	Delete(ctx context.Context, sessionID string) error
}

// NewPostgresProvider returns a Postgres-backed Provider implementation.
// Implemented in postgres.go.
func NewPostgresProvider(db database.DBTX, cfg Config) Provider {
	return newPostgresProvider(db, cfg)
}