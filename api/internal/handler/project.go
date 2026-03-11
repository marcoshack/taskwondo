package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	baseURL  string
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(projects *service.ProjectService, baseURL string) *ProjectHandler {
	return &ProjectHandler{projects: projects, baseURL: strings.TrimRight(baseURL, "/")}
}

// --- Request DTOs ---

type createProjectRequest struct {
	Name              string  `json:"name"`
	Key               string  `json:"key"`
	Description       *string `json:"description,omitempty"`
	DefaultWorkflowID *string `json:"default_workflow_id,omitempty"`
}

type updateProjectRequest struct {
	Name                    *string                  `json:"name,omitempty"`
	Key                     *string                  `json:"key,omitempty"`
	Description             *string                  `json:"description"`
	DefaultWorkflowID       *string                  `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues *[]int                   `json:"allowed_complexity_values"`
	BusinessHours           *model.BusinessHoursConfig `json:"business_hours,omitempty"`
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
	ID                      uuid.UUID                   `json:"id"`
	Name                    string                      `json:"name"`
	Key                     string                      `json:"key"`
	Description             *string                     `json:"description,omitempty"`
	Namespace               *string                     `json:"namespace,omitempty"`
	DefaultWorkflowID       *uuid.UUID                  `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues []int                       `json:"allowed_complexity_values"`
	BusinessHours           *model.BusinessHoursConfig  `json:"business_hours,omitempty"`
	ItemCounter             int                         `json:"item_counter"`
	CreatedAt               time.Time                   `json:"created_at"`
	UpdatedAt               time.Time                   `json:"updated_at"`
}

type projectListItemResponse struct {
	projectResponse
	MemberCount    int    `json:"member_count"`
	OpenCount      int    `json:"open_count"`
	InProgressCount int   `json:"in_progress_count"`
	NamespaceSlug  string `json:"namespace_slug,omitempty"`
	NamespaceIcon  string `json:"namespace_icon,omitempty"`
	NamespaceColor string `json:"namespace_color,omitempty"`
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
		BusinessHours:           p.BusinessHours,
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
		AvatarURL:   avatarURL(m.AvatarURL, m.UserID, 0),
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

	ownedCount, err := h.projects.CountOwnedByUser(r.Context(), info.UserID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to count owned projects")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	effectiveLimit, err := h.projects.ResolveEffectiveLimit(r.Context(), info)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to resolve project limit")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	// Resolve namespace info for all projects (keyed by project ID to avoid key collision across namespaces)
	ids := make([]uuid.UUID, len(projects))
	for i := range projects {
		ids[i] = projects[i].ID
	}
	nsMap, err := h.projects.ResolveProjectNamespacesByIDs(r.Context(), ids)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to resolve project namespaces")
		// Non-fatal: continue without namespace info
		nsMap = nil
	}

	resp := make([]projectListItemResponse, len(projects))
	for i := range projects {
		resp[i] = projectListItemResponse{
			projectResponse: toProjectResponse(&projects[i].Project),
			MemberCount:     projects[i].MemberCount,
			OpenCount:       projects[i].OpenCount,
			InProgressCount: projects[i].InProgressCount,
		}
		if nsInfo, ok := nsMap[projects[i].ID]; ok {
			resp[i].NamespaceSlug = nsInfo.NamespaceSlug
			resp[i].NamespaceIcon = nsInfo.NamespaceIcon
			resp[i].NamespaceColor = nsInfo.NamespaceColor
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": resp,
		"meta": map[string]interface{}{
			"owned_project_count": ownedCount,
			"max_projects":        effectiveLimit,
		},
	})
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

	project, err := h.projects.Update(r.Context(), info, projectKey, req.Name, req.Key, req.Description, clearDescription, workflowID, allowedComplexityValues, clearAllowedComplexityValues, req.BusinessHours, false)
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

	members, totalCount, err := h.projects.ListMembers(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to list members")
		return
	}

	resp := make([]memberResponse, len(members))
	for i := range members {
		resp[i] = toMemberResponse(&members[i])
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":        resp,
		"total_count": totalCount,
	})
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

// --- Invite Handlers ---

type createInviteRequest struct {
	Role      string `json:"role"`
	ExpiresIn string `json:"expires_in,omitempty"`
	MaxUses   *int   `json:"max_uses,omitempty"`
}

