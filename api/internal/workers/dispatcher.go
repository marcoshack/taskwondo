package workers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

const (
	streamName    = "TASKWONDO"
	subjectPrefix = "taskwondo."
)

// Dispatcher consumes messages from NATS JetStream and dispatches them
// to registered task handlers via the worker pool.
type Dispatcher struct {
	js     nats.JetStreamContext
	pool   *Pool
	tasks  map[string]Task
	subs   []*nats.Subscription
	logger zerolog.Logger
}

// NewDispatcher creates a new dispatcher backed by NATS JetStream.
func NewDispatcher(nc *nats.Conn, pool *Pool, logger zerolog.Logger) (*Dispatcher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("creating jetstream context: %w", err)
	}
	return &Dispatcher{
		js:     js,
		pool:   pool,
		tasks:  make(map[string]Task),
		logger: logger,
	}, nil
}

// Register adds a task handler. Must be called before Start.
func (d *Dispatcher) Register(task Task) {
	d.tasks[task.Name()] = task
	d.logger.Info().Str("task", task.Name()).Msg("registered task handler")
}

// Start creates the JetStream stream (if needed) and begins consuming
// messages for all registered tasks.
func (d *Dispatcher) Start(ctx context.Context) error {
	// Create or update the stream
	_, err := d.js.AddStream(&nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subjectPrefix + ">"},
		Storage:   nats.FileStorage,
		Retention: nats.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	})
	if err != nil {
		return fmt.Errorf("creating stream: %w", err)
	}

	// Subscribe to each registered task's subject
	for name, task := range d.tasks {
		subject := subjectPrefix + name
		// NATS consumer/durable names cannot contain dots
		durableName := strings.ReplaceAll(name, ".", "-")
		sub, err := d.js.QueueSubscribe(
			subject,
			durableName, // queue group for competing consumers
			d.messageHandler(ctx, task),
			nats.Durable(durableName),
			nats.ManualAck(),
			nats.AckWait(30*time.Second),
			nats.MaxDeliver(3),
		)
		if err != nil {
			return fmt.Errorf("subscribing to %s: %w", subject, err)
		}
		d.subs = append(d.subs, sub)
		d.logger.Info().Str("subject", subject).Msg("subscribed to stream")
	}

	d.logger.Info().Int("tasks", len(d.tasks)).Msg("dispatcher started")
	return nil
}

func (d *Dispatcher) messageHandler(ctx context.Context, task Task) nats.MsgHandler {
	return func(msg *nats.Msg) {
		d.pool.Submit(func() {
			logger := d.logger.With().Str("task", task.Name()).Logger()
			logger.Debug().Msg("executing task")

			if err := task.Execute(ctx, msg.Data); err != nil {
				logger.Error().Err(err).Msg("task execution failed")
				if nakErr := msg.Nak(); nakErr != nil {
					logger.Error().Err(nakErr).Msg("failed to nak message")
				}
				return
			}

			if err := msg.Ack(); err != nil {
				logger.Error().Err(err).Msg("failed to ack message")
			}
			logger.Debug().Msg("task completed")
		})
	}
}

// Shutdown drains all subscriptions and waits for in-flight work to complete.
func (d *Dispatcher) Shutdown(_ context.Context) {
	for _, sub := range d.subs {
		if err := sub.Drain(); err != nil {
			d.logger.Error().Err(err).Msg("failed to drain subscription")
		}
	}
	d.pool.Shutdown()
	d.logger.Info().Msg("dispatcher stopped")
}
