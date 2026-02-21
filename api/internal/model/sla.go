package model

import (
	"time"

	"github.com/google/uuid"
)

// Calendar mode constants.
const (
	CalendarMode24x7          = "24x7"
	CalendarModeBusinessHours = "business_hours"
)

// SLA status constants.
const (
	SLAStatusOnTrack = "on_track"
	SLAStatusWarning = "warning"
	SLAStatusBreached = "breached"
)

// SLAStatusTarget defines the maximum time a work item can stay in a given
// status, scoped by project, work item type, and workflow.
type SLAStatusTarget struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	WorkItemType  string    `json:"work_item_type"`
	WorkflowID    uuid.UUID `json:"workflow_id"`
	StatusName    string    `json:"status_name"`
	TargetSeconds int       `json:"target_seconds"`
	CalendarMode  string    `json:"calendar_mode"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SLAElapsed tracks accumulated time a work item has spent in a given status.
// This supports anti-gaming: if an item leaves and re-enters a status,
// elapsed_seconds carries forward (no reset).
type SLAElapsed struct {
	WorkItemID     uuid.UUID  `json:"work_item_id"`
	StatusName     string     `json:"status_name"`
	ElapsedSeconds int        `json:"elapsed_seconds"`
	LastEnteredAt  *time.Time `json:"last_entered_at,omitempty"`
}

// SLAInfo is the computed SLA status attached to work item responses.
type SLAInfo struct {
	TargetSeconds    int    `json:"target_seconds"`
	ElapsedSeconds   int    `json:"elapsed_seconds"`
	RemainingSeconds int    `json:"remaining_seconds"`
	Percentage       int    `json:"percentage"`
	Status           string `json:"status"` // on_track, warning, breached
}

// BusinessHoursConfig defines business hours for SLA calculations.
type BusinessHoursConfig struct {
	Days      []int  `json:"days"`       // 0=Sun, 1=Mon, ..., 6=Sat
	StartHour int    `json:"start_hour"` // 0-23
	EndHour   int    `json:"end_hour"`   // 0-23
	Timezone  string `json:"timezone"`   // IANA timezone name
}
