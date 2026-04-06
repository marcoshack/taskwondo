package model

import (
	"time"

	"github.com/google/uuid"
)

// OncallRotation represents an on-call rotation configuration for a team.
type OncallRotation struct {
	ID              uuid.UUID  `json:"id"`
	TeamID          uuid.UUID  `json:"team_id"`
	PeriodDays      int        `json:"period_days"`
	RotationTime    string     `json:"rotation_time"`
	Timezone        string     `json:"timezone"`
	StartDate       string     `json:"start_date"`
	CurrentUserID   *uuid.UUID `json:"current_user_id"`
	CurrentPosition int        `json:"current_position"`
	NextRotationAt  *time.Time `json:"next_rotation_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// OncallRotationMember associates a user with a position in the rotation.
type OncallRotationMember struct {
	ID         uuid.UUID `json:"id"`
	RotationID uuid.UUID `json:"rotation_id"`
	UserID     uuid.UUID `json:"user_id"`
	Position   int       `json:"position"`
}

// OncallRotationMemberWithUser includes user details alongside the rotation membership.
type OncallRotationMemberWithUser struct {
	OncallRotationMember
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// OncallRotationHistory records a user's on-call shift.
type OncallRotationHistory struct {
	ID         uuid.UUID  `json:"id"`
	RotationID uuid.UUID  `json:"rotation_id"`
	UserID     uuid.UUID  `json:"user_id"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// OncallRotationHistoryWithUser includes user details alongside the history entry.
type OncallRotationHistoryWithUser struct {
	OncallRotationHistory
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// OncallOverride represents a temporary on-call override for a rotation.
type OncallOverride struct {
	ID             uuid.UUID `json:"id"`
	RotationID     uuid.UUID `json:"rotation_id"`
	OverrideUserID uuid.UUID `json:"override_user_id"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	Reason         *string   `json:"reason,omitempty"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// OncallOverrideWithUser includes user details alongside the override.
type OncallOverrideWithUser struct {
	OncallOverride
	OverrideUserName string  `json:"override_user_name"`
	OverrideAvatar   *string `json:"override_avatar_url,omitempty"`
	CreatedByName    string  `json:"created_by_name"`
}

// OncallOverrideCreatedEvent is published when an on-call override is created.
type OncallOverrideCreatedEvent struct {
	OverrideID     uuid.UUID `json:"override_id"`
	RotationID     uuid.UUID `json:"rotation_id"`
	TeamID         uuid.UUID `json:"team_id"`
	TeamName       string    `json:"team_name"`
	OverrideUserID uuid.UUID `json:"override_user_id"`
	ScheduledUser  uuid.UUID `json:"scheduled_user_id"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	Reason         *string   `json:"reason,omitempty"`
}

// OncallOverrideCancelledEvent is published when an on-call override is cancelled.
type OncallOverrideCancelledEvent struct {
	OverrideID     uuid.UUID `json:"override_id"`
	RotationID     uuid.UUID `json:"rotation_id"`
	TeamID         uuid.UUID `json:"team_id"`
	TeamName       string    `json:"team_name"`
	OverrideUserID uuid.UUID `json:"override_user_id"`
	ScheduledUser  uuid.UUID `json:"scheduled_user_id"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
}

// OncallRotationAdvancedEvent is published when an on-call rotation advances.
type OncallRotationAdvancedEvent struct {
	RotationID     uuid.UUID  `json:"rotation_id"`
	TeamID         uuid.UUID  `json:"team_id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	TeamName       string     `json:"team_name"`
	OldUserID      uuid.UUID  `json:"old_user_id"`
	NewUserID      uuid.UUID  `json:"new_user_id"`
	NextRotationAt *time.Time `json:"next_rotation_at"`
}
