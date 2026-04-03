package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectKeyResolver looks up a project by key.
type ProjectKeyResolver interface {
	GetByKey(ctx context.Context, key string) (*model.Project, error)
}

// MemberRoleResolver looks up a user's project membership.
type MemberRoleResolver interface {
	GetByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error)
}

// ExcludeCustomer blocks users who have the "customer" role in the project
// resolved from the {projectKey} URL parameter. Global admins and system keys
// are always allowed through.
func ExcludeCustomer(projects ProjectKeyResolver, members MemberRoleResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := model.AuthInfoFromContext(r.Context())
			if info == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Global admins and system keys bypass the check.
			if info.GlobalRole == model.RoleAdmin || info.IsSystemKey() {
				next.ServeHTTP(w, r)
				return
			}

			projectKey := chi.URLParam(r, "projectKey")
			if projectKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			project, err := projects.GetByKey(r.Context(), projectKey)
			if err != nil {
				// Let the handler deal with not-found projects.
				next.ServeHTTP(w, r)
				return
			}

			member, err := members.GetByProjectAndUser(r.Context(), project.ID, info.UserID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if member.Role == model.ProjectRoleCustomer {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    "NOT_FOUND",
						"message": "not found",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
