package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	apphttpx "github.com/delordemm1/go-api-simple-starter/internal/httpx"
	"github.com/delordemm1/go-api-simple-starter/internal/modules/user"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTAuthHuma is a router-agnostic Huma middleware that validates a JWT and injects
// the user ID into the request context using user.UserIDKey for downstream handlers.
// On failure it writes an RFC7807 problem+json response with code ErrUnauthorized.
func JWTAuthHuma(jwtSecret string, logger *slog.Logger) huma.Middleware {
	return func(ctx huma.Context, next func(huma.Context)) {
		w, r := humachi.Unwrap(ctx)

		writeUnauthorized := func(detail string) {
			reqID := chimw.GetReqID(r.Context())
			p := &apphttpx.Problem{
				Type:      "urn:problem:auth/err-unauthorized",
				Title:     http.StatusText(http.StatusUnauthorized),
				Status:    http.StatusUnauthorized,
				Detail:    detail,
				Code:      "ErrUnauthorized",
				RequestID: reqID,
			}
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(p.GetStatus())
			_ = json.NewEncoder(w).Encode(p)
		}

		// 1. Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized("missing authorization header")
			return
		}

		// 2. Bearer token.
		tokenString, found := strings.CutPrefix(authHeader, "Bearer ")
		if !found {
			writeUnauthorized("invalid authorization header format")
			return
		}

		// 3. Parse and validate the token.
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			logger.Warn("invalid jwt token", "error", err)
			writeUnauthorized("invalid or expired token")
			return
		}

		// 4. Extract subject as UUID.
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			logger.Error("invalid user id in jwt claims", "subject", claims.Subject, "error", err)
			writeUnauthorized("invalid token claims")
			return
		}

		// 5. Inject user ID into context for downstream handlers.
		ctx = huma.WithValue(ctx, user.UserIDKey, userID)
		next(ctx)
	}
}