package model

import (
	"time"

	"github.com/google/uuid"
)

// Comment represents a comment on a work item.
type Comment struct {
	ID              uuid.UUID  `json:"id"`
	WorkItemID      uuid.UUID  `json:"work_item_id"`
	AuthorID        *uuid.UUID `json:"author_id,omitempty"`
	PortalContactID *uuid.UUID `json:"portal_contact_id,omitempty"`
	Body            string     `json:"body"`
	Visibility      string     `json:"visibility"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
