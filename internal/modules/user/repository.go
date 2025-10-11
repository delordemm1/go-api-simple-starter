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
	// Users
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error

	// Password (legacy token fields retained but not used in new 6-digit flow)
	UpdatePassword(ctx context.Context, userID string, newPasswordHash string) error
	FindByPasswordResetToken(ctx context.Context, tokenHash string) (*User, error)
	UpdatePasswordResetInfo(ctx context.Context, userID string, tokenHash string, expiry time.Time) error

	// Verification codes (6-digit OTP)
	CreateVerificationCode(ctx context.Context, vc *VerificationCode) error
	GetActiveVerificationCodeByContact(ctx context.Context, contact string, purpose VerificationPurpose, channel VerificationChannel) (*VerificationCode, error)
	GetActiveVerificationCodeByUser(ctx context.Context, userID string, purpose VerificationPurpose, channel VerificationChannel) (*VerificationCode, error)
	UpdateVerificationCodeForResend(ctx context.Context, id string, newCodeHash string, newExpiresAt time.Time, lastSentAt time.Time, maxAttempts int) error
	IncrementVerificationAttempt(ctx context.Context, id string) (attempts int, maxAttempts int, err error)
	ConsumeVerificationCode(ctx context.Context, id string) error

	// Internal action tokens (e.g., password reset)
	CreateActionToken(ctx context.Context, t *ActionToken) error
	FindActionTokenByHash(ctx context.Context, tokenHash string, purpose string) (*ActionToken, error)
	ConsumeActionToken(ctx context.Context, id string) error
	DeleteUserActionTokensByPurpose(ctx context.Context, userID string, purpose string) error

	// Session/token (auth sessions)
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
