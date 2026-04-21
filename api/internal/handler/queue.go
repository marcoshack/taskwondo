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

// QueueHandler handles queue endpoints.
type QueueHandler struct {
	queues *service.QueueService
}

// NewQueueHandler creates a new QueueHandler.
func NewQueueHandler(queues *service.QueueService) *QueueHandler {
	return &QueueHandler{queues: queues}
}

// --- Request DTOs ---

type createQueueRequest struct {
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	QueueType         string  `json:"queue_type"`
	IsPublic          bool    `json:"is_public"`
	DefaultPriority   string  `json:"default_priority,omitempty"`
	DefaultAssigneeID *string `json:"default_assignee_id,omitempty"`
	WorkflowID        *string `json:"workflow_id,omitempty"`
}

type updateQueueRequest struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	QueueType         *string `json:"queue_type,omitempty"`
	IsPublic          *bool   `json:"is_public,omitempty"`
	DefaultPriority   *string `json:"default_priority,omitempty"`
	DefaultAssigneeID *string `json:"default_assignee_id,omitempty"`
	WorkflowID        *string `json:"workflow_id,omitempty"`
}

// --- Response DTOs ---

type queueResponse struct {
	ID                uuid.UUID  `json:"id"`
	ProjectID         uuid.UUID  `json:"project_id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	QueueType         string     `json:"queue_type"`
	IsPublic          bool       `json:"is_public"`
	DefaultPriority   string     `json:"default_priority"`
	DefaultAssigneeID *uuid.UUID `json:"default_assignee_id,omitempty"`
	WorkflowID        *uuid.UUID `json:"workflow_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func toQueueResponse(q *model.Queue) queueResponse {
	return queueResponse{
		ID:                q.ID,
		ProjectID:         q.ProjectID,
		Name:              q.Name,
		Description:       q.Description,
		QueueType:         q.QueueType,
		IsPublic:          q.IsPublic,
		DefaultPriority:   q.DefaultPriority,
		DefaultAssigneeID: q.DefaultAssigneeID,
		WorkflowID:        q.WorkflowID,
		CreatedAt:         q.CreatedAt,
		UpdatedAt:         q.UpdatedAt,
	}
}

// --- Handlers ---

