package user

import (
	"context"

	"github.com/delordemm1/go-api-simple-starter/internal/httpx"
	"github.com/delordemm1/go-api-simple-starter/internal/validation"
)

// --- DTOs (Data Transfer Objects) ---

// RegisterRequest defines the structure for the user registration request body.
type RegisterRequest struct {
	Body struct {
		FirstName       string `json:"firstName" validate:"required,min=2"`
		LastName        string `json:"lastName" validate:"required,min=2"`
		Email           string `json:"email" validate:"required,email"`
		Password        string `json:"password" validate:"required,min=8"`
		ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password"`
		AcceptTerms     bool   `json:"acceptTerms" validate:"required,eq=true"`
	}
}

// RegisterResponse defines the structure for a successful registration response.
type RegisterResponse struct {
	Body struct {
		ID        string `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	}
}

// LoginRequest defines the structure for the user login request body.
type LoginRequest struct {
	Body struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
}

// LoginResponse defines the structure for a successful login response.
type LoginResponse struct {
	Body struct {
		Token string `json:"token"`
	}
}

// --- Mapper ---

// toRegisterResponse converts a domain User object to a RegisterResponse DTO.
func toRegisterResponse(user *User) *RegisterResponse {

	return &RegisterResponse{
		Body: struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Email     string `json:"email"`
		}{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
		},
	}
}

// --- Handlers ---

// RegisterHandler handles the user registration endpoint.
func (h *Handler) RegisterHandler(ctx context.Context, input *RegisterRequest) (*RegisterResponse, error) {
	h.logger.Info("handling user registration request", "email", input.Body.Email)
	if verr := validation.ValidateStruct(&input.Body); verr != nil {
		return nil, httpx.ToProblem(ctx, verr)
	}

	user, err := h.service.Register(ctx, input.Body.FirstName, input.Body.LastName, input.Body.Email, input.Body.Password)
	if err != nil {
		h.logger.Error("registration failed", "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	h.logger.Info("user registered successfully", "user_id", user.ID)
	return toRegisterResponse(user), nil
}

// LoginHandler handles the user login endpoint.
func (h *Handler) LoginHandler(ctx context.Context, input *LoginRequest) (*LoginResponse, error) {
	h.logger.Info("handling user login request", "email", input.Body.Email)
	if verr := validation.ValidateStruct(&input.Body); verr != nil {
		return nil, httpx.ToProblem(ctx, verr)
	}

	token, err := h.service.Login(ctx, input.Body.Email, input.Body.Password)
	if err != nil {
		h.logger.Warn("login attempt failed", "email", input.Body.Email, "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	h.logger.Info("user logged in successfully", "email", input.Body.Email)
	return &LoginResponse{
		Body: struct {
			Token string `json:"token"`
		}{
			Token: token,
		},
	}, nil
}
