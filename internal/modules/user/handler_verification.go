package user

import (
	"context"

	"github.com/delordemm1/go-api-simple-starter/internal/httpx"
	"github.com/delordemm1/go-api-simple-starter/internal/validation"
)

// --- DTOs ---

type ResendEmailVerificationRequest struct {
	Body struct {
		Email string `json:"email" validate:"required,email"`
	}
}

type ResendEmailVerificationResponse struct{}

// ConfirmEmailVerificationRequest defines the structure for confirming an email with a 6-digit code.
type ConfirmEmailVerificationRequest struct {
	Body struct {
		Email string `json:"email" validate:"required,email"`
		Code  string `json:"code" validate:"required,len=6"`
	}
}

type ConfirmEmailVerificationResponse struct{}

// --- Handlers ---

// ResendEmailVerificationHandler triggers sending a 6-digit code for email verification.
// It enforces cooldown in the service layer and does not leak user enumeration.
func (h *Handler) ResendEmailVerificationHandler(ctx context.Context, input *ResendEmailVerificationRequest) (*ResendEmailVerificationResponse, error) {
	if verr := validation.ValidateStruct(&input.Body); verr != nil {
		return nil, httpx.ToProblem(ctx, verr)
	}

	if err := h.service.ResendEmailVerification(ctx, input.Body.Email); err != nil {
		return nil, httpx.ToProblem(ctx, err)
	}

	return &ResendEmailVerificationResponse{}, nil
}

// ConfirmEmailVerificationHandler validates the 6-digit code and marks the user's email as verified.
func (h *Handler) ConfirmEmailVerificationHandler(ctx context.Context, input *ConfirmEmailVerificationRequest) (*ConfirmEmailVerificationResponse, error) {
	if verr := validation.ValidateStruct(&input.Body); verr != nil {
		return nil, httpx.ToProblem(ctx, verr)
	}

	if err := h.service.ConfirmEmailVerification(ctx, input.Body.Email, input.Body.Code); err != nil {
		return nil, httpx.ToProblem(ctx, err)
	}

	return &ConfirmEmailVerificationResponse{}, nil
}