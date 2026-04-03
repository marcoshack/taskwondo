package model

import (
	"time"

	"github.com/google/uuid"
)

// QueueCategory represents a classification category within a queue.
type QueueCategory struct {
	ID          uuid.UUID `json:"id"`
	QueueID     uuid.UUID `json:"queue_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
