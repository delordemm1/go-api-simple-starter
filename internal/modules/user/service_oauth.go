package user

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// --- OAuth Provider Abstraction ---

// oAuthUserInfo holds the standardized user information extracted from a provider.
type oAuthUserInfo struct {
	ID    string
	Email string
	Name  string
}

// OAuthProvider defines the interface for an OAuth provider like Google or Apple.
type OAuth interface {
	getOAuthConfig() *oauth2.Config
	getUserInfo(ctx context.Context, token *oauth2.Token) (*oAuthUserInfo, error)
}

func parseApplePrivateKey(key string) (*ecdsa.PrivateKey, error) {
	// Replace the literal '\n' characters with actual newlines
	formattedKey := strings.ReplaceAll(key, "\\n", "\n")
	return jwt.ParseECPrivateKeyFromPEM([]byte(formattedKey))
}

// newOAuthProvider is a factory function that returns the correct provider implementation.
func (s *service) newOAuthProvider(provider string) (OAuth, error) {
	switch provider {
	case "google":
		return &googleProvider{
			config: &oauth2.Config{
				ClientID:     s.config.Google.ClientID,
				ClientSecret: s.config.Google.ClientSecret,
				RedirectURL:  s.config.Google.RedirectURL,
				Endpoint:     google.Endpoint,
				Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			},
		}, nil
	case "apple":
		privateKey, err := parseApplePrivateKey(s.config.Apple.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse apple private key: %w", err)
		}

		return &appleProvider{
			config: &oauth2.Config{
				ClientID:     s.config.Apple.ClientID,
				ClientSecret: "",
				RedirectURL:  s.config.Apple.RedirectURL,
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://appleid.apple.com/auth/authorize",
					TokenURL: "https://appleid.apple.com/auth/token",
				},
				Scopes: []string{"name", "email"},
			},
			teamID: s.config.Apple.TeamID,
			keyID:  s.config.Apple.KeyID,
			prvKey: privateKey,
		}, nil
	default:
		return nil, ErrUnsupportedOAuthProvider.WithDetail(fmt.Sprintf("unsupported oauth provider: %s", provider))
	}
}

// --- Google Provider Implementation ---

type googleProvider struct {
	config *oauth2.Config
}
type appleProvider struct {
	config *oauth2.Config
	teamID string
	keyID  string
	prvKey *ecdsa.PrivateKey
}

func (g *googleProvider) getOAuthConfig() *oauth2.Config {
	return g.config
}
func (a *appleProvider) getOAuthConfig() *oauth2.Config {
	return a.config
}

// Apple's user info is not fetched from a separate endpoint.
// It's encoded in the ID Token that comes back in the token exchange.
func (a *appleProvider) getUserInfo(ctx context.Context, token *oauth2.Token) (*oAuthUserInfo, error) {
	// 1. Extract the id_token from the token response.
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, errors.New("id_token not found in apple oauth token")
	}

	// 2. Parse the JWT without verification, as we trust the source (Apple's token endpoint).
	// For higher security, you could verify the token's signature against Apple's public key.
	var claims struct {
		jwt.RegisteredClaims
		Email string `json:"email"`
	}

	// The parser needs a key function, but we're skipping verification for this step.
	_, _, err := jwt.NewParser().ParseUnverified(idToken, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse apple id_token: %w", err)
	}

	// 3. The unique user ID is in the 'Subject' claim.
	if claims.Subject == "" {
		return nil, errors.New("subject (user id) claim missing from apple id_token")
	}

	// Note: Apple only sends the user's name on the VERY FIRST login.
	// Your application MUST save it then, as it won't be sent again.
	// The name is not part of the id_token; it's sent as a separate `user` form parameter
	// to your redirect URI, which is not handled by the standard oauth2.Exchange.
	// The best practice is to ask the user for their name on the next screen if it's missing.

	return &oAuthUserInfo{
		ID:    claims.Subject, // This is the stable unique identifier for the user.
		Email: claims.Email,
		Name:  "", // Name must be handled separately (see note above).
	}, nil
}

// generateAppleClientSecret creates the JWT used as the client_secret.
func (a *appleProvider) generateAppleClientSecret() (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer:    a.teamID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		Audience:  jwt.ClaimStrings{"https://appleid.apple.com"},
		Subject:   a.config.ClientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = a.keyID

	return token.SignedString(a.prvKey)
}

func (g *googleProvider) getUserInfo(ctx context.Context, token *oauth2.Token) (*oAuthUserInfo, error) {
	client := g.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info from google: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response body: %w", err)
	}

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &oAuthUserInfo{
		ID:    userInfo.ID,
		Email: userInfo.Email,
		Name:  userInfo.Name,
	}, nil
}

// --- Main Service Methods ---

