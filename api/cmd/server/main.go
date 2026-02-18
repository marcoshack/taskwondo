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
	"github.com/marcoshack/trackforge/internal/storage"
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
	projectRepo := repository.NewProjectRepository(db)
	projectMemberRepo := repository.NewProjectMemberRepository(db)
	workItemRepo := repository.NewWorkItemRepository(db)
	workItemEventRepo := repository.NewWorkItemEventRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	relationRepo := repository.NewWorkItemRelationRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	queueRepo := repository.NewQueueRepository(db)
	milestoneRepo := repository.NewMilestoneRepository(db)
	userSettingRepo := repository.NewUserSettingRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)

	// Initialize storage
	store, err := storage.NewMinIOStorage(
		cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey,
		cfg.StorageBucket, cfg.StorageRegion, cfg.StorageUseSSL,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage")
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, apiKeyRepo, cfg.JWTSecret, cfg.JWTExpiry)
	projectService := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, workflowRepo)
	workflowService := service.NewWorkflowService(workflowRepo)
	queueService := service.NewQueueService(queueRepo, projectRepo, projectMemberRepo)
	milestoneService := service.NewMilestoneService(milestoneRepo, projectRepo, projectMemberRepo)
	workItemService := service.NewWorkItemService(workItemRepo, workItemEventRepo, commentRepo, relationRepo, attachmentRepo, projectRepo, projectMemberRepo, workflowRepo, queueRepo, milestoneRepo, store, cfg.MaxUploadSize)
	userSettingService := service.NewUserSettingService(userSettingRepo, projectRepo, projectMemberRepo)

	// Seed admin user if configured
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if err := authService.SeedAdminUser(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatal().Err(err).Msg("failed to seed admin user")
		}
	}

	// Seed default workflows
	if err := workflowService.SeedDefaultWorkflows(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to seed default workflows")
	}

	// Initialize handlers
	health := handler.NewHealthHandler(db)
	auth := handler.NewAuthHandler(authService)
	projects := handler.NewProjectHandler(projectService)
	workflows := handler.NewWorkflowHandler(workflowService)
	queues := handler.NewQueueHandler(queueService)
	milestones := handler.NewMilestoneHandler(milestoneService)
	items := handler.NewWorkItemHandler(workItemService, cfg.MaxUploadSize)
	userSettings := handler.NewUserSettingHandler(userSettingService)

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

			// User search
			r.Get("/users/search", auth.SearchUsers)

			// Global user preferences
			r.Route("/user/preferences", func(r chi.Router) {
				r.Get("/", userSettings.ListGlobal)
				r.Route("/{key}", func(r chi.Router) {
					r.Get("/", userSettings.GetGlobal)
					r.Put("/", userSettings.SetGlobal)
					r.Delete("/", userSettings.DeleteGlobal)
				})
			})

			// Workflows
			r.Route("/workflows", func(r chi.Router) {
				r.Get("/", workflows.List)
				r.Post("/", workflows.Create)
				r.Route("/{workflowId}", func(r chi.Router) {
					r.Get("/", workflows.Get)
					r.Patch("/", workflows.Update)
					r.Get("/transitions", workflows.ListTransitions)
				})
			})

			// Projects
			r.Route("/projects", func(r chi.Router) {
				r.Get("/", projects.List)
				r.Post("/", projects.Create)
				r.Route("/{projectKey}", func(r chi.Router) {
					r.Get("/", projects.Get)
					r.Patch("/", projects.Update)
					r.Delete("/", projects.Delete)
					r.Route("/members", func(r chi.Router) {
						r.Get("/", projects.ListMembers)
						r.Post("/", projects.AddMember)
						r.Patch("/{userId}", projects.UpdateMemberRole)
						r.Delete("/{userId}", projects.RemoveMember)
					})
					r.Route("/queues", func(r chi.Router) {
						r.Get("/", queues.List)
						r.Post("/", queues.Create)
						r.Route("/{queueId}", func(r chi.Router) {
							r.Get("/", queues.Get)
							r.Patch("/", queues.Update)
							r.Delete("/", queues.Delete)
						})
					})
					r.Route("/milestones", func(r chi.Router) {
						r.Get("/", milestones.List)
						r.Post("/", milestones.Create)
						r.Route("/{milestoneId}", func(r chi.Router) {
							r.Get("/", milestones.Get)
							r.Patch("/", milestones.Update)
							r.Delete("/", milestones.Delete)
						})
					})
					r.Route("/user-settings", func(r chi.Router) {
						r.Get("/", userSettings.List)
						r.Route("/{key}", func(r chi.Router) {
							r.Get("/", userSettings.Get)
							r.Put("/", userSettings.Set)
							r.Delete("/", userSettings.Delete)
						})
					})
					r.Route("/items", func(r chi.Router) {
						r.Get("/", items.List)
						r.Post("/", items.Create)
						r.Route("/{itemNumber}", func(r chi.Router) {
							r.Get("/", items.Get)
							r.Patch("/", items.Update)
							r.Delete("/", items.Delete)
							r.Route("/comments", func(r chi.Router) {
								r.Get("/", items.ListComments)
								r.Post("/", items.CreateComment)
								r.Patch("/{commentId}", items.UpdateComment)
								r.Delete("/{commentId}", items.DeleteComment)
							})
							r.Route("/relations", func(r chi.Router) {
								r.Get("/", items.ListRelations)
								r.Post("/", items.CreateRelation)
								r.Delete("/{relationId}", items.DeleteRelation)
							})
							r.Route("/attachments", func(r chi.Router) {
								r.Get("/", items.ListAttachments)
								r.Post("/", items.UploadAttachment)
								r.Get("/{attachmentId}", items.DownloadAttachment)
								r.Delete("/{attachmentId}", items.DeleteAttachment)
							})
							r.Get("/events", items.ListEvents)
						})
					})
				})
			})
		})
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:         cfg.ListenAddr(),
		Handler:      r,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
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
