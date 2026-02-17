package model

import (
	"time"

	"github.com/google/uuid"
)

// Queue type constants.
const (
	QueueTypeSupport  = "support"
	QueueTypeAlerts   = "alerts"
	QueueTypeFeedback = "feedback"
	QueueTypeGeneral  = "general"
)

// Queue represents an inbound work channel within a project.
type Queue struct {
	ID                uuid.UUID  `json:"id"`
	ProjectID         uuid.UUID  `json:"project_id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	QueueType         string     `json:"queue_type"`
	IsPublic          bool       `json:"is_public"`
	DefaultPriority   string     `json:"default_priority"`
	DefaultAssigneeID *uuid.UUID `json:"default_assignee_id,omitempty"`
	WorkflowID        *uuid.UUID `json:"workflow_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
