package model

import (
	"time"

	"github.com/google/uuid"
)

// EscalationList is a named, reusable escalation configuration scoped to a project.
type EscalationList struct {
	ID        uuid.UUID         `json:"id"`
	ProjectID uuid.UUID         `json:"project_id"`
	Name      string            `json:"name"`
	Levels    []EscalationLevel `json:"levels"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// EscalationLevel defines a threshold percentage and users to notify.
type EscalationLevel struct {
	ID           uuid.UUID             `json:"id"`
	ListID       uuid.UUID             `json:"escalation_list_id"`
	ThresholdPct int                   `json:"threshold_pct"`
	Position     int                   `json:"position"`
	Users        []EscalationLevelUser `json:"users"`
	CreatedAt    time.Time             `json:"created_at"`
}

// EscalationLevelUser represents a user assigned to an escalation level.
type EscalationLevelUser struct {
	UserID      uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}

// TypeEscalationMapping maps a work item type to an escalation list within a project.
type TypeEscalationMapping struct {
	ProjectID        uuid.UUID `json:"project_id"`
	WorkItemType     string    `json:"work_item_type"`
	EscalationListID uuid.UUID `json:"escalation_list_id"`
}
