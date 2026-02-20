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

// WorkflowHandler handles workflow endpoints.
type WorkflowHandler struct {
	workflows *service.WorkflowService
	projects  *service.ProjectService
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(workflows *service.WorkflowService, projects *service.ProjectService) *WorkflowHandler {
	return &WorkflowHandler{workflows: workflows, projects: projects}
}

// --- Request DTOs ---

type createWorkflowRequest struct {
	Name        string                    `json:"name"`
	Description *string                   `json:"description,omitempty"`
	Statuses    []createWorkflowStatusDTO `json:"statuses"`
	Transitions []createWorkflowTransDTO  `json:"transitions"`
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
	Name        *string                   `json:"name,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Statuses    []createWorkflowStatusDTO `json:"statuses,omitempty"`
	Transitions []createWorkflowTransDTO  `json:"transitions,omitempty"`
}

// --- Response DTOs ---

type workflowResponse struct {
	ID          uuid.UUID                `json:"id"`
	ProjectID   *uuid.UUID               `json:"project_id,omitempty"`
	Name        string                   `json:"name"`
	Description *string                  `json:"description,omitempty"`
	IsDefault   bool                     `json:"is_default"`
	Statuses    []workflowStatusResponse `json:"statuses"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

type workflowDetailResponse struct {
	ID          uuid.UUID                    `json:"id"`
	ProjectID   *uuid.UUID                   `json:"project_id,omitempty"`
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
	statuses := make([]workflowStatusResponse, 0, len(wf.Statuses))
	for _, s := range wf.Statuses {
		statuses = append(statuses, workflowStatusResponse{
			ID:          s.ID,
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Category:    s.Category,
			Position:    s.Position,
			Color:       s.Color,
		})
	}
	return workflowResponse{
		ID:          wf.ID,
		ProjectID:   wf.ProjectID,
		Name:        wf.Name,
		Description: wf.Description,
		IsDefault:   wf.IsDefault,
		Statuses:    statuses,
		CreatedAt:   wf.CreatedAt,
		UpdatedAt:   wf.UpdatedAt,
	}
}

func toWorkflowDetailResponse(wf *model.Workflow) workflowDetailResponse {
	statuses := make([]workflowStatusResponse, 0, len(wf.Statuses))
	for _, s := range wf.Statuses {
		statuses = append(statuses, workflowStatusResponse{
			ID:          s.ID,
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Category:    s.Category,
			Position:    s.Position,
			Color:       s.Color,
		})
	}
	transitions := make([]workflowTransitionResponse, 0, len(wf.Transitions))
	for _, t := range wf.Transitions {
		transitions = append(transitions, workflowTransitionResponse{
			ID:         t.ID,
			FromStatus: t.FromStatus,
			ToStatus:   t.ToStatus,
			Name:       t.Name,
		})
	}
	return workflowDetailResponse{
		ID:          wf.ID,
		ProjectID:   wf.ProjectID,
		Name:        wf.Name,
		Description: wf.Description,
		IsDefault:   wf.IsDefault,
		Statuses:    statuses,
		Transitions: transitions,
		CreatedAt:   wf.CreatedAt,
		UpdatedAt:   wf.UpdatedAt,
	}
}

// --- System workflow handlers ---

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

// --- Project workflow handlers ---

// ListProjectWorkflows handles GET /api/v1/projects/{projectKey}/workflows
func (h *WorkflowHandler) ListProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	project, err := h.projects.Get(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to get project")
		return
	}

	workflows, err := h.workflows.ListByProject(r.Context(), project.ID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list project workflows")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]workflowResponse, len(workflows))
	for i := range workflows {
		resp[i] = toWorkflowResponse(&workflows[i])
	}

	writeData(w, http.StatusOK, resp)
}

// GetProjectWorkflow handles GET /api/v1/projects/{projectKey}/workflows/{workflowId}
func (h *WorkflowHandler) GetProjectWorkflow(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	project, err := h.projects.Get(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to get project")
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

	// Verify the workflow is either a system workflow or belongs to this project
	if wf.ProjectID != nil && *wf.ProjectID != project.ID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
		return
	}

	writeData(w, http.StatusOK, toWorkflowDetailResponse(wf))
}

// CreateProjectWorkflow handles POST /api/v1/projects/{projectKey}/workflows
func (h *WorkflowHandler) CreateProjectWorkflow(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	project, err := h.projects.RequireProjectRole(r.Context(), info, projectKey, model.ProjectRoleOwner, model.ProjectRoleAdmin)
	if err != nil {
		handleProjectError(w, r, err, "failed to check project role")
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
		ProjectID:   &project.ID,
		Name:        req.Name,
		Description: req.Description,
		Statuses:    statuses,
		Transitions: transitions,
	})
	if err != nil {
		handleWorkflowError(w, r, err, "failed to create project workflow")
		return
	}

	writeData(w, http.StatusCreated, toWorkflowDetailResponse(wf))
}

// UpdateProjectWorkflow handles PATCH /api/v1/projects/{projectKey}/workflows/{workflowId}
func (h *WorkflowHandler) UpdateProjectWorkflow(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	project, err := h.projects.RequireProjectRole(r.Context(), info, projectKey, model.ProjectRoleOwner, model.ProjectRoleAdmin)
	if err != nil {
		handleProjectError(w, r, err, "failed to check project role")
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "workflowId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID")
		return
	}

	// Verify the workflow is a project workflow (not system) and belongs to this project
	existing, err := h.workflows.GetByID(r.Context(), workflowID)
	if err != nil {
		handleWorkflowError(w, r, err, "failed to get workflow")
		return
	}
	if existing.IsDefault || existing.ProjectID == nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "system workflows cannot be edited from project settings")
		return
	}
	if *existing.ProjectID != project.ID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
		return
	}

	var req updateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input := service.UpdateWorkflowInput{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.Statuses != nil {
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
		input.Statuses = statuses

		transitions := make([]model.WorkflowTransition, len(req.Transitions))
		for i, t := range req.Transitions {
			transitions[i] = model.WorkflowTransition{
				FromStatus: t.FromStatus,
				ToStatus:   t.ToStatus,
				Name:       t.Name,
			}
		}
		input.Transitions = transitions
	}

	wf, err := h.workflows.Update(r.Context(), workflowID, input)
	if err != nil {
		handleWorkflowError(w, r, err, "failed to update project workflow")
		return
	}

	writeData(w, http.StatusOK, toWorkflowDetailResponse(wf))
}

