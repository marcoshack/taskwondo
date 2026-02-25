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
	"golang.org/x/time/rate"

	"github.com/marcoshack/taskwondo/internal/config"
	"github.com/marcoshack/taskwondo/internal/database"
	"github.com/marcoshack/taskwondo/internal/dataport"
	"github.com/marcoshack/taskwondo/internal/handler"
	"github.com/marcoshack/taskwondo/internal/middleware"
	"github.com/marcoshack/taskwondo/internal/repository"
	"github.com/marcoshack/taskwondo/internal/service"
	"github.com/marcoshack/taskwondo/internal/storage"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit")
	exportPath := flag.String("export", "", "Export all data to the specified tar.gz file and exit")
	importPath := flag.String("import", "", "Import data from the specified tar.gz file and exit")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Configure logger
	setupLogger(cfg.LogLevel, cfg.LogFormat)

	ctx := log.Logger.WithContext(context.Background())

	log.Info().Msg("starting taskwondo")

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

	// Export/import modes: initialize storage and run, then exit.
	if *exportPath != "" || *importPath != "" {
		store, err := storage.NewMinIOStorage(
			cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey,
			cfg.StorageBucket, cfg.StorageRegion, cfg.StorageUseSSL,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize storage")
		}

		if *exportPath != "" {
			exporter := dataport.NewExporter(db, store)
			f, err := os.Create(*exportPath)
			if err != nil {
				log.Fatal().Err(err).Str("path", *exportPath).Msg("failed to create export file")
			}
			if err := exporter.Export(ctx, f); err != nil {
				f.Close()
				os.Remove(*exportPath)
				log.Fatal().Err(err).Msg("export failed")
			}
			f.Close()
			log.Info().Str("path", *exportPath).Msg("export complete")
			return
		}

		if *importPath != "" {
			importer := dataport.NewImporter(db, store)
			f, err := os.Open(*importPath)
			if err != nil {
				log.Fatal().Err(err).Str("path", *importPath).Msg("failed to open import file")
			}
			if err := importer.Import(ctx, f); err != nil {
				f.Close()
				log.Fatal().Err(err).Msg("import failed")
			}
			f.Close()
			log.Info().Str("path", *importPath).Msg("import complete")
			return
		}
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
	systemSettingRepo := repository.NewSystemSettingRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)
	timeEntryRepo := repository.NewTimeEntryRepository(db)
	inboxRepo := repository.NewInboxRepository(db)
	oauthAccountRepo := repository.NewOAuthAccountRepository(db)
	typeWorkflowRepo := repository.NewProjectTypeWorkflowRepository(db)
	inviteRepo := repository.NewProjectInviteRepository(db)
	slaRepo := repository.NewSLARepository(db)

	// Initialize storage
	store, err := storage.NewMinIOStorage(
		cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey,
		cfg.StorageBucket, cfg.StorageRegion, cfg.StorageUseSSL,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage")
	}

	// Initialize services
	// Build OAuth providers from config
	var oauthProviders []service.OAuthProvider
	if cfg.DiscordClientID != "" && cfg.DiscordClientSecret != "" && cfg.DiscordRedirectURI != "" {
		oauthProviders = append(oauthProviders, service.NewDiscordProvider(
			cfg.DiscordClientID, cfg.DiscordClientSecret, cfg.DiscordRedirectURI, nil,
		))
	}
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" && cfg.GoogleRedirectURI != "" {
		oauthProviders = append(oauthProviders, service.NewGoogleProvider(
			cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURI, nil,
		))
	}

	authService := service.NewAuthService(
		userRepo, apiKeyRepo, oauthAccountRepo,
		cfg.JWTSecret, cfg.JWTExpiry,
		oauthProviders,
	)
	projectService := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, workflowRepo, typeWorkflowRepo, systemSettingRepo, inviteRepo)
	workflowService := service.NewWorkflowService(workflowRepo)
	queueService := service.NewQueueService(queueRepo, projectRepo, projectMemberRepo)
	milestoneService := service.NewMilestoneService(milestoneRepo, projectRepo, projectMemberRepo)
	slaService := service.NewSLAService(slaRepo, projectRepo, projectMemberRepo, workflowRepo)
	workItemService := service.NewWorkItemService(workItemRepo, workItemEventRepo, commentRepo, relationRepo, attachmentRepo, timeEntryRepo, projectRepo, projectMemberRepo, workflowRepo, typeWorkflowRepo, queueRepo, milestoneRepo, slaRepo, slaService, store, cfg.MaxUploadSize)
	inboxService := service.NewInboxService(inboxRepo, projectMemberRepo)
	userSettingService := service.NewUserSettingService(userSettingRepo, projectRepo, projectMemberRepo)
	systemSettingService := service.NewSystemSettingService(systemSettingRepo)
	adminService := service.NewAdminService(userRepo, projectRepo, projectMemberRepo)

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

	// Seed default type-workflow setting
	if err := projectService.SeedDefaultTypeWorkflows(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to seed default type workflows")
	}

	// Backfill type-workflow mappings for existing projects
	if err := projectService.SeedExistingProjectTypeWorkflows(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to seed existing project type workflows")
	}

	// Initialize handlers
	health := handler.NewHealthHandler(db)
	auth := handler.NewAuthHandler(authService)
	projects := handler.NewProjectHandler(projectService, cfg.BaseURL)
	workflows := handler.NewWorkflowHandler(workflowService, projectService)
	queues := handler.NewQueueHandler(queueService)
	milestones := handler.NewMilestoneHandler(milestoneService)
	items := handler.NewWorkItemHandler(workItemService, slaService, cfg.MaxUploadSize)
	userSettings := handler.NewUserSettingHandler(userSettingService)
	systemSettings := handler.NewSystemSettingHandler(systemSettingService)
	admin := handler.NewAdminHandler(adminService)
	sla := handler.NewSLAHandler(slaService)
	inbox := handler.NewInboxHandler(inboxService, slaService)

	// Set up router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(log.Logger))
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(cfg.BaseURL))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.BodyLimit(1 << 20)) // 1MB limit for non-multipart requests

	// Health checks (unauthenticated)
	r.Get("/healthz", health.Healthz)
	r.Get("/readyz", health.Readyz)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (rate-limited)
		authLimiter := middleware.RateLimit(rate.Limit(cfg.AuthRateLimit)/60, cfg.AuthRateBurst)
		r.With(authLimiter).Post("/auth/login", auth.Login)
		r.Get("/auth/providers", auth.AuthProviders)
		r.Get("/auth/{provider}", auth.OAuthAuth)
		r.With(authLimiter).Post("/auth/{provider}/callback", auth.OAuthCallback)

		// Public settings (unauthenticated)
		r.Get("/settings/public", systemSettings.GetPublic)

		// Public invite info (unauthenticated)
		r.Get("/invites/{code}", projects.GetInviteInfo)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			r.Post("/auth/refresh", auth.Refresh)
			r.Post("/auth/logout", auth.Logout)
			r.Get("/auth/me", auth.Me)
			r.Post("/auth/change-password", auth.ChangePassword)

			// API key management
			r.Get("/user/api-keys", auth.ListAPIKeys)
			r.Post("/user/api-keys", auth.CreateAPIKey)
			r.Delete("/user/api-keys/{keyId}", auth.DeleteAPIKey)

			// User search
			r.Get("/users/search", auth.SearchUsers)

			// Accept invite (authenticated)
			r.Post("/invites/{code}/accept", projects.AcceptInvite)

			// Global user preferences
			r.Route("/user/preferences", func(r chi.Router) {
				r.Get("/", userSettings.ListGlobal)
				r.Route("/{key}", func(r chi.Router) {
					r.Get("/", userSettings.GetGlobal)
					r.Put("/", userSettings.SetGlobal)
					r.Delete("/", userSettings.DeleteGlobal)
				})
			})

			// User inbox
			r.Route("/user/inbox", func(r chi.Router) {
				r.Get("/", inbox.List)
				r.Post("/", inbox.Add)
				r.Get("/count", inbox.Count)
				r.Delete("/completed", inbox.ClearCompleted)
				r.Route("/{inboxItemId}", func(r chi.Router) {
					r.Delete("/", inbox.Remove)
					r.Patch("/", inbox.Reorder)
				})
			})

			// Workflows
			r.Route("/workflows", func(r chi.Router) {
				r.Get("/", workflows.List)
				r.Get("/statuses", workflows.ListSystemStatuses)
				r.Route("/{workflowId}", func(r chi.Router) {
					r.Get("/", workflows.Get)
					r.Get("/transitions", workflows.ListTransitions)
				})
				// Create/Update/Delete require admin role
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireAdmin)
					r.Post("/", workflows.Create)
					r.Patch("/{workflowId}", workflows.Update)
					r.Delete("/{workflowId}", workflows.Delete)
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
					r.Route("/invites", func(r chi.Router) {
						r.Get("/", projects.ListInvites)
						r.Post("/", projects.CreateInvite)
						r.Delete("/{inviteId}", projects.DeleteInvite)
					})
					r.Route("/type-workflows", func(r chi.Router) {
						r.Get("/", projects.ListTypeWorkflows)
						r.Put("/{type}", projects.UpdateTypeWorkflow)
					})
					r.Route("/workflows", func(r chi.Router) {
						r.Get("/", workflows.ListProjectWorkflows)
						r.Get("/statuses", workflows.ListAvailableStatuses)
						r.Post("/", workflows.CreateProjectWorkflow)
						r.Route("/{workflowId}", func(r chi.Router) {
							r.Get("/", workflows.GetProjectWorkflow)
							r.Patch("/", workflows.UpdateProjectWorkflow)
							r.Delete("/", workflows.DeleteProjectWorkflow)
						})
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
					r.Route("/sla-targets", func(r chi.Router) {
						r.Get("/", sla.List)
						r.Put("/", sla.BulkUpsert)
						r.Delete("/{targetId}", sla.Delete)
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
								r.Patch("/{attachmentId}", items.UpdateAttachmentComment)
								r.Delete("/{attachmentId}", items.DeleteAttachment)
							})
							r.Route("/time-entries", func(r chi.Router) {
								r.Get("/", items.ListTimeEntries)
								r.Post("/", items.CreateTimeEntry)
								r.Patch("/{timeEntryId}", items.UpdateTimeEntry)
								r.Delete("/{timeEntryId}", items.DeleteTimeEntry)
							})
							r.Get("/events", items.ListEvents)
						})
					})
				})
			})

			// Admin routes (requires admin role)
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Route("/users", func(r chi.Router) {
					r.Get("/", admin.ListUsers)
					r.Post("/", admin.CreateUser)
					r.Route("/{userId}", func(r chi.Router) {
						r.Patch("/", admin.UpdateUser)
						r.Post("/reset-password", admin.ResetUserPassword)
						r.Route("/projects", func(r chi.Router) {
							r.Get("/", admin.ListUserProjects)
							r.Post("/", admin.AddUserToProject)
							r.Patch("/{projectId}", admin.UpdateUserProjectRole)
							r.Delete("/{projectId}", admin.RemoveUserFromProject)
						})
					})
				})
				r.Route("/settings", func(r chi.Router) {
					r.Get("/", systemSettings.List)
					r.Route("/{key}", func(r chi.Router) {
						r.Get("/", systemSettings.Get)
						r.Put("/", systemSettings.Set)
						r.Delete("/", systemSettings.Delete)
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
