package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/config"
	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/database"
	"github.com/marcoshack/taskwondo/internal/email"
	applog "github.com/marcoshack/taskwondo/internal/log"
	"github.com/marcoshack/taskwondo/internal/repository"
	"github.com/marcoshack/taskwondo/internal/service"
	"github.com/marcoshack/taskwondo/internal/workers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	if cfg.NatsURL == "" {
		log.Fatal().Msg("NATS_URL environment variable is required")
	}

	applog.Setup(cfg.LogLevel, cfg.LogFormat, "worker")
	baseCtx := log.Logger.WithContext(context.Background())
	// Cancel ctx on SIGINT (Ctrl+C) / SIGTERM; baseCtx stays live for the
	// graceful-shutdown drain below.
	ctx, stop := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Info().Msg("starting taskwondo worker")

	// Connect to database with worker-specific pool size
	db, err := database.ConnectWithPool(ctx, cfg.DatabaseURL, cfg.WorkerDBPool, cfg.WorkerDBPool/2+1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Int("max_open", cfg.WorkerDBPool).Msg("connected to database")

	// Initialize repositories
	statsRepo := repository.NewStatsRepository(db)
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	userSettingRepo := repository.NewUserSettingRepository(db)
	systemSettingRepo := repository.NewSystemSettingRepository(db)

	workItemRepo := repository.NewWorkItemRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	milestoneRepo := repository.NewMilestoneRepository(db)
	queueRepo := repository.NewQueueRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	embeddingRepo := repository.NewEmbeddingRepository(db)
	escalationRepo := repository.NewEscalationRepository(db)
	slaNotificationRepo := repository.NewSLANotificationRepository(db)
	slaRepo := repository.NewSLARepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	memberRepo := repository.NewProjectMemberRepository(db)

	// Initialize embedding and indexer services
	embeddingService := service.NewEmbeddingService(cfg.OllamaURL, cfg.OllamaModel)
	indexerService := service.NewIndexerService(
		embeddingService, embeddingRepo,
		workItemRepo, commentRepo, projectRepo, milestoneRepo, queueRepo, attachmentRepo, teamRepo,
	)

	// Initialize encryption (same derivation as API server)
	var encKey []byte
	if v := os.Getenv("ENCRYPTION_KEY"); v != "" {
		encKey = []byte(v)
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

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL,
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
		log.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer nc.Close()
	log.Info().Str("url", cfg.NatsURL).Msg("connected to NATS")

	// Create worker pool
	pool := workers.NewPool(cfg.WorkerPoolSize)

	// Create dispatcher (NATS JetStream consumer)
	dispatcher, err := workers.NewDispatcher(nc, pool, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create dispatcher")
	}

	// Initialize watcher repository
	watcherRepo := repository.NewWatcherRepository(db)

	// URL builder: single source of truth for notification/email URLs.
	// Resolves the correct namespace segment per project so links point at the
	// right tenant (e.g. /mhack/projects/... rather than /d/projects/...).
	urlBuilder := workers.NewURLBuilder(cfg.BaseURL, projectRepo)

	// Register event-driven tasks
	notifyAssignment := workers.NewNotificationAssignmentTask(
		userRepo, projectRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyAssignment)

	notifyWatcher := workers.NewNotificationWatcherTask(
		watcherRepo, userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyWatcher)

	notifyNewItem := workers.NewNotificationNewItemTask(
		memberRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyNewItem)

	notifyCommentAssigned := workers.NewNotificationCommentOnAssignedTask(
		userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyCommentAssigned)

	notifyStatusChange := workers.NewNotificationStatusChangeTask(
		userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyStatusChange)

	notifyMemberAdded := workers.NewNotificationMemberAddedTask(
		userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyMemberAdded)

	notifyInviteEmail := workers.NewNotificationInviteEmailTask(
		emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyInviteEmail)

	notifyNamespaceInviteEmail := workers.NewNotificationNamespaceInviteEmailTask(
		emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyNamespaceInviteEmail)

	// Register SLA breach notification task
	notifySLABreach := workers.NewNotificationSLABreachTask(
		escalationRepo, slaNotificationRepo, teamRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifySLABreach)

	// Register on-call rotation notification task
	notifyOncallRotation := workers.NewNotificationOncallRotationTask(
		userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyOncallRotation)

	// Register on-call override notification tasks
	notifyOncallOverrideCreated := workers.NewNotificationOncallOverrideCreatedTask(
		userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyOncallOverrideCreated)

	notifyOncallOverrideCancelled := workers.NewNotificationOncallOverrideCancelledTask(
		userRepo, userSettingRepo, emailSender, urlBuilder, log.Logger,
	)
	dispatcher.Register(notifyOncallOverrideCancelled)

	// Register embedding tasks
	embedIndex := workers.NewEmbedIndexTask(indexerService, systemSettingRepo, log.Logger)
	dispatcher.Register(embedIndex)

	embedDelete := workers.NewEmbedDeleteTask(indexerService, systemSettingRepo, log.Logger)
	dispatcher.Register(embedDelete)

	embedBackfill := workers.NewEmbedBackfillTask(indexerService, systemSettingRepo, log.Logger)
	dispatcher.Register(embedBackfill)

	// Start dispatcher. Task execution uses baseCtx, not the signal-cancellable
	// ctx, so in-flight work runs to completion during the Shutdown drain below
	// rather than being aborted the instant a signal arrives.
	if err := dispatcher.Start(baseCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to start dispatcher")
	}

	// Create event publisher for SLA monitor (publishes back to NATS for the consumer)
	eventPublisher, err := workers.NewEventPublisher(nc, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create event publisher")
	}

	// Initialize SLA service for the monitor
	slaService := service.NewSLAService(slaRepo, projectRepo, memberRepo, workflowRepo)

	// Initialize token repos for cleanup. Both tables hold email addresses —
	// including of people who never completed signup — so expired rows are purged.
	emailVerifRepo := repository.NewEmailVerificationRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)

	// Set up periodic scheduler
	scheduler := workers.NewScheduler(log.Logger)

	statsSummarize := workers.NewStatsSummarizeTask(statsRepo, log.Logger)

	scheduler.Add(workers.PeriodicTask{
		Name:     "stats.summarize",
		Interval: 5 * time.Minute,
		Fn:       statsSummarize.Run,
	})

	// SLA monitor: periodic scan for SLA threshold crossings
	typeWorkflowRepo := repository.NewProjectTypeWorkflowRepository(db)
	slaMonitor := workers.NewSLAMonitorTask(
		projectRepo, slaService, escalationRepo, slaNotificationRepo,
		workItemRepo, workflowRepo, typeWorkflowRepo, eventPublisher, log.Logger,
	)
	scheduler.Add(workers.PeriodicTask{
		Name:     "sla.monitor",
		Interval: cfg.SLAMonitorInterval,
		Fn:       slaMonitor.Run,
	})

	// On-call rotation: periodic scan for due rotations
	oncallRepo := repository.NewOncallRotationRepository(db)
	oncallService := service.NewOncallService(oncallRepo, teamRepo, projectRepo, memberRepo)
	oncallTask := workers.NewOncallRotationTask(
		oncallRepo, oncallService, teamRepo, projectRepo, eventPublisher, log.Logger,
	)
	scheduler.Add(workers.PeriodicTask{
		Name:     "oncall.rotation",
		Interval: 60 * time.Second,
		Fn:       oncallTask.Run,
	})

	// Token cleanup: purge expired email verification and password reset tokens
	tokenCleanup := workers.NewTokenCleanupTask(log.Logger,
		workers.TokenStore{Name: "email_verification", Repo: emailVerifRepo},
		workers.TokenStore{Name: "password_reset", Repo: passwordResetRepo},
	)
	scheduler.Add(workers.PeriodicTask{
		Name:     "token.cleanup",
		Interval: cfg.TokenCleanupInterval,
		Fn:       tokenCleanup.Run,
	})

	// Run backfill if requested (before starting periodic tasks)
	if cfg.BackfillStats {
		log.Info().Msg("backfilling historical stats snapshots")
		inserted, err := statsRepo.Backfill(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("stats backfill failed")
		}
		log.Info().Int64("snapshots_inserted", inserted).Msg("stats backfill completed")
	}

	scheduler.Start(baseCtx)

	// Graceful shutdown: block until a signal cancels ctx.
	<-ctx.Done()
	log.Info().Msg("shutting down worker")

	shutdownCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
	defer cancel()

	dispatcher.Shutdown(shutdownCtx)
	scheduler.Shutdown()

	log.Info().Msg("worker stopped")
}