// DeleteProjectWorkflow handles DELETE /api/v1/projects/{projectKey}/workflows/{workflowId}
func (h *WorkflowHandler) DeleteProjectWorkflow(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	project, err := h.projects.RequireProjectRole(r.Context(), info, projectKey, model.ProjectRoleOwner, model.ProjectRoleAdmin)
	if err != nil {
		handleProjectError(w, r, err, "failed to check project role")
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "workflowId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID")
		return
	}

	// Verify the workflow belongs to this project before deleting
	existing, err := h.workflows.GetByID(r.Context(), workflowID)
	if err != nil {
		handleWorkflowError(w, r, err, "failed to get workflow")
		return
	}
	if existing.ProjectID == nil || *existing.ProjectID != project.ID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
		return
	}

	if err := h.workflows.DeleteProjectWorkflow(r.Context(), workflowID); err != nil {
		handleWorkflowError(w, r, err, "failed to delete project workflow")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListAvailableStatuses handles GET /api/v1/projects/{projectKey}/workflows/statuses
func (h *WorkflowHandler) ListAvailableStatuses(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	if _, err := h.projects.Get(r.Context(), info, projectKey); err != nil {
		handleProjectError(w, r, err, "failed to get project")
		return
	}

	statuses, err := h.workflows.ListAllStatuses(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list available statuses")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]workflowStatusResponse, len(statuses))
	for i, s := range statuses {
		resp[i] = workflowStatusResponse{
			ID:          s.ID,
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Category:    s.Category,
			Position:    s.Position,
			Color:       s.Color,
		}
	}

	writeData(w, http.StatusOK, resp)
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
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
