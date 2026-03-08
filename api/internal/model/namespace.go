package model

import (
	"time"

	"github.com/google/uuid"
)

// Namespace member roles.
const (
	NamespaceRoleOwner  = "owner"
	NamespaceRoleAdmin  = "admin"
	NamespaceRoleMember = "member"
)

// DefaultNamespaceSlug is the slug of the built-in default namespace.
const DefaultNamespaceSlug = "default"

// Namespace represents a tenant grouping that scopes project key uniqueness.
type Namespace struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	IsDefault   bool      `json:"is_default"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NamespaceMember associates a user with a namespace and their role within it.
type NamespaceMember struct {
	NamespaceID uuid.UUID `json:"namespace_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// NamespaceMemberWithUser includes user details alongside the namespace membership.
type NamespaceMemberWithUser struct {
	NamespaceMember
	DisplayName string  `json:"display_name"`
	Email       string  `json:"email"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}
