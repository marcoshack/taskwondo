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

// AdminHandler handles admin endpoints.
type AdminHandler struct {
	admin *service.AdminService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(admin *service.AdminService) *AdminHandler {
	return &AdminHandler{admin: admin}
}

type adminUserResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Email               string     `json:"email"`
	DisplayName         string     `json:"display_name"`
	GlobalRole          string     `json:"global_role"`
	AvatarURL           *string    `json:"avatar_url,omitempty"`
	IsActive            bool       `json:"is_active"`
	ForcePasswordChange bool       `json:"force_password_change"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

func toAdminUserResponse(u *model.User) adminUserResponse {
	return adminUserResponse{
		ID:                  u.ID,
		Email:               u.Email,
		DisplayName:         u.DisplayName,
		GlobalRole:          u.GlobalRole,
		AvatarURL:           u.AvatarURL,
		IsActive:            u.IsActive,
		ForcePasswordChange: u.ForcePasswordChange,
		LastLoginAt:         u.LastLoginAt,
		CreatedAt:           u.CreatedAt,
	}
}

type userProjectResponse struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ProjectKey  string    `json:"project_key"`
	Role        string    `json:"role"`
	OwnerCount  int       `json:"owner_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListUsers handles GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.admin.ListUsers(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list users")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]adminUserResponse, len(users))
	for i := range users {
		resp[i] = toAdminUserResponse(&users[i])
	}
	writeData(w, http.StatusOK, resp)
}

type createUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type createUserResponse struct {
	User              adminUserResponse `json:"user"`
	TemporaryPassword string            `json:"temporary_password"`
}

type resetPasswordResponse struct {
	TemporaryPassword string `json:"temporary_password"`
}

// CreateUser handles POST /api/v1/admin/users.
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Email == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and display_name are required")
		return
	}

	user, password, err := h.admin.CreateUser(r.Context(), req.Email, req.DisplayName)
	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "CONFLICT", "a user with this email already exists")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to create user")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusCreated, createUserResponse{
		User:              toAdminUserResponse(user),
		TemporaryPassword: password,
	})
}

// ResetUserPassword handles POST /api/v1/admin/users/{userId}/reset-password.
func (h *AdminHandler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	password, err := h.admin.ResetUserPassword(r.Context(), userID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to reset user password")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, resetPasswordResponse{
		TemporaryPassword: password,
	})
}

type updateUserRequest struct {
	GlobalRole *string `json:"global_role,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

// UpdateUser handles PATCH /api/v1/admin/users/{userId}.
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.GlobalRole == nil && req.IsActive == nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "at least one field must be provided")
		return
	}

	user, err := h.admin.UpdateUser(r.Context(), info.UserID, userID, req.GlobalRole, req.IsActive)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to update user")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, toAdminUserResponse(user))
}

// ListUserProjects handles GET /api/v1/admin/users/{userId}/projects.
func (h *AdminHandler) ListUserProjects(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	memberships, err := h.admin.ListUserProjects(r.Context(), userID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list user projects")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]userProjectResponse, len(memberships))
	for i, m := range memberships {
		resp[i] = userProjectResponse{
			ProjectID:   m.ProjectID,
			ProjectName: m.ProjectName,
			ProjectKey:  m.ProjectKey,
			Role:        m.Role,
			OwnerCount:  m.OwnerCount,
			CreatedAt:   m.CreatedAt,
		}
	}
	writeData(w, http.StatusOK, resp)
}

type addUserToProjectRequest struct {
	ProjectID string `json:"project_id"`
	Role      string `json:"role"`
}

// AddUserToProject handles POST /api/v1/admin/users/{userId}/projects.
func (h *AdminHandler) AddUserToProject(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	var req addUserToProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.ProjectID == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "project_id and role are required")
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project ID")
		return
	}

	if err := h.admin.AddUserToProject(r.Context(), userID, projectID, req.Role); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user or project not found")
			return
		}
		if errors.Is(err, model.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "CONFLICT", "user is already a member of this project")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to add user to project")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateUserProjectRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserProjectRole handles PATCH /api/v1/admin/users/{userId}/projects/{projectId}.
func (h *AdminHandler) UpdateUserProjectRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project ID")
		return
	}

	var req updateUserProjectRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role is required")
		return
	}

	if err := h.admin.UpdateUserProjectRole(r.Context(), userID, projectID, req.Role); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "membership not found")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to update user project role")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveUserFromProject handles DELETE /api/v1/admin/users/{userId}/projects/{projectId}.
func (h *AdminHandler) RemoveUserFromProject(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project ID")
		return
	}

	if err := h.admin.RemoveUserFromProject(r.Context(), userID, projectID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "membership not found")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to remove user from project")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
