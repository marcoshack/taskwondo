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

// WorkItemRelationWithDetails is a relation enriched with display info from JOINed work items and projects.
type WorkItemRelationWithDetails struct {
	WorkItemRelation
	SourceProjectKey string
	SourceItemNumber int
	SourceTitle      string
	TargetProjectKey string
	TargetItemNumber int
	TargetTitle      string
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
