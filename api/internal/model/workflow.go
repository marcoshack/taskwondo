package model

import (
	"time"

	"github.com/google/uuid"
)

// Status category constants.
const (
	CategoryTodo       = "todo"
	CategoryInProgress = "in_progress"
	CategoryDone       = "done"
	CategoryCancelled  = "cancelled"
)

// Workflow defines a set of statuses and transitions for work items.
type Workflow struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	IsDefault   bool                 `json:"is_default"`
	Statuses    []WorkflowStatus     `json:"statuses,omitempty"`
	Transitions []WorkflowTransition `json:"transitions,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// WorkflowStatus is a single status within a workflow.
type WorkflowStatus struct {
	ID          uuid.UUID `json:"id"`
	WorkflowID  uuid.UUID `json:"workflow_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Category    string    `json:"category"`
	Position    int       `json:"position"`
	Color       *string   `json:"color,omitempty"`
}

// WorkflowTransition defines a valid status change within a workflow.
type WorkflowTransition struct {
	ID         uuid.UUID `json:"id"`
	WorkflowID uuid.UUID `json:"workflow_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Name       *string   `json:"name,omitempty"`
}
