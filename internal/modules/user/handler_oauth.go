package user

import (
	"context"

	"github.com/delordemm1/go-api-simple-starter/internal/httpx"
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

	redirectURL, err := h.service.InitiateOAuthLogin(ctx, input.Provider)
	if err != nil {
		h.logger.Error("failed to initiate oauth login", "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	resp := &OAuthLoginResponse{}
	resp.Body.RedirectURL = redirectURL

	return resp, nil
}

// OAuthCallbackHandler handles the callback from the proxy.
// On success, it returns the JWT in a custom header for the proxy to handle.
func (h *Handler) OAuthCallbackHandler(ctx context.Context, input *OAuthCallbackRequest) (*OAuthCallbackResponse, error) {
	h.logger.Info("handling oauth callback", "provider", input.Provider)

	jwtToken, err := h.service.HandleOAuthCallback(ctx, input.Provider, input.State, input.Code)
	if err != nil {
		h.logger.Error("oauth callback processing failed", "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	h.logger.Info("oauth login successful, returning token in header")

	resp := &OAuthCallbackResponse{}
	resp.XSessionToken = jwtToken
	resp.Body.Message = "Authentication successful."
	return resp, nil
}
