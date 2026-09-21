package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/model"
)

// ssoStateTTL bounds how long a login may sit at the identity provider.
const ssoStateTTL = 10 * time.Minute

// ssoDiscoveryTTL is how long a discovery document is reused before refetching.
const ssoDiscoveryTTL = time.Hour

// ssoDiscoveryMaxEntries caps the discovery cache so a changed issuer cannot
// grow it without bound.
const ssoDiscoveryMaxEntries = 8

// SSODiscoveryCache memoises OIDC discovery documents. The oidc.Provider value
// also lazily caches its JWKS key set, so reusing one across logins avoids a
// metadata and key fetch on every sign-in. Safe for concurrent use.
type SSODiscoveryCache struct {
	mu      sync.Mutex
	entries map[string]*ssoDiscoveryEntry
	now     func() time.Time
}

type ssoDiscoveryEntry struct {
	provider *oidc.Provider
	expires  time.Time
}

// NewSSODiscoveryCache creates an empty discovery cache.
func NewSSODiscoveryCache() *SSODiscoveryCache {
	return &SSODiscoveryCache{
		entries: make(map[string]*ssoDiscoveryEntry),
		now:     time.Now,
	}
}

// get returns the cached provider for issuer, discovering it when absent or stale.
func (c *SSODiscoveryCache) get(ctx context.Context, issuer string, client *http.Client) (*oidc.Provider, error) {
	c.mu.Lock()
	if e, ok := c.entries[issuer]; ok && c.now().Before(e.expires) {
		provider := e.provider
		c.mu.Unlock()
		return provider, nil
	}
	c.mu.Unlock()

	dctx := ctx
	if client != nil {
		dctx = oidc.ClientContext(ctx, client)
	}
	provider, err := oidc.NewProvider(dctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	c.mu.Lock()
	if len(c.entries) >= ssoDiscoveryMaxEntries {
		c.entries = make(map[string]*ssoDiscoveryEntry)
	}
	c.entries[issuer] = &ssoDiscoveryEntry{provider: provider, expires: c.now().Add(ssoDiscoveryTTL)}
	c.mu.Unlock()

	return provider, nil
}

// invalidate drops the cached document for an issuer so the next attempt
// refetches it. Used when ID-token verification fails, which is what a rotated
// JWKS looks like before the cached keys go stale.
func (c *SSODiscoveryCache) invalidate(issuer string) {
	c.mu.Lock()
	delete(c.entries, issuer)
	c.mu.Unlock()
}

// SSOProvider implements OAuthProvider for a custom OpenID Connect identity
// provider configured by an administrator. Unlike the built-in providers it
// discovers its endpoints from the issuer URL, validates the ID token
// signature, and carries per-login secrets (nonce, PKCE verifier) inside the
// sealed state parameter.
//
// Account identity is resolved by email address in
// AuthService.findOrCreateOAuthUser; new accounts are gated by the
// sso_auto_provision_enabled setting.
type SSOProvider struct {
	cfg        model.OAuthProviderConfig
	redirect   string
	httpClient *http.Client
	sealer     *crypto.Encryptor
	cache      *SSODiscoveryCache
	now        func() time.Time
}

// NewSSOProvider creates a generic OIDC provider. sealer must not be nil: it
// seals the state parameter, and without it a login cannot be bound to the
// browser that started it.
func NewSSOProvider(cfg model.OAuthProviderConfig, redirectURI string, sealer *crypto.Encryptor, cache *SSODiscoveryCache, httpClient *http.Client) *SSOProvider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if cache == nil {
		cache = NewSSODiscoveryCache()
	}
	return &SSOProvider{
		cfg:        cfg,
		redirect:   redirectURI,
		httpClient: httpClient,
		sealer:     sealer,
		cache:      cache,
		now:        time.Now,
	}
}

func (p *SSOProvider) Name() string { return model.OAuthProviderSSO }

// ButtonLabel returns the operator-supplied login button text, if configured.
func (p *SSOProvider) ButtonLabel() string { return p.cfg.ButtonLabel }

// ssoState is the payload sealed into the OIDC state parameter.
type ssoState struct {
	Provider string `json:"p"`
	Nonce    string `json:"n"`
	Verifier string `json:"v,omitempty"`
	Expires  int64  `json:"e"`
}

