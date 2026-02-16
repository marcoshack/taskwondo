package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/config"
	"github.com/marcoshack/trackforge/internal/database"
	"github.com/marcoshack/trackforge/internal/handler"
	"github.com/marcoshack/trackforge/internal/middleware"
	"github.com/marcoshack/trackforge/internal/repository"
	"github.com/marcoshack/trackforge/internal/service"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Configure logger
	setupLogger(cfg.LogLevel, cfg.LogFormat)

	ctx := log.Logger.WithContext(context.Background())

	log.Info().Msg("starting trackforge")

	// Connect to database
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	log.Info().Msg("connected to database")

	// Run migrations
	if err := database.Migrate(ctx, db); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	if *migrateOnly {
		log.Info().Msg("migrations complete, exiting")
		return
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, apiKeyRepo, cfg.JWTSecret, cfg.JWTExpiry)

	// Seed admin user if configured
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if err := authService.SeedAdminUser(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatal().Err(err).Msg("failed to seed admin user")
		}
	}

	// Initialize handlers
	health := handler.NewHealthHandler(db)
	auth := handler.NewAuthHandler(authService)

	// Set up router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(log.Logger))
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(cfg.BaseURL))

	// Health checks (unauthenticated)
	r.Get("/healthz", health.Healthz)
	r.Get("/readyz", health.Readyz)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Post("/auth/login", auth.Login)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			r.Post("/auth/refresh", auth.Refresh)
			r.Post("/auth/logout", auth.Logout)
			r.Get("/auth/me", auth.Me)

			// API key management
			r.Get("/user/api-keys", auth.ListAPIKeys)
			r.Post("/user/api-keys", auth.CreateAPIKey)
			r.Delete("/user/api-keys/{keyId}", auth.DeleteAPIKey)
		})
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:         cfg.ListenAddr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("http server listening")
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutting down")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server error")
		}
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server shutdown error")
	}

	log.Info().Msg("server stopped")
}

func setupLogger(level, format string) {
	// Parse level
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	// Configure output format
	if format == "text" {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(os.Stderr).
			With().Timestamp().Logger()
	}
}