// List handles GET /api/v1/projects/{projectKey}/queues
func (h *QueueHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	queues, err := h.queues.List(r.Context(), info, projectKey)
	if err != nil {
		handleQueueError(w, r, err, "failed to list queues")
		return
	}

	resp := make([]queueResponse, len(queues))
	for i := range queues {
		resp[i] = toQueueResponse(&queues[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/projects/{projectKey}/queues
func (h *QueueHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	input := service.CreateQueueInput{
		Name:            req.Name,
		Description:     req.Description,
		QueueType:       req.QueueType,
		IsPublic:        req.IsPublic,
		DefaultPriority: req.DefaultPriority,
	}

	if req.DefaultAssigneeID != nil {
		id, ok := parseOptionalUUID(w, req.DefaultAssigneeID, "invalid default_assignee_id")
		if !ok {
			return
		}
		input.DefaultAssigneeID = &id
	}

	if req.WorkflowID != nil {
		id, ok := parseOptionalUUID(w, req.WorkflowID, "invalid workflow_id")
		if !ok {
			return
		}
		input.WorkflowID = &id
	}

	q, err := h.queues.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleQueueError(w, r, err, "failed to create queue")
		return
	}

	writeData(w, http.StatusCreated, toQueueResponse(q))
}

// Get handles GET /api/v1/projects/{projectKey}/queues/{queueId}
func (h *QueueHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	q, err := h.queues.Get(r.Context(), info, projectKey, queueID)
	if err != nil {
		handleQueueError(w, r, err, "failed to get queue")
		return
	}

	writeData(w, http.StatusOK, toQueueResponse(q))
}

// Update handles PATCH /api/v1/projects/{projectKey}/queues/{queueId}
func (h *QueueHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	// Decode to raw map for explicit null detection
	raw := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	var input service.UpdateQueueInput

	if v, ok := raw["name"]; ok {
		name, ok := unmarshalField[string](w, v, "name")
		if !ok {
			return
		}
		input.Name = &name
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			input.ClearDescription = true
		} else {
			desc, ok := unmarshalField[string](w, v, "description")
			if !ok {
				return
			}
			input.Description = &desc
		}
	}

	if v, ok := raw["queue_type"]; ok {
		qt, ok := unmarshalField[string](w, v, "queue_type")
		if !ok {
			return
		}
		input.QueueType = &qt
	}

	if v, ok := raw["is_public"]; ok {
		ip, ok := unmarshalField[bool](w, v, "is_public")
		if !ok {
			return
		}
		input.IsPublic = &ip
	}

	if v, ok := raw["default_priority"]; ok {
		dp, ok := unmarshalField[string](w, v, "default_priority")
		if !ok {
			return
		}
		input.DefaultPriority = &dp
	}

	if v, ok := raw["default_assignee_id"]; ok {
		id, cleared, ok := unmarshalNullableUUID(w, v, "default_assignee_id")
		if !ok {
			return
		}
		if cleared {
			input.ClearDefaultAssignee = true
		} else {
			input.DefaultAssigneeID = &id
		}
	}

	if v, ok := raw["workflow_id"]; ok {
		id, cleared, ok := unmarshalNullableUUID(w, v, "workflow_id")
		if !ok {
			return
		}
		if cleared {
			input.ClearWorkflow = true
		} else {
			input.WorkflowID = &id
		}
	}

	q, err := h.queues.Update(r.Context(), info, projectKey, queueID, input)
	if err != nil {
		handleQueueError(w, r, err, "failed to update queue")
		return
	}

	writeData(w, http.StatusOK, toQueueResponse(q))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/queues/{queueId}
func (h *QueueHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	if err := h.queues.Delete(r.Context(), info, projectKey, queueID); err != nil {
		handleQueueError(w, r, err, "failed to delete queue")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Category Request/Response DTOs ---

type createCategoryRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Position    int     `json:"position"`
}

type updateCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Position    *int    `json:"position,omitempty"`
}

type categoryResponse struct {
	ID          uuid.UUID `json:"id"`
	QueueID     uuid.UUID `json:"queue_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toCategoryResponse(cat *model.QueueCategory) categoryResponse {
	return categoryResponse{
		ID:          cat.ID,
		QueueID:     cat.QueueID,
		Name:        cat.Name,
		Description: cat.Description,
		Position:    cat.Position,
		CreatedAt:   cat.CreatedAt,
		UpdatedAt:   cat.UpdatedAt,
	}
}

// --- Queue Team Request/Response DTOs ---

type addQueueTeamRequest struct {
	TeamID string `json:"team_id"`
}

type queueTeamResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toQueueTeamResponse(t *model.Team) queueTeamResponse {
	return queueTeamResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Name:        t.Name,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// --- Category Handlers ---

// ListCategories handles GET /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/categories
func (h *QueueHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	categories, err := h.queues.ListCategories(r.Context(), info, projectKey, queueID)
	if err != nil {
		handleQueueError(w, r, err, "failed to list categories")
		return
	}

	resp := make([]categoryResponse, len(categories))
	for i := range categories {
		resp[i] = toCategoryResponse(&categories[i])
	}

	writeData(w, http.StatusOK, resp)
}

// CreateCategory handles POST /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/categories
func (h *QueueHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	input := service.CreateCategoryInput{
		Name:        req.Name,
		Description: req.Description,
		Position:    req.Position,
	}

	cat, err := h.queues.CreateCategory(r.Context(), info, projectKey, queueID, input)
	if err != nil {
		handleQueueError(w, r, err, "failed to create category")
		return
	}

	writeData(w, http.StatusCreated, toCategoryResponse(cat))
}

// UpdateCategory handles PATCH /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/categories/{categoryId}
func (h *QueueHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}
	categoryID, ok := parseUUIDParam(w, r, "categoryId", "invalid category ID")
	if !ok {
		return
	}

	// Decode to raw map for explicit null detection on description
	raw := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	var input service.UpdateCategoryInput

	if v, ok := raw["name"]; ok {
		name, ok := unmarshalField[string](w, v, "name")
		if !ok {
			return
		}
		input.Name = &name
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			input.ClearDescription = true
		} else {
			desc, ok := unmarshalField[string](w, v, "description")
			if !ok {
				return
			}
			input.Description = &desc
		}
	}

	if v, ok := raw["position"]; ok {
		pos, ok := unmarshalField[int](w, v, "position")
		if !ok {
			return
		}
		input.Position = &pos
	}

	cat, err := h.queues.UpdateCategory(r.Context(), info, projectKey, queueID, categoryID, input)
	if err != nil {
		handleQueueError(w, r, err, "failed to update category")
		return
	}

	writeData(w, http.StatusOK, toCategoryResponse(cat))
}

// DeleteCategory handles DELETE /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/categories/{categoryId}
func (h *QueueHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}
	categoryID, ok := parseUUIDParam(w, r, "categoryId", "invalid category ID")
	if !ok {
		return
	}

	if err := h.queues.DeleteCategory(r.Context(), info, projectKey, queueID, categoryID); err != nil {
		handleQueueError(w, r, err, "failed to delete category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Queue Team Handlers ---

// ListQueueTeams handles GET /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/teams
func (h *QueueHandler) ListQueueTeams(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	teams, err := h.queues.ListQueueTeams(r.Context(), info, projectKey, queueID)
	if err != nil {
		handleQueueError(w, r, err, "failed to list queue teams")
		return
	}

	resp := make([]queueTeamResponse, len(teams))
	for i := range teams {
		resp[i] = toQueueTeamResponse(&teams[i])
	}

	writeData(w, http.StatusOK, resp)
}

// AssignQueueTeam handles POST /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/teams
func (h *QueueHandler) AssignQueueTeam(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}

	var req addQueueTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	teamID, err := uuid.Parse(req.TeamID)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid team_id")
		return
	}

	if err := h.queues.AssignTeam(r.Context(), info, projectKey, queueID, teamID); err != nil {
		handleQueueError(w, r, err, "failed to assign team to queue")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UnassignQueueTeam handles DELETE /api/v1/{namespace}/projects/{projectKey}/queues/{queueId}/teams/{teamId}
func (h *QueueHandler) UnassignQueueTeam(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, ok := parseUUIDParam(w, r, "queueId", "invalid queue ID")
	if !ok {
		return
	}
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	if err := h.queues.UnassignTeam(r.Context(), info, projectKey, queueID, teamID); err != nil {
		handleQueueError(w, r, err, "failed to unassign team from queue")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleQueueError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "queue not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, CodeForbidden, "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeErrorFromService(w, http.StatusBadRequest, CodeValidationError, err)
		return
	}
	if errors.Is(err, model.ErrAlreadyExists) || errors.Is(err, model.ErrConflict) {
		writeErrorFromService(w, http.StatusConflict, CodeConflict, err)
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}
