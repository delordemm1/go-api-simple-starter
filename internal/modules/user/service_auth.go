package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Register handles the business logic for creating a new user.
func (s *service) Register(ctx context.Context, firstName, lastName, email, password string) (*User, error) {
	// Check if a user with the given email already exists
	_, err := s.repo.FindByEmail(ctx, email)
	if err == nil {
		// A user was found, so the email is already taken.
		return nil, ErrEmailExists
	}
	// We expect a "not found" error, so if it's any other error, we return it.
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// Hash the password for security
	hashedPassword, err := hashPassword(password)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		return nil, errors.New("internal server error")
	}

	newUserID, err := uuid.NewV7()
	// Create the new user entity
	newUser := &User{
		ID:            newUserID.String(),
		FirstName:     firstName,
		LastName:      lastName,
		Email:         email,
		PasswordHash:  hashedPassword,
		EmailVerified: false, // Email is not verified upon registration
	}

	// Persist the user to the database
	if err := s.repo.Create(ctx, newUser); err != nil {
		s.logger.Error("failed to create user", "error", err)
		return nil, errors.New("internal server error")
	}

	s.logger.Info("user registered successfully", "user_id", newUser.ID)

	// In a real application, you would also trigger an email verification flow here.

	return newUser, nil
}

// Login handles the business logic for authenticating a user.
func (s *service) Login(ctx context.Context, email, password string) (string, error) {
	// Find the user by their email address
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Use a generic error to avoid telling attackers that the email exists.
			return "", ErrInvalidCredentials
		}
		s.logger.Error("failed to find user by email", "error", err)
		return "", errors.New("internal server error")
	}

	// Check if the provided password matches the stored hash
	if !checkPasswordHash(password, user.PasswordHash) {
		return "", ErrInvalidCredentials
	}

	// Generate a JWT token for the authenticated user
	token, err := generateJWT(user.ID)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		return "", errors.New("internal server error")
	}

	s.logger.Info("user logged in successfully", "user_id", user.ID)

	return token, nil
}
