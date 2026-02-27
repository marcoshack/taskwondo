package workers

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// compactRepository is the minimal interface for the compaction task.
type compactRepository interface {
	CompactOlderThan(ctx context.Context, threshold time.Time) (int64, error)
}

// StatsCompactTask rolls up fine-grained stats snapshots to hourly granularity.
type StatsCompactTask struct {
	repo      compactRepository
	retention time.Duration
	logger    zerolog.Logger
}

// NewStatsCompactTask creates a new stats compaction task.
// retention is how long to keep fine-grained (5-min) data before rolling up to hourly.
func NewStatsCompactTask(repo compactRepository, retention time.Duration, logger zerolog.Logger) *StatsCompactTask {
	return &StatsCompactTask{repo: repo, retention: retention, logger: logger}
}

// Run executes the stats compaction.
func (t *StatsCompactTask) Run(ctx context.Context) error {
	threshold := time.Now().Add(-t.retention)
	deleted, err := t.repo.CompactOlderThan(ctx, threshold)
	if err != nil {
		return err
	}
	t.logger.Info().Int64("compacted_rows", deleted).Time("threshold", threshold).Msg("stats compaction complete")
	return nil
}
