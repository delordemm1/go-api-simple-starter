package user

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

// --- DTOs ---

// ForgotPasswordRequest defines the structure for initiating a password reset.
type ForgotPasswordRequest struct {
	Body struct {
		Email string `json:"email" validate:"required,email"`
	}
}

// ForgotPasswordResponse is an empty successful response.
type ForgotPasswordResponse struct{}

// ResetPasswordRequest defines the structure for finalizing a password reset.
type ResetPasswordRequest struct {
	Body struct {
		Token           string `json:"token" validate:"required"`
		Password        string `json:"password" validate:"required,min=8"`
		ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password"`
	}
}

// ResetPasswordResponse is an empty successful response.
type ResetPasswordResponse struct{}

// --- Handlers ---

// ForgotPasswordHandler handles the request to initiate a password reset.
func (h *Handler) ForgotPasswordHandler(ctx context.Context, input *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	h.logger.Info("handling forgot password request", "email", input.Body.Email)

	err := h.service.InitiatePasswordReset(ctx, input.Body.Email)
	if err != nil {
		// IMPORTANT: To prevent email enumeration attacks, we do not reveal if the
		// email was found or not. We log the real error for debugging but return
		// a generic success response to the client in all cases. The actual email
		// sending happens in the service layer, and if that fails, it's a system issue.
		h.logger.Error("failed to initiate password reset", "email", input.Body.Email, "error", err)
	}

	// Always return a successful-looking response.
	return &ForgotPasswordResponse{}, nil
}

// ResetPasswordHandler handles the request to set a new password using a reset token.
func (h *Handler) ResetPasswordHandler(ctx context.Context, input *ResetPasswordRequest) (*ResetPasswordResponse, error) {
	h.logger.Info("handling reset password request")

	err := h.service.FinalizePasswordReset(ctx, input.Body.Token, input.Body.Password)
	if err != nil {
		h.logger.Warn("failed to reset password", "error", err)
		if errors.Is(err, ErrInvalidResetToken) {
			return nil, huma.Error400BadRequest("the provided token is invalid or has expired")
		}
		// For any other error, return a generic internal server error.
		return nil, huma.Error500InternalServerError("could not reset password")
	}

	h.logger.Info("password reset successfully")
	return &ResetPasswordResponse{}, nil
}
