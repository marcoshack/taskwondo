package model

import (
	"time"

	"github.com/google/uuid"
)

// AdminProject is an admin-facing view of a project with aggregated counts.
type AdminProject struct {
	ID                   uuid.UUID `json:"id"`
	Key                  string    `json:"key"`
	Name                 string    `json:"name"`
	NamespaceSlug        string    `json:"namespace_slug"`
	NamespaceDisplayName string    `json:"namespace_display_name"`
	OwnerDisplayName     string    `json:"owner_display_name"`
	OwnerEmail           string    `json:"owner_email"`
	MemberCount          int       `json:"member_count"`
	ItemCount            int       `json:"item_count"`
	StorageBytes         int64     `json:"storage_bytes"`
	CreatedAt            time.Time `json:"created_at"`
}

// AdminProjectList is the paginated result for admin project listings.
type AdminProjectList struct {
	Items   []AdminProject `json:"items"`
	Cursor  string         `json:"cursor"`
	HasMore bool           `json:"has_more"`
}

// AdminNamespace is an admin-facing view of a namespace with aggregated counts.
type AdminNamespace struct {
	ID           uuid.UUID `json:"id"`
	Slug         string    `json:"slug"`
	DisplayName  string    `json:"display_name"`
	IsDefault    bool      `json:"is_default"`
	ProjectCount int       `json:"project_count"`
	MemberCount  int       `json:"member_count"`
	StorageBytes int64     `json:"storage_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

// AdminNamespaceList is the paginated result for admin namespace listings.
type AdminNamespaceList struct {
	Items   []AdminNamespace `json:"items"`
	Cursor  string           `json:"cursor"`
	HasMore bool             `json:"has_more"`
}

// AdminStats contains aggregated system-wide counts for the admin dashboard.
type AdminStats struct {
	Projects     int   `json:"projects"`
	Namespaces   int   `json:"namespaces"`
	Users        int   `json:"users"`
	StorageBytes int64 `json:"storage_bytes"`
}
