package model

import (
	"context"

	"github.com/google/uuid"
)

// API key permission scopes.
const (
	PermissionRead  = "read"
	PermissionWrite = "write"
)

// ValidPermissions defines the set of recognized permission scopes.
var ValidPermissions = map[string]bool{
	PermissionRead:  true,
	PermissionWrite: true,
}

// AuthInfo holds the authenticated user's identity extracted from JWT or API key.
type AuthInfo struct {
	UserID              uuid.UUID
	Email               string
	GlobalRole          string
	ForcePasswordChange bool
	Permissions         []string // API key scopes; empty = full access
}

// HasPermission checks whether the auth info allows the given HTTP method.
// Empty permissions = full access (backward compatible with existing keys).
func (a *AuthInfo) HasPermission(method string) bool {
	if len(a.Permissions) == 0 {
		return true
	}
	for _, p := range a.Permissions {
		if p == PermissionWrite {
			return true
		}
		if p == PermissionRead {
			switch method {
			case "GET", "HEAD", "OPTIONS":
				return true
			}
		}
	}
	return false
}

type contextKey string

const authInfoKey contextKey = "auth_info"
const namespaceIDKey contextKey = "namespace_id"

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

// ContextWithNamespaceID stores the resolved namespace ID in the context.
func ContextWithNamespaceID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, namespaceIDKey, id)
}

// NamespaceIDFromContext retrieves the resolved namespace ID from the context.
// Returns uuid.Nil if not set.
func NamespaceIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(namespaceIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}
