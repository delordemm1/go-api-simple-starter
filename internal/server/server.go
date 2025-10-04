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
func New(cfg *config.Config, log *slog.Logger, userService *user.Service) chi.Router {
	// Create a new Chi router and Huma API.
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("Go API Starter", "1.0.0"))

	// Add standard middleware.
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger) // Chi's built-in logger, can be replaced with a custom slog one.
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))
	userHandler := user.NewHandler(*userService, log)
	userHandler.RegisterRoutes(api)
	// Create the main server struct.
	// srv := &Server{
	// 	Router: router,
	// 	log:    log,
	// 	config: cfg,
	// }

	// // Create the underlying http.Server.
	// srv.server = &http.Server{
	// 	Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
	// 	Handler: router,
	// 	// Good practice to set timeouts to avoid Slowloris attacks.
	// 	WriteTimeout: 15 * time.Second,
	// 	ReadTimeout:  15 * time.Second,
	// 	IdleTimeout:  60 * time.Second,
	// }

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

// Start runs the HTTP server and handles graceful shutdown.
// func (s *Server) Start(ctx context.Context) error {
// 	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
// 	defer stop()

// 	// Start the server in a separate goroutine.
// 	go func() {
// 		s.log.Info("server starting", "address", s.server.Addr)
// 		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
// 			s.log.Error("server failed to start", "error", err)
// 			stop() // Trigger shutdown on start failure.
// 		}
// 	}()

// 	// Wait for the shutdown signal.
// 	<-shutdownCtx.Done()

// 	s.log.Info("shutdown signal received, starting graceful shutdown")

// 	// Create a context with a timeout for the shutdown process.
// 	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Attempt to gracefully shut down the server.
// 	if err := s.server.Shutdown(timeoutCtx); err != nil {
// 		s.log.Error("graceful shutdown failed", "error", err)
// 		return err
// 	}

// 	s.log.Info("server stopped gracefully")
// 	return nil
// }
