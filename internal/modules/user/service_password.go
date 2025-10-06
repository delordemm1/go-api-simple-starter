package user

import (
	"context"
	"errors"
	"time"
)

// InitiatePasswordReset handles the logic for initiating a password reset.
// It finds a user by email, generates a secure reset token, stores the token's hash,
// and simulates sending a password reset email.
func (s *service) InitiatePasswordReset(ctx context.Context, email string) error {
	// 1. Find user by email.
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// If the user is not found, we return nil to prevent email enumeration.
		// An attacker should not be able to determine if an email is registered.
		if errors.Is(err, ErrNotFound) {
			s.logger.Info("password reset requested for non-existent email", "email", email)
			return nil
		}
		// For other errors, log and return a generic domain error.
		s.logger.Error("failed to find user by email for password reset", "error", err)
		return ErrInternal.WithCause(err)
	}

	// 2. Generate a secure, unique password reset token.
	token, err := generateSecureToken(32)
	if err != nil {
		s.logger.Error("failed to generate secure token for password reset", "error", err)
		return ErrInternal.WithCause(err)
	}

	// 3. Hash the token before storing it in the database for security.
	tokenHash := hashToken(token)

	// 4. Set an expiry time for the token (e.g., 15 minutes from now).
	expiryTime := time.Now().Add(15 * time.Minute)

	// 5. Update the user record in the database with the token hash and expiry.
	// NOTE: This requires the repository to have an `UpdatePasswordResetInfo` method.
	err = s.repo.UpdatePasswordResetInfo(ctx, user.ID, tokenHash, expiryTime)
	if err != nil {
		s.logger.Error("failed to update user with password reset token", "error", err)
		return ErrInternal.WithCause(err)
	}

	// 6. Send the password reset email to the user.
	// This is a placeholder. A real implementation would use an email service client.
	s.logger.Info("Simulating sending password reset email", "email", user.Email, "raw_token_for_dev", token)
	// Example: emailService.SendPasswordResetEmail(user.Email, token)

	return nil
}

// ResetPassword validates a password reset token and updates the user's password.
func (s *service) FinalizePasswordReset(ctx context.Context, token, newPassword string) error {
	// 1. Hash the provided raw token to find its matching hash in the database.
	if token == "" {
		return ErrInvalidResetToken
	}
	tokenHash := hashToken(token)

	// 2. Find the user by the hashed password reset token.
	// NOTE: This requires the repository to have a `FindByPasswordResetToken` method.
	user, err := s.repo.FindByPasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Use the same error for not found, expired, or invalid tokens.
			return ErrInvalidResetToken
		}
		s.logger.Error("failed to find user by password reset token", "error", err)
		return ErrInternal.WithCause(err)
	}

	// 3. Check if the token has expired.
	// NOTE: This assumes the User entity has a `PasswordResetTokenExpiry *time.Time` field.
	if user.PasswordResetTokenExpiry == nil || time.Now().After(*user.PasswordResetTokenExpiry) {
		return ErrInvalidResetToken
	}

	// 4. Hash the new password.
	newPasswordHash, err := hashPassword(newPassword)
	if err != nil {
		s.logger.Error("failed to hash new password during reset", "error", err)
		return ErrInternal.WithCause(err)
	}

	// 5. Update the user's password and clear the reset token fields.
	// NOTE: This requires the repository's UpdatePassword method to also clear the token fields.
	err = s.repo.UpdatePassword(ctx, user.ID, newPasswordHash)
	if err != nil {
		s.logger.Error("failed to update user password after reset", "error", err)
		return ErrInternal.WithCause(err)
	}

	s.logger.Info("user password has been reset successfully", "user_id", user.ID)

	return nil
}
