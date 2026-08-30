package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Well-known system setting keys.
const (
	SettingMaxProjectsPerUser    = "max_projects_per_user"
	SettingMaxNamespacesPerUser = "max_namespaces_per_user"
	SettingDefaultTypeWorkflows = "default_type_workflows"
	SettingSMTPConfig           = "smtp_config"

	// SettingMaxUploadSize is published with the public settings but is sourced
	// from the MAX_UPLOAD_SIZE env config, not the settings table.
	SettingMaxUploadSize = "max_upload_size"

	// Authentication settings
	SettingAuthEmailLoginEnabled        = "auth_email_login_enabled"
	SettingAuthEmailRegistrationEnabled = "auth_email_registration_enabled"
	SettingAuthDiscordEnabled           = "auth_discord_enabled"
	SettingAuthGoogleEnabled            = "auth_google_enabled"
	SettingAuthGitHubEnabled            = "auth_github_enabled"
	SettingAuthMicrosoftEnabled         = "auth_microsoft_enabled"
	SettingAuthSSOEnabled               = "auth_sso_enabled"

	// SettingSSOAutoProvision gates account creation for SSO logins whose email
	// does not match an existing user. When false (the default), SSO can only
	// sign in users that already exist.
	SettingSSOAutoProvision = "sso_auto_provision_enabled"

	// OAuth provider ordering (JSON array of provider names, e.g. ["discord","google","github"])
	SettingOAuthProviderOrder = "oauth_provider_order"

	// OAuth provider configuration
	SettingOAuthDiscordConfig = "oauth_discord_config"
	SettingOAuthGoogleConfig  = "oauth_google_config"
	SettingOAuthGitHubConfig     = "oauth_github_config"
	SettingOAuthMicrosoftConfig  = "oauth_microsoft_config"
	SettingOAuthSSOConfig        = "oauth_sso_config"

	// Deny lists (JSON arrays of strings)
	SettingReservedNamespaceSlugs = "reserved_namespace_slugs"
	SettingReservedProjectKeys   = "reserved_project_keys"

	// Feature flags
	SettingFeatureStatsTimeline  = "feature_stats_timeline"
	SettingFeatureSemanticSearch = "feature_semantic_search"
	SettingOllamaAvailable      = "ollama_available"
	SettingNamespacesEnabled     = "namespaces_enabled"
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
	// SkipCertVerify disables TLS certificate verification for SMTP connections.
	// Intended for self-hosted servers with self-signed or expired certificates.
	SkipCertVerify bool `json:"skip_cert_verify"`
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
// The redirect URI is derived automatically from BaseURL + "/auth/{provider}/callback".
//
// The Issuer/Scopes/ButtonLabel/DisablePKCE/RequireVerifiedEmail fields are only
// used by the generic SSO provider (OAuthProviderSSO). They are omitted from the
// stored JSON for the built-in providers, whose behaviour is unchanged.
type OAuthProviderConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	// Issuer is the OIDC issuer URL used for discovery (SSO only).
	Issuer string `json:"issuer,omitempty"`
	// Scopes overrides the requested scopes; empty means DefaultSSOScopes.
	Scopes []string `json:"scopes,omitempty"`
	// ButtonLabel overrides the login button text (SSO only).
	ButtonLabel string `json:"button_label,omitempty"`
	// DisablePKCE turns off the S256 code challenge for IdPs that reject it.
	DisablePKCE bool `json:"disable_pkce,omitempty"`
	// RequireVerifiedEmail gates logins on the email_verified claim. nil = required.
	RequireVerifiedEmail *bool `json:"require_verified_email,omitempty"`
}

// DefaultSSOScopes are requested when Scopes is empty.
var DefaultSSOScopes = []string{"openid", "profile", "email"}

// MaxSSOButtonLabel is the length cap for the login button override.
const MaxSSOButtonLabel = 40

// RequiresVerifiedEmail reports whether the email_verified claim must be true.
func (c *OAuthProviderConfig) RequiresVerifiedEmail() bool {
	return c.RequireVerifiedEmail == nil || *c.RequireVerifiedEmail
}

// ScopeList returns the configured scopes or the defaults.
func (c *OAuthProviderConfig) ScopeList() []string {
	if len(c.Scopes) == 0 {
		return DefaultSSOScopes
	}
	return c.Scopes
}

// Validate checks that all required fields are present.
func (c *OAuthProviderConfig) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("%w: client_id is required", ErrValidation)
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("%w: client_secret is required", ErrValidation)
	}
	return nil
}

