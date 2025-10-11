package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/delordemm1/go-api-simple-starter/internal/contextx"
	apphttpx "github.com/delordemm1/go-api-simple-starter/internal/httpx"
	"github.com/delordemm1/go-api-simple-starter/internal/session"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// JWTAuthHuma (now session-based) is a router-agnostic Huma middleware that validates
// an opaque Bearer session ID, injects the user ID and session ID into the context,
// and extends the session TTL. On failure, it writes an RFC7807 problem+json response.
func JWTAuthHuma(provider session.Provider, logger *slog.Logger) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		r, w := humachi.Unwrap(ctx)

		writeUnauthorized := func(detail string) {
			reqID := chimw.GetReqID(r.Context())
			p := &apphttpx.Problem{
				Type:      "urn:problem:auth/err-unauthorized",
				Title:     http.StatusText(http.StatusUnauthorized),
				Status:    http.StatusUnauthorized,
				Detail:    detail,
				Code:      "ErrUnauthorized",
				RequestID: reqID,
				Message:   detail, // alias to support {code,message,data}
			}
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(p.GetStatus())
			_ = json.NewEncoder(w).Encode(p)
		}

		// 1) Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized("missing authorization header")
			return
		}

		// 2) Expect Bearer & opaque session ID
		sessionID, found := strings.CutPrefix(authHeader, "Bearer ")
		if !found || strings.TrimSpace(sessionID) == "" {
			writeUnauthorized("invalid authorization header format")
			return
		}

		// 3) Validate session & extend sliding TTL
		userID, err := provider.GetAndExtend(r.Context(), sessionID)
		if err != nil {
			logger.Warn("invalid session", "error", err)
			writeUnauthorized("invalid or expired session")
			return
		}

		// 4) Inject into context for downstream handlers
		ctx = huma.WithValue(ctx, contextx.UserIDKey, userID)
		ctx = huma.WithValue(ctx, contextx.SessionIDKey, sessionID)

		// 5) Continue
		next(ctx)
	}
}