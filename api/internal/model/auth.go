package model

import (
	"context"

	"github.com/google/uuid"
)

// AuthInfo holds the authenticated user's identity extracted from JWT or API key.
type AuthInfo struct {
	UserID     uuid.UUID
	Email      string
	GlobalRole string
}

type contextKey string

const authInfoKey contextKey = "auth_info"

// ContextWithAuthInfo stores auth info in the context.
func ContextWithAuthInfo(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey, info)
}

// AuthInfoFromContext retrieves auth info from the context.
func AuthInfoFromContext(ctx context.Context) *AuthInfo {
	if info, ok := ctx.Value(authInfoKey).(*AuthInfo); ok {
		return info
	}
	return nil
}
