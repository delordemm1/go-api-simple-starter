package user

import (
	"context"
	"time"

	"github.com/delordemm1/go-api-simple-starter/internal/contextx"
	"github.com/delordemm1/go-api-simple-starter/internal/httpx"
	"github.com/delordemm1/go-api-simple-starter/internal/validation"
)

// --- Context Key ---
// Now provided by internal/contextx to avoid import cycles and ensure consistency.

// --- DTOs & Mappers ---

// ProfileResponse is the DTO for a user's public profile.
type ProfileResponse struct {
	Body struct {
		ID        string    `json:"id"`
		FirstName string    `json:"firstName"`
		LastName  string    `json:"lastName"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"createdAt"`
	}
}

// toProfileResponse maps a domain User object to a ProfileResponse DTO.
func toProfileResponse(user *User) *ProfileResponse {
	var resp ProfileResponse
	resp.Body.ID = user.ID
	resp.Body.FirstName = user.FirstName
	resp.Body.LastName = user.LastName
	resp.Body.Email = user.Email
	resp.Body.CreatedAt = user.CreatedAt
	return &resp
}

// UpdateProfileRequest defines the fields that can be updated on a user's profile.
type UpdateProfileRequest struct {
	Body struct {
		FirstName string `json:"firstName" validate:"required,min=2"`
		LastName  string `json:"lastName" validate:"required,min=2"`
	}
}

// --- Handlers ---

// GetProfileHandler retrieves the profile of the currently authenticated user.
// It relies on an authentication middleware to have set the user's ID in the context.
func (h *Handler) GetProfileHandler(ctx context.Context, input *struct{}) (*ProfileResponse, error) {
	// Extract user ID from the context, which is set by the auth middleware.
	userIDVal := ctx.Value(contextx.UserIDKey)
	h.logger.Info("userIDVal", "userIDVal", userIDVal)
	userID, ok := userIDVal.(string)
	if !ok {
		h.logger.Error("user ID not found in context or is of wrong type")
		// This indicates a misconfiguration in the middleware chain.
		return nil, httpx.ToProblem(ctx, ErrUnauthorized.WithDetail("invalid authentication context"))
	}

	h.logger.Info("handling get profile request", "user_id", userID)

	user, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user profile", "user_id", userID, "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	return toProfileResponse(user), nil
}

// UpdateProfileHandler updates the profile of the currently authenticated user.
func (h *Handler) UpdateProfileHandler(ctx context.Context, input *UpdateProfileRequest) (*ProfileResponse, error) {
	userIDVal := ctx.Value(contextx.UserIDKey)
	userID, ok := userIDVal.(string)
	if !ok {
		h.logger.Error("user ID not found in context for update profile")
		return nil, httpx.ToProblem(ctx, ErrUnauthorized.WithDetail("invalid authentication context"))
	}

	// Validate request body
	if verr := validation.ValidateStruct(&input.Body); verr != nil {
		return nil, httpx.ToProblem(ctx, verr)
	}

	h.logger.Info("handling update profile request", "user_id", userID)

	updatedUser, err := h.service.UpdateProfile(ctx, userID, UpdateProfileInput{FirstName: &input.Body.FirstName, LastName: &input.Body.LastName})
	if err != nil {
		h.logger.Error("failed to update user profile", "user_id", userID, "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	h.logger.Info("profile updated successfully", "user_id", userID)
	return toProfileResponse(updatedUser), nil
}
