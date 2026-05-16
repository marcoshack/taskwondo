package model

import (
	"time"

	"github.com/google/uuid"
)

// Comment represents a comment on a work item.
type Comment struct {
	ID              uuid.UUID      `json:"id"`
	WorkItemID      uuid.UUID      `json:"work_item_id"`
	AuthorID        *uuid.UUID     `json:"author_id,omitempty"`
	PortalContactID *uuid.UUID     `json:"portal_contact_id,omitempty"`
	Body            string         `json:"body"`
	Visibility      string         `json:"visibility"`
	EditCount       int            `json:"edit_count"`
	Anchor          *CommentAnchor `json:"anchor,omitempty"`
	// ParentCommentID threads a reply onto a root inline comment. Root
	// comments (regular or inline) have a nil parent.
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
