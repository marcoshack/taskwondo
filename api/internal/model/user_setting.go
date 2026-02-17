package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserSetting stores a key-value preference for a user, optionally scoped to a project.
type UserSetting struct {
	UserID    uuid.UUID        `json:"user_id"`
	ProjectID *uuid.UUID       `json:"project_id,omitempty"`
	Key       string           `json:"key"`
	Value     json.RawMessage  `json:"value"`
	UpdatedAt time.Time        `json:"updated_at"`
}
