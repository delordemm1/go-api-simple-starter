package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// UpdateProfileInput defines the updatable fields for a user's profile.
// Using pointers allows us to distinguish between a field not being provided (nil)
// and a field being set to its zero value (e.g., an empty string).
type UpdateProfileInput struct {
	FirstName *string
	LastName  *string
}

// GetProfile retrieves a single user's profile by their ID.
func (s *service) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound.WithCause(err)
		}
		s.logger.Error("failed to get user profile from repository", "error", err, "user_id", userID)
		return nil, errors.New("internal server error")
	}
	return user, nil
}

// UpdateProfile updates a user's profile information.
func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*User, error) {
	// 1. Retrieve the existing user to ensure they exist and to apply changes.
	user, err := s.repo.FindByID(ctx, userID.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound.WithCause(err)
		}
		s.logger.Error("failed to find user for profile update", "error", err, "user_id", userID)
		return nil, errors.New("internal server error")
	}

	// 2. Apply updates from the input struct.
	if input.FirstName != nil {
		user.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		user.LastName = *input.LastName
	}

	// 3. Set the updated timestamp.
	user.UpdatedAt = time.Now()

	// 4. Persist the changes to the database.
	// NOTE: This requires the repository to have a general `Update` method.
	if err := s.repo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update user profile in repository", "error", err, "user_id", userID)
		return nil, errors.New("internal server error")
	}

	s.logger.Info("user profile updated successfully", "user_id", user.ID)

	return user, nil
}
