package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/config"
	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/database"
	"github.com/marcoshack/taskwondo/internal/email"
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

	setupLogger(cfg.LogLevel, cfg.LogFormat)
	ctx := log.Logger.WithContext(context.Background())
	log.Info().Msg("starting taskwondo worker")

	// Connect to database with worker-specific pool size
	db, err := database.ConnectWithPool(ctx, cfg.DatabaseURL, cfg.WorkerDBPool, cfg.WorkerDBPool/2+1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Int("max_open", cfg.WorkerDBPool).Msg("connected to database")

	// Run migrations (idempotent)
	if err := database.Migrate(ctx, db); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

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
	embeddingRepo := repository.NewEmbeddingRepository(db)

	// Initialize embedding and indexer services
	embeddingService := service.NewEmbeddingService(cfg.OllamaURL, cfg.OllamaModel)
	indexerService := service.NewIndexerService(
		embeddingService, embeddingRepo,
		workItemRepo, commentRepo, projectRepo, milestoneRepo, queueRepo, attachmentRepo,
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

	// Register event-driven tasks
	notifyAssignment := workers.NewNotificationAssignmentTask(
		userRepo, projectRepo, userSettingRepo, emailSender, cfg.BaseURL, log.Logger,
	)
	dispatcher.Register(notifyAssignment)

	notifyWatcher := workers.NewNotificationWatcherTask(
		watcherRepo, userRepo, userSettingRepo, emailSender, cfg.BaseURL, log.Logger,
	)
	dispatcher.Register(notifyWatcher)

	memberRepo := repository.NewProjectMemberRepository(db)

	notifyNewItem := workers.NewNotificationNewItemTask(
		memberRepo, userSettingRepo, emailSender, cfg.BaseURL, log.Logger,
	)
	dispatcher.Register(notifyNewItem)

	notifyCommentAssigned := workers.NewNotificationCommentOnAssignedTask(
		userRepo, userSettingRepo, emailSender, cfg.BaseURL, log.Logger,
	)
	dispatcher.Register(notifyCommentAssigned)

	notifyStatusChange := workers.NewNotificationStatusChangeTask(
		userRepo, userSettingRepo, emailSender, cfg.BaseURL, log.Logger,
	)
	dispatcher.Register(notifyStatusChange)

	notifyMemberAdded := workers.NewNotificationMemberAddedTask(
		userRepo, userSettingRepo, emailSender, cfg.BaseURL, log.Logger,
	)
	dispatcher.Register(notifyMemberAdded)

	// Register embedding tasks
	embedIndex := workers.NewEmbedIndexTask(indexerService, systemSettingRepo, log.Logger)
	dispatcher.Register(embedIndex)

	embedDelete := workers.NewEmbedDeleteTask(indexerService, systemSettingRepo, log.Logger)
	dispatcher.Register(embedDelete)

	embedBackfill := workers.NewEmbedBackfillTask(indexerService, systemSettingRepo, log.Logger)
	dispatcher.Register(embedBackfill)

	// Start dispatcher
	if err := dispatcher.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start dispatcher")
	}

	// Set up periodic scheduler
	scheduler := workers.NewScheduler(log.Logger)

	statsSummarize := workers.NewStatsSummarizeTask(statsRepo, log.Logger)

	scheduler.Add(workers.PeriodicTask{
		Name:     "stats.summarize",
		Interval: 5 * time.Minute,
		Fn:       statsSummarize.Run,
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

	scheduler.Start(ctx)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("shutting down worker")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	dispatcher.Shutdown(shutdownCtx)
	scheduler.Shutdown()

	log.Info().Msg("worker stopped")
}

func setupLogger(level, format string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	if format == "text" {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(os.Stderr).
			With().Timestamp().Logger()
	}
}
