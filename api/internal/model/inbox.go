package model

import (
	"time"

	"github.com/google/uuid"
)

// MaxInboxItems is the hard limit on how many items a user can have in their inbox.
const MaxInboxItems = 100

// InboxPositionGap is the gap between consecutive inbox item positions.
const InboxPositionGap = 1000

// InboxItem represents a work item in a user's personal inbox.
type InboxItem struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	WorkItemID uuid.UUID `json:"work_item_id"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

// InboxItemWithWorkItem is an InboxItem enriched with joined work item and project data.
type InboxItemWithWorkItem struct {
	InboxItem
	DisplayID      string     `json:"display_id"`
	Title          string     `json:"title"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	StatusCategory string     `json:"status_category"`
	Priority       string     `json:"priority"`
	ProjectKey     string     `json:"project_key"`
	ProjectName    string     `json:"project_name"`
	AssigneeID          *uuid.UUID `json:"assignee_id,omitempty"`
	AssigneeDisplayName string     `json:"assignee_display_name,omitempty"`
	Description         string     `json:"description,omitempty"`
	DueDate             *time.Time `json:"due_date,omitempty"`
	SLATargetAt         *time.Time `json:"sla_target_at,omitempty"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// InboxItemList is the paginated result for inbox item listings.
type InboxItemList struct {
	Items   []InboxItemWithWorkItem `json:"items"`
	Cursor  string                 `json:"cursor"`
	HasMore bool                   `json:"has_more"`
	Total   int                    `json:"total"`
}
