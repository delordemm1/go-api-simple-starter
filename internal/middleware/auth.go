package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/delordemm1/go-api-simple-starter/internal/modules/user"
	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the structure of the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
}

// Authenticator is a middleware that validates a JWT and adds the user ID to the request context.
func Authenticator(jwtSecret string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get the token from the Authorization header.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// 2. Check for the "Bearer " prefix.
			tokenString, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			// 3. Parse and validate the token.
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Ensure the signing method is what we expect.
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				logger.Warn("invalid jwt token", "error", err)
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// 4. Extract the user ID (subject) from the claims.
			// userID, err := uuid.Parse(claims.Subject)
			// if err != nil {
			// 	logger.Error("invalid user id in jwt claims", "subject", claims.Subject, "error", err)
			// 	http.Error(w, "invalid token claims", http.StatusUnauthorized)
			// 	return
			// }

			// 5. Add the user ID to the request's context for downstream handlers.
			logger.Info("claims.Subject", "claims.Subject", claims.Subject)
			ctx := context.WithValue(r.Context(), user.UserIDKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
