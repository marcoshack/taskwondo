package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// SearchHandler handles the search endpoint (FTS + semantic).
type SearchHandler struct {
	search *service.SearchService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(search *service.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

// --- SSE/JSON response DTOs ---

type ftsSection struct {
	Results []model.SearchResult `json:"results"`
	Total   int                  `json:"total"`
}

type semanticSection struct {
	Results   []model.SearchResult `json:"results,omitempty"`
	Total     int                  `json:"total"`
	Available bool                 `json:"available"`
	Status    string               `json:"status"`
}

type ftsEventPayload struct {
	FTS      ftsSection      `json:"fts"`
	Semantic semanticSection `json:"semantic"`
}

type semanticEventPayload struct {
	Semantic semanticSection `json:"semantic"`
}

type errorEventPayload struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Search handles GET /api/v1/search?q=...&entity_type=work_item,comment&project=TF&limit=20
// Supports SSE streaming (Accept: text/event-stream) and regular JSON responses.
//
// The optional `project` parameter takes a project key and scopes the
// project-scoped entity types (work_item, milestone, queue, team, comment,
// attachment) to that project. Results of type `project` stay global so a
// client can navigate across projects. Omitting it searches everything the
// caller can reach, which is the historical behaviour.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "q parameter is required")
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

	// Optional project scope, given as a project key like the rest of the API.
	// Resolved before any streaming starts so a bad scope is reported as a
	// normal JSON error even for SSE requests, and treated as fatal — a scope
	// that cannot be resolved must never degrade into an unscoped search.
	if projectKey := strings.TrimSpace(r.URL.Query().Get("project")); projectKey != "" {
		projectID, err := h.search.ResolveProjectScope(r.Context(), info, projectKey)
		if err != nil {
			if errors.Is(err, model.ErrForbidden) || errors.Is(err, model.ErrNotFound) {
				// Identical response whether the project does not exist or the
				// caller simply cannot see it — no existence leak.
				writeError(w, http.StatusForbidden, CodeForbidden, "project not found or not accessible")
				return
			}
			log.Ctx(r.Context()).Error().Err(err).Str("project_key", projectKey).Msg("failed to resolve search project scope")
			writeError(w, http.StatusInternalServerError, CodeInternalError, "search failed")
			return
		}
		filter.ScopeProjectID = &projectID
	}

	// Content negotiation: SSE vs JSON
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		h.searchSSE(w, r, info, filter)
		return
	}
	h.searchJSON(w, r, info, filter)
}

// searchSSE streams results using Server-Sent Events.
func (h *SearchHandler) searchSSE(w http.ResponseWriter, r *http.Request, info *model.AuthInfo, filter *model.SearchFilter) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, CodeInternalError, "streaming unsupported")
		return
	}

	ctx := r.Context()
	semanticAvailable := h.search.SemanticEnabled(ctx)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	// Launch both searches concurrently
	type ftsResult struct {
		results []model.SearchResult
		err     error
	}
	type semResult struct {
		results []model.SearchResult
		err     error
	}

	ftsCh := make(chan ftsResult, 1)
	semCh := make(chan semResult, 1)

	go func() {
		res, err := h.search.SearchFTS(ctx, info, filter)
		ftsCh <- ftsResult{res, err}
	}()

	if semanticAvailable {
		go func() {
			res, err := h.search.SearchSemantic(ctx, info, filter)
			semCh <- semResult{res, err}
		}()
	}

	// Phase 1: FTS results (fast)
	fts := <-ftsCh
	if fts.err != nil {
		log.Ctx(ctx).Error().Err(fts.err).Msg("FTS search failed")
		writeSSEEvent(w, "error", errorEventPayload{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{Code: "FTS_SEARCH_FAILED", Message: "Full-text search failed"},
		})
		writeSSEEvent(w, "done", struct{}{})
		flusher.Flush()
		return
	}

	ftsResults := fts.results
	if ftsResults == nil {
		ftsResults = []model.SearchResult{}
	}
	enrichResourcePaths(ftsResults)

	semStatus := "pending"
	if !semanticAvailable {
		semStatus = "complete"
	}

	writeSSEEvent(w, "fts", ftsEventPayload{
		FTS: ftsSection{Results: ftsResults, Total: len(ftsResults)},
		Semantic: semanticSection{
			Available: semanticAvailable,
			Status:    semStatus,
		},
	})
	flusher.Flush()

	// Phase 2: Semantic results (slower, if available)
	if semanticAvailable {
		sem := <-semCh
		if sem.err != nil {
			log.Ctx(ctx).Warn().Err(sem.err).Msg("semantic search failed mid-stream")
			writeSSEEvent(w, "error", errorEventPayload{
				Error: struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}{Code: "SEMANTIC_SEARCH_FAILED", Message: "Semantic search is temporarily unavailable"},
			})
		} else {
			semResults := sem.results
			if semResults == nil {
				semResults = []model.SearchResult{}
			}
			enrichResourcePaths(semResults)
			writeSSEEvent(w, "semantic", semanticEventPayload{
				Semantic: semanticSection{
					Results:   semResults,
					Total:     len(semResults),
					Available: true,
					Status:    "complete",
				},
			})
		}
	}

	writeSSEEvent(w, "done", struct{}{})
	flusher.Flush()
}

