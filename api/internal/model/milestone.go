package model

import (
	"time"

	"github.com/google/uuid"
)

// Milestone status constants.
const (
	MilestoneStatusOpen   = "open"
	MilestoneStatusClosed = "closed"
)

// Milestone represents a progress tracking target within a project.
type Milestone struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MilestoneWithProgress is a Milestone enriched with work item counts and time aggregates.
type MilestoneWithProgress struct {
	Milestone
	OpenCount             int `json:"open_count"`
	ClosedCount           int `json:"closed_count"`
	TotalCount            int `json:"total_count"`
	TotalEstimatedSeconds int `json:"total_estimated_seconds"`
	TotalSpentSeconds     int `json:"total_spent_seconds"`
}
