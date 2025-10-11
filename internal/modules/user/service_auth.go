package user

import (
	"context"
	"errors"

	"github.com/delordemm1/go-api-simple-starter/internal/notification"
	"github.com/delordemm1/go-api-simple-starter/internal/notification/templates"
	"github.com/google/uuid"
)

// Register handles the business logic for creating a new user.
func (s *service) Register(ctx context.Context, firstName, lastName, email, password string) (*User, error) {
	// 1) Check if a user with the given email already exists.
	existing, err := s.repo.FindByEmail(ctx, email)
	if err == nil {
		// User exists
		if existing.EmailVerified {
			return nil, ErrEmailExists
		}
		// Re-register allowed for unverified: update names only, keep password as-is
		changed := false
		if existing.FirstName != firstName {
			existing.FirstName = firstName
			changed = true
		}
		if existing.LastName != lastName {
			existing.LastName = lastName
			changed = true
		}
		if changed {
			if uerr := s.repo.Update(ctx, existing); uerr != nil {
				s.logger.Error("failed to update unverified user names", "error", uerr, "user_id", existing.ID)
				return nil, ErrInternal.WithCause(uerr)
			}
		}

		// Generate or refresh 6-digit verification code (respect cooldown)
		code, cerr := s.createOrRefreshVerificationCode(ctx, existing, existing.Email, VerificationPurposeEmailVerify, VerificationChannelEmail)
		if cerr != nil {
			if errors.Is(cerr, ErrResendTooSoon) {
				s.logger.Info("verification code resend cooldown active", "email", email)
			} else {
				s.logger.Error("failed to create/refresh verification code", "error", cerr, "user_id", existing.ID)
			}
		} else if code != "" {
			// Fire-and-forget notification
			go func(u *User, c string) {
				data := templates.VerifyEmailData{
					FirstName:    u.FirstName,
					Code:         c,
					SupportEmail: s.config.SMTP.From,
				}
				if err := notification.SendTemplate(ctx, s.notification, templates.VerifyEmail, u.Email, []notification.Channel{notification.ChannelEmail}, notification.PriorityHigh, data); err != nil {
					s.logger.Error("failed to send verify email", "error", err, "user_id", u.ID)
				}
			}(existing, code)
		}

		s.logger.Info("user re-registered; awaiting email verification", "user_id", existing.ID)
		return existing, nil
	}
	// We expect "not found"; if it's any other error, map to internal.
	if !errors.Is(err, ErrNotFound) {
		s.logger.Error("failed to check existing user by email", "error", err)
		return nil, ErrInternal.WithCause(err)
	}

	// 2) Hash the password for security.
	hashedPassword, err := hashPassword(password)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		return nil, ErrInternal.WithCause(err)
	}

	// 3) Generate a new user ID.
	newUserID, err := uuid.NewV7()
	if err != nil {
		s.logger.Error("failed to generate user ID", "error", err)
		return nil, ErrInternal.WithCause(err)
	}

	// 4) Create the new user entity.
	newUser := &User{
		ID:            newUserID.String(),
		FirstName:     firstName,
		LastName:      lastName,
		Email:         email,
		PasswordHash:  hashedPassword,
		EmailVerified: false, // Email is not verified upon registration
	}

	// 5) Persist the user to the database.
	if err := s.repo.Create(ctx, newUser); err != nil {
		s.logger.Error("failed to create user", "error", err)
		return nil, ErrInternal.WithCause(err)
	}

	// 6) Issue a 6-digit verification code and send email
	code, cerr := s.createOrRefreshVerificationCode(ctx, newUser, newUser.Email, VerificationPurposeEmailVerify, VerificationChannelEmail)
	if cerr != nil {
		if errors.Is(cerr, ErrResendTooSoon) {
			s.logger.Info("verification code resend cooldown active (new user)", "email", email)
		} else {
			s.logger.Error("failed to create verification code for new user", "error", cerr, "user_id", newUser.ID)
		}
	} else if code != "" {
		go func(u *User, c string) {
			data := templates.VerifyEmailData{
				FirstName:    u.FirstName,
				Code:         c,
				SupportEmail: s.config.SMTP.From,
			}
			if err := notification.SendTemplate(ctx, s.notification, templates.VerifyEmail, u.Email, []notification.Channel{notification.ChannelEmail}, notification.PriorityHigh, data); err != nil {
				s.logger.Error("failed to send verify email", "error", err, "user_id", u.ID)
			}
		}(newUser, code)
	}

	s.logger.Info("user registered successfully", "user_id", newUser.ID)
	return newUser, nil
}

// Login handles the business logic for authenticating a user.
func (s *service) Login(ctx context.Context, email, password string) (string, error) {
	// 1) Find the user by their email address.
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Use a generic error to avoid telling attackers that the email exists.
			return "", ErrInvalidCredentials
		}
		s.logger.Error("failed to find user by email", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	// 2) Check if the provided password matches the stored hash.
	if !checkPasswordHash(password, user.PasswordHash) {
		return "", ErrInvalidCredentials
	}

	// 2b) Block login until email is verified
	if !user.EmailVerified {
		return "", ErrEmailNotVerified
	}

	// 3) Create an auth session and return the session ID.
	sessionID, err := s.sessions.CreateAuthSession(ctx, user.ID, "", "")
	if err != nil {
		s.logger.Error("failed to create auth session", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	s.logger.Info("user logged in successfully", "user_id", user.ID)
	return sessionID, nil
}
