package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/delordemm1/go-api-simple-starter/internal/cache"
	"github.com/delordemm1/go-api-simple-starter/internal/config"
	"github.com/delordemm1/go-api-simple-starter/internal/database"
	"github.com/delordemm1/go-api-simple-starter/internal/modules/user"
	"github.com/delordemm1/go-api-simple-starter/internal/server"
)

// Options for the CLI.
type Options struct {
	Port int `help:"Port to listen on" short:"p"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		// Use a structured logger
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		cfg := config.Load()
		if cfg == nil {
			logger.Error("failed to load configuration")
			os.Exit(1)
		}
		logger.Info("configuration loaded successfully", "env", cfg)

		// --- Database & Cache ---
		dbPool := database.NewPostgresPool(cfg.Database.URL)
		if dbPool == nil {
			logger.Error("failed to connect to postgres")
			os.Exit(1)
		}
		hooks.OnStop(dbPool.Close)
		logger.Info("successfully connected to postgres database")
		redisClient := cache.NewRedisClient(cfg.Redis.URL)
		if redisClient == nil {
			logger.Error("failed to connect to redis")
			os.Exit(1)
		}
		hooks.OnStop(func() { redisClient.Close() })
		logger.Info("successfully connected to redis")

		// --- Module Initialization (Bottom-Up) ---

		// User Module
		userRepo := user.NewRepository(dbPool)
		userService := user.NewService(&user.Config{
			Repo:   userRepo,
			Logger: logger,
			Config: cfg,
		})
		router := server.New(cfg, logger, &userService)
		hooks.OnStart(func() {
			// Determine port: CLI -p overrides, else cfg.Server.Port, else 8080
			port := options.Port
			if port <= 0 {
				if cfg.Server.Port != "" {
					if p, err := strconv.Atoi(cfg.Server.Port); err == nil {
						port = p
						logger.Info("using port from config", "port", port)
					} else {
						logger.Warn("invalid port in config, falling back to default", "cfgPort", cfg.Server.Port)
					}
				}
			} else {
				logger.Info("using port from CLI", "port", port)
			}
			if port <= 0 {
				port = 8080
				logger.Info("using default port", "port", port)
			}

			logger.Info(fmt.Sprintf("Starting server on port %d...", port))
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), router); err != nil {
				slog.Error("Server failed to start", "error", err)
				os.Exit(1)
			}
		})
	})
	cli.Run()
}

// func xmain() {
// 	// Use a structured logger
// 	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// 	// --- Configuration ---
// 	cfg := config.Load()
// 	if cfg == nil {
// 		logger.Error("failed to load configuration")
// 		os.Exit(1)
// 	}
// 	logger.Info("configuration loaded successfully", "env", cfg.Server.Env)

// 	// --- Database & Cache ---
// 	dbPool := database.NewPostgresPool(cfg.Database.URL)
// 	if dbPool == nil {
// 		logger.Error("failed to connect to postgres")
// 		os.Exit(1)
// 	}
// 	defer dbPool.Close()
// 	logger.Info("successfully connected to postgres database")

// 	redisClient := cache.NewRedisClient(cfg.Redis.URL)
// 	if redisClient == nil {
// 		logger.Error("failed to connect to redis")
// 		os.Exit(1)
// 	}
// 	defer redisClient.Close()
// 	logger.Info("successfully connected to redis")

// 	// --- Module Initialization (Bottom-Up) ---

// 	// User Module
// 	userRepo := user.NewRepository(dbPool)
// 	userService := user.NewService(&user.Config{
// 		Repo:   userRepo,
// 		Logger: logger,
// 		Config: cfg,
// 	})
// 	userHandler := user.NewHandler(userService, logger)

// 	// --- Server Setup ---
// 	srv := server.New(cfg, logger)

// 	// Register module routes
// 	userHandler.RegisterRoutes(srv.Router)

// 	// --- Start Server ---
// 	ctx := context.Background()
// 	if err := srv.Start(ctx); err != nil {
// 		logger.Error("server failed to start", "error", err)
// 		os.Exit(1)
// 	}
// }
