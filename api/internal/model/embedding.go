package model

import (
	"time"

	"github.com/google/uuid"
)

// Entity type constants for the embeddings table.
const (
	EntityTypeWorkItem   = "work_item"
	EntityTypeComment    = "comment"
	EntityTypeProject    = "project"
	EntityTypeMilestone  = "milestone"
	EntityTypeQueue      = "queue"
	EntityTypeAttachment = "attachment"
)

// Embedding represents a vector embedding stored in the database.
type Embedding struct {
	ID         uuid.UUID  `json:"id"`
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty"`
	Path       string     `json:"path"`
	Content    string     `json:"-"`
	Embedding  []float32  `json:"-"`
	IndexedAt  time.Time  `json:"indexed_at"`
}

// EmbedIndexEvent is the NATS payload for embed.index events.
type EmbedIndexEvent struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty"`
}

// EmbedDeleteEvent is the NATS payload for embed.delete events.
type EmbedDeleteEvent struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
}

// EmbedBackfillEvent is the NATS payload for embed.backfill events.
type EmbedBackfillEvent struct {
	Backfill bool `json:"backfill"`
}

// SearchResult represents a single result from a semantic search.
type SearchResult struct {
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty"`
	Score      float64    `json:"score"`
	Content    string     `json:"snippet"`
	Path       string     `json:"path"`
}

// SearchFilter holds the parameters for a semantic search query.
type SearchFilter struct {
	Query       string
	EntityTypes []string
	ProjectIDs  []uuid.UUID
	Limit       int
}
