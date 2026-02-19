package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/marcoshack/trackforge/internal/model"
)

// RequireAdmin returns 403 if the authenticated user is not a global admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := model.AuthInfoFromContext(r.Context())
		if info == nil || info.GlobalRole != model.RoleAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "FORBIDDEN",
					"message": "admin access required",
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
