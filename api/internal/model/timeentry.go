package model

import (
	"time"

	"github.com/google/uuid"
)

// TimeEntry records time spent on a work item.
type TimeEntry struct {
	ID              uuid.UUID  `json:"id"`
	WorkItemID      uuid.UUID  `json:"work_item_id"`
	UserID          uuid.UUID  `json:"user_id"`
	StartedAt       time.Time  `json:"started_at"`
	DurationSeconds int        `json:"duration_seconds"`
	Description     *string    `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
