package model

import "github.com/google/uuid"

// AssignmentEvent is published when a work item is assigned or reassigned.
type AssignmentEvent struct {
	WorkItemID uuid.UUID  `json:"work_item_id"`
	ProjectKey string     `json:"project_key"`
	ItemNumber int        `json:"item_number"`
	Title      string     `json:"title"`
	AssigneeID uuid.UUID  `json:"assignee_id"`
	AssignerID uuid.UUID  `json:"assigner_id"`
	ProjectID  uuid.UUID  `json:"project_id"`
}

// NotificationPreferences holds per-project notification toggles stored in user_settings.
type NotificationPreferences struct {
	AssignedToMe              bool `json:"assigned_to_me"`
	AnyUpdateOnWatched        bool `json:"any_update_on_watched"`
	NewItemCreated            bool `json:"new_item_created"`
	CommentsOnAssigned        bool `json:"comments_on_assigned"`
	CommentsOnWatched         bool `json:"comments_on_watched"`
	StatusChangesIntermediate bool `json:"status_changes_intermediate"`
	StatusChangesFinal        bool `json:"status_changes_final"`
}

// DefaultNotificationPreferences returns the default notification settings.
func DefaultNotificationPreferences() NotificationPreferences {
	return NotificationPreferences{
		AssignedToMe: true,
	}
}

// GlobalNotificationPreferences holds global (non-project-scoped) notification toggles.
type GlobalNotificationPreferences struct {
	AddedToProject bool `json:"added_to_project"`
}

// DefaultGlobalNotificationPreferences returns the default global notification settings.
func DefaultGlobalNotificationPreferences() GlobalNotificationPreferences {
	return GlobalNotificationPreferences{}
}

// NewItemEvent is published when a work item is created in a project.
type NewItemEvent struct {
	WorkItemID uuid.UUID `json:"work_item_id"`
	ProjectKey string    `json:"project_key"`
	ProjectID  uuid.UUID `json:"project_id"`
	ItemNumber int       `json:"item_number"`
	Title      string    `json:"title"`
	CreatorID  uuid.UUID `json:"creator_id"`
	Type       string    `json:"type"`
}

// CommentOnAssignedEvent is published when a comment is added to an assigned work item.
type CommentOnAssignedEvent struct {
	WorkItemID  uuid.UUID `json:"work_item_id"`
	ProjectKey  string    `json:"project_key"`
	ProjectID   uuid.UUID `json:"project_id"`
	ItemNumber  int       `json:"item_number"`
	Title       string    `json:"title"`
	AssigneeID  uuid.UUID `json:"assignee_id"`
	CommenterID uuid.UUID `json:"commenter_id"`
	Preview     string    `json:"preview"`
}

// StatusChangeEvent is published when a work item's status changes.
type StatusChangeEvent struct {
	WorkItemID uuid.UUID `json:"work_item_id"`
	ProjectKey string    `json:"project_key"`
	ProjectID  uuid.UUID `json:"project_id"`
	ItemNumber int       `json:"item_number"`
	Title      string    `json:"title"`
	AssigneeID uuid.UUID `json:"assignee_id"`
	ActorID    uuid.UUID `json:"actor_id"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	Category   string    `json:"category"` // "in_progress", "done", "cancelled"
}

// MemberAddedEvent is published when a user is added to a project.
type MemberAddedEvent struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectKey  string    `json:"project_key"`
	ProjectName string    `json:"project_name"`
	UserID      uuid.UUID `json:"user_id"`
	AddedByID   uuid.UUID `json:"added_by_id"`
	Role        string    `json:"role"`
}

// InviteEmailEvent is published when an email invite is created for a non-existing user.
type InviteEmailEvent struct {
	ProjectKey   string `json:"project_key"`
	ProjectName  string `json:"project_name"`
	InviteeEmail string `json:"invitee_email"`
	InviterName  string `json:"inviter_name"`
	InviteCode   string `json:"invite_code"`
	Role         string `json:"role"`
}

// WatcherEvent is published when a watched work item is modified.
type WatcherEvent struct {
	WorkItemID uuid.UUID `json:"work_item_id"`
	ProjectKey string    `json:"project_key"`
	ProjectID  uuid.UUID `json:"project_id"`
	ItemNumber int       `json:"item_number"`
	Title      string    `json:"title"`
	ActorID    uuid.UUID `json:"actor_id"`
	EventType  string    `json:"event_type"`            // "field_change", "comment_added"
	FieldName  string    `json:"field_name,omitempty"`   // e.g. "status", "priority"
	OldValue   string    `json:"old_value,omitempty"`
	NewValue   string    `json:"new_value,omitempty"`
	Summary    string    `json:"summary,omitempty"`      // human-readable change summary
}
