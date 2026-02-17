package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/model"
	"github.com/marcoshack/trackforge/internal/service"
)

// WorkflowHandler handles workflow endpoints.
type WorkflowHandler struct {
	workflows *service.WorkflowService
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(workflows *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{workflows: workflows}
}

// --- Request DTOs ---

type createWorkflowRequest struct {
	Name        string                      `json:"name"`
	Description *string                     `json:"description,omitempty"`
	Statuses    []createWorkflowStatusDTO   `json:"statuses"`
	Transitions []createWorkflowTransDTO    `json:"transitions"`
}

type createWorkflowStatusDTO struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Category    string  `json:"category"`
	Position    int     `json:"position"`
	Color       *string `json:"color,omitempty"`
}

type createWorkflowTransDTO struct {
	FromStatus string  `json:"from_status"`
	ToStatus   string  `json:"to_status"`
	Name       *string `json:"name,omitempty"`
}

type updateWorkflowRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// --- Response DTOs ---

type workflowResponse struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	Description *string                      `json:"description,omitempty"`
	IsDefault   bool                         `json:"is_default"`
	Statuses    []workflowStatusResponse     `json:"statuses"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
}

type workflowDetailResponse struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	Description *string                      `json:"description,omitempty"`
	IsDefault   bool                         `json:"is_default"`
	Statuses    []workflowStatusResponse     `json:"statuses"`
	Transitions []workflowTransitionResponse `json:"transitions"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
}

type workflowStatusResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Category    string    `json:"category"`
	Position    int       `json:"position"`
	Color       *string   `json:"color,omitempty"`
}

type workflowTransitionResponse struct {
	ID         uuid.UUID `json:"id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Name       *string   `json:"name,omitempty"`
}

func toWorkflowResponse(wf *model.Workflow) workflowResponse {
	statuses := make([]workflowStatusResponse, len(wf.Statuses))
	for i, s := range wf.Statuses {
		statuses[i] = workflowStatusResponse{
			ID:          s.ID,
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Category:    s.Category,
			Position:    s.Position,
			Color:       s.Color,
		}
	}
	if statuses == nil {
		statuses = []workflowStatusResponse{}
	}
	return workflowResponse{
		ID:          wf.ID,
		Name:        wf.Name,
		Description: wf.Description,
		IsDefault:   wf.IsDefault,
		Statuses:    statuses,
		CreatedAt:   wf.CreatedAt,
		UpdatedAt:   wf.UpdatedAt,
	}
}

func toWorkflowDetailResponse(wf *model.Workflow) workflowDetailResponse {
	statuses := make([]workflowStatusResponse, len(wf.Statuses))
	for i, s := range wf.Statuses {
		statuses[i] = workflowStatusResponse{
			ID:          s.ID,
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Category:    s.Category,
			Position:    s.Position,
			Color:       s.Color,
		}
	}
	transitions := make([]workflowTransitionResponse, len(wf.Transitions))
	for i, t := range wf.Transitions {
		transitions[i] = workflowTransitionResponse{
			ID:         t.ID,
			FromStatus: t.FromStatus,
			ToStatus:   t.ToStatus,
			Name:       t.Name,
		}
	}
	if statuses == nil {
		statuses = []workflowStatusResponse{}
	}
	if transitions == nil {
		transitions = []workflowTransitionResponse{}
	}
	return workflowDetailResponse{
		ID:          wf.ID,
		Name:        wf.Name,
		Description: wf.Description,
		IsDefault:   wf.IsDefault,
		Statuses:    statuses,
		Transitions: transitions,
		CreatedAt:   wf.CreatedAt,
		UpdatedAt:   wf.UpdatedAt,
	}
}

// --- Handlers ---

// List handles GET /api/v1/workflows
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	workflows, err := h.workflows.List(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list workflows")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]workflowResponse, len(workflows))
	for i := range workflows {
		resp[i] = toWorkflowResponse(&workflows[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/workflows
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req createWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	statuses := make([]model.WorkflowStatus, len(req.Statuses))
	for i, s := range req.Statuses {
		statuses[i] = model.WorkflowStatus{
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Category:    s.Category,
			Position:    s.Position,
			Color:       s.Color,
		}
	}

	transitions := make([]model.WorkflowTransition, len(req.Transitions))
	for i, t := range req.Transitions {
		transitions[i] = model.WorkflowTransition{
			FromStatus: t.FromStatus,
			ToStatus:   t.ToStatus,
			Name:       t.Name,
		}
	}

	wf, err := h.workflows.Create(r.Context(), service.CreateWorkflowInput{
		Name:        req.Name,
		Description: req.Description,
		Statuses:    statuses,
		Transitions: transitions,
	})
	if err != nil {
		handleWorkflowError(w, r, err, "failed to create workflow")
		return
	}

	writeData(w, http.StatusCreated, toWorkflowDetailResponse(wf))
}

// Get handles GET /api/v1/workflows/{workflowId}
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "workflowId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID")
		return
	}

	wf, err := h.workflows.GetByID(r.Context(), workflowID)
	if err != nil {
		handleWorkflowError(w, r, err, "failed to get workflow")
		return
	}

	writeData(w, http.StatusOK, toWorkflowDetailResponse(wf))
}

// Update handles PATCH /api/v1/workflows/{workflowId}
func (h *WorkflowHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "workflowId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID")
		return
	}

	var req updateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	wf, err := h.workflows.Update(r.Context(), workflowID, service.UpdateWorkflowInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		handleWorkflowError(w, r, err, "failed to update workflow")
		return
	}

	writeData(w, http.StatusOK, toWorkflowDetailResponse(wf))
}

// ListTransitions handles GET /api/v1/workflows/{workflowId}/transitions
func (h *WorkflowHandler) ListTransitions(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "workflowId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID")
		return
	}

	transMap, err := h.workflows.GetTransitionsMap(r.Context(), workflowID)
	if err != nil {
		handleWorkflowError(w, r, err, "failed to list transitions")
		return
	}

	// Convert to response format
	result := make(map[string][]workflowTransitionResponse)
	for fromStatus, transitions := range transMap {
		resp := make([]workflowTransitionResponse, len(transitions))
		for i, t := range transitions {
			resp[i] = workflowTransitionResponse{
				ID:         t.ID,
				FromStatus: t.FromStatus,
				ToStatus:   t.ToStatus,
				Name:       t.Name,
			}
		}
		result[fromStatus] = resp
	}

	writeData(w, http.StatusOK, result)
}

func handleWorkflowError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
