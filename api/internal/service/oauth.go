package service

import (
	"context"

	"github.com/marcoshack/taskwondo/internal/model"
)

// OAuthProvider abstracts a single OAuth 2.0 provider.
type OAuthProvider interface {
	// Name returns the provider identifier (e.g. "discord", "google").
	Name() string
	// AuthURL builds the authorization URL using the given signed state parameter.
	AuthURL(state string) string
	// ExchangeCode exchanges an authorization code for user info.
	ExchangeCode(ctx context.Context, code string) (model.OAuthUserInfo, error)
}

// ContextualAuthURL is implemented by providers that must reach the network to
// build an authorization URL (OIDC discovery) and therefore need the request
// context rather than only the state string.
type ContextualAuthURL interface {
	AuthURLContext(ctx context.Context, state string) (string, error)
}

// StateBinder is implemented by providers that carry per-login secrets (an OIDC
// nonce and a PKCE verifier) through the authorization redirect.
//
// A provider that implements it takes over state generation and validation:
// AuthService calls NewState instead of its own HMAC state and ValidateState
// instead of validateOAuthState. ValidateState returns a context carrying the
// unsealed secrets, which the following ExchangeCode call reads back. Binding
// the state to the secrets this way is what makes the state parameter CSRF
// protection real for OIDC — an attacker who replays someone else's callback
// fails the nonce check rather than logging in as them.
type StateBinder interface {
	NewState(ctx context.Context) (string, error)
	ValidateState(ctx context.Context, state string) (context.Context, error)
}

// SSOLabeledProvider is implemented by providers whose login button text is
// operator-configured rather than derived from a fixed i18n key.
type SSOLabeledProvider interface {
	ButtonLabel() string
}

type ssoFlowKey struct{}

// ssoFlow carries the values unsealed from one login's state parameter.
type ssoFlow struct {
	nonce    string
	verifier string
}

func ssoFlowContext(ctx context.Context, flow ssoFlow) context.Context {
	return context.WithValue(ctx, ssoFlowKey{}, flow)
}

func ssoFlowFromContext(ctx context.Context) (ssoFlow, bool) {
	f, ok := ctx.Value(ssoFlowKey{}).(ssoFlow)
	return f, ok && f.nonce != ""
}
