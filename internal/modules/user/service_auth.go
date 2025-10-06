package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Register handles the business logic for creating a new user.
func (s *service) Register(ctx context.Context, firstName, lastName, email, password string) (*User, error) {
	// 1) Check if a user with the given email already exists.
	_, err := s.repo.FindByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
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

	// 3) Generate a JWT token for the authenticated user.
	token, err := generateJWT(user.ID)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	s.logger.Info("user logged in successfully", "user_id", user.ID)
	return token, nil
}
