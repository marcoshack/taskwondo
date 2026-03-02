package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"github.com/marcoshack/taskwondo/internal/config"
	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/database"
	"github.com/marcoshack/taskwondo/internal/email"
	"github.com/marcoshack/taskwondo/internal/handler"
	"github.com/marcoshack/taskwondo/internal/middleware"
	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/repository"
	"github.com/marcoshack/taskwondo/internal/service"
	"github.com/marcoshack/taskwondo/internal/storage"
	"github.com/marcoshack/taskwondo/internal/workers"
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
	savedSearchRepo := repository.NewSavedSearchRepository(db)
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
	watcherRepo := repository.NewWatcherRepository(db)
	emailVerificationRepo := repository.NewEmailVerificationRepository(db)
	statsRepo := repository.NewStatsRepository(db)

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
	if cfg.DiscordClientID != "" && cfg.DiscordClientSecret != "" {
		oauthProviders = append(oauthProviders, service.NewDiscordProvider(
			cfg.DiscordClientID, cfg.DiscordClientSecret, cfg.BaseURL+"/auth/discord/callback", nil,
		))
	}
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		oauthProviders = append(oauthProviders, service.NewGoogleProvider(
			cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.BaseURL+"/auth/google/callback", nil,
		))
	}
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		oauthProviders = append(oauthProviders, service.NewGitHubProvider(
			cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.BaseURL+"/auth/github/callback", nil,
		))
	}
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		oauthProviders = append(oauthProviders, service.NewMicrosoftProvider(
			cfg.MicrosoftClientID, cfg.MicrosoftClientSecret, cfg.BaseURL+"/auth/microsoft/callback", nil,
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
	savedSearchService := service.NewSavedSearchService(savedSearchRepo, projectRepo, projectMemberRepo)
	milestoneService := service.NewMilestoneService(milestoneRepo, projectRepo, projectMemberRepo)
	slaService := service.NewSLAService(slaRepo, projectRepo, projectMemberRepo, workflowRepo)
	workItemService := service.NewWorkItemService(workItemRepo, workItemEventRepo, commentRepo, relationRepo, attachmentRepo, timeEntryRepo, watcherRepo, projectRepo, projectMemberRepo, workflowRepo, typeWorkflowRepo, queueRepo, milestoneRepo, slaRepo, slaService, store, cfg.MaxUploadSize)
	inboxService := service.NewInboxService(inboxRepo, projectMemberRepo)
	userSettingService := service.NewUserSettingService(userSettingRepo, projectRepo, projectMemberRepo)
	systemSettingService := service.NewSystemSettingService(systemSettingRepo)
	adminService := service.NewAdminService(userRepo, projectRepo, projectMemberRepo)
	statsService := service.NewStatsService(statsRepo, projectRepo, projectMemberRepo)

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

	// Initialize encryption
	var encKey []byte
	if cfg.EncryptionKey != "" {
		encKey = []byte(cfg.EncryptionKey)
		if len(encKey) != 32 {
			log.Fatal().Msg("ENCRYPTION_KEY must be exactly 32 bytes")
		}
	} else {
		var err error
		encKey, err = crypto.DeriveKey(cfg.JWTSecret)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to derive encryption key")
		}
	}
	encryptor, err := crypto.NewEncryptor(encKey)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create encryptor")
	}

	// Initialize email sender
	emailSender := email.NewSender(encryptor, systemSettingRepo)

	// Configure email verification on auth service
	authService.SetEmailVerification(emailVerificationRepo, systemSettingRepo, emailSender, cfg.BaseURL)

	// Configure encryptor on auth service (for decrypting OAuth secrets from DB)
	authService.SetEncryptor(encryptor)

	// Configure storage for avatar uploads
	authService.SetStorage(store)

	// Seed OAuth config from env vars if not already in DB
	seedOAuthConfig(ctx, systemSettingRepo, encryptor, cfg)

	// Connect to NATS for event publishing (optional — notifications degrade gracefully)
	publisher, natsCleanup := initEventPublisher(cfg.NatsURL)
	defer natsCleanup()
	workItemService.SetPublisher(publisher)

	// Initialize handlers
	health := handler.NewHealthHandler(db)
	auth := handler.NewAuthHandler(authService, projectService)
	projects := handler.NewProjectHandler(projectService, cfg.BaseURL)
	workflows := handler.NewWorkflowHandler(workflowService, projectService)
	queues := handler.NewQueueHandler(queueService)
	savedSearches := handler.NewSavedSearchHandler(savedSearchService)
	milestones := handler.NewMilestoneHandler(milestoneService)
	items := handler.NewWorkItemHandler(workItemService, slaService, cfg.MaxUploadSize)
	userSettings := handler.NewUserSettingHandler(userSettingService)
	systemSettings := handler.NewSystemSettingHandler(systemSettingService, encryptor, emailSender)
	admin := handler.NewAdminHandler(adminService)
	sla := handler.NewSLAHandler(slaService)
	inbox := handler.NewInboxHandler(inboxService, slaService)
	stats := handler.NewStatsHandler(statsService)

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
		r.With(authLimiter).Post("/auth/register", auth.Register)
		r.With(authLimiter).Post("/auth/verify-email", auth.VerifyEmail)
		r.Get("/auth/providers", auth.AuthProviders)
		r.Get("/auth/{provider}", auth.OAuthAuth)
		r.With(authLimiter).Post("/auth/{provider}/callback", auth.OAuthCallback)

		// Public settings (unauthenticated)
		r.Get("/settings/public", systemSettings.GetPublic)

		// Public invite info (unauthenticated)
		r.Get("/invites/{code}", projects.GetInviteInfo)

		// Public avatar serving (loaded by <img> tags which can't send JWT)
		r.Get("/users/{userId}/avatar", auth.GetUserAvatar)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			r.Post("/auth/refresh", auth.Refresh)
			r.Post("/auth/logout", auth.Logout)
			r.Get("/auth/me", auth.Me)
			r.Post("/auth/change-password", auth.ChangePassword)

			// User profile
			r.Patch("/user/profile", auth.UpdateProfile)
			r.Post("/user/avatar", auth.UploadAvatar)
			r.Delete("/user/avatar", auth.DeleteAvatar)

			// API key management
			r.Get("/user/api-keys", auth.ListAPIKeys)
			r.Post("/user/api-keys", auth.CreateAPIKey)
			r.Patch("/user/api-keys/{keyId}", auth.RenameAPIKey)
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

			// User watchlist
			r.Get("/user/watchlist", items.ListWatchedItemIDs)

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
					r.Route("/saved-searches", func(r chi.Router) {
						r.Get("/", savedSearches.List)
						r.Post("/", savedSearches.Create)
						r.Route("/{searchId}", func(r chi.Router) {
							r.Patch("/", savedSearches.Update)
							r.Delete("/", savedSearches.Delete)
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
							r.Get("/stats", milestones.Stats)
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
					r.Route("/stats", func(r chi.Router) {
						r.Get("/timeline", stats.Timeline)
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
							r.Route("/watchers", func(r chi.Router) {
							r.Get("/", items.ListWatchers)
							r.Post("/", items.AddWatcher)
							r.Delete("/{userId}", items.RemoveWatcher)
						})
						r.Post("/watch", items.ToggleWatch)
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
					r.Route("/smtp_config", func(r chi.Router) {
						r.Get("/", systemSettings.GetSMTP)
						r.Put("/", systemSettings.SetSMTP)
						r.Post("/test", systemSettings.TestSMTP)
					})
					r.Route("/oauth_config/{provider}", func(r chi.Router) {
						r.Get("/", systemSettings.GetOAuthConfig)
						r.Put("/", systemSettings.SetOAuthConfig)
					})
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

func initEventPublisher(natsURL string) (service.EventPublisher, func()) {
	noop := func() {}
	if natsURL == "" {
		log.Warn().Msg("NATS_URL not configured, notifications disabled")
		return workers.NoopPublisher{}, noop
	}

	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Error().Err(err).Msg("nats disconnected")
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Info().Msg("nats reconnected")
		}),
	)
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to NATS, notifications disabled")
		return workers.NoopPublisher{}, noop
	}

	pub, err := workers.NewEventPublisher(nc, log.Logger)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create event publisher, notifications disabled")
		nc.Close()
		return workers.NoopPublisher{}, noop
	}

	log.Info().Str("url", natsURL).Msg("connected to NATS for event publishing")
	return pub, nc.Close
}

// seedOAuthConfig seeds OAuth provider config from env vars into the database
// if no DB config exists yet. This provides backward compatibility — env vars
// are used only for initial seeding, then the DB config takes over.
func seedOAuthConfig(ctx context.Context, settings service.SystemSettingRepositoryInterface, encryptor *crypto.Encryptor, cfg *config.Config) {
	type providerEnv struct {
		name       string
		settingKey string
		enabledKey string
		clientID   string
		secret     string
	}

	providers := []providerEnv{
		{
			name:       "discord",
			settingKey: model.SettingOAuthDiscordConfig,
			enabledKey: model.SettingAuthDiscordEnabled,
			clientID:   cfg.DiscordClientID,
			secret:     cfg.DiscordClientSecret,
		},
		{
			name:       "google",
			settingKey: model.SettingOAuthGoogleConfig,
			enabledKey: model.SettingAuthGoogleEnabled,
			clientID:   cfg.GoogleClientID,
			secret:     cfg.GoogleClientSecret,
		},
		{
			name:       "github",
			settingKey: model.SettingOAuthGitHubConfig,
			enabledKey: model.SettingAuthGitHubEnabled,
			clientID:   cfg.GitHubClientID,
			secret:     cfg.GitHubClientSecret,
		},
		{
			name:       "microsoft",
			settingKey: model.SettingOAuthMicrosoftConfig,
			enabledKey: model.SettingAuthMicrosoftEnabled,
			clientID:   cfg.MicrosoftClientID,
			secret:     cfg.MicrosoftClientSecret,
		},
	}

	for _, p := range providers {
		if p.clientID == "" || p.secret == "" {
			continue // env vars not set, skip
		}

		// Check if DB config already exists
		_, err := settings.Get(ctx, p.settingKey)
		if err == nil {
			log.Debug().Str("provider", p.name).Msg("oauth config already in database, skipping seed")
			continue
		}

		// Encrypt the client secret
		encryptedSecret, err := encryptor.Encrypt(p.secret)
		if err != nil {
			log.Error().Err(err).Str("provider", p.name).Msg("failed to encrypt oauth client secret for seeding")
			continue
		}

		oauthCfg := model.OAuthProviderConfig{
			ClientID:     p.clientID,
			ClientSecret: encryptedSecret,
		}

		value, err := json.Marshal(oauthCfg)
		if err != nil {
			log.Error().Err(err).Str("provider", p.name).Msg("failed to marshal oauth config for seeding")
			continue
		}

		if err := settings.Upsert(ctx, &model.SystemSetting{Key: p.settingKey, Value: value}); err != nil {
			log.Error().Err(err).Str("provider", p.name).Msg("failed to seed oauth config")
			continue
		}

		// Also seed the enabled setting
		enabledValue, _ := json.Marshal(true)
		if err := settings.Upsert(ctx, &model.SystemSetting{Key: p.enabledKey, Value: enabledValue}); err != nil {
			log.Error().Err(err).Str("provider", p.name).Msg("failed to seed oauth enabled setting")
			continue
		}

		log.Info().Str("provider", p.name).Msg("seeded oauth config from environment variables")
	}
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