// ValidateAs validates the config in the context of a specific provider,
// enforcing the extra fields the generic SSO provider needs.
func (c *OAuthProviderConfig) ValidateAs(provider string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if provider != OAuthProviderSSO {
		return nil
	}

	issuer, err := NormalizeOIDCIssuer(c.Issuer)
	if err != nil {
		return err
	}
	c.Issuer = issuer

	label := strings.TrimSpace(c.ButtonLabel)
	if len([]rune(label)) > MaxSSOButtonLabel {
		return fmt.Errorf("%w: button_label must be %d characters or fewer", ErrValidation, MaxSSOButtonLabel)
	}
	c.ButtonLabel = label

	for _, s := range c.Scopes {
		if s == "" || strings.ContainsAny(s, " \t") {
			return fmt.Errorf("%w: scopes must be individual non-empty strings", ErrValidation)
		}
	}
	if len(c.Scopes) > 0 && !containsString(c.Scopes, "openid") {
		return fmt.Errorf("%w: scopes must include openid", ErrValidation)
	}
	return nil
}

// NormalizeOIDCIssuer validates and canonicalises an OIDC issuer URL.
// Issuers are compared exactly by the discovery and verification code, so
// trailing slashes are stripped and only https (or http for local dev) is kept.
func NormalizeOIDCIssuer(raw string) (string, error) {
	issuer := strings.TrimSpace(raw)
	if issuer == "" {
		return "", fmt.Errorf("%w: issuer is required", ErrValidation)
	}
	u, err := url.Parse(issuer)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%w: issuer must be an absolute URL", ErrValidation)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || strings.HasPrefix(u.Host, "127."))) {
		return "", fmt.Errorf("%w: issuer must use https", ErrValidation)
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%w: issuer must not contain credentials, query or fragment", ErrValidation)
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// KnownOAuthProviders lists every provider whose credentials live in an
// oauth_<name>_config setting and whose switch is an auth_<name>_enabled
// setting. Login-page ordering and enablement are driven off this list.
var KnownOAuthProviders = []string{
	OAuthProviderDiscord,
	OAuthProviderGoogle,
	OAuthProviderGitHub,
	OAuthProviderMicrosoft,
	OAuthProviderSSO,
}

// OAuthConfigSettingKey returns the system setting key for a given provider name.
func OAuthConfigSettingKey(provider string) string {
	switch provider {
	case OAuthProviderDiscord:
		return SettingOAuthDiscordConfig
	case OAuthProviderGoogle:
		return SettingOAuthGoogleConfig
	case OAuthProviderGitHub:
		return SettingOAuthGitHubConfig
	case OAuthProviderMicrosoft:
		return SettingOAuthMicrosoftConfig
	case OAuthProviderSSO:
		return SettingOAuthSSOConfig
	default:
		return ""
	}
}

// OAuthEnabledSettingKey returns the auth_<provider>_enabled setting key for a
// provider name, or empty string for unknown providers.
func OAuthEnabledSettingKey(provider string) string {
	switch provider {
	case OAuthProviderDiscord:
		return SettingAuthDiscordEnabled
	case OAuthProviderGoogle:
		return SettingAuthGoogleEnabled
	case OAuthProviderGitHub:
		return SettingAuthGitHubEnabled
	case OAuthProviderMicrosoft:
		return SettingAuthMicrosoftEnabled
	case OAuthProviderSSO:
		return SettingAuthSSOEnabled
	default:
		return ""
	}
}

// OAuthEnabledToConfigKey maps an auth_*_enabled setting key to its corresponding
// oauth_*_config setting key. Returns empty string for non-OAuth enabled keys.
func OAuthEnabledToConfigKey(enabledKey string) string {
	switch enabledKey {
	case SettingAuthDiscordEnabled:
		return SettingOAuthDiscordConfig
	case SettingAuthGoogleEnabled:
		return SettingOAuthGoogleConfig
	case SettingAuthGitHubEnabled:
		return SettingOAuthGitHubConfig
	case SettingAuthMicrosoftEnabled:
		return SettingOAuthMicrosoftConfig
	case SettingAuthSSOEnabled:
		return SettingOAuthSSOConfig
	default:
		return ""
	}
}

// DefaultMaxProjectsPerUser is the fallback when the setting is not configured.
const DefaultMaxProjectsPerUser = 5

// DefaultMaxNamespacesPerUser is the fallback when the setting is not configured.
const DefaultMaxNamespacesPerUser = 1

// SystemSetting stores a global key-value setting (not scoped to any user or project).
type SystemSetting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}
