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
		SessionToken string `json:"sessionToken"`
	}
}

// --- Handlers ---

// OAuthLoginHandler initiates the OAuth flow by returning a redirect URL to the proxy.
func (h *Handler) OAuthLoginHandler(ctx context.Context, input *OAuthLoginRequest) (*OAuthLoginResponse, error) {
	h.logger.Info("initiating oauth login", "provider", input.Provider)

	redirectURL, err := h.service.InitiateOAuthLogin(ctx, OAuthProvider(input.Provider))
	if err != nil {
		h.logger.Error("failed to initiate oauth login", "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	resp := &OAuthLoginResponse{}
	resp.Body.RedirectURL = redirectURL

	return resp, nil
}

// OAuthCallbackHandler handles the callback from the proxy.
// On success, it returns the session token in a custom header for the proxy to handle.
func (h *Handler) OAuthCallbackHandler(ctx context.Context, input *OAuthCallbackRequest) (*OAuthCallbackResponse, error) {
	h.logger.Info("handling oauth callback", "provider", input.Provider)

	sessionToken, err := h.service.HandleOAuthCallback(ctx, OAuthProvider(input.Provider), input.State, input.Code)
	if err != nil {
		h.logger.Error("oauth callback processing failed", "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	h.logger.Info("oauth login successful, returning session token in header")

	resp := &OAuthCallbackResponse{}
	resp.Body.SessionToken = sessionToken
	return resp, nil
}


// OAuthCallbackPostRequest supports both form_post (Apple) and optional JSON via proxy.
type OAuthCallbackPostRequest struct {
	Provider string `path:"provider"`
	// Form-encoded fields from Apple (response_mode=form_post)
	Code  string `form:"code"`
	State string `form:"state"`
	// Optional JSON body if a proxy forwards as JSON
	Body struct {
		Code  string `json:"code"`
		State string `json:"state"`
		User  string `json:"user,omitempty"`
	}
}

// OAuthCallbackPostHandler handles POST callbacks (Apple form_post or JSON proxy).
func (h *Handler) OAuthCallbackPostHandler(ctx context.Context, input *OAuthCallbackPostRequest) (*OAuthCallbackResponse, error) {
	h.logger.Info("handling oauth callback (POST)", "provider", input.Provider)

	code := input.Code
	state := input.State
	if code == "" {
		code = input.Body.Code
	}
	if state == "" {
		state = input.Body.State
	}

	sessionToken, err := h.service.HandleOAuthCallback(ctx, OAuthProvider(input.Provider), state, code)
	if err != nil {
		h.logger.Error("oauth callback processing failed (POST)", "error", err)
		return nil, httpx.ToProblem(ctx, err)
	}

	resp := &OAuthCallbackResponse{}
	resp.Body.SessionToken = sessionToken
	return resp, nil
}