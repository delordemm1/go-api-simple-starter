package user

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

// This header key must match the one your SvelteKit proxy is looking for.

// --- DTOs ---

// OAuthLoginRequest defines the provider being requested from the URL path.
type OAuthLoginRequest struct {
	Provider string `path:"provider"`
}

// OAuthLoginResponse is the JSON response sent to the proxy.
type OAuthLoginResponse struct {
	Body struct {
		RedirectURL string `json:"redirectUrl"`
	}
}

// OAuthCallbackRequest defines the query parameters sent by the OAuth provider,
// which are forwarded by the proxy.
type OAuthCallbackRequest struct {
	Provider string `path:"provider"`
	Code     string `query:"code"`
	State    string `query:"state"`
}

// OAuthCallbackResponse is the JSON response for a successful callback.
type OAuthCallbackResponse struct {
	Body struct {
		Message string `json:"message"`
	}
	XSessionToken string `header:"x-session-token"`
}

// --- Handlers ---

// OAuthLoginHandler initiates the OAuth flow by returning a redirect URL to the proxy.
func (h *Handler) OAuthLoginHandler(ctx context.Context, input *OAuthLoginRequest) (*OAuthLoginResponse, error) {
	h.logger.Info("initiating oauth login", "provider", input.Provider)

	// The service now just generates the URL and a state.
	// The state must be handled by your SvelteKit server (e.g., stored in its session).
	// For this example, we assume the service doesn't need to persist the state itself.
	redirectURL, err := h.service.InitiateOAuthLogin(ctx, input.Provider)
	if err != nil {
		h.logger.Error("failed to initiate oauth login", "error", err)
		return nil, huma.Error400BadRequest("Invalid OAuth provider")
	}

	// Instead of redirecting, we return the URL in a JSON response.
	resp := &OAuthLoginResponse{}
	resp.Body.RedirectURL = redirectURL

	return resp, nil
}

// OAuthCallbackHandler handles the callback from the proxy.
// On success, it returns the JWT in a custom header for the proxy to handle.
func (h *Handler) OAuthCallbackHandler(ctx context.Context, input *OAuthCallbackRequest) (*OAuthCallbackResponse, error) {
	h.logger.Info("handling oauth callback", "provider", input.Provider)

	// Your SvelteKit server must validate the 'state' parameter against what it stored
	// before calling this backend endpoint. The Go backend now trusts the proxy.

	// Call the service to handle the logic of exchanging the code for a token and getting user info.
	// Note: The service signature needs to be updated to no longer require the 'storedState'.
	jwtToken, err := h.service.HandleOAuthCallback(ctx, input.Provider, input.State, input.Code)
	if err != nil {
		h.logger.Error("oauth callback processing failed", "error", err)
		if errors.Is(err, ErrEmailExists) {
			return nil, huma.Error409Conflict("An account with this email already exists.")
		}
		return nil, huma.Error401Unauthorized("OAuth authentication failed.")
	}

	// On success, set the custom header that the SvelteKit proxy expects.
	h.logger.Info("oauth login successful, returning token in header")

	// Return a standard 200 OK JSON response.
	resp := &OAuthCallbackResponse{}
	resp.XSessionToken = jwtToken
	resp.Body.Message = "Authentication successful."
	return resp, nil
}
