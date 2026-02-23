package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/marcoshack/taskwondo/internal/model"
)

// DiscordProvider implements OAuthProvider for Discord.
type DiscordProvider struct {
	clientID    string
	secret      string
	redirectURI string
	httpClient  *http.Client
	baseURL     string // overridable for tests; defaults to "https://discord.com"
}

// NewDiscordProvider creates a Discord OAuth provider.
func NewDiscordProvider(clientID, secret, redirectURI string, httpClient *http.Client) *DiscordProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &DiscordProvider{
		clientID:    clientID,
		secret:      secret,
		redirectURI: redirectURI,
		httpClient:  httpClient,
		baseURL:     "https://discord.com",
	}
}

func (p *DiscordProvider) Name() string { return model.OAuthProviderDiscord }

func (p *DiscordProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {p.clientID},
		"redirect_uri":  {p.redirectURI},
		"response_type": {"code"},
		"scope":         {"identify email"},
		"state":         {state},
	}
	return p.baseURL + "/oauth2/authorize?" + params.Encode()
}

func (p *DiscordProvider) ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error) {
	accessToken, err := p.exchangeCode(ctx, code)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("exchanging code: %w", err)
	}

	discordUser, err := p.fetchUser(ctx, accessToken)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("fetching discord user: %w", err)
	}

	return discordToOAuthUserInfo(discordUser), nil
}

func (p *DiscordProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/api/oauth2/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discord token error: status=%d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

func (p *DiscordProvider) fetchUser(ctx context.Context, accessToken string) (*model.DiscordUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.baseURL+"/api/v10/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user error: status=%d", resp.StatusCode)
	}

	var discordUser model.DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return nil, fmt.Errorf("decoding discord user: %w", err)
	}

	return &discordUser, nil
}

func discordToOAuthUserInfo(d *model.DiscordUser) model.OAuthUserInfo {
	info := model.OAuthUserInfo{
		ProviderUserID: d.ID,
		DisplayName:    d.DisplayName(),
		AvatarURL:      d.AvatarURL(),
		Username:       d.Username,
	}
	if d.Avatar != nil {
		info.RawAvatar = *d.Avatar
	}
	if d.Email != nil {
		info.Email = *d.Email
	}
	if d.Verified != nil {
		info.EmailVerified = *d.Verified
	}
	return info
}
