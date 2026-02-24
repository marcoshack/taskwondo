package model

import (
	"encoding/json"
	"time"
)

// Well-known system setting keys.
const (
	SettingMaxProjectsPerUser     = "max_projects_per_user"
	SettingDefaultTypeWorkflows   = "default_type_workflows"
)

// DefaultMaxProjectsPerUser is the fallback when the setting is not configured.
const DefaultMaxProjectsPerUser = 5

// SystemSetting stores a global key-value setting (not scoped to any user or project).
type SystemSetting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}
