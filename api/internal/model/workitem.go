package model

import (
	"time"

	"github.com/google/uuid"
)

// Work item type constants.
const (
	WorkItemTypeTask     = "task"
	WorkItemTypeTicket   = "ticket"
	WorkItemTypeBug      = "bug"
	WorkItemTypeFeedback = "feedback"
	WorkItemTypeEpic     = "epic"
)

// Work item priority constants.
const (
	PriorityCritical = "critical"
	PriorityHigh     = "high"
	PriorityMedium   = "medium"
	PriorityLow      = "low"
)

// Work item visibility constants.
const (
	VisibilityInternal = "internal"
	VisibilityPortal   = "portal"
	VisibilityPublic   = "public"
)

// WorkItem represents a task, ticket, bug, feedback item, or epic.
type WorkItem struct {
	ID              uuid.UUID              `json:"id"`
	ProjectID       uuid.UUID              `json:"project_id"`
	QueueID         *uuid.UUID             `json:"queue_id,omitempty"`
	MilestoneID     *uuid.UUID             `json:"milestone_id,omitempty"`
	ParentID        *uuid.UUID             `json:"parent_id,omitempty"`
	ItemNumber      int                    `json:"item_number"`
	DisplayID       string                 `json:"display_id"`
	Type            string                 `json:"type"`
	Title           string                 `json:"title"`
	Description     *string                `json:"description,omitempty"`
	Status          string                 `json:"status"`
	Priority        string                 `json:"priority"`
	AssigneeID      *uuid.UUID             `json:"assignee_id,omitempty"`
	ReporterID      uuid.UUID              `json:"reporter_id"`
	PortalContactID *uuid.UUID             `json:"portal_contact_id,omitempty"`
	Visibility      string                 `json:"visibility"`
	Labels          []string               `json:"labels"`
	CustomFields    map[string]interface{} `json:"custom_fields"`
	DueDate         *time.Time             `json:"due_date,omitempty"`
	SLADeadline     *time.Time             `json:"sla_deadline,omitempty"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       *time.Time             `json:"-"`
}

// WorkItemEvent records a state change on a work item (audit trail).
type WorkItemEvent struct {
	ID         uuid.UUID              `json:"id"`
	WorkItemID uuid.UUID              `json:"work_item_id"`
	ActorID    *uuid.UUID             `json:"actor_id,omitempty"`
	EventType  string                 `json:"event_type"`
	FieldName  *string                `json:"field_name,omitempty"`
	OldValue   *string                `json:"old_value,omitempty"`
	NewValue   *string                `json:"new_value,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Visibility string                 `json:"visibility"`
	CreatedAt  time.Time              `json:"created_at"`
}

// WorkItemFilter holds all possible filter criteria for listing work items.
type WorkItemFilter struct {
	Types      []string   // filter by type (multiple allowed)
	Statuses   []string   // filter by status (multiple allowed)
	Priorities []string   // filter by priority (multiple allowed)
	AssigneeID *uuid.UUID // specific assignee
	Unassigned bool       // true = WHERE assignee_id IS NULL
	AssigneeMe bool       // true = use the caller's user ID
	QueueID    *uuid.UUID // filter by queue
	MilestoneID *uuid.UUID // filter by milestone
	Labels     []string   // filter items that contain ALL these labels
	ParentID   *uuid.UUID // children of a specific parent
	ParentNone bool       // true = top-level items only (parent_id IS NULL)
	Search     string     // full-text search query
	Sort       string     // sort field: created_at, updated_at, priority, due_date, item_number
	Order      string     // asc or desc
	Cursor     *uuid.UUID // cursor-based pagination: items after this ID
	Limit      int        // page size (max 100, default 50)
}

// WorkItemList is the paginated result for work item listings.
type WorkItemList struct {
	Items   []WorkItem `json:"items"`
	Cursor  string     `json:"cursor"`
	HasMore bool       `json:"has_more"`
	Total   int        `json:"total"`
}

// WorkItemEventWithActor is a WorkItemEvent enriched with actor display info.
type WorkItemEventWithActor struct {
	WorkItemEvent
	ActorDisplayName *string `json:"actor_display_name,omitempty"`
}
