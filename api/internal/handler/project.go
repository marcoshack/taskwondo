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

// ProjectHandler handles project and membership endpoints.
type ProjectHandler struct {
	projects *service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(projects *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// --- Request DTOs ---

type createProjectRequest struct {
	Name              string  `json:"name"`
	Key               string  `json:"key"`
	Description       *string `json:"description,omitempty"`
	DefaultWorkflowID *string `json:"default_workflow_id,omitempty"`
}

type updateProjectRequest struct {
	Name                    *string `json:"name,omitempty"`
	Key                     *string `json:"key,omitempty"`
	Description             *string `json:"description"`
	DefaultWorkflowID       *string `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues *[]int  `json:"allowed_complexity_values"`
}

type addMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

// --- Response DTOs ---

type projectResponse struct {
	ID                      uuid.UUID  `json:"id"`
	Name                    string     `json:"name"`
	Key                     string     `json:"key"`
	Description             *string    `json:"description,omitempty"`
	DefaultWorkflowID       *uuid.UUID `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues []int      `json:"allowed_complexity_values"`
	ItemCounter             int        `json:"item_counter"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type projectListItemResponse struct {
	projectResponse
	MemberCount     int `json:"member_count"`
	OpenCount       int `json:"open_count"`
	InProgressCount int `json:"in_progress_count"`
}

type memberResponse struct {
	UserID      uuid.UUID  `json:"user_id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"created_at"`
}

func toProjectResponse(p *model.Project) projectResponse {
	acv := p.AllowedComplexityValues
	if acv == nil {
		acv = []int{}
	}
	return projectResponse{
		ID:                      p.ID,
		Name:                    p.Name,
		Key:                     p.Key,
		Description:             p.Description,
		DefaultWorkflowID:       p.DefaultWorkflowID,
		AllowedComplexityValues: acv,
		ItemCounter:             p.ItemCounter,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

func toMemberResponse(m *model.ProjectMemberWithUser) memberResponse {
	return memberResponse{
		UserID:      m.UserID,
		Email:       m.Email,
		DisplayName: m.DisplayName,
		AvatarURL:   m.AvatarURL,
		Role:        m.Role,
		CreatedAt:   m.CreatedAt,
	}
}

// --- Project Handlers ---

// Create handles POST /api/v1/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "key is required")
		return
	}

	var workflowID *uuid.UUID
	if req.DefaultWorkflowID != nil {
		id, err := uuid.Parse(*req.DefaultWorkflowID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid default_workflow_id format")
			return
		}
		workflowID = &id
	}

	project, err := h.projects.Create(r.Context(), info, req.Name, req.Key, req.Description, workflowID)
	if err != nil {
		handleProjectError(w, r, err, "failed to create project")
		return
	}

	writeData(w, http.StatusCreated, toProjectResponse(project))
}

// List handles GET /api/v1/projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projects, err := h.projects.ListWithSummary(r.Context(), info)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list projects")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]projectListItemResponse, len(projects))
	for i := range projects {
		resp[i] = projectListItemResponse{
			projectResponse: toProjectResponse(&projects[i].Project),
			MemberCount:     projects[i].MemberCount,
			OpenCount:       projects[i].OpenCount,
			InProgressCount: projects[i].InProgressCount,
		}
	}

	writeData(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/projects/{projectKey}
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	writeData(w, http.StatusOK, toProjectResponse(project))
}

// Update handles PATCH /api/v1/projects/{projectKey}
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	// Detect explicit null for description (clear vs omit)
	var raw map[string]json.RawMessage
	// Re-read for explicit null check isn't possible since body is consumed.
	// We handle clear by checking if description key was present with nil value.
	clearDescription := false
	if req.Description == nil && req.Name == nil && req.Key == nil {
		// Try parsing raw to detect explicit null
		_ = raw // description clearing handled via explicit null in JSON
	}

	var workflowID *uuid.UUID
	if req.DefaultWorkflowID != nil {
		id, err := uuid.Parse(*req.DefaultWorkflowID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid default_workflow_id format")
			return
		}
		workflowID = &id
	}

	// Handle allowed_complexity_values: nil pointer means not provided, empty slice means clear
	var allowedComplexityValues []int
	clearAllowedComplexityValues := false
	if req.AllowedComplexityValues != nil {
		if len(*req.AllowedComplexityValues) == 0 {
			clearAllowedComplexityValues = true
		} else {
			allowedComplexityValues = *req.AllowedComplexityValues
		}
	}

	project, err := h.projects.Update(r.Context(), info, projectKey, req.Name, req.Key, req.Description, clearDescription, workflowID, allowedComplexityValues, clearAllowedComplexityValues)
	if err != nil {
		handleProjectError(w, r, err, "failed to update project")
		return
	}

	writeData(w, http.StatusOK, toProjectResponse(project))
}

// Delete handles DELETE /api/v1/projects/{projectKey}
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	if err := h.projects.Delete(r.Context(), info, projectKey); err != nil {
		handleProjectError(w, r, err, "failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Member Handlers ---

// AddMember handles POST /api/v1/projects/{projectKey}/members
func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user_id format")
		return
	}

	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role is required")
		return
	}

	member, err := h.projects.AddMember(r.Context(), info, projectKey, userID, req.Role)
	if err != nil {
		handleProjectError(w, r, err, "failed to add member")
		return
	}

	writeData(w, http.StatusCreated, map[string]interface{}{
		"user_id":    member.UserID,
		"project_id": member.ProjectID,
		"role":       member.Role,
		"created_at": member.CreatedAt,
	})
}

