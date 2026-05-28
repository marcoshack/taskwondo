package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/marcoshack/taskwondo/internal/model"
)

// oidcDiscovery holds the relevant fields from an OpenID Connect discovery document.
type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

// OIDCProvider implements OAuthProvider for any standard OpenID Connect provider
// (e.g. Keycloak, Auth0, Okta).
type OIDCProvider struct {
	clientID    string
	secret      string
	redirectURI string
	issuerURL   string
	httpClient  *http.Client

	once      sync.Once
	discovery *oidcDiscovery
	discErr   error
}

// NewOIDCProvider creates an OIDC OAuth provider. It lazily fetches the discovery
// document on the first call to AuthURL or ExchangeCode.
func NewOIDCProvider(clientID, secret, redirectURI, issuerURL string, httpClient *http.Client) *OIDCProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OIDCProvider{
		clientID:    clientID,
		secret:      secret,
		redirectURI: redirectURI,
		issuerURL:   strings.TrimRight(issuerURL, "/"),
		httpClient:  httpClient,
	}
}

func (p *OIDCProvider) Name() string { return model.OAuthProviderOIDC }

func (p *OIDCProvider) AuthURL(state string) string {
	disc, err := p.discover()
	if err != nil {
		// Return a best-effort URL — the error will surface during ExchangeCode.
		return p.issuerURL + "/protocol/openid-connect/auth"
	}

	params := url.Values{
		"client_id":     {p.clientID},
		"redirect_uri":  {p.redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
	}
	return disc.AuthorizationEndpoint + "?" + params.Encode()
}

func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error) {
	disc, err := p.discover()
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("oidc discovery failed: %w", err)
	}

	accessToken, err := p.exchangeCode(ctx, disc, code)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("exchanging code: %w", err)
	}

	userInfo, err := p.fetchUserInfo(ctx, disc, accessToken)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("fetching userinfo: %w", err)
	}

	return userInfo, nil
}

// discover fetches and caches the OIDC discovery document.
func (p *OIDCProvider) discover() (*oidcDiscovery, error) {
	p.once.Do(func() {
		p.discovery, p.discErr = p.fetchDiscovery()
	})
	return p.discovery, p.discErr
}

func (p *OIDCProvider) fetchDiscovery() (*oidcDiscovery, error) {
	discoveryURL := p.issuerURL + "/.well-known/openid-configuration"

	resp, err := p.httpClient.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("fetching discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}

	var disc oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, fmt.Errorf("decoding discovery document: %w", err)
	}

	if disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" || disc.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("discovery document missing required endpoints")
	}

	return &disc, nil
}

func (p *OIDCProvider) exchangeCode(ctx context.Context, disc *oidcDiscovery, code string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		disc.TokenEndpoint,
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
		return "", fmt.Errorf("oidc token error: status=%d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("oidc token response missing access_token")
	}

	return tokenResp.AccessToken, nil
}

func (p *OIDCProvider) fetchUserInfo(ctx context.Context, disc *oidcDiscovery, accessToken string) (model.OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		disc.UserinfoEndpoint, nil)
	if err != nil {
		return model.OAuthUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.OAuthUserInfo{}, fmt.Errorf("userinfo error: status=%d", resp.StatusCode)
	}

	// Standard OIDC userinfo claims
	var claims struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername  string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("decoding userinfo: %w", err)
	}

	if claims.Sub == "" {
		return model.OAuthUserInfo{}, fmt.Errorf("userinfo missing sub claim")
	}

	displayName := claims.Name
	if displayName == "" {
		displayName = claims.PreferredUsername
	}
	if displayName == "" {
		displayName = claims.Email
	}

	return model.OAuthUserInfo{
		ProviderUserID: claims.Sub,
		Email:          claims.Email,
		EmailVerified:  claims.EmailVerified,
		DisplayName:    displayName,
		AvatarURL:      claims.Picture,
		Username:       claims.PreferredUsername,
	}, nil
}