// NewState seals a fresh nonce and PKCE verifier into the state parameter.
func (p *SSOProvider) NewState(_ context.Context) (string, error) {
	if p.sealer == nil {
		return "", fmt.Errorf("sso provider: state sealer is not configured")
	}
	nonce, err := ssoRandomToken()
	if err != nil {
		return "", fmt.Errorf("generating sso nonce: %w", err)
	}
	state := ssoState{
		Provider: model.OAuthProviderSSO,
		Nonce:    nonce,
		Expires:  p.now().Add(ssoStateTTL).Unix(),
	}
	if !p.cfg.DisablePKCE {
		state.Verifier = oauth2.GenerateVerifier()
	}
	sealed, err := p.sealer.SealJSON(state)
	if err != nil {
		return "", fmt.Errorf("sealing sso state: %w", err)
	}
	return sealed, nil
}

// ValidateState unseals and checks the state parameter, returning a context
// carrying the nonce and verifier for the following ExchangeCode call.
func (p *SSOProvider) ValidateState(ctx context.Context, state string) (context.Context, error) {
	if p.sealer == nil {
		return ctx, fmt.Errorf("sso provider: state sealer is not configured")
	}
	var s ssoState
	if err := p.sealer.OpenJSON(state, &s); err != nil {
		return ctx, fmt.Errorf("decoding state: %w", err)
	}
	if s.Nonce == "" || s.Provider != model.OAuthProviderSSO {
		return ctx, fmt.Errorf("malformed state")
	}
	if p.now().Unix() > s.Expires {
		return ctx, fmt.Errorf("state expired")
	}
	return ssoFlowContext(ctx, ssoFlow{nonce: s.Nonce, verifier: s.Verifier}), nil
}

// AuthURL is unused for SSO: building the URL requires provider discovery, so
// AuthService calls AuthURLContext through the ContextualAuthURL interface.
func (p *SSOProvider) AuthURL(state string) string {
	url, err := p.AuthURLContext(context.Background(), state)
	if err != nil {
		return ""
	}
	return url
}

// AuthURLContext discovers the provider and builds the authorization request,
// attaching the nonce and PKCE challenge recovered from the sealed state.
func (p *SSOProvider) AuthURLContext(ctx context.Context, state string) (string, error) {
	flow, ok := ssoFlowFromContext(ctx)
	if !ok {
		if p.sealer == nil {
			return "", fmt.Errorf("sso provider: state sealer is not configured")
		}
		var s ssoState
		if err := p.sealer.OpenJSON(state, &s); err != nil {
			return "", fmt.Errorf("decoding state: %w", err)
		}
		if s.Provider != model.OAuthProviderSSO || s.Nonce == "" {
			return "", fmt.Errorf("malformed state")
		}
		flow = ssoFlow{nonce: s.Nonce, verifier: s.Verifier}
		ctx = ssoFlowContext(ctx, flow)
	}

	provider, err := p.discover(ctx)
	if err != nil {
		return "", err
	}

	opts := []oauth2.AuthCodeOption{oidc.Nonce(flow.nonce)}
	if flow.verifier != "" {
		opts = append(opts, oauth2.S256ChallengeOption(flow.verifier))
	}
	return p.oauthConfig(provider).AuthCodeURL(state, opts...), nil
}

