package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/marcoshack/taskwondo/internal/model"
)

// GitHubProvider implements OAuthProvider for GitHub.
type GitHubProvider struct {
	clientID    string
	secret      string
	redirectURI string
	httpClient  *http.Client
	baseURL     string // overridable for tests; defaults to "https://github.com"
	apiBaseURL  string // overridable for tests; defaults to "https://api.github.com"
}

// NewGitHubProvider creates a GitHub OAuth provider.
func NewGitHubProvider(clientID, secret, redirectURI string, httpClient *http.Client) *GitHubProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GitHubProvider{
		clientID:    clientID,
		secret:      secret,
		redirectURI: redirectURI,
		httpClient:  httpClient,
		baseURL:     "https://github.com",
		apiBaseURL:  "https://api.github.com",
	}
}

func (p *GitHubProvider) Name() string { return model.OAuthProviderGitHub }

func (p *GitHubProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":    {p.clientID},
		"redirect_uri": {p.redirectURI},
		"scope":        {"user:email"},
		"state":        {state},
	}
	return p.baseURL + "/login/oauth/authorize?" + params.Encode()
}

func (p *GitHubProvider) ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error) {
	accessToken, err := p.exchangeCode(ctx, code)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("exchanging code: %w", err)
	}

	ghUser, err := p.fetchUser(ctx, accessToken)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("fetching github user: %w", err)
	}

	info := githubToOAuthUserInfo(ghUser)

	// If the user profile doesn't have a public email, fetch from /user/emails
	if info.Email == "" {
		emails, err := p.fetchEmails(ctx, accessToken)
		if err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					info.Email = e.Email
					info.EmailVerified = true
					break
				}
			}
		}
	}

	return info, nil
}

func (p *GitHubProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"code":          {code},
		"redirect_uri":  {p.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/login/oauth/access_token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github token error: status=%d", resp.StatusCode)
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

func (p *GitHubProvider) fetchUser(ctx context.Context, accessToken string) (*model.GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.apiBaseURL+"/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user error: status=%d", resp.StatusCode)
	}

	var ghUser model.GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decoding github user: %w", err)
	}

	return &ghUser, nil
}

func (p *GitHubProvider) fetchEmails(ctx context.Context, accessToken string) ([]model.GitHubEmail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.apiBaseURL+"/user/emails", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github emails request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github emails error: status=%d", resp.StatusCode)
	}

	var emails []model.GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("decoding github emails: %w", err)
	}

	return emails, nil
}

func githubToOAuthUserInfo(u *model.GitHubUser) model.OAuthUserInfo {
	info := model.OAuthUserInfo{
		ProviderUserID: strconv.FormatInt(u.ID, 10),
		AvatarURL:      u.AvatarURL,
		Username:       u.Login,
	}
	if u.Name != nil && *u.Name != "" {
		info.DisplayName = *u.Name
	} else {
		info.DisplayName = u.Login
	}
	if u.Email != nil && *u.Email != "" {
		info.Email = *u.Email
		info.EmailVerified = true
	}
	return info
}
