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
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
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
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
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
		id, err := uuid.Parse(*req.DefaultAssigneeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid default_assignee_id")
			return
		}
		input.DefaultAssigneeID = &id
	}

	if req.WorkflowID != nil {
		id, err := uuid.Parse(*req.WorkflowID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow_id")
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
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, err := uuid.Parse(chi.URLParam(r, "queueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid queue ID")
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
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, err := uuid.Parse(chi.URLParam(r, "queueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid queue ID")
		return
	}

	// Decode to raw map for explicit null detection
	raw := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	var input service.UpdateQueueInput

	if v, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(v, &name); err == nil {
			input.Name = &name
		}
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			input.ClearDescription = true
		} else {
			var desc string
			if err := json.Unmarshal(v, &desc); err == nil {
				input.Description = &desc
			}
		}
	}

	if v, ok := raw["queue_type"]; ok {
		var qt string
		if err := json.Unmarshal(v, &qt); err == nil {
			input.QueueType = &qt
		}
	}

	if v, ok := raw["is_public"]; ok {
		var ip bool
		if err := json.Unmarshal(v, &ip); err == nil {
			input.IsPublic = &ip
		}
	}

	if v, ok := raw["default_priority"]; ok {
		var dp string
		if err := json.Unmarshal(v, &dp); err == nil {
			input.DefaultPriority = &dp
		}
	}

	if v, ok := raw["default_assignee_id"]; ok {
		if string(v) == "null" {
			input.ClearDefaultAssignee = true
		} else {
			var idStr string
			if err := json.Unmarshal(v, &idStr); err == nil {
				id, err := uuid.Parse(idStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid default_assignee_id")
					return
				}
				input.DefaultAssigneeID = &id
			}
		}
	}

	if v, ok := raw["workflow_id"]; ok {
		if string(v) == "null" {
			input.ClearWorkflow = true
		} else {
			var idStr string
			if err := json.Unmarshal(v, &idStr); err == nil {
				id, err := uuid.Parse(idStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow_id")
					return
				}
				input.WorkflowID = &id
			}
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
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	queueID, err := uuid.Parse(chi.URLParam(r, "queueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid queue ID")
		return
	}

	if err := h.queues.Delete(r.Context(), info, projectKey, queueID); err != nil {
		handleQueueError(w, r, err, "failed to delete queue")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleQueueError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "queue not found")
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
	if errors.Is(err, model.ErrAlreadyExists) || errors.Is(err, model.ErrConflict) {
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