// ExchangeCode redeems the authorization code, verifies the ID token
// (signature, issuer, audience, expiry, nonce and at_hash) and, when the ID
// token omits the email claim, backfills it from the UserInfo endpoint.
func (p *SSOProvider) ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error) {
	flow, ok := ssoFlowFromContext(ctx)
	if !ok {
		return model.OAuthUserInfo{}, fmt.Errorf("missing sso login context")
	}

	provider, err := p.discover(ctx)
	if err != nil {
		return model.OAuthUserInfo{}, err
	}

	ecfg := p.oauthConfig(provider)
	exchangeCtx := oidc.ClientContext(ctx, p.httpClient)

	var opts []oauth2.AuthCodeOption
	if flow.verifier != "" {
		opts = append(opts, oauth2.VerifierOption(flow.verifier))
	}
	oauth2Token, err := ecfg.Exchange(exchangeCtx, code, opts...)
	if err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("exchanging code: %w", err)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return model.OAuthUserInfo{}, fmt.Errorf("token response did not contain an id_token")
	}

	// Verifier (not VerifierContext) reuses the key set cached on the provider,
	// so JWKS is fetched at most once per discovery refresh.
	idToken, err := provider.Verifier(&oidc.Config{ClientID: p.cfg.ClientID}).Verify(exchangeCtx, rawIDToken)
	if err != nil {
		p.cache.invalidate(p.cfg.Issuer)
		return model.OAuthUserInfo{}, fmt.Errorf("verifying id_token: %w", err)
	}
	if idToken.Nonce != flow.nonce {
		return model.OAuthUserInfo{}, fmt.Errorf("id_token nonce mismatch")
	}
	// at_hash binds the ID token to the access token. Optional per spec, but
	// when present it must match, otherwise a stolen access token is undetectable.
	if idToken.AccessTokenHash != "" {
		if err := idToken.VerifyAccessToken(oauth2Token.AccessToken); err != nil {
			return model.OAuthUserInfo{}, fmt.Errorf("verifying access token hash: %w", err)
		}
	}

	var claims ssoClaims
	if err := idToken.Claims(&claims); err != nil {
		return model.OAuthUserInfo{}, fmt.Errorf("decoding id_token claims: %w", err)
	}

	if claims.Email == "" && provider.UserInfoEndpoint() != "" {
		ui, err := provider.UserInfo(exchangeCtx, oauth2.StaticTokenSource(oauth2Token))
		if err != nil {
			return model.OAuthUserInfo{}, fmt.Errorf("fetching userinfo: %w", err)
		}
		claims.Email = ui.Email
		claims.EmailVerified = ui.EmailVerified
		if claims.Name == "" || claims.Picture == "" {
			var extra struct {
				Name              string `json:"name"`
				Picture           string `json:"picture"`
				PreferredUsername string `json:"preferred_username"`
			}
			if err := ui.Claims(&extra); err == nil {
				claims.Name = firstNonEmpty(claims.Name, extra.Name)
				claims.Picture = firstNonEmpty(claims.Picture, extra.Picture)
				claims.PreferredUsername = firstNonEmpty(claims.PreferredUsername, extra.PreferredUsername)
			}
		}
	}

	return p.userInfo(claims)
}

// userInfo normalises the ID token claims into the shared OAuth user shape,
// applying the email policy configured for this provider.
func (p *SSOProvider) userInfo(claims ssoClaims) (model.OAuthUserInfo, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return model.OAuthUserInfo{}, &SSOError{
			Sentinel: model.ErrOAuthEmailMissing,
			Key:      "sso_email_missing",
			Message:  "the identity provider did not return an email address",
		}
	}

	// email_verified is only meaningful when the claim is present; the config
	// switch defaults to requiring it, because an unverified address would let
	// anyone at the IdP claim another user's mailbox and inherit their account.
	verified := claims.EmailVerified || !p.cfg.RequiresVerifiedEmail()
	if !verified {
		return model.OAuthUserInfo{}, &SSOError{
			Sentinel: model.ErrOAuthEmailUnverified,
			Key:      "sso_email_unverified",
			Message:  "the identity provider reported this email address as unverified",
		}
	}

	display := firstNonEmpty(claims.Name, claims.Nickname, claims.PreferredUsername, email)

	return model.OAuthUserInfo{
		ProviderUserID: claims.Subject,
		Email:          email,
		EmailVerified:  true,
		DisplayName:    display,
		AvatarURL:      claims.Picture,
		Username:       claims.PreferredUsername,
		RawAvatar:      claims.Picture,
	}, nil
}

// ssoClaims are the OIDC standard claims consumed by the SSO provider.
type ssoClaims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	Nickname          string `json:"nickname"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
}

// SSOError carries a stable error key so the login page can localise why a
// single sign-in was rejected instead of showing a generic failure.
type SSOError struct {
	Sentinel error
	Key      string
	Message  string
}

func (e *SSOError) Error() string { return e.Message }
func (e *SSOError) Unwrap() error { return e.Sentinel }

func (p *SSOProvider) discover(ctx context.Context) (*oidc.Provider, error) {
	return p.cache.get(ctx, p.cfg.Issuer, p.httpClient)
}

func (p *SSOProvider) oauthConfig(provider *oidc.Provider) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  p.redirect,
		Scopes:       p.cfg.ScopeList(),
		Endpoint:     provider.Endpoint(),
	}
}

func ssoRandomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

var (
	_ OAuthProvider      = (*SSOProvider)(nil)
	_ StateBinder        = (*SSOProvider)(nil)
	_ ContextualAuthURL  = (*SSOProvider)(nil)
	_ SSOLabeledProvider = (*SSOProvider)(nil)
)
