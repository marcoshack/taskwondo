package model

import (
	"time"

	"github.com/google/uuid"
)

// WorkItemRelation represents a relationship between two work items.
type WorkItemRelation struct {
	ID           uuid.UUID `json:"id"`
	SourceID     uuid.UUID `json:"source_id"`
	TargetID     uuid.UUID `json:"target_id"`
	RelationType string    `json:"relation_type"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// Relation type constants.
const (
	RelationBlocks     = "blocks"
	RelationBlockedBy  = "blocked_by"
	RelationRelatesTo  = "relates_to"
	RelationDuplicates = "duplicates"
	RelationCausedBy   = "caused_by"
	RelationParentOf   = "parent_of"
	RelationChildOf    = "child_of"
)
