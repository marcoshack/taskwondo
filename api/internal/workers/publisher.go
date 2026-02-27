package workers

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// EventPublisher publishes messages to NATS JetStream subjects.
type EventPublisher struct {
	js     nats.JetStreamContext
	logger zerolog.Logger
}

// NewEventPublisher creates a publisher backed by a NATS connection.
// It ensures the TASKWONDO stream exists.
func NewEventPublisher(nc *nats.Conn, logger zerolog.Logger) (*EventPublisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("creating jetstream context: %w", err)
	}

	// Ensure the stream exists (idempotent — same config as dispatcher)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subjectPrefix + ">"},
		Storage:   nats.FileStorage,
		Retention: nats.WorkQueuePolicy,
		MaxAge:    24 * 60 * 60 * 1e9, // 24h in nanoseconds
	})
	if err != nil {
		return nil, fmt.Errorf("ensuring stream: %w", err)
	}

	return &EventPublisher{js: js, logger: logger}, nil
}

// Publish serializes data to JSON and publishes to the given subject.
func (p *EventPublisher) Publish(subject string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	fullSubject := subjectPrefix + subject
	if _, err := p.js.Publish(fullSubject, payload); err != nil {
		p.logger.Error().Err(err).Str("subject", fullSubject).Msg("failed to publish event")
		return fmt.Errorf("publishing to %s: %w", fullSubject, err)
	}

	p.logger.Debug().Str("subject", fullSubject).Msg("event published")
	return nil
}

// NoopPublisher is a publisher that does nothing, used when NATS is not configured.
type NoopPublisher struct{}

// Publish is a no-op.
func (NoopPublisher) Publish(string, any) error { return nil }
