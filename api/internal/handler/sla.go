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

// SLAHandler handles SLA target endpoints.
type SLAHandler struct {
	sla *service.SLAService
}

// NewSLAHandler creates a new SLAHandler.
func NewSLAHandler(sla *service.SLAService) *SLAHandler {
	return &SLAHandler{sla: sla}
}

// --- Request DTOs ---

type bulkUpsertSLARequest struct {
	WorkItemType string           `json:"work_item_type"`
	WorkflowID   string           `json:"workflow_id"`
	Targets      []slaTargetInput `json:"targets"`
}

type slaTargetInput struct {
	StatusName    string `json:"status_name"`
	Priority      string `json:"priority"`
	TargetSeconds int    `json:"target_seconds"`
	CalendarMode  string `json:"calendar_mode"`
}

// --- Response DTOs ---

type slaTargetResponse struct {
	ID            uuid.UUID `json:"id"`
	WorkItemType  string    `json:"work_item_type"`
	WorkflowID    uuid.UUID `json:"workflow_id"`
	StatusName    string    `json:"status_name"`
	Priority      string    `json:"priority"`
	TargetSeconds int       `json:"target_seconds"`
	CalendarMode  string    `json:"calendar_mode"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toSLATargetResponse(t *model.SLAStatusTarget) slaTargetResponse {
	return slaTargetResponse{
		ID:            t.ID,
		WorkItemType:  t.WorkItemType,
		WorkflowID:    t.WorkflowID,
		StatusName:    t.StatusName,
		Priority:      t.Priority,
		TargetSeconds: t.TargetSeconds,
		CalendarMode:  t.CalendarMode,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

// --- Handlers ---

// List handles GET /api/v1/projects/{projectKey}/sla-targets
func (h *SLAHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	targets, err := h.sla.ListTargets(r.Context(), info, projectKey)
	if err != nil {
		handleSLAError(w, r, err, "failed to list SLA targets")
		return
	}

	resp := make([]slaTargetResponse, len(targets))
	for i := range targets {
		resp[i] = toSLATargetResponse(&targets[i])
	}

	writeData(w, http.StatusOK, resp)
}

// BulkUpsert handles PUT /api/v1/projects/{projectKey}/sla-targets
func (h *SLAHandler) BulkUpsert(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req bulkUpsertSLARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.WorkItemType == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "work_item_type is required")
		return
	}

	workflowID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow_id format")
		return
	}

	// Build service input
	slaTargets := make([]service.SLATargetInput, len(req.Targets))
	for i, t := range req.Targets {
		calendarMode := t.CalendarMode
		if calendarMode == "" {
			calendarMode = model.CalendarMode24x7
		}
		if t.Priority == "" {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "priority is required for each target")
			return
		}
		slaTargets[i] = service.SLATargetInput{
			StatusName:    t.StatusName,
			Priority:      t.Priority,
			TargetSeconds: t.TargetSeconds,
			CalendarMode:  calendarMode,
		}
	}

	input := service.BulkUpsertSLAInput{
		WorkItemType: req.WorkItemType,
		WorkflowID:   workflowID,
		Targets:      slaTargets,
	}

	targets, err := h.sla.BulkUpsertTargets(r.Context(), info, projectKey, input)
	if err != nil {
		handleSLAError(w, r, err, "failed to upsert SLA targets")
		return
	}

	resp := make([]slaTargetResponse, len(targets))
	for i := range targets {
		resp[i] = toSLATargetResponse(&targets[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Delete handles DELETE /api/v1/projects/{projectKey}/sla-targets/{targetId}
func (h *SLAHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	targetID, err := uuid.Parse(chi.URLParam(r, "targetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid target ID format")
		return
	}

	if err := h.sla.DeleteTarget(r.Context(), info, projectKey, targetID); err != nil {
		handleSLAError(w, r, err, "failed to delete SLA target")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleSLAError(w http.ResponseWriter, r *http.Request, err error, msg string) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, model.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	case errors.Is(err, model.ErrValidation):
		writeErrorFromService(w, http.StatusBadRequest, "VALIDATION_ERROR", err)
	default:
		log.Ctx(r.Context()).Error().Err(err).Msg(msg)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
