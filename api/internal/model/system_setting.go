package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Well-known system setting keys.
const (
	SettingMaxProjectsPerUser   = "max_projects_per_user"
	SettingDefaultTypeWorkflows = "default_type_workflows"
	SettingSMTPConfig           = "smtp_config"
)

// SMTPEncryption constants for the Encryption field of SMTPConfig.
const (
	SMTPEncryptionSTARTTLS = "starttls"
	SMTPEncryptionTLS      = "tls"
	SMTPEncryptionNone     = "none"
)

// PasswordMask is the placeholder returned in API responses instead of the real password.
const PasswordMask = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022"

// SMTPConfig holds SMTP and IMAP configuration stored as a system setting.
type SMTPConfig struct {
	Enabled     bool   `json:"enabled"`
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	IMAPHost    string `json:"imap_host"`
	IMAPPort    int    `json:"imap_port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Encryption  string `json:"encryption"` // "starttls", "tls", "none"
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
}

// Validate checks that all required fields are present when SMTP is enabled.
func (c *SMTPConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SMTPHost == "" {
		return fmt.Errorf("%w: smtp_host is required when enabled", ErrValidation)
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return fmt.Errorf("%w: smtp_port must be between 1 and 65535", ErrValidation)
	}
	if c.Username == "" {
		return fmt.Errorf("%w: username is required when enabled", ErrValidation)
	}
	if c.FromAddress == "" {
		return fmt.Errorf("%w: from_address is required when enabled", ErrValidation)
	}
	switch c.Encryption {
	case SMTPEncryptionSTARTTLS, SMTPEncryptionTLS, SMTPEncryptionNone:
		// valid
	default:
		return fmt.Errorf("%w: encryption must be one of: starttls, tls, none", ErrValidation)
	}
	return nil
}

// DefaultMaxProjectsPerUser is the fallback when the setting is not configured.
const DefaultMaxProjectsPerUser = 5

// SystemSetting stores a global key-value setting (not scoped to any user or project).
type SystemSetting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}
