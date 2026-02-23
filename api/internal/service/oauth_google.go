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

// GoogleProvider implements OAuthProvider for Google.
type GoogleProvider struct {
	clientID    string
	secret      string
	redirectURI string
	httpClient  *http.Client
	baseURL     string // overridable for tests; defaults to "https://accounts.google.com"
	apiBaseURL  string // overridable for tests; defaults to "https://www.googleapis.com"
	tokenURL    string // overridable for tests; defaults to "https://oauth2.googleapis.com"
}

// NewGoogleProvider creates a Google OAuth provider.
func NewGoogleProvider(clientID, secret, redirectURI string, httpClient *http.Client) *GoogleProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GoogleProvider{
		clientID:    clientID,
		secret:      secret,
		redirectURI: redirectURI,
		httpClient:  httpClient,
		baseURL:     "https://accounts.google.com",
		apiBaseURL:  "https://www.googleapis.com",
		tokenURL:    "https://oauth2.googleapis.com",
	}
}

func (p *GoogleProvider) Name() string { return model.OAuthProviderGoogle }

func (p *GoogleProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {p.clientID},
		"redirect_uri":  {p.redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
	}
	return p.baseURL + "/o/oauth2/v2/auth?" + params.Encode()
}

func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error) {
	accessToken, err := p.exchangeCode(ctx, code)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("exchanging code: %w", err)
	}

	googleUser, err := p.fetchUser(ctx, accessToken)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("fetching google user: %w", err)
	}

	return model.OAuthUserInfo{
		ProviderUserID: googleUser.ID,
		Email:          googleUser.Email,
		EmailVerified:  googleUser.VerifiedEmail,
		DisplayName:    googleUser.Name,
		AvatarURL:      googleUser.Picture,
		Username:       googleUser.Email,
	}, nil
}

func (p *GoogleProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.tokenURL+"/token",
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
		return "", fmt.Errorf("google token error: status=%d", resp.StatusCode)
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

func (p *GoogleProvider) fetchUser(ctx context.Context, accessToken string) (*model.GoogleUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.apiBaseURL+"/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google user error: status=%d", resp.StatusCode)
	}

	var googleUser model.GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("decoding google user: %w", err)
	}

	return &googleUser, nil
}
