package model

import (
	"time"

	"github.com/google/uuid"
)

// WorkItemWatcher represents a user watching a work item for updates.
type WorkItemWatcher struct {
	ID         uuid.UUID `json:"id"`
	WorkItemID uuid.UUID `json:"work_item_id"`
	UserID     uuid.UUID `json:"user_id"`
	AddedBy    uuid.UUID `json:"added_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// WorkItemWatcherWithUser is a watcher enriched with user display info.
type WorkItemWatcherWithUser struct {
	WorkItemWatcher
	DisplayName string  `json:"display_name"`
	Email       string  `json:"email"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	AddedByName string  `json:"added_by_name"`
}
