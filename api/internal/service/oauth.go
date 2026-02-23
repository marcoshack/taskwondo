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
