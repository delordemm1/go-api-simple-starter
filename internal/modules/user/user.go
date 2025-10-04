package user

import (
	"time"
)

// User represents a user in the system.
// This is the core entity for the user module, used across the repository, service, and handler layers.
type User struct {
	ID                       string     `db:"id"`
	FirstName                string     `db:"firstName"`
	LastName                 string     `db:"lastName"`
	Email                    string     `db:"email"`
	PasswordHash             string     `db:"password_hash"`
	EmailVerified            bool       `db:"email_verified"`
	PasswordResetToken       string     `db:"password_reset_token"`
	PasswordResetTokenExpiry *time.Time `db:"password_reset_token_expiry"`
	CreatedAt                time.Time  `db:"created_at"`
	UpdatedAt                time.Time  `db:"updated_at"`
}

type OAuthProvider string

const (
	OAuthProviderGOOGLE   OAuthProvider = "google"
	OAuthProviderFACEBOOK OAuthProvider = "facebook"
	OAuthProviderGITHUB   OAuthProvider = "github"
	OAuthProviderX        OAuthProvider = "x"
	OAuthProviderLINKEDIN OAuthProvider = "linkedin"
)

type OAuthState struct {
	State     string        `db:"state"`
	Provider  OAuthProvider `db:"provider"`
	UserID    *string       `db:"user_id"`
	Verifier  string        `db:"verifier"`
	ExpiresAt time.Time     `db:"expires_at"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}

type UserActiveSession struct {
	ID           string    `db:"id"`
	UserID       string    `db:"user_id"`
	SessionToken string    `db:"session_token"`
	UserAgent    string    `db:"user_agent"`
	IpAddress    string    `db:"ip_address"`
	LastActiveAt time.Time `db:"last_active_at"`
	CreatedAt    time.Time `db:"created_at"`
}
