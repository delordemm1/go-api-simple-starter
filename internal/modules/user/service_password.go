package user

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/delordemm1/go-api-simple-starter/internal/notification"
	"github.com/delordemm1/go-api-simple-starter/internal/notification/templates"
)

// InitiatePasswordReset sends a 6-digit reset code to the user's email if it exists.
// Always returns nil to avoid email enumeration.
func (s *service) InitiatePasswordReset(ctx context.Context, email string) error {
	// 1. Find user by email.
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// Hide enumeration
		if errors.Is(err, ErrNotFound) {
			s.logger.Info("password reset requested for non-existent email", "email", email)
			return nil
		}
		s.logger.Error("failed to find user by email for password reset", "error", err)
		return ErrInternal.WithCause(err)
	}

	// 2. Create or refresh a 6-digit code with TTL & cooldown.
	code, err := s.createOrRefreshVerificationCode(ctx, user, user.Email, VerificationPurposePasswordReset, VerificationChannelEmail)
	if err != nil {
		if errors.Is(err, ErrResendTooSoon) {
			// Surface rate-limit error to let client throttle (still does not reveal existence)
			return err
		}
		return err
	}

	// 3. Send via templates.
	go func() {
		data := templates.PasswordResetCodeData{
			FirstName:    user.FirstName,
			Code:         code,
			SupportEmail: s.config.SMTP.From,
		}
		if err := notification.SendTemplate(ctx, s.notification, templates.PasswordResetCode, user.Email, []notification.Channel{notification.ChannelEmail}, notification.PriorityHigh, data); err != nil {
			s.logger.Error("failed to send password reset code", "error", err, "user_id", user.ID)
		}
	}()

	return nil
}

// VerifyPasswordResetCode validates the 6-digit code and issues a short-lived internal reset token.
// The raw token is returned to the client; only its hash is stored.
func (s *service) VerifyPasswordResetCode(ctx context.Context, email, code string) (string, error) {
	if code == "" {
		return "", ErrInvalidOTP
	}

	// 1) Lookup user
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrInvalidOTP
		}
		s.logger.Error("verify reset code: find user failed", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	// 2) Fetch active verification code
	vc, err := s.repo.GetActiveVerificationCodeByUser(ctx, user.ID, VerificationPurposePasswordReset, VerificationChannelEmail)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrInvalidOTP
		}
		s.logger.Error("verify reset code: get active code failed", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	// 3) TTL check
	if time.Now().After(vc.ExpiresAt) {
		return "", ErrInvalidOTP
	}

	// 4) Constant-time compare
	hashed := hashToken(code)
	if subtle.ConstantTimeCompare([]byte(hashed), []byte(vc.CodeHash)) != 1 {
		attempts, max, incErr := s.repo.IncrementVerificationAttempt(ctx, vc.ID)
		if incErr != nil && !errors.Is(incErr, ErrNotFound) {
			s.logger.Error("verify reset code: increment attempts failed", "error", incErr)
			return "", ErrInternal.WithCause(incErr)
		}
		if attempts >= max {
			return "", ErrTooManyAttempts
		}
		return "", ErrInvalidOTP
	}

	// 5) Consume the code
	if err := s.repo.ConsumeVerificationCode(ctx, vc.ID); err != nil && !errors.Is(err, ErrNotFound) {
		s.logger.Error("verify reset code: consume code failed", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	// 6) Issue internal action token (short-lived)
	rawToken, err := generateSecureToken(32)
	if err != nil {
		s.logger.Error("verify reset code: generate action token failed", "error", err)
		return "", ErrInternal.WithCause(err)
	}
	tokenHash := hashToken(rawToken)

	ttlMin := s.config.ResetToken.TTLMinutes
	if ttlMin <= 0 {
		ttlMin = 15
	}
	expiresAt := time.Now().Add(time.Duration(ttlMin) * time.Minute)

	// Ensure only one active token per user/purpose
	if err := s.repo.DeleteUserActionTokensByPurpose(ctx, user.ID, "password_reset"); err != nil {
		s.logger.Warn("verify reset code: cleanup old action tokens failed", "error", err)
	}

	at := &ActionToken{
		UserID:    user.ID,
		Purpose:   "password_reset",
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		ConsumedAt: nil,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateActionToken(ctx, at); err != nil {
		s.logger.Error("verify reset code: create action token failed", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	return rawToken, nil
}

// FinalizePasswordReset accepts an internal reset token and the new password.
// It validates and consumes the token, then updates the user's password.
func (s *service) FinalizePasswordReset(ctx context.Context, resetToken, newPassword string) error {
	if resetToken == "" {
		return ErrInvalidResetToken
	}

	// Hash provided token
	tokenHash := hashToken(resetToken)

	// Find action token
	at, err := s.repo.FindActionTokenByHash(ctx, tokenHash, "password_reset")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidResetToken
		}
		s.logger.Error("finalize reset: find token failed", "error", err)
		return ErrInternal.WithCause(err)
	}

	// Expiry check
	if time.Now().After(at.ExpiresAt) {
		return ErrInvalidResetToken
	}

	// Hash new password
	newPasswordHash, err := hashPassword(newPassword)
	if err != nil {
		s.logger.Error("finalize reset: hash password failed", "error", err)
		return ErrInternal.WithCause(err)
	}

	// Update password
	if err := s.repo.UpdatePassword(ctx, at.UserID, newPasswordHash); err != nil {
		s.logger.Error("finalize reset: update password failed", "error", err)
		return ErrInternal.WithCause(err)
	}

	// Consume the action token
	if err := s.repo.ConsumeActionToken(ctx, at.ID); err != nil && !errors.Is(err, ErrNotFound) {
		s.logger.Warn("finalize reset: consume action token failed", "error", err)
	}

	s.logger.Info("user password has been reset successfully", "user_id", at.UserID)
	return nil
}