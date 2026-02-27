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
	"github.com/marcoshack/taskwondo/internal/database"
	"github.com/marcoshack/taskwondo/internal/repository"
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

	// Register event-driven tasks here as they are added
	// dispatcher.Register(someTask)

	// Start dispatcher
	if err := dispatcher.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start dispatcher")
	}

	// Set up periodic scheduler
	scheduler := workers.NewScheduler(log.Logger)

	statsSummarize := workers.NewStatsSummarizeTask(statsRepo, log.Logger)
	statsCompact := workers.NewStatsCompactTask(statsRepo, 7*24*time.Hour, log.Logger)

	scheduler.Add(workers.PeriodicTask{
		Name:     "stats.summarize",
		Interval: 5 * time.Minute,
		Fn:       statsSummarize.Run,
	})
	scheduler.Add(workers.PeriodicTask{
		Name:     "stats.compact",
		Interval: 1 * time.Hour,
		Fn:       statsCompact.Run,
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
