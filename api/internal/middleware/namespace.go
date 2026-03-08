package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// NamespaceResolver resolves namespace context for requests.
type NamespaceResolver interface {
	GetDefault(ctx context.Context) (*model.Namespace, error)
	GetBySlug(ctx context.Context, slug string) (*model.Namespace, error)
	IsNamespacesEnabled(ctx context.Context) (bool, error)
}

// Namespace middleware extracts the namespace from the {namespace} URL path
// parameter. The resolved namespace ID is stored in the request context.
func Namespace(resolver NamespaceResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := chi.URLParam(r, "namespace")

			var nsID uuid.UUID

			if slug == "" || slug == model.DefaultNamespaceSlug {
				// Resolve to default namespace
				ns, err := resolver.GetDefault(r.Context())
				if err != nil {
					log.Ctx(r.Context()).Error().Err(err).Msg("failed to resolve default namespace")
					writeNamespaceError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resolve namespace")
					return
				}
				nsID = ns.ID
			} else {
				// Check if namespaces are enabled
				enabled, err := resolver.IsNamespacesEnabled(r.Context())
				if err != nil {
					log.Ctx(r.Context()).Error().Err(err).Msg("failed to check namespaces feature flag")
					writeNamespaceError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resolve namespace")
					return
				}
				if !enabled {
					writeNamespaceError(w, http.StatusNotFound, "NOT_FOUND", "namespace not found")
					return
				}

				ns, err := resolver.GetBySlug(r.Context(), slug)
				if err != nil {
					if err == model.ErrNotFound {
						writeNamespaceError(w, http.StatusNotFound, "NOT_FOUND", "namespace not found")
						return
					}
					log.Ctx(r.Context()).Error().Err(err).Str("slug", slug).Msg("failed to resolve namespace")
					writeNamespaceError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resolve namespace")
					return
				}
				nsID = ns.ID
			}

			ctx := model.ContextWithNamespaceID(r.Context(), nsID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeNamespaceError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}
