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

// DefaultBrandName is the fallback display name for the default namespace when no brand is configured.
const DefaultBrandName = "Taskwondo"

// Valid namespace icons.
var ValidNamespaceIcons = map[string]bool{
	"building2":  true,
	"users":      true,
	"briefcase":  true,
	"code":       true,
	"rocket":     true,
	"shield":     true,
	"heart":      true,
	"zap":        true,
	"book-open":  true,
	"star":       true,
	"layers":     true,
	"compass":    true,
	"target":     true,
	"lightbulb":  true,
	"globe":      true,
	"palette":    true,
	"cpu":        true,
	"leaf":       true,
	"music":      true,
	"anchor":     true,
}

// Valid namespace colors.
var ValidNamespaceColors = map[string]bool{
	"slate":  true,
	"red":    true,
	"orange": true,
	"amber":  true,
	"green":  true,
	"teal":   true,
	"blue":   true,
	"indigo": true,
	"purple": true,
	"pink":   true,
}

// Namespace represents a tenant grouping that scopes project key uniqueness.
type Namespace struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
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