// InitiateOAuthLogin generates the redirect URL and a state for CSRF protection.
// The handler is responsible for storing the state (e.g., in a secure, short-lived cookie).
func (s *service) InitiateOAuthLogin(ctx context.Context, provider OAuthProvider) (redirectURL string, err error) {
	oauthProvider, err := s.newOAuthProvider((string(provider)))
	if err != nil {
		return "", err
	}

	// Generate a random state string for CSRF protection.
	state, err := generateSecureToken(32)
	if err != nil {
		return "", ErrInternal.WithCause(fmt.Errorf("failed to generate oauth state: %w", err))
	}
	verifier := oauth2.GenerateVerifier()
	err = s.repo.InsertOAuthState(ctx, &OAuthState{
		Verifier:  verifier,
		State:     state,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		UpdatedAt: time.Now(),
		Provider:  provider,
	})
	if err != nil {
		return "", ErrInternal.WithCause(fmt.Errorf("failed to generate oauth state: %w", err))
	}

	// The `oauth2.AccessTypeOffline` prompts the user for consent to get a refresh token.
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	}
	// Apple requires response_mode=form_post when requesting name/email scopes.
	if provider == "apple" {
		opts = append(opts,
			oauth2.SetAuthURLParam("response_mode", "form_post"),
			oauth2.SetAuthURLParam("response_type", "code"),
		)
	}
	url := oauthProvider.getOAuthConfig().AuthCodeURL(state, opts...)

	return url, nil
}

// HandleOAuthCallback processes the callback from the OAuth provider. It verifies the state,
// exchanges the code for a token, fetches user info, finds or creates a local user,
// and returns a session ID.
func (s *service) HandleOAuthCallback(ctx context.Context, provider OAuthProvider, state, code string) (sessionID string, err error) {
	oauthProvider, err := s.newOAuthProvider(string(provider))
	if err != nil {
		return "", err
	}

	token, err := s.repo.GetOAuthStateByState(ctx, state)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.Error("oauth state not found", "state", state, "error", err)
			return "", ErrOAuthStateInvalid.WithCause(err)
		}
		s.logger.Error("error getting oauth state", "error", err)
		return "", ErrInternal.WithCause(err)
	}
	if time.Now().After(token.ExpiresAt) {
		s.logger.Error("oauth state expired", "state", state)
		return "", ErrOAuthStateExpired
	}
	defer s.repo.DeleteOAuthState(ctx, state)

	var exchangeOptions []oauth2.AuthCodeOption
	exchangeOptions = append(exchangeOptions, oauth2.VerifierOption(token.Verifier))

	if provider == "apple" {
		appleP, ok := oauthProvider.(*appleProvider)
		if !ok {
			return "", ErrInternal.WithDetail("provider is not a valid apple provider")
		}

		clientSecret, err := appleP.generateAppleClientSecret()
		if err != nil {
			return "", ErrInternal.WithCause(fmt.Errorf("failed to generate apple client secret: %w", err))
		}
		exchangeOptions = append(exchangeOptions, oauth2.SetAuthURLParam("client_secret", clientSecret))
	}

	// Exchange the authorization code for an access token.
	oauthToken, err := oauthProvider.getOAuthConfig().Exchange(ctx, code, exchangeOptions...)
	if err != nil {
		return "", ErrOAuthExchangeFailed.WithCause(fmt.Errorf("failed to exchange oauth code for token: %w", err))
	}

	// 3. Fetch the user's information from the provider.
	userInfo, err := oauthProvider.getUserInfo(ctx, oauthToken)
	if err != nil {
		return "", ErrOAuthExchangeFailed.WithCause(err)
	}
	if userInfo.Email == "" {
		return "", ErrOAuthEmailMissing
	}

	// 4. Find or create the user in the local database.
	user, err := s.repo.FindByEmail(ctx, userInfo.Email)
	firstName, lastName := "", ""
	nameParts := strings.SplitN(userInfo.Name, " ", 2)
	if len(nameParts) > 0 {
		firstName = nameParts[0]
	}
	if len(nameParts) > 1 {
		lastName = nameParts[1]
	}
	if err != nil {
		// If the user doesn't exist, create a new one (provisioning).
		if errors.Is(err, ErrNotFound) {
			id, err := uuid.NewV7()
			if err != nil {
				return "", ErrInternal.WithCause(err)
			}
			newUser := &User{
				ID:            id.String(),
				Email:         userInfo.Email,
				FirstName:     firstName,
				LastName:      lastName,
				EmailVerified: true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			if err := s.repo.Create(ctx, newUser); err != nil {
				s.logger.Error("failed to create new user from oauth", "error", err)
				return "", ErrInternal.WithCause(err)
			}
			s.logger.Info("new user created via oauth", "user_id", newUser.ID, "email", newUser.Email)
			user = newUser
		} else {
			// Handle other database errors.
			s.logger.Error("failed to find user by email during oauth callback", "error", err)
			return "", ErrInternal.WithCause(err)
		}
	}

	// 5. Create a session for the user.
	sessionID, err = s.sessions.CreateAuthSession(ctx, user.ID, "", "")
	if err != nil {
		s.logger.Error("failed to create auth session after oauth login", "error", err)
		return "", ErrInternal.WithCause(err)
	}

	s.logger.Info("user logged in successfully via oauth", "provider", provider, "user_id", user.ID)

	return sessionID, nil
}
