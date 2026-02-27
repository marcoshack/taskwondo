package workers

import (
	"context"

	"github.com/rs/zerolog"
)

// StatsSummarizeTask snapshots work item counts for all projects.
type StatsSummarizeTask struct {
	repo   snapshotRepository
	logger zerolog.Logger
}

// snapshotRepository is the minimal interface for the summarization task.
type snapshotRepository interface {
	SnapshotAll(ctx context.Context) error
}

// NewStatsSummarizeTask creates a new stats summarization task.
func NewStatsSummarizeTask(repo snapshotRepository, logger zerolog.Logger) *StatsSummarizeTask {
	return &StatsSummarizeTask{repo: repo, logger: logger}
}

// Run executes the stats summarization.
func (t *StatsSummarizeTask) Run(ctx context.Context) error {
	return t.repo.SnapshotAll(ctx)
}
