package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/delordemm1/go-api-simple-starter/internal/config"
	"github.com/delordemm1/go-api-simple-starter/internal/modules/user"
	"github.com/delordemm1/go-api-simple-starter/internal/session"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the dependencies for the HTTP server.
// type Server struct {
// 	chi.Router
// 	server *http.Server
// 	log    *slog.Logger
// 	config *config.Config
// }

// New creates and configures a new server instance.
func New(cfg *config.Config, log *slog.Logger, userService user.Service, sessions session.Provider) chi.Router {
	// Create a new Chi router and Huma API.
	router := chi.NewMux()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger) // Chi's built-in logger, can be replaced with a custom slog one.
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))
	apiConfig := huma.DefaultConfig("Go API Starter", "1.0.0")
	apiConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "Opaque",
		},
	}
	api := humachi.New(router, apiConfig)

	// Add standard middleware.
	userHandler := user.NewHandler(userService, log, sessions)
	userHandler.RegisterRoutes(api)

	// Register a simple health check endpoint.
	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health Check",
		Description: "Responds with the server's health status.",
	}, func(ctx context.Context, input *struct{}) (*struct {
		Body struct {
			Status string `json:"status"`
		}
	}, error) {
		resp := &struct {
			Body struct {
				Status string `json:"status"`
			}
		}{}
		resp.Body.Status = "ok"
		return resp, nil
	})

	return router
}
