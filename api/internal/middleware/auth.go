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
}

// Auth extracts and validates authentication from the Authorization header.
// Supports both JWT tokens and API keys (prefixed with "twk_").
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

			if strings.HasPrefix(token, "twk_") {
				authInfo, err = authenticator.ValidateAPIKey(r.Context(), token)
			} else {
				authInfo, err = authenticator.ValidateJWT(token)
			}

			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := model.ContextWithAuthInfo(r.Context(), authInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
