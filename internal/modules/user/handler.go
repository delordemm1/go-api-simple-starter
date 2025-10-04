package user

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Handler holds the dependencies for the user module's HTTP handlers.
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler creates a new handler for the user module.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes sets up the routing for the user module.
// It defines all the API endpoints and connects them to their respective handler functions.
func (h *Handler) RegisterRoutes(api huma.API) {
	// --- Authentication Routes ---
	huma.Register(api, huma.Operation{
		Method:  http.MethodPost,
		Path:    "/users/register",
		Summary: "Register a new user",
	}, h.RegisterHandler)

	huma.Register(api, huma.Operation{
		Method:  http.MethodPost,
		Path:    "/users/login",
		Summary: "Log in a user",
	}, h.LoginHandler)

	// --- Password Management Routes ---
	huma.Register(api, huma.Operation{
		Method:  http.MethodPost,
		Path:    "/users/password/forgot",
		Summary: "Initiate password reset",
	}, h.ForgotPasswordHandler)

	huma.Register(api, huma.Operation{
		Method:  http.MethodPost,
		Path:    "/users/password/reset",
		Summary: "Reset password with a token",
	}, h.ResetPasswordHandler)

	// --- OAuth Routes ---
	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/users/oauth/{provider}",
		Summary: "Initiate OAuth login",
	}, h.OAuthLoginHandler)

	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/users/oauth/{provider}/callback",
		Summary: "Handle OAuth callback",
	}, h.OAuthCallbackHandler)

	// --- Profile Routes (requires authentication middleware) ---
	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/users/profile",
		Summary: "Get the current user's profile",
		// Security: []huma.SecurityRequirement{{ID: "BearerAuth"}}, // Example of protected route
	}, h.GetProfileHandler)

	huma.Register(api, huma.Operation{
		Method:  http.MethodPatch,
		Path:    "/users/profile",
		Summary: "Update the current user's profile",
		// Security: []huma.SecurityRequirement{{ID: "BearerAuth"}}, // Example of protected route
	}, h.UpdateProfileHandler)
}
