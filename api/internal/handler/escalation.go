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

// EscalationHandler handles escalation list endpoints.
type EscalationHandler struct {
	escalations *service.EscalationService
}

// NewEscalationHandler creates a new EscalationHandler.
func NewEscalationHandler(escalations *service.EscalationService) *EscalationHandler {
	return &EscalationHandler{escalations: escalations}
}

// --- Request DTOs ---

type createEscalationListRequest struct {
	Name   string                    `json:"name"`
	Levels []escalationLevelRequest  `json:"levels"`
}

type escalationLevelRequest struct {
	ThresholdPct int      `json:"threshold_pct"`
	UserIDs      []string `json:"user_ids"`
}

type updateMappingRequest struct {
	EscalationListID string `json:"escalation_list_id"`
}

// --- Response DTOs ---

type escalationListResponse struct {
	ID        uuid.UUID                  `json:"id"`
	Name      string                     `json:"name"`
	Levels    []escalationLevelResponse  `json:"levels"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

type escalationLevelResponse struct {
	ID           uuid.UUID                     `json:"id"`
	ThresholdPct int                           `json:"threshold_pct"`
	Position     int                           `json:"position"`
	Users        []escalationLevelUserResponse `json:"users"`
}

type escalationLevelUserResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}

type typeEscalationMappingResponse struct {
	WorkItemType     string    `json:"work_item_type"`
	EscalationListID uuid.UUID `json:"escalation_list_id"`
}

func toEscalationListResponse(el *model.EscalationList) escalationListResponse {
	levels := make([]escalationLevelResponse, len(el.Levels))
	for i, lv := range el.Levels {
		users := make([]escalationLevelUserResponse, len(lv.Users))
		for j, u := range lv.Users {
			users[j] = escalationLevelUserResponse{
				ID:          u.UserID,
				DisplayName: u.DisplayName,
				Email:       u.Email,
			}
		}
		levels[i] = escalationLevelResponse{
			ID:           lv.ID,
			ThresholdPct: lv.ThresholdPct,
			Position:     lv.Position,
			Users:        users,
		}
	}
	return escalationListResponse{
		ID:        el.ID,
		Name:      el.Name,
		Levels:    levels,
		CreatedAt: el.CreatedAt,
		UpdatedAt: el.UpdatedAt,
	}
}

// --- Handlers ---

// List handles GET /api/v1/projects/{projectKey}/escalation-lists
func (h *EscalationHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	lists, err := h.escalations.List(r.Context(), info, projectKey)
	if err != nil {
		handleEscalationError(w, r, err, "failed to list escalation lists")
		return
	}

	resp := make([]escalationListResponse, len(lists))
	for i := range lists {
		resp[i] = toEscalationListResponse(&lists[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/projects/{projectKey}/escalation-lists
func (h *EscalationHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createEscalationListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input, err := toEscalationInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	el, err := h.escalations.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleEscalationError(w, r, err, "failed to create escalation list")
		return
	}

	writeData(w, http.StatusCreated, toEscalationListResponse(el))
}

// Get handles GET /api/v1/projects/{projectKey}/escalation-lists/{listId}
func (h *EscalationHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	listID, err := uuid.Parse(chi.URLParam(r, "listId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid escalation list ID")
		return
	}

	el, err := h.escalations.Get(r.Context(), info, projectKey, listID)
	if err != nil {
		handleEscalationError(w, r, err, "failed to get escalation list")
		return
	}

	writeData(w, http.StatusOK, toEscalationListResponse(el))
}

// Update handles PUT /api/v1/projects/{projectKey}/escalation-lists/{listId}
func (h *EscalationHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	listID, err := uuid.Parse(chi.URLParam(r, "listId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid escalation list ID")
		return
	}

	var req createEscalationListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input, err := toEscalationInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	el, err := h.escalations.Update(r.Context(), info, projectKey, listID, input)
	if err != nil {
		handleEscalationError(w, r, err, "failed to update escalation list")
		return
	}

	writeData(w, http.StatusOK, toEscalationListResponse(el))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/escalation-lists/{listId}
func (h *EscalationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	listID, err := uuid.Parse(chi.URLParam(r, "listId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid escalation list ID")
		return
	}

	if err := h.escalations.Delete(r.Context(), info, projectKey, listID); err != nil {
		handleEscalationError(w, r, err, "failed to delete escalation list")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMappings handles GET /api/v1/projects/{projectKey}/escalation-lists/mappings
func (h *EscalationHandler) ListMappings(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	mappings, err := h.escalations.ListMappings(r.Context(), info, projectKey)
	if err != nil {
		handleEscalationError(w, r, err, "failed to list escalation mappings")
		return
	}

	resp := make([]typeEscalationMappingResponse, len(mappings))
	for i, m := range mappings {
		resp[i] = typeEscalationMappingResponse{
			WorkItemType:     m.WorkItemType,
			EscalationListID: m.EscalationListID,
		}
	}

	writeData(w, http.StatusOK, resp)
}

// UpdateMapping handles PUT /api/v1/projects/{projectKey}/escalation-lists/mappings/{type}
func (h *EscalationHandler) UpdateMapping(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	workItemType := chi.URLParam(r, "type")

	var req updateMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	listID, err := uuid.Parse(req.EscalationListID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid escalation_list_id")
		return
	}

	m, err := h.escalations.UpdateMapping(r.Context(), info, projectKey, workItemType, listID)
	if err != nil {
		handleEscalationError(w, r, err, "failed to update escalation mapping")
		return
	}

	writeData(w, http.StatusOK, typeEscalationMappingResponse{
		WorkItemType:     m.WorkItemType,
		EscalationListID: m.EscalationListID,
	})
}

// DeleteMapping handles DELETE /api/v1/projects/{projectKey}/escalation-lists/mappings/{type}
func (h *EscalationHandler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	workItemType := chi.URLParam(r, "type")

	if err := h.escalations.DeleteMapping(r.Context(), info, projectKey, workItemType); err != nil {
		handleEscalationError(w, r, err, "failed to delete escalation mapping")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func toEscalationInput(req createEscalationListRequest) (service.CreateEscalationListInput, error) {
	levels := make([]service.EscalationLevelInput, len(req.Levels))
	for i, lv := range req.Levels {
		userIDs := make([]uuid.UUID, len(lv.UserIDs))
		for j, idStr := range lv.UserIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return service.CreateEscalationListInput{}, errors.New("invalid user_id: " + idStr)
			}
			userIDs[j] = id
		}
		levels[i] = service.EscalationLevelInput{
			ThresholdPct: lv.ThresholdPct,
			UserIDs:      userIDs,
		}
	}
	return service.CreateEscalationListInput{
		Name:   req.Name,
		Levels: levels,
	}, nil
}

func handleEscalationError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "escalation list not found")
	case errors.Is(err, model.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	case errors.Is(err, model.ErrValidation):
		writeErrorFromService(w, http.StatusBadRequest, "VALIDATION_ERROR", err)
	case errors.Is(err, model.ErrAlreadyExists) || errors.Is(err, model.ErrConflict):
		writeErrorFromService(w, http.StatusConflict, "CONFLICT", err)
	default:
		log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
