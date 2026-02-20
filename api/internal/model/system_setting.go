package model

import (
	"encoding/json"
	"time"
)

// SystemSetting stores a global key-value setting (not scoped to any user or project).
type SystemSetting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}
