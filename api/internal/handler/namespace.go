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

// NamespaceHandler handles namespace and namespace member endpoints.
type NamespaceHandler struct {
	namespaces *service.NamespaceService
}

// NewNamespaceHandler creates a new NamespaceHandler.
func NewNamespaceHandler(namespaces *service.NamespaceService) *NamespaceHandler {
	return &NamespaceHandler{namespaces: namespaces}
}

// --- Request DTOs ---

type createNamespaceRequest struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}

type updateNamespaceRequest struct {
	Slug        *string `json:"slug,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type addNamespaceMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type updateNamespaceMemberRoleRequest struct {
	Role string `json:"role"`
}

type migrateProjectRequest struct {
	TargetNamespace string `json:"target_namespace"`
}

// --- Response DTOs ---

type namespaceResponse struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type namespaceMemberResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func toNamespaceResponse(ns *model.Namespace) namespaceResponse {
	return namespaceResponse{
		ID:          ns.ID,
		Slug:        ns.Slug,
		DisplayName: ns.DisplayName,
		Icon:        ns.Icon,
		Color:       ns.Color,
		IsDefault:   ns.IsDefault,
		CreatedAt:   ns.CreatedAt,
		UpdatedAt:   ns.UpdatedAt,
	}
}

func toNamespaceMemberResponse(m *model.NamespaceMemberWithUser) namespaceMemberResponse {
	return namespaceMemberResponse{
		UserID:      m.UserID,
		Email:       m.Email,
		DisplayName: m.DisplayName,
		AvatarURL:   avatarURL(m.AvatarURL, m.UserID, 0),
		Role:        m.Role,
		CreatedAt:   m.CreatedAt,
	}
}

// --- Namespace Handlers ---

// Create handles POST /api/v1/namespaces
func (h *NamespaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req createNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Slug == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "slug is required")
		return
	}
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "display_name is required")
		return
	}

	ns, err := h.namespaces.CreateNamespace(r.Context(), info, req.Slug, req.DisplayName)
	if err != nil {
		handleNamespaceError(w, r, err, "failed to create namespace")
		return
	}

	writeData(w, http.StatusCreated, toNamespaceResponse(ns))
}

// List handles GET /api/v1/namespaces
func (h *NamespaceHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	namespaces, err := h.namespaces.ListUserNamespaces(r.Context(), info)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list namespaces")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	ownedCount, err := h.namespaces.CountOwnedByUser(r.Context(), info.UserID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to count owned namespaces")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	effectiveLimit, err := h.namespaces.ResolveEffectiveLimit(r.Context(), info)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to resolve namespace limit")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]namespaceResponse, len(namespaces))
	for i := range namespaces {
		resp[i] = toNamespaceResponse(&namespaces[i])
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": resp,
		"meta": map[string]interface{}{
			"owned_namespace_count": ownedCount,
			"max_namespaces":        effectiveLimit,
		},
	})
}

// Get handles GET /api/v1/namespaces/{slug}
func (h *NamespaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	ns, err := h.namespaces.GetNamespace(r.Context(), info, slug)
	if err != nil {
		handleNamespaceError(w, r, err, "failed to get namespace")
		return
	}

	writeData(w, http.StatusOK, toNamespaceResponse(ns))
}

// Update handles PATCH /api/v1/namespaces/{slug}
func (h *NamespaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	var req updateNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	ns, err := h.namespaces.UpdateNamespace(r.Context(), info, slug, req.Slug, req.DisplayName, req.Icon, req.Color)
	if err != nil {
		handleNamespaceError(w, r, err, "failed to update namespace")
		return
	}

	writeData(w, http.StatusOK, toNamespaceResponse(ns))
}

// Delete handles DELETE /api/v1/namespaces/{slug}
func (h *NamespaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	if err := h.namespaces.DeleteNamespace(r.Context(), info, slug); err != nil {
		handleNamespaceError(w, r, err, "failed to delete namespace")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Namespace Member Handlers ---

// AddMember handles POST /api/v1/namespaces/{slug}/members
func (h *NamespaceHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	var req addNamespaceMemberRequest
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

	member, err := h.namespaces.AddNamespaceMember(r.Context(), info, slug, userID, req.Role)
	if err != nil {
		handleNamespaceError(w, r, err, "failed to add namespace member")
		return
	}

	writeData(w, http.StatusCreated, map[string]interface{}{
		"namespace_id": member.NamespaceID,
		"user_id":      member.UserID,
		"role":         member.Role,
	})
}

// ListMembers handles GET /api/v1/namespaces/{slug}/members
func (h *NamespaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	members, err := h.namespaces.ListNamespaceMembers(r.Context(), info, slug)
	if err != nil {
		handleNamespaceError(w, r, err, "failed to list namespace members")
		return
	}

	resp := make([]namespaceMemberResponse, len(members))
	for i := range members {
		resp[i] = toNamespaceMemberResponse(&members[i])
	}

	writeData(w, http.StatusOK, resp)
}

// UpdateMemberRole handles PUT /api/v1/namespaces/{slug}/members/{userId}
func (h *NamespaceHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	var req updateNamespaceMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role is required")
		return
	}

	if err := h.namespaces.UpdateNamespaceMemberRole(r.Context(), info, slug, userID, req.Role); err != nil {
		handleNamespaceError(w, r, err, "failed to update namespace member role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember handles DELETE /api/v1/namespaces/{slug}/members/{userId}
func (h *NamespaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	if err := h.namespaces.RemoveNamespaceMember(r.Context(), info, slug, userID); err != nil {
		handleNamespaceError(w, r, err, "failed to remove namespace member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Project Migration Handler ---

// MigrateProject handles POST /api/v1/namespaces/{slug}/projects/{projectKey}/migrate
func (h *NamespaceHandler) MigrateProject(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	slug := chi.URLParam(r, "slug")
	projectKey := chi.URLParam(r, "projectKey")

	var req migrateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.TargetNamespace == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "target_namespace is required")
		return
	}

	if err := h.namespaces.MigrateProject(r.Context(), info, projectKey, slug, req.TargetNamespace); err != nil {
		handleNamespaceError(w, r, err, "failed to migrate project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleNamespaceError maps service errors to HTTP responses.
func handleNamespaceError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrAlreadyExists) {
		writeErrorFromService(w, http.StatusConflict, "CONFLICT", err)
		return
	}
	if errors.Is(err, model.ErrNamespacesDisabled) {
		writeError(w, http.StatusForbidden, "NAMESPACES_DISABLED", "namespace feature is not enabled")
		return
	}
	if errors.Is(err, model.ErrNamespaceNotEmpty) {
		writeErrorFromService(w, http.StatusConflict, "NAMESPACE_NOT_EMPTY", err)
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeErrorFromService(w, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}
	if errors.Is(err, model.ErrConflict) {
		writeErrorFromService(w, http.StatusConflict, "CONFLICT", err)
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
