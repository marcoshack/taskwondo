package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// SearchHandler handles the semantic search endpoint.
type SearchHandler struct {
	search *service.SearchService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(search *service.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

// Search handles GET /api/v1/search?q=...&entity_type=work_item,comment&limit=20
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "q parameter is required")
		return
	}

	filter := &model.SearchFilter{
		Query: query,
		Limit: 20,
	}

	if et := r.URL.Query().Get("entity_type"); et != "" {
		filter.EntityTypes = strings.Split(et, ",")
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			filter.Limit = parsed
		}
	}

	results, err := h.search.Search(r.Context(), info, filter)
	if err != nil {
		if errors.Is(err, model.ErrFeatureDisabled) {
			writeError(w, http.StatusNotFound, "FEATURE_DISABLED", "semantic search is not enabled")
			return
		}
		if errors.Is(err, model.ErrEmbeddingUnavailable) {
			writeError(w, http.StatusServiceUnavailable, "SEARCH_UNAVAILABLE", "embedding service is unavailable")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "search failed")
		return
	}

	if results == nil {
		results = []model.SearchResult{}
	}

	writeData(w, http.StatusOK, map[string]any{
		"results": results,
		"query":   query,
		"total":   len(results),
	})
}
