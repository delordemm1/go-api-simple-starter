package user

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/delordemm1/go-api-simple-starter/internal/notification"
	"github.com/delordemm1/go-api-simple-starter/internal/notification/templates"
)

// ResendEmailVerification generates or refreshes a 6-digit code for email verification and sends it.
// It enforces resend cooldown and hides user enumeration by returning nil when the email is unknown or already verified.
func (s *service) ResendEmailVerification(ctx context.Context, email string) error {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Hide enumeration
			return nil
		}
		s.logger.Error("resend verify: find user failed", "error", err)
		return ErrInternal.WithCause(err)
	}
	if user.EmailVerified {
		// Already verified, treat as success
		return nil
	}

	code, err := s.createOrRefreshVerificationCode(ctx, user, user.Email, VerificationPurposeEmailVerify, VerificationChannelEmail)
	if err != nil {
		return err
	}

	// Fire-and-forget notification
	go func() {
		data := templates.VerifyEmailData{
			FirstName:    user.FirstName,
			Code:         code,
			SupportEmail: s.config.SMTP.From,
		}
		if err := notification.SendTemplate(ctx, s.notification, templates.VerifyEmail, user.Email, []notification.Channel{notification.ChannelEmail}, notification.PriorityHigh, data); err != nil {
			s.logger.Error("failed to send verify email", "error", err, "user_id", user.ID)
		}
	}()
	return nil
}

// ConfirmEmailVerification validates a 6-digit code, marks the user's email as verified, and consumes the code.
func (s *service) ConfirmEmailVerification(ctx context.Context, email, code string) error {
	if strings.TrimSpace(code) == "" {
		return ErrInvalidOTP
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Avoid enumeration
			return ErrInvalidOTP
		}
		s.logger.Error("confirm verify: find user failed", "error", err)
		return ErrInternal.WithCause(err)
	}

	if user.EmailVerified {
		// Already verified - idempotent success
		return nil
	}

	vc, err := s.repo.GetActiveVerificationCodeByUser(ctx, user.ID, VerificationPurposeEmailVerify, VerificationChannelEmail)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidOTP
		}
		s.logger.Error("confirm verify: get active code failed", "error", err)
		return ErrInternal.WithCause(err)
	}

	// TTL check
	if time.Now().After(vc.ExpiresAt) {
		return ErrInvalidOTP
	}

	// Compare hash in constant time
	hashed := hashToken(code)
	if subtle.ConstantTimeCompare([]byte(hashed), []byte(vc.CodeHash)) != 1 {
		attempts, max, incErr := s.repo.IncrementVerificationAttempt(ctx, vc.ID)
		if incErr != nil && !errors.Is(incErr, ErrNotFound) {
			s.logger.Error("confirm verify: increment attempts failed", "error", incErr)
			return ErrInternal.WithCause(incErr)
		}
		if attempts >= max {
			return ErrTooManyAttempts
		}
		return ErrInvalidOTP
	}

	// Success: consume the code and mark verified
	if err := s.repo.ConsumeVerificationCode(ctx, vc.ID); err != nil && !errors.Is(err, ErrNotFound) {
		s.logger.Error("confirm verify: consume code failed", "error", err)
		return ErrInternal.WithCause(err)
	}
	user.EmailVerified = true
	if err := s.repo.Update(ctx, user); err != nil {
		s.logger.Error("confirm verify: update user failed", "error", err)
		return ErrInternal.WithCause(err)
	}

	return nil
}

// createOrRefreshVerificationCode enforces cooldown and returns the plaintext code (never stored).
func (s *service) createOrRefreshVerificationCode(ctx context.Context, user *User, contact string, purpose VerificationPurpose, channel VerificationChannel) (string, error) {
	ttlMinutes := s.config.Verification.TTLMinutes
	if ttlMinutes <= 0 {
		ttlMinutes = 10
	}
	resendCooldownSecs := s.config.Verification.ResendCooldownSeconds
	if resendCooldownSecs <= 0 {
		resendCooldownSecs = 60
	}
	maxAttempts := s.config.Verification.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	// Prefer user scoped active code
	var active *VerificationCode
	var err error
	if user != nil {
		active, err = s.repo.GetActiveVerificationCodeByUser(ctx, user.ID, purpose, channel)
		if err != nil && !errors.Is(err, ErrNotFound) {
			s.logger.Error("createOrRefresh: get active by user failed", "error", err)
			return "", ErrInternal.WithCause(err)
		}
	}
	if active == nil {
		active, err = s.repo.GetActiveVerificationCodeByContact(ctx, contact, purpose, channel)
		if err != nil && !errors.Is(err, ErrNotFound) {
			s.logger.Error("createOrRefresh: get active by contact failed", "error", err)
			return "", ErrInternal.WithCause(err)
		}
	}

	now := time.Now()
	// Cooldown check if there is an active code
	if active != nil && now.Sub(active.LastSentAt) < time.Duration(resendCooldownSecs)*time.Second {
		return "", ErrResendTooSoon
	}

	// Generate new 6-digit code
	code, genErr := generateNumericCode(6)
	if genErr != nil {
		s.logger.Error("createOrRefresh: generate code failed", "error", genErr)
		return "", ErrInternal.WithCause(genErr)
	}
	hash := hashToken(code)
	expiresAt := now.Add(time.Duration(ttlMinutes) * time.Minute)

	if active != nil {
		// Refresh existing record: reset attempts, update hash, expiry, last_sent_at, max_attempts
		if err := s.repo.UpdateVerificationCodeForResend(ctx, active.ID, hash, expiresAt, now, maxAttempts); err != nil {
			if !errors.Is(err, ErrNotFound) {
				s.logger.Error("createOrRefresh: update for resend failed", "error", err)
				return "", ErrInternal.WithCause(err)
			}
			// Fall through to create new if race marked it consumed
		} else {
			return code, nil
		}
	}

	// Create a new verification code
	var uid *string
	if user != nil {
		uid = &user.ID
	}
	vc := &VerificationCode{
		UserID:      uid,
		Contact:     contact,
		Purpose:     purpose,
		Channel:     channel,
		CodeHash:    hash,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		LastSentAt:  now,
		ExpiresAt:   expiresAt,
		ConsumedAt:  nil,
		CreatedAt:   now,
	}
	if err := s.repo.CreateVerificationCode(ctx, vc); err != nil {
		s.logger.Error("createOrRefresh: create code failed", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	return code, nil
}