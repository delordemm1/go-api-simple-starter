package user

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/delordemm1/go-api-simple-starter/internal/database"
)

// Repository defines the interface for database operations for the user module.
// This abstraction allows the service layer to be independent of the database implementation.
type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error

	UpdatePassword(ctx context.Context, userID string, newPasswordHash string) error
	FindByPasswordResetToken(ctx context.Context, tokenHash string) (*User, error)
	UpdatePasswordResetInfo(ctx context.Context, userID string, tokenHash string, expiry time.Time) error

	// Session/token
	CreateUserActiveSession(ctx context.Context, sess *UserActiveSession) error
	UpdateUserActiveSessionTimestamp(ctx context.Context, sessionToken string) error
	DeleteSessionByToken(ctx context.Context, sessionToken string) error

	// Oauth states (for social login)
	InsertOAuthState(ctx context.Context, state *OAuthState) error
	GetOAuthStateByState(ctx context.Context, state string) (*OAuthState, error)
	UpdateOAuthStateUserID(ctx context.Context, state string, userID string) (*OAuthState, error)
	DeleteOAuthState(ctx context.Context, state string) error
	DeleteExpiredOAuthStates(ctx context.Context) error
}

// repository implements the Repository interface using pgx and squirrel.
type repository struct {
	db   database.DBTX
	psql squirrel.StatementBuilderType
}

// NewRepository creates a new user repository with the given database connection.
func NewRepository(db database.DBTX) Repository {
	return &repository{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
