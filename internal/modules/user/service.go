package user

import (
	"context"
	"log/slog"

	"github.com/delordemm1/go-api-simple-starter/internal/config"
	"github.com/google/uuid"
)

// Service defines the interface for the user module's business logic.
// It orchestrates the flow of data between the handlers and the repository,
// and contains the core business rules.
type Service interface {
	// Auth-related methods
	Register(ctx context.Context, firstName, lastName, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (string, error) // Returns a token

	// Profile-related methods
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*User, error)

	// Password-related methods (placeholders for now)
	InitiatePasswordReset(ctx context.Context, email string) error
	FinalizePasswordReset(ctx context.Context, token, newPassword string) error

	// OAuth-related methods (placeholders for now)
	InitiateOAuthLogin(ctx context.Context, provider string) (redirectURL string, state string, err error)
	HandleOAuthCallback(ctx context.Context, provider, state, code, storedState string) (jwtToken string, err error)
}

// service implements the Service interface.
type service struct {
	repo   Repository
	logger *slog.Logger
	config *config.Config
	// cache redis.Client // Example of adding a cache dependency
}

// Config holds the dependencies for the user service.
type Config struct {
	Repo   Repository
	Logger *slog.Logger
	Config *config.Config
}

// NewService creates a new user service with the given dependencies.
func NewService(cfg *Config) Service {
	return &service{
		repo:   cfg.Repo,
		logger: cfg.Logger,
		config: cfg.Config,
	}
}
