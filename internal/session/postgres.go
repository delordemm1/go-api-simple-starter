package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/delordemm1/go-api-simple-starter/internal/database"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("session not found")
	ErrExpired  = errors.New("session expired")
)

type postgresProvider struct {
	db  database.DBTX
	cfg Config
}

func newPostgresProvider(db database.DBTX, cfg Config) *postgresProvider {
	// Defaults
	if cfg.SlidingTTL == 0 {
		cfg.SlidingTTL = 7 * 24 * time.Hour // 7 days
	}
	if cfg.AbsoluteTTL == 0 {
		cfg.AbsoluteTTL = 30 * 24 * time.Hour // 30 days
	}
	return &postgresProvider{db: db, cfg: cfg}
}

func (p *postgresProvider) CreateAuthSession(ctx context.Context, userID string, userAgent string, ip string) (string, error) {
	raw, err := randomOpaque(32)
	if err != nil {
		return "", err
	}
	sessionID := "auth:" + raw

	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate session row id: %w", err)
	}

	now := time.Now()
	sql := `
		INSERT INTO user_active_sessions
			(id, user_id, session_token, user_agent, ip_address, last_active_at, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
	`
	_, execErr := p.db.Exec(ctx, sql, id.String(), userID, sessionID, nullable(userAgent), nullable(ip), now, now)
	if execErr != nil {
		return "", fmt.Errorf("failed to insert session: %w", execErr)
	}

	return sessionID, nil
}

func (p *postgresProvider) GetAndExtend(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" || !strings.Contains(sessionID, ":") {
		return "", ErrNotFound
	}

	var (
		userID       string
		createdAt    time.Time
		lastActiveAt time.Time
	)

	query := `
		SELECT user_id, created_at, last_active_at
		FROM user_active_sessions
		WHERE session_token = $1
		LIMIT 1
	`
	row := p.db.QueryRow(ctx, query, sessionID)
	if err := row.Scan(&userID, &createdAt, &lastActiveAt); err != nil {
		return "", ErrNotFound
	}

	now := time.Now()
	// Absolute TTL
	if now.Sub(createdAt) > p.cfg.AbsoluteTTL {
		// Best effort cleanup
		_, _ = p.db.Exec(ctx, `DELETE FROM user_active_sessions WHERE session_token = $1`, sessionID)
		return "", ErrExpired
	}
	// Sliding TTL
	if now.Sub(lastActiveAt) > p.cfg.SlidingTTL {
		// Best effort cleanup
		_, _ = p.db.Exec(ctx, `DELETE FROM user_active_sessions WHERE session_token = $1`, sessionID)
		return "", ErrExpired
	}

	// Extend sliding TTL
	_, _ = p.db.Exec(ctx, `UPDATE user_active_sessions SET last_active_at = $1 WHERE session_token = $2`, now, sessionID)

	return userID, nil
}

func (p *postgresProvider) Delete(ctx context.Context, sessionID string) error {
	_, err := p.db.Exec(ctx, `DELETE FROM user_active_sessions WHERE session_token = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func randomOpaque(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	// base64url without padding
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
