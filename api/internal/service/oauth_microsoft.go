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

// MicrosoftProvider implements OAuthProvider for Microsoft (Azure AD / Entra ID).
// Uses the "common" tenant to support both personal Microsoft accounts and org accounts.
type MicrosoftProvider struct {
	clientID    string
	secret      string
	redirectURI string
	httpClient  *http.Client
	authBaseURL string // overridable for tests; defaults to "https://login.microsoftonline.com"
	graphURL    string // overridable for tests; defaults to "https://graph.microsoft.com"
}

// NewMicrosoftProvider creates a Microsoft OAuth provider.
func NewMicrosoftProvider(clientID, secret, redirectURI string, httpClient *http.Client) *MicrosoftProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &MicrosoftProvider{
		clientID:    clientID,
		secret:      secret,
		redirectURI: redirectURI,
		httpClient:  httpClient,
		authBaseURL: "https://login.microsoftonline.com",
		graphURL:    "https://graph.microsoft.com",
	}
}

func (p *MicrosoftProvider) Name() string { return model.OAuthProviderMicrosoft }

func (p *MicrosoftProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {p.clientID},
		"redirect_uri":  {p.redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile User.Read"},
		"state":         {state},
	}
	return p.authBaseURL + "/common/oauth2/v2.0/authorize?" + params.Encode()
}

func (p *MicrosoftProvider) ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error) {
	accessToken, err := p.exchangeCode(ctx, code)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("exchanging code: %w", err)
	}

	msUser, err := p.fetchUser(ctx, accessToken)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("fetching microsoft user: %w", err)
	}

	return microsoftToOAuthUserInfo(msUser), nil
}

func (p *MicrosoftProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"code":          {code},
		"redirect_uri":  {p.redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.authBaseURL+"/common/oauth2/v2.0/token",
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
		return "", fmt.Errorf("microsoft token error: status=%d", resp.StatusCode)
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

func (p *MicrosoftProvider) fetchUser(ctx context.Context, accessToken string) (*model.MicrosoftUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.graphURL+"/v1.0/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoft user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("microsoft user error: status=%d", resp.StatusCode)
	}

	var msUser model.MicrosoftUser
	if err := json.NewDecoder(resp.Body).Decode(&msUser); err != nil {
		return nil, fmt.Errorf("decoding microsoft user: %w", err)
	}

	return &msUser, nil
}

func microsoftToOAuthUserInfo(u *model.MicrosoftUser) model.OAuthUserInfo {
	info := model.OAuthUserInfo{
		ProviderUserID: u.ID,
		DisplayName:    u.DisplayName,
	}

	// Microsoft Graph returns email in "mail" field, or falls back to userPrincipalName
	if u.Mail != "" {
		info.Email = u.Mail
		info.EmailVerified = true
	} else if u.UserPrincipalName != "" && strings.Contains(u.UserPrincipalName, "@") {
		info.Email = u.UserPrincipalName
		info.EmailVerified = true
	}

	return info
}
