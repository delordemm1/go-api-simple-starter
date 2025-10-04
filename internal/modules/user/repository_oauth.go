package user

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
)

// InsertOAuthState inserts a new OAuth state record into the database.
func (r *repository) InsertOAuthState(ctx context.Context, state *OAuthState) error {
	state.CreatedAt = time.Now()
	state.UpdatedAt = time.Now()

	query, args, err := r.psql.Insert("oauth_states").
		Columns("state", "provider", "user_id", "verifier", "expires_at", "created_at", "updated_at").
		Values(state.State, state.Provider, state.UserID, state.Verifier, state.ExpiresAt, state.CreatedAt, state.UpdatedAt).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

// GetOAuthStateByState retrieves an OAuth state record by its state string.
func (r *repository) GetOAuthStateByState(ctx context.Context, state string) (*OAuthState, error) {
	query, args, err := r.psql.Select("*").
		From("oauth_states").
		Where(squirrel.Eq{"state": state}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}

	var oauthState OAuthState
	err = pgxscan.Get(ctx, r.db, &oauthState, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound.WithCause(err)
		}
		return nil, err
	}

	return &oauthState, nil
}

// UpdateOAuthStateUserID updates the user_id field of an OAuth state.
// This is used to link an OAuth flow to a specific user after authentication.
func (r *repository) UpdateOAuthStateUserID(ctx context.Context, state string, userID string) (*OAuthState, error) {
	query, args, err := r.psql.Update("oauth_states").
		Set("user_id", userID).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"state": state}).
		ToSql()
	if err != nil {
		return nil, err
	}

	cmdTag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	if cmdTag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	// Retrieve and return the updated OAuth state
	return r.GetOAuthStateByState(ctx, state)
}

// DeleteOAuthState removes an OAuth state record from the database.
func (r *repository) DeleteOAuthState(ctx context.Context, state string) error {
	query, args, err := r.psql.Delete("oauth_states").
		Where(squirrel.Eq{"state": state}).
		ToSql()
	if err != nil {
		return err
	}

	cmdTag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteExpiredOAuthStates removes all OAuth state records that have expired.
// This should be called periodically as a cleanup operation.
func (r *repository) DeleteExpiredOAuthStates(ctx context.Context) error {
	query, args, err := r.psql.Delete("oauth_states").
		Where(squirrel.Lt{"expires_at": time.Now()}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Note: We don't return an error if no rows were deleted,
	// as it's normal for there to be no expired states to clean up.
	return nil
}