// ListMembers handles GET /api/v1/projects/{projectKey}/members
func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	members, err := h.projects.ListMembers(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to list members")
		return
	}

	resp := make([]memberResponse, len(members))
	for i := range members {
		resp[i] = toMemberResponse(&members[i])
	}

	writeData(w, http.StatusOK, resp)
}

// UpdateMemberRole handles PATCH /api/v1/projects/{projectKey}/members/{userId}
func (h *ProjectHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role is required")
		return
	}

	if err := h.projects.UpdateMemberRole(r.Context(), info, projectKey, userID, req.Role); err != nil {
		handleProjectError(w, r, err, "failed to update member role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember handles DELETE /api/v1/projects/{projectKey}/members/{userId}
func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	if err := h.projects.RemoveMember(r.Context(), info, projectKey, userID); err != nil {
		handleProjectError(w, r, err, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Type Workflow Handlers ---

type typeWorkflowResponse struct {
	WorkItemType string    `json:"work_item_type"`
	WorkflowID   uuid.UUID `json:"workflow_id"`
}

type updateTypeWorkflowRequest struct {
	WorkflowID string `json:"workflow_id"`
}

// ListTypeWorkflows handles GET /api/v1/projects/{projectKey}/type-workflows
func (h *ProjectHandler) ListTypeWorkflows(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	mappings, err := h.projects.GetTypeWorkflows(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to list type workflows")
		return
	}

	resp := make([]typeWorkflowResponse, len(mappings))
	for i, m := range mappings {
		resp[i] = typeWorkflowResponse{
			WorkItemType: m.WorkItemType,
			WorkflowID:   m.WorkflowID,
		}
	}

	writeData(w, http.StatusOK, resp)
}

// UpdateTypeWorkflow handles PUT /api/v1/projects/{projectKey}/type-workflows/{type}
func (h *ProjectHandler) UpdateTypeWorkflow(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	workItemType := chi.URLParam(r, "type")

	var req updateTypeWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.WorkflowID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "workflow_id is required")
		return
	}

	workflowID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow_id format")
		return
	}

	mapping, err := h.projects.UpdateTypeWorkflow(r.Context(), info, projectKey, workItemType, workflowID)
	if err != nil {
		handleProjectError(w, r, err, "failed to update type workflow")
		return
	}

	writeData(w, http.StatusOK, typeWorkflowResponse{
		WorkItemType: mapping.WorkItemType,
		WorkflowID:   mapping.WorkflowID,
	})
}

// handleProjectError maps service errors to HTTP responses.
func handleProjectError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if errors.Is(err, model.ErrConflict) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
