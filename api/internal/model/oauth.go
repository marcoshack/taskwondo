package model

import (
	"time"

	"github.com/google/uuid"
)

// OAuth provider constants.
const (
	OAuthProviderDiscord = "discord"
	OAuthProviderGoogle  = "google"
	OAuthProviderGitHub  = "github"
)

// OAuthAccount represents a linked external identity.
type OAuthAccount struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	Provider         string    `json:"provider"`
	ProviderUserID   string    `json:"provider_user_id"`
	ProviderEmail    string    `json:"provider_email,omitempty"`
	ProviderUsername string    `json:"provider_username,omitempty"`
	ProviderAvatar   string    `json:"provider_avatar,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// OAuthUserInfo holds normalized user data from any OAuth provider.
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
	Username       string // provider-specific (for OAuthAccount.ProviderUsername)
	RawAvatar      string // provider-specific (for OAuthAccount.ProviderAvatar)
}

// GoogleUser is the response from Google's userinfo endpoint.
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// DiscordUser is the response from Discord's /users/@me endpoint.
type DiscordUser struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	GlobalName    *string `json:"global_name"`
	Avatar        *string `json:"avatar"`
	Email         *string `json:"email"`
	Verified      *bool   `json:"verified"`
}

// AvatarURL returns the full CDN URL for the Discord avatar.
func (d *DiscordUser) AvatarURL() string {
	if d.Avatar == nil || *d.Avatar == "" {
		return "https://cdn.discordapp.com/embed/avatars/0.png"
	}
	return "https://cdn.discordapp.com/avatars/" + d.ID + "/" + *d.Avatar + ".png"
}

// DisplayName returns the best available display name.
func (d *DiscordUser) DisplayName() string {
	if d.GlobalName != nil && *d.GlobalName != "" {
		return *d.GlobalName
	}
	return d.Username
}

// GitHubUser is the response from GitHub's /user endpoint.
type GitHubUser struct {
	ID        int64   `json:"id"`
	Login     string  `json:"login"`
	Name      *string `json:"name"`
	Email     *string `json:"email"`
	AvatarURL string  `json:"avatar_url"`
}

// GitHubEmail is a single entry from GitHub's /user/emails endpoint.
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}
