package model

import (
	"time"

	"github.com/google/uuid"
)

// Attachment represents a file attached to a work item.
type Attachment struct {
	ID          uuid.UUID `json:"id"`
	WorkItemID  uuid.UUID `json:"work_item_id"`
	UploaderID  uuid.UUID `json:"uploader_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StorageKey  string    `json:"-"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}
