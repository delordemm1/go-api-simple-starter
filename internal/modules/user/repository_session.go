package user

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// CreateUserActiveSession inserts a new user session record into the database.
func (r *repository) CreateUserActiveSession(ctx context.Context, sess *UserActiveSession) error {
	sess.CreatedAt = time.Now()
	sess.LastActiveAt = time.Now()

	query, args, err := r.psql.Insert("user_active_sessions").
		Columns("id", "user_id", "session_token", "user_agent", "ip_address", "last_active_at", "created_at").
		Values(sess.ID, sess.UserID, sess.SessionToken, sess.UserAgent, sess.IpAddress, sess.LastActiveAt, sess.CreatedAt).
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

// UpdateUserActiveSessionTimestamp updates the last_active_at timestamp for a session.
func (r *repository) UpdateUserActiveSessionTimestamp(ctx context.Context, sessionToken string) error {
	query, args, err := r.psql.Update("user_active_sessions").
		Set("last_active_at", time.Now()).
		Where(squirrel.Eq{"session_token": sessionToken}).
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

// DeleteSessionByToken removes a session from the database by its token.
func (r *repository) DeleteSessionByToken(ctx context.Context, sessionToken string) error {
	query, args, err := r.psql.Delete("user_active_sessions").
		Where(squirrel.Eq{"session_token": sessionToken}).
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
