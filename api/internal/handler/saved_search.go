package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// SavedSearchHandler handles saved search endpoints.
type SavedSearchHandler struct {
	savedSearches *service.SavedSearchService
}

// NewSavedSearchHandler creates a new SavedSearchHandler.
func NewSavedSearchHandler(savedSearches *service.SavedSearchService) *SavedSearchHandler {
	return &SavedSearchHandler{savedSearches: savedSearches}
}

// --- Request DTOs ---

type createSavedSearchRequest struct {
	Name     string                    `json:"name"`
	Filters  model.SavedSearchFilters  `json:"filters"`
	ViewMode string                    `json:"view_mode"`
	Shared   bool                      `json:"shared"`
}

type updateSavedSearchRequest struct {
	Name     *string                   `json:"name,omitempty"`
	Filters  *model.SavedSearchFilters `json:"filters,omitempty"`
	ViewMode *string                   `json:"view_mode,omitempty"`
}

// --- Response DTOs ---

type savedSearchResponse struct {
	ID        uuid.UUID                `json:"id"`
	ProjectID uuid.UUID                `json:"project_id"`
	UserID    *uuid.UUID               `json:"user_id,omitempty"`
	Scope     string                   `json:"scope"`
	Name      string                   `json:"name"`
	Filters   model.SavedSearchFilters `json:"filters"`
	ViewMode  string                   `json:"view_mode"`
	Position  int                      `json:"position"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

func toSavedSearchResponse(s *model.SavedSearch) savedSearchResponse {
	return savedSearchResponse{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		UserID:    s.UserID,
		Scope:     s.Scope(),
		Name:      s.Name,
		Filters:   s.Filters,
		ViewMode:  s.ViewMode,
		Position:  s.Position,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// --- Handlers ---

// List handles GET /api/v1/projects/{projectKey}/saved-searches
func (h *SavedSearchHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	searches, err := h.savedSearches.List(r.Context(), info, projectKey)
	if err != nil {
		handleSavedSearchError(w, r, err, "failed to list saved searches")
		return
	}

	resp := make([]savedSearchResponse, len(searches))
	for i := range searches {
		resp[i] = toSavedSearchResponse(&searches[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/projects/{projectKey}/saved-searches
func (h *SavedSearchHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createSavedSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input := service.CreateSavedSearchInput{
		Name:     req.Name,
		Filters:  req.Filters,
		ViewMode: req.ViewMode,
		Shared:   req.Shared,
	}

	ss, err := h.savedSearches.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleSavedSearchError(w, r, err, "failed to create saved search")
		return
	}

	writeData(w, http.StatusCreated, toSavedSearchResponse(ss))
}

// Update handles PATCH /api/v1/projects/{projectKey}/saved-searches/{searchId}
func (h *SavedSearchHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	searchID, err := uuid.Parse(chi.URLParam(r, "searchId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid search ID")
		return
	}

	var req updateSavedSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input := service.UpdateSavedSearchInput{
		Name:     req.Name,
		Filters:  req.Filters,
		ViewMode: req.ViewMode,
	}

	ss, err := h.savedSearches.Update(r.Context(), info, projectKey, searchID, input)
	if err != nil {
		handleSavedSearchError(w, r, err, "failed to update saved search")
		return
	}

	writeData(w, http.StatusOK, toSavedSearchResponse(ss))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/saved-searches/{searchId}
func (h *SavedSearchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	searchID, err := uuid.Parse(chi.URLParam(r, "searchId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid search ID")
		return
	}

	if err := h.savedSearches.Delete(r.Context(), info, projectKey, searchID); err != nil {
		handleSavedSearchError(w, r, err, "failed to delete saved search")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleSavedSearchError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "saved search not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
