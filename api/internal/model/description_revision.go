package model

import (
	"time"

	"github.com/google/uuid"
)

// Comment anchor status values.
const (
	AnchorStatusActive   = "active"
	AnchorStatusOutdated = "outdated"
)

// DescriptionRevision is a snapshot of a work item's markdown description
// taken at a point in time. New revisions are written every time a save
// produces a different normalized body.
type DescriptionRevision struct {
	ID             uuid.UUID  `json:"id"`
	WorkItemID     uuid.UUID  `json:"work_item_id"`
	RevisionNumber int        `json:"revision_number"`
	Content        string     `json:"content"`
	ContentHash    string     `json:"content_hash"`
	AuthorID       *uuid.UUID `json:"author_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CommentAnchor describes the position an inline comment is attached to
// inside a particular description revision. A comment whose anchor is nil
// is a regular (non-inline) comment.
//
// StartLine/EndLine are 1-based inclusive source line numbers. StartCol/EndCol
// are 1-based character offsets within their respective lines; EndCol is
// exclusive (one past the last selected character). Columns are 0 when the
// anchor covers whole blocks rather than a sub-line text selection.
type CommentAnchor struct {
	RevisionID     uuid.UUID `json:"revision_id"`
	RevisionNumber int       `json:"revision_number"`
	StartLine      int       `json:"start_line"`
	StartCol       int       `json:"start_col"`
	EndLine        int       `json:"end_line"`
	EndCol         int       `json:"end_col"`
	Snippet        string    `json:"snippet"`
	SnippetHash    string    `json:"snippet_hash"`
	Status         string    `json:"status"`
}
