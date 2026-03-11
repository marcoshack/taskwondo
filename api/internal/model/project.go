package model

import (
	"time"

	"github.com/google/uuid"
)

// Project member roles
const (
	ProjectRoleOwner  = "owner"
	ProjectRoleAdmin  = "admin"
	ProjectRoleMember = "member"
	ProjectRoleViewer = "viewer"
)

// Project represents a top-level organizational unit.
type Project struct {
	ID                      uuid.UUID  `json:"id"`
	Name                    string     `json:"name"`
	Key                     string     `json:"key"`
	Description             *string    `json:"description,omitempty"`
	NamespaceID             *uuid.UUID `json:"namespace_id,omitempty"`
	DefaultWorkflowID       *uuid.UUID `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues []int                `json:"allowed_complexity_values"`
	BusinessHours           *BusinessHoursConfig `json:"business_hours,omitempty"`
	ItemCounter             int                  `json:"item_counter"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	DeletedAt               *time.Time `json:"-"`
}

// ProjectNamespaceInfo maps a project key to its namespace slug, display name, icon, and color.
type ProjectNamespaceInfo struct {
	ProjectKey     string `json:"project_key"`
	NamespaceSlug  string `json:"namespace_slug"`
	NamespaceName  string `json:"namespace_name"`
	NamespaceIcon  string `json:"namespace_icon"`
	NamespaceColor string `json:"namespace_color"`
}

// ProjectSummary holds aggregate counts for a project list view.
type ProjectSummary struct {
	MemberCount     int `json:"member_count"`
	OpenCount       int `json:"open_count"`
	InProgressCount int `json:"in_progress_count"`
}

// ProjectWithSummary combines a project with its summary counts.
type ProjectWithSummary struct {
	Project
	ProjectSummary
}

// ProjectMember associates a user with a project and their role within it.
type ProjectMember struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// ProjectMemberWithUser includes user details alongside the membership.
type ProjectMemberWithUser struct {
	ProjectMember
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// ProjectTypeWorkflow maps a work item type to a specific workflow within a project.
type ProjectTypeWorkflow struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    uuid.UUID `json:"project_id"`
	WorkItemType string    `json:"work_item_type"`
	WorkflowID   uuid.UUID `json:"workflow_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProjectMemberWithProject includes project details alongside the membership.
type ProjectMemberWithProject struct {
	ProjectMember
	ProjectName string `json:"project_name"`
	ProjectKey  string `json:"project_key"`
	OwnerCount  int    `json:"owner_count"`
}

// ProjectInvite represents a shareable invite link to join a project.
type ProjectInvite struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	Code          string     `json:"code"`
	Role          string     `json:"role"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedByName string     `json:"created_by_name"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	MaxUses       int        `json:"max_uses"`
	UseCount      int        `json:"use_count"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ProjectInviteInfo is a public-facing view of an invite for the join page.
type ProjectInviteInfo struct {
	ProjectName string `json:"project_name"`
	ProjectKey  string `json:"project_key"`
	Role        string `json:"role"`
	Expired     bool   `json:"expired"`
	Full        bool   `json:"full"`
}
