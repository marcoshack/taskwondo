package model

import (
	"time"

	"github.com/google/uuid"
)

// Global roles
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User represents an internal user of the system.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	PasswordHash string     `json:"-"`
	GlobalRole   string     `json:"global_role"`
	AvatarURL           *string    `json:"avatar_url,omitempty"`
	IsActive            bool       `json:"is_active"`
	ForcePasswordChange bool       `json:"force_password_change"`
	MaxProjects         *int       `json:"max_projects,omitempty"`
	MaxNamespaces       *int       `json:"max_namespaces,omitempty"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// APIKey represents an API key for programmatic access (user or system).
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