type inviteResponse struct {
	ID            uuid.UUID  `json:"id"`
	Code          string     `json:"code"`
	Role          string     `json:"role"`
	URL           string     `json:"url"`
	CreatedByName string     `json:"created_by_name"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	MaxUses       int        `json:"max_uses"`
	UseCount      int        `json:"use_count"`
	CreatedAt     time.Time  `json:"created_at"`
}

type inviteInfoResponse struct {
	ProjectName string `json:"project_name"`
	ProjectKey  string `json:"project_key"`
	Role        string `json:"role"`
	Expired     bool   `json:"expired"`
	Full        bool   `json:"full"`
}

type acceptInviteResponse struct {
	projectResponse
	RoleNotApplied bool   `json:"role_not_applied,omitempty"`
	ExistingRole   string `json:"existing_role,omitempty"`
	InviteRole     string `json:"invite_role,omitempty"`
}

func (h *ProjectHandler) toInviteResponse(inv *model.ProjectInvite) inviteResponse {
	return inviteResponse{
		ID:            inv.ID,
		Code:          inv.Code,
		Role:          inv.Role,
		URL:           fmt.Sprintf("%s/invite/%s", h.baseURL, inv.Code),
		CreatedByName: inv.CreatedByName,
		ExpiresAt:     inv.ExpiresAt,
		MaxUses:       inv.MaxUses,
		UseCount:      inv.UseCount,
		CreatedAt:     inv.CreatedAt,
	}
}

// CreateInvite handles POST /api/v1/projects/{projectKey}/invites
func (h *ProjectHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role is required")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		d, err := parseExpiresIn(req.ExpiresIn)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	maxUses := 0
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
	}

	invite, err := h.projects.CreateInvite(r.Context(), info, projectKey, req.Role, expiresAt, maxUses)
	if err != nil {
		handleProjectError(w, r, err, "failed to create invite")
		return
	}

	writeData(w, http.StatusCreated, h.toInviteResponse(invite))
}

// ListInvites handles GET /api/v1/projects/{projectKey}/invites
func (h *ProjectHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	invites, err := h.projects.ListInvites(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to list invites")
		return
	}

	resp := make([]inviteResponse, len(invites))
	for i := range invites {
		resp[i] = h.toInviteResponse(&invites[i])
	}

	writeData(w, http.StatusOK, resp)
}

// DeleteInvite handles DELETE /api/v1/projects/{projectKey}/invites/{inviteId}
func (h *ProjectHandler) DeleteInvite(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	inviteID, err := uuid.Parse(chi.URLParam(r, "inviteId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid invite ID")
		return
	}

	if err := h.projects.DeleteInvite(r.Context(), info, projectKey, inviteID); err != nil {
		handleProjectError(w, r, err, "failed to delete invite")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetInviteInfo handles GET /api/v1/invites/{code}
func (h *ProjectHandler) GetInviteInfo(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	inviteInfo, err := h.projects.GetInviteInfo(r.Context(), code)
	if err != nil {
		handleProjectError(w, r, err, "failed to get invite info")
		return
	}

	writeData(w, http.StatusOK, inviteInfoResponse{
		ProjectName: inviteInfo.ProjectName,
		ProjectKey:  inviteInfo.ProjectKey,
		Role:        inviteInfo.Role,
		Expired:     inviteInfo.Expired,
		Full:        inviteInfo.Full,
	})
}

// AcceptInvite handles POST /api/v1/invites/{code}/accept
func (h *ProjectHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	code := chi.URLParam(r, "code")

	result, err := h.projects.AcceptInvite(r.Context(), info, code)
	if err != nil {
		handleProjectError(w, r, err, "failed to accept invite")
		return
	}

	resp := acceptInviteResponse{
		projectResponse: toProjectResponse(result.Project),
	}
	if result.RoleNotApplied {
		resp.RoleNotApplied = true
		resp.ExistingRole = result.ExistingRole
		resp.InviteRole = result.InviteRole
	}

	writeData(w, http.StatusOK, resp)
}

func parseExpiresIn(s string) (time.Duration, error) {
	switch s {
	case "1h":
		return time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid expires_in value %q; allowed: 1h, 1d, 7d, 30d", s)
	}
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
