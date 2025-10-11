package user

import (
	"context"
	"log/slog"

	"github.com/delordemm1/go-api-simple-starter/internal/config"
	"github.com/delordemm1/go-api-simple-starter/internal/notification"
	"github.com/delordemm1/go-api-simple-starter/internal/session"
)

// Service defines the interface for the user module's business logic.
// It orchestrates the flow of data between the handlers and the repository,
// and contains the core business rules.
type Service interface {
	// Auth-related methods
	Register(ctx context.Context, firstName, lastName, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (string, error) // Returns a session ID

	// Profile-related methods
	GetProfile(ctx context.Context, userID string) (*User, error)
	UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*User, error)

	// Email verification (6-digit code)
	ResendEmailVerification(ctx context.Context, email string) error
	ConfirmEmailVerification(ctx context.Context, email, code string) error

	// Password reset (6-digit code + internal reset token)
	InitiatePasswordReset(ctx context.Context, email string) error
	VerifyPasswordResetCode(ctx context.Context, email, code string) (resetToken string, err error)
	FinalizePasswordReset(ctx context.Context, resetToken, newPassword string) error

	// OAuth-related methods
	InitiateOAuthLogin(ctx context.Context, provider OAuthProvider) (redirectURL string, err error)
	HandleOAuthCallback(ctx context.Context, provider OAuthProvider, state, code string) (sessionID string, err error)
}

// service implements the Service interface.
type service struct {
	repo         Repository
	logger       *slog.Logger
	config       *config.Config
	sessions     session.Provider
	notification notification.Service
	// cache redis.Client // Example of adding a cache dependency
}

// Config holds the dependencies for the user service.
type Config struct {
	Repo         Repository
	Logger       *slog.Logger
	Config       *config.Config
	Sessions     session.Provider
	Notification notification.Service
}

// NewService creates a new user service with the given dependencies.
func NewService(cfg *Config) Service {
	return &service{
		repo:         cfg.Repo,
		logger:       cfg.Logger,
		config:       cfg.Config,
		sessions:     cfg.Sessions,
		notification: cfg.Notification,
	}
}
