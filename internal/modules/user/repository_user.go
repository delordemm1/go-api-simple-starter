package user

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
)

// Create inserts a new user record into the database.
func (r *repository) Create(ctx context.Context, user *User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query, args, err := r.psql.Insert("users").
		Columns("id", "firstName", "lastName", "email", "password_hash", "email_verified", "created_at", "updated_at").
		Values(user.ID, user.FirstName, user.LastName, user.Email, user.PasswordHash, user.EmailVerified, user.CreatedAt, user.UpdatedAt).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		// Here you might check for a unique constraint violation and return a domain-specific error
		return err
	}

	return nil
}

// FindByEmail retrieves a user by their email address.
// It returns ErrNotFound if no user is found.
func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query, args, err := r.psql.Select("*").
		From("users").
		Where(squirrel.Eq{"email": email}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}

	var user User
	err = pgxscan.Get(ctx, r.db, &user, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound.WithCause(err)
		}
		return nil, err
	}

	return &user, nil
}

// FindByID retrieves a user by their unique ID.
// It returns ErrNotFound if no user is found.
func (r *repository) FindByID(ctx context.Context, id string) (*User, error) {
	query, args, err := r.psql.Select("*").
		From("users").
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}

	var user User
	err = pgxscan.Get(ctx, r.db, &user, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound.WithCause(err)
		}
		return nil, err
	}

	return &user, nil
}

// Update modifies an existing user's details in the database.
func (r *repository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	query, args, err := r.psql.Update("users").
		Set("firstName", user.FirstName).
		Set("lastName", user.LastName).
		Set("email", user.Email).
		Set("password_hash", user.PasswordHash).
		Set("email_verified", user.EmailVerified).
		Set("updated_at", user.UpdatedAt).
		Where(squirrel.Eq{"id": user.ID}).
		ToSql()
	if err != nil {
		return err
	}

	ct, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdatePasswordResetInfo stores the hashed reset token and its expiry for a given user.
func (r *repository) UpdatePasswordResetInfo(ctx context.Context, userID string, tokenHash string, expiry time.Time) error {
	sql, args, err := r.psql.Update("users").
		Set("password_reset_token", tokenHash).
		Set("password_reset_token_expiry", expiry).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return err
	}

	cmdTag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePassword sets a new password hash for a user and clears any password reset tokens.
func (r *repository) UpdatePassword(ctx context.Context, userID string, newPasswordHash string) error {
	sql, args, err := r.psql.Update("users").
		Set("password_hash", newPasswordHash).
		Set("password_reset_token", nil).
		Set("password_reset_token_expiry", nil).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return err
	}

	cmdTag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByPasswordResetToken finds a user by their hashed password reset token.
func (r *repository) FindByPasswordResetToken(ctx context.Context, tokenHash string) (*User, error) {
	return r.findOne(ctx, squirrel.Eq{"password_reset_token": tokenHash})
}

// findOne is a helper method to find a single user by a given condition.
func (r *repository) findOne(ctx context.Context, condition squirrel.Sqlizer) (*User, error) {
	sql, args, err := r.psql.Select(
		"id", "firstName", "lastName", "email", "password_hash", "is_active",
		"password_reset_token", "password_reset_token_expiry",
		"created_at", "updated_at",
	).From("users").Where(condition).Limit(1).ToSql()

	if err != nil {
		return nil, err
	}

	user := &User{}
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&user.ID, &user.FirstName, user.LastName, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.PasswordResetToken, &user.PasswordResetTokenExpiry,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return user, nil
}
