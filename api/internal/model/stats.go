package model

import (
	"time"

	"github.com/google/uuid"
)

// ProjectStatsSnapshot represents a point-in-time snapshot of work item counts
// for a project. When UserID is nil, it represents project-level aggregates.
// When UserID is set, it represents per-assignee counts within the project.
type ProjectStatsSnapshot struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	TodoCount       int        `json:"todo_count"`
	InProgressCount int        `json:"in_progress_count"`
	DoneCount       int        `json:"done_count"`
	CancelledCount  int        `json:"cancelled_count"`
	CapturedAt     time.Time  `json:"captured_at"`
}
