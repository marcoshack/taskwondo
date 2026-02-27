package workers

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// PeriodicTask defines a task that runs on a fixed interval.
type PeriodicTask struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

// Scheduler runs periodic tasks on fixed intervals.
type Scheduler struct {
	tasks  []PeriodicTask
	logger zerolog.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler creates a new periodic task scheduler.
func NewScheduler(logger zerolog.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

// Add registers a periodic task. Must be called before Start.
func (s *Scheduler) Add(task PeriodicTask) {
	s.tasks = append(s.tasks, task)
}

// Start begins running all registered periodic tasks.
// Each task runs immediately on start, then on its interval.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	for _, task := range s.tasks {
		s.wg.Add(1)
		go s.runLoop(ctx, task)
	}
	s.logger.Info().Int("tasks", len(s.tasks)).Msg("scheduler started")
}

func (s *Scheduler) runLoop(ctx context.Context, task PeriodicTask) {
	defer s.wg.Done()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	// Run immediately on start
	s.execute(ctx, task)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.execute(ctx, task)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, task PeriodicTask) {
	s.logger.Info().Str("task", task.Name).Msg("running periodic task")
	if err := task.Fn(ctx); err != nil {
		s.logger.Error().Err(err).Str("task", task.Name).Msg("periodic task failed")
		return
	}
	s.logger.Info().Str("task", task.Name).Msg("periodic task completed")
}

// Shutdown stops all periodic tasks and waits for them to finish.
func (s *Scheduler) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.logger.Info().Msg("scheduler stopped")
}
