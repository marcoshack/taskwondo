package model

import (
	"context"

	"github.com/google/uuid"
)

// API key permission scopes (user keys).
const (
	PermissionRead  = "read"
	PermissionWrite = "write"
)

// API key type constants.
const (
	KeyTypeUser   = "user"
	KeyTypeSystem = "system"
)

// System key resource constants.
const (
	ResourceMetrics = "metrics"
	ResourceItems   = "items"
)

// ValidPermissions defines the set of recognized permission scopes for user keys.
var ValidPermissions = map[string]bool{
	PermissionRead:  true,
	PermissionWrite: true,
}

// ValidSystemResources defines the set of recognized resource names for system keys.
var ValidSystemResources = map[string]bool{
	ResourceMetrics: true,
	ResourceItems:   true,
}

// ValidSystemAccess defines the set of recognized access levels for system keys.
var ValidSystemAccess = map[string]bool{
	"r":  true,
	"w":  true,
	"rw": true,
}

// AuthInfo holds the authenticated user's identity extracted from JWT or API key.
type AuthInfo struct {
	UserID              uuid.UUID
	Email               string
	GlobalRole          string
	ForcePasswordChange bool
	Permissions         []string // API key scopes; empty = full access
	KeyType             string   // "", "user", or "system"
	KeyID               uuid.UUID
	KeyName             string
}

// IsSystemKey returns true if this auth info represents a system API key.
func (a *AuthInfo) IsSystemKey() bool {
	return a.KeyType == KeyTypeSystem
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

// HasResourcePermission checks whether a system key has permission for the given resource and method.
func (a *AuthInfo) HasResourcePermission(resource, method string) bool {
	for _, p := range a.Permissions {
		parts := splitPermission(p)
		if len(parts) != 2 || parts[0] != resource {
			continue
		}
		access := parts[1]
		switch method {
		case "GET", "HEAD", "OPTIONS":
			return access == "r" || access == "rw"
		default:
			return access == "w" || access == "rw"
		}
	}
	return false
}

func splitPermission(p string) []string {
	for i := 0; i < len(p); i++ {
		if p[i] == ':' {
			return []string{p[:i], p[i+1:]}
		}
	}
	return []string{p}
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
