package workers

import "context"

// Task is the interface for all async job handlers dispatched via NATS JetStream.
type Task interface {
	// Name returns a unique identifier for this task type (e.g. "stats.summarize").
	// Used as the NATS subject suffix: taskwondo.<name>
	Name() string

	// Execute runs the task with the given payload (decoded from the NATS message).
	Execute(ctx context.Context, payload []byte) error
}
