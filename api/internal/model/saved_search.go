package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SavedSearchScopeUser   = "user"
	SavedSearchScopeShared = "shared"
)

// SavedSearchFilters holds the filter criteria persisted as JSONB.
type SavedSearchFilters struct {
	Search    string   `json:"q,omitempty"`
	Type      []string `json:"type,omitempty"`
	Status    []string `json:"status,omitempty"`
	Priority  []string `json:"priority,omitempty"`
	Assignee  []string `json:"assignee,omitempty"`
	Milestone []string `json:"milestone,omitempty"`
}

// SavedSearch represents a persisted filter+view configuration.
type SavedSearch struct {
	ID        uuid.UUID          `json:"id"`
	ProjectID uuid.UUID          `json:"project_id"`
	UserID    *uuid.UUID         `json:"user_id,omitempty"`
	Name      string             `json:"name"`
	Filters   SavedSearchFilters `json:"filters"`
	ViewMode  string             `json:"view_mode"`
	Position  int                `json:"position"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// Scope returns "user" or "shared".
func (s *SavedSearch) Scope() string {
	if s.UserID != nil {
		return SavedSearchScopeUser
	}
	return SavedSearchScopeShared
}
