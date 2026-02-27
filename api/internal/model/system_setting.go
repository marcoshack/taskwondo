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

	// Authentication settings
	SettingAuthEmailLoginEnabled        = "auth_email_login_enabled"
	SettingAuthEmailRegistrationEnabled = "auth_email_registration_enabled"
	SettingAuthDiscordEnabled           = "auth_discord_enabled"
	SettingAuthGoogleEnabled            = "auth_google_enabled"

	// OAuth provider configuration
	SettingOAuthDiscordConfig = "oauth_discord_config"
	SettingOAuthGoogleConfig  = "oauth_google_config"

	// Feature flags
	SettingFeatureStatsTimeline = "feature_stats_timeline"
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

// OAuthProviderConfig holds OAuth provider credentials stored as a system setting.
// The enabled/disabled state is stored separately in auth_*_enabled settings.
type OAuthProviderConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

// Validate checks that all required fields are present.
func (c *OAuthProviderConfig) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("%w: client_id is required", ErrValidation)
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("%w: client_secret is required", ErrValidation)
	}
	if c.RedirectURI == "" {
		return fmt.Errorf("%w: redirect_uri is required", ErrValidation)
	}
	return nil
}

// OAuthConfigSettingKey returns the system setting key for a given provider name.
func OAuthConfigSettingKey(provider string) string {
	switch provider {
	case OAuthProviderDiscord:
		return SettingOAuthDiscordConfig
	case OAuthProviderGoogle:
		return SettingOAuthGoogleConfig
	default:
		return ""
	}
}

// DefaultMaxProjectsPerUser is the fallback when the setting is not configured.
const DefaultMaxProjectsPerUser = 5

// SystemSetting stores a global key-value setting (not scoped to any user or project).
type SystemSetting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}