// searchJSON returns a single JSON response with both FTS and semantic results.
func (h *SearchHandler) searchJSON(w http.ResponseWriter, r *http.Request, info *model.AuthInfo, filter *model.SearchFilter) {
	ctx := r.Context()
	semanticAvailable := h.search.SemanticEnabled(ctx)

	// Run both searches concurrently
	var (
		ftsResults []model.SearchResult
		ftsErr     error
		semResults []model.SearchResult
		semErr     error
		wg         sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		ftsResults, ftsErr = h.search.SearchFTS(ctx, info, filter)
	}()

	if semanticAvailable {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semResults, semErr = h.search.SearchSemantic(ctx, info, filter)
		}()
	}

	wg.Wait()

	if ftsErr != nil {
		log.Ctx(ctx).Error().Err(ftsErr).Msg("FTS search failed")
		writeError(w, http.StatusInternalServerError, CodeInternalError, "search failed")
		return
	}

	if ftsResults == nil {
		ftsResults = []model.SearchResult{}
	}
	enrichResourcePaths(ftsResults)

	semSection := semanticSection{
		Available: semanticAvailable,
		Status:    "complete",
	}

	if semanticAvailable {
		if semErr != nil {
			log.Ctx(ctx).Warn().Err(semErr).Msg("semantic search failed")
			semSection.Status = "error"
			semSection.Results = []model.SearchResult{}
		} else {
			if semResults == nil {
				semResults = []model.SearchResult{}
			}
			enrichResourcePaths(semResults)
			semSection.Results = semResults
			semSection.Total = len(semResults)
		}
	}

	writeData(w, http.StatusOK, map[string]any{
		"query": filter.Query,
		"fts": ftsSection{
			Results: ftsResults,
			Total:   len(ftsResults),
		},
		"semantic": semSection,
	})
}

// enrichResourcePaths populates the ResourcePath field for each search result.
func enrichResourcePaths(results []model.SearchResult) {
	for i := range results {
		r := &results[i]
		ns := r.NamespaceSlug
		if ns == "" {
			ns = "default"
		}
		base := fmt.Sprintf("/api/v1/%s/%s/%s", ns, PathProjects, r.ProjectKey)
		switch r.EntityType {
		case model.EntityTypeWorkItem:
			if r.ProjectKey != "" && r.ItemNumber != nil {
				r.ResourcePath = fmt.Sprintf("%s/%s/%d", base, PathItems, *r.ItemNumber)
			}
		case model.EntityTypeComment:
			if r.ProjectKey != "" && r.ItemNumber != nil {
				r.ResourcePath = fmt.Sprintf("%s/%s/%d/%s/%s", base, PathItems, *r.ItemNumber, PathComments, r.EntityID)
			}
		case model.EntityTypeAttachment:
			if r.ProjectKey != "" && r.ItemNumber != nil {
				r.ResourcePath = fmt.Sprintf("%s/%s/%d/%s/%s", base, PathItems, *r.ItemNumber, PathAttachments, r.EntityID)
			}
		case model.EntityTypeProject:
			if r.ProjectKey != "" {
				r.ResourcePath = base
			}
		case model.EntityTypeMilestone:
			if r.ProjectKey != "" {
				r.ResourcePath = fmt.Sprintf("%s/%s/%s", base, PathMilestones, r.EntityID)
			}
		case model.EntityTypeQueue:
			if r.ProjectKey != "" {
				r.ResourcePath = fmt.Sprintf("%s/%s/%s", base, PathQueues, r.EntityID)
			}
		case model.EntityTypeTeam:
			if r.ProjectKey != "" {
				r.ResourcePath = fmt.Sprintf("%s/%s/%s", base, PathTeams, r.EntityID)
			}
		}
	}
}

// writeSSEEvent writes a single SSE event in the format: event: <name>\ndata: <json>\n\n
func writeSSEEvent(w http.ResponseWriter, eventName string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, jsonData)
}
