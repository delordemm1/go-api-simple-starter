package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

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
	// case "apple":
	//  // Add Apple implementation here
	default:
		return nil, fmt.Errorf("unsupported oauth provider: %s", provider)
	}
}

// --- Google Provider Implementation ---

type googleProvider struct {
	config *oauth2.Config
}

func (g *googleProvider) getOAuthConfig() *oauth2.Config {
	return g.config
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
func (s *service) InitiateOAuthLogin(ctx context.Context, provider string) (redirectURL string, state string, err error) {
	oauthProvider, err := s.newOAuthProvider(provider)
	if err != nil {
		return "", "", err
	}

	// Generate a random state string for CSRF protection.
	state, err = generateSecureToken(32)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate oauth state: %w", err)
	}

	// The `oauth2.AccessTypeOffline` prompts the user for consent to get a refresh token.
	url := oauthProvider.getOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)

	return url, state, nil
}

// HandleOAuthCallback processes the callback from the OAuth provider. It verifies the state,
// exchanges the code for a token, fetches user info, finds or creates a local user,
// and returns a JWT for the session.
func (s *service) HandleOAuthCallback(ctx context.Context, provider, state, code, storedState string) (jwtToken string, err error) {
	// 1. Validate the state to prevent CSRF attacks.
	if state == "" || state != storedState {
		return "", errors.New("invalid oauth state")
	}

	oauthProvider, err := s.newOAuthProvider(provider)
	if err != nil {
		return "", err
	}

	// 2. Exchange the authorization code for an access token.
	token, err := oauthProvider.getOAuthConfig().Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange oauth code for token: %w", err)
	}

	// 3. Fetch the user's information from the provider.
	userInfo, err := oauthProvider.getUserInfo(ctx, token)
	if err != nil {
		return "", err
	}
	if userInfo.Email == "" {
		return "", errors.New("email not provided by oauth provider")
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
				return "", err
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
				return "", errors.New("failed to create user")
			}
			s.logger.Info("new user created via oauth", "user_id", newUser.ID, "email", newUser.Email)
			user = newUser
		} else {
			// Handle other database errors.
			s.logger.Error("failed to find user by email during oauth callback", "error", err)
			return "", errors.New("database error during login")
		}
	}

	// 5. Generate a JWT for the user session.
	sessionToken, err := generateJWT(user.ID)
	if err != nil {
		s.logger.Error("failed to generate JWT after oauth login", "error", err)
		return "", errors.New("failed to create session")
	}

	s.logger.Info("user logged in successfully via oauth", "provider", provider, "user_id", user.ID)

	return sessionToken, nil
}
