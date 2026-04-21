package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/marcoshack/taskwondo/internal/model"
)

// Authenticator validates authentication tokens.
type Authenticator interface {
	ValidateJWT(tokenString string) (*model.AuthInfo, error)
	ValidateAPIKey(ctx context.Context, key string) (*model.AuthInfo, error)
	ValidateSystemAPIKey(ctx context.Context, key string) (*model.AuthInfo, error)
}

// Auth extracts and validates authentication from the Authorization header.
// Supports JWT tokens, user API keys (twk_), and system API keys (twks_).
func Auth(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeAuthError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing token")
				return
			}

			var authInfo *model.AuthInfo
			var err error

			if strings.HasPrefix(token, "twks_") {
				// System API key
				authInfo, err = authenticator.ValidateSystemAPIKey(r.Context(), token)
				if err != nil {
					writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}

				// Check resource-level permission
				resource := resourceFromPath(r.URL.Path)
				if resource == "" {
					writeAuthError(w, http.StatusForbidden, "system key not authorized for this resource")
					return
				}
				if !authInfo.HasResourcePermission(resource, r.Method) {
					writeAuthError(w, http.StatusForbidden, "system key does not have sufficient permissions for this resource")
					return
				}
			} else if strings.HasPrefix(token, "twk_") {
				authInfo, err = authenticator.ValidateAPIKey(r.Context(), token)
				if err != nil {
					writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}
				if !authInfo.HasPermission(r.Method) {
					writeAuthError(w, http.StatusForbidden, "api key does not have sufficient permissions")
					return
				}
			} else {
				authInfo, err = authenticator.ValidateJWT(token)
				if err != nil {
					writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}
			}

			ctx := model.ContextWithAuthInfo(r.Context(), authInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resourceFromPath maps an API request path to a resource constant for system
// key authorization. Matches are strict on path shape:
//   - /metrics[...]          → ResourceMetrics
//   - /api/v1/{ns}/projects/{key}/items[/...] → ResourceItems
//
// Earlier versions of this function scanned every path segment for the literal
// "items", which would have treated a namespace slug or project key of "items"
// as a work-item route. The explicit index checks below prevent that.
func resourceFromPath(path string) string {
	if strings.HasPrefix(path, "/metrics") {
		return model.ResourceMetrics
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	// Expected layout: ["api", "v1", "{ns}", "projects", "{key}", "items", ...]
	if len(parts) >= 6 &&
		parts[0] == "api" && parts[1] == "v1" &&
		parts[3] == "projects" && parts[5] == "items" {
		return model.ResourceItems
	}

	return ""
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}
