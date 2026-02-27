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
	NewItemCreated            bool `json:"new_item_created"`
	CommentsOnAssigned        bool `json:"comments_on_assigned"`
	CommentsOnWatched         bool `json:"comments_on_watched"`
	StatusChangesIntermediate bool `json:"status_changes_intermediate"`
	StatusChangesFinal        bool `json:"status_changes_final"`
	AddedToProject            bool `json:"added_to_project"`
}

// DefaultNotificationPreferences returns the default notification settings.
func DefaultNotificationPreferences() NotificationPreferences {
	return NotificationPreferences{
		AssignedToMe: true,
	}
}
