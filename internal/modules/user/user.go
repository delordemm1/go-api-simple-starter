package user

import (
	"time"
)

// User represents a user in the system.
// This is the core entity for the user module, used across the repository, service, and handler layers.
type User struct {
	ID                       string     `db:"id"`
	FirstName                string     `db:"first_name"`
	LastName                 string     `db:"last_name"`
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


// --- Verification & Reset Types ---

// VerificationPurpose defines the reason a 6-digit code is issued.
type VerificationPurpose string

const (
	VerificationPurposeEmailVerify  VerificationPurpose = "email_verify"
	VerificationPurposePasswordReset VerificationPurpose = "password_reset"
)

// VerificationChannel defines the medium used to deliver a verification code.
type VerificationChannel string

const (
	VerificationChannelEmail VerificationChannel = "email"
)

// VerificationCode represents a one-time 6-digit verification code issued to a user/contact.
type VerificationCode struct {
	ID          string               `db:"id"`
	UserID      *string              `db:"user_id"`
	Contact     string               `db:"contact"`
	Purpose     VerificationPurpose  `db:"purpose"`
	Channel     VerificationChannel  `db:"channel"`
	CodeHash    string               `db:"code_hash"`
	Attempts    int                  `db:"attempts"`
	MaxAttempts int                  `db:"max_attempts"`
	LastSentAt  time.Time            `db:"last_sent_at"`
	ExpiresAt   time.Time            `db:"expires_at"`
	ConsumedAt  *time.Time           `db:"consumed_at"`
	CreatedAt   time.Time            `db:"created_at"`
}

// ActionToken represents a short-lived opaque token used to authorize sensitive actions (e.g., password reset).
type ActionToken struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	Purpose   string     `db:"purpose"` // e.g., "password_reset"
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	ConsumedAt *time.Time `db:"consumed_at"`
	CreatedAt time.Time  `db:"created_at"`
}