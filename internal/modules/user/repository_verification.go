package user

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- Verification Codes (6-digit OTP) ---

func (r *repository) CreateVerificationCode(ctx context.Context, vc *VerificationCode) error {
	if vc.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		vc.ID = id.String()
	}
	now := time.Now()
	if vc.CreatedAt.IsZero() {
		vc.CreatedAt = now
	}
	if vc.LastSentAt.IsZero() {
		vc.LastSentAt = now
	}

	sql, args, err := r.psql.Insert("verification_codes").
		Columns("id", "user_id", "contact", "purpose", "channel", "code_hash", "attempts", "max_attempts", "last_sent_at", "expires_at", "consumed_at", "created_at").
		Values(vc.ID, vc.UserID, vc.Contact, string(vc.Purpose), string(vc.Channel), vc.CodeHash, vc.Attempts, vc.MaxAttempts, vc.LastSentAt, vc.ExpiresAt, vc.ConsumedAt, vc.CreatedAt).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *repository) GetActiveVerificationCodeByContact(ctx context.Context, contact string, purpose VerificationPurpose, channel VerificationChannel) (*VerificationCode, error) {
	sql, args, err := r.psql.Select(
		"id", "user_id", "contact", "purpose", "channel", "code_hash", "attempts", "max_attempts", "last_sent_at", "expires_at", "consumed_at", "created_at",
	).From("verification_codes").
		Where(squirrel.Eq{"contact": contact, "purpose": string(purpose), "channel": string(channel)}).
		Where("consumed_at IS NULL").
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}
	var vc VerificationCode
	if err := pgxscan.Get(ctx, r.db, &vc, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound.WithCause(err)
		}
		return nil, err
	}
	return &vc, nil
}

func (r *repository) GetActiveVerificationCodeByUser(ctx context.Context, userID string, purpose VerificationPurpose, channel VerificationChannel) (*VerificationCode, error) {
	sql, args, err := r.psql.Select(
		"id", "user_id", "contact", "purpose", "channel", "code_hash", "attempts", "max_attempts", "last_sent_at", "expires_at", "consumed_at", "created_at",
	).From("verification_codes").
		Where(squirrel.Eq{"user_id": userID, "purpose": string(purpose), "channel": string(channel)}).
		Where("consumed_at IS NULL").
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}
	var vc VerificationCode
	if err := pgxscan.Get(ctx, r.db, &vc, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound.WithCause(err)
		}
		return nil, err
	}
	return &vc, nil
}

func (r *repository) UpdateVerificationCodeForResend(ctx context.Context, id string, newCodeHash string, newExpiresAt time.Time, lastSentAt time.Time, maxAttempts int) error {
	sql, args, err := r.psql.Update("verification_codes").
		Set("code_hash", newCodeHash).
		Set("expires_at", newExpiresAt).
		Set("last_sent_at", lastSentAt).
		Set("attempts", 0).
		Set("max_attempts", maxAttempts).
		Where(squirrel.Eq{"id": id}).
		Where("consumed_at IS NULL").
		ToSql()
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) IncrementVerificationAttempt(ctx context.Context, id string) (int, int, error) {
	sql := `
        UPDATE verification_codes
        SET attempts = attempts + 1
        WHERE id = $1 AND consumed_at IS NULL
        RETURNING attempts, max_attempts
    `
	var attempts int
	var maxAttempts int
	if err := r.db.QueryRow(ctx, sql, id).Scan(&attempts, &maxAttempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrNotFound.WithCause(err)
		}
		return 0, 0, err
	}
	return attempts, maxAttempts, nil
}

func (r *repository) ConsumeVerificationCode(ctx context.Context, id string) error {
	sql, args, err := r.psql.Update("verification_codes").
		Set("consumed_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		Where("consumed_at IS NULL").
		ToSql()
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Internal Action Tokens (short-lived tokens, e.g., password reset) ---

func (r *repository) CreateActionToken(ctx context.Context, t *ActionToken) error {
	if t.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		t.ID = id.String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}

	sql, args, err := r.psql.Insert("action_tokens").
		Columns("id", "user_id", "purpose", "token_hash", "expires_at", "consumed_at", "created_at").
		Values(t.ID, t.UserID, t.Purpose, t.TokenHash, t.ExpiresAt, t.ConsumedAt, t.CreatedAt).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *repository) FindActionTokenByHash(ctx context.Context, tokenHash string, purpose string) (*ActionToken, error) {
	sql, args, err := r.psql.Select("id", "user_id", "purpose", "token_hash", "expires_at", "consumed_at", "created_at").
		From("action_tokens").
		Where(squirrel.Eq{"token_hash": tokenHash, "purpose": purpose}).
		Where("consumed_at IS NULL").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, err
	}
	var t ActionToken
	if err := pgxscan.Get(ctx, r.db, &t, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound.WithCause(err)
		}
		return nil, err
	}
	return &t, nil
}

func (r *repository) ConsumeActionToken(ctx context.Context, id string) error {
	sql, args, err := r.psql.Update("action_tokens").
		Set("consumed_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		Where("consumed_at IS NULL").
		ToSql()
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) DeleteUserActionTokensByPurpose(ctx context.Context, userID string, purpose string) error {
	sql, args, err := r.psql.Delete("action_tokens").
		Where(squirrel.Eq{"user_id": userID, "purpose": purpose}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}