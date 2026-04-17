package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// InviteHandler resolves invite codes against both project and namespace invites.
// Invites use the same URL format (/invite/:code) regardless of resource type.
type InviteHandler struct {
	projects   *service.ProjectService
	namespaces *service.NamespaceService
}

// NewInviteHandler creates a new InviteHandler.
func NewInviteHandler(projects *service.ProjectService, namespaces *service.NamespaceService) *InviteHandler {
	return &InviteHandler{projects: projects, namespaces: namespaces}
}

// inviteInfoResponseUnified describes an invite for the join page. It includes a
// `type` discriminator plus the resource-specific fields.
type inviteInfoResponseUnified struct {
	Type string `json:"type"` // "project" or "namespace"
	Role string `json:"role"`
	// Project fields (populated when type == "project")
	ProjectName string `json:"project_name,omitempty"`
	ProjectKey  string `json:"project_key,omitempty"`
	// Namespace fields (populated when type == "namespace")
	NamespaceSlug        string `json:"namespace_slug,omitempty"`
	NamespaceDisplayName string `json:"namespace_display_name,omitempty"`
	Expired              bool   `json:"expired"`
	Full                 bool   `json:"full"`
}

// GetInviteInfo handles GET /api/v1/invites/{code}. Tries project invites first,
// then falls back to namespace invites.
func (h *InviteHandler) GetInviteInfo(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if info, err := h.projects.GetInviteInfo(r.Context(), code); err == nil {
		writeData(w, http.StatusOK, inviteInfoResponseUnified{
			Type:        "project",
			Role:        info.Role,
			ProjectName: info.ProjectName,
			ProjectKey:  info.ProjectKey,
			Expired:     info.Expired,
			Full:        info.Full,
		})
		return
	} else if !errors.Is(err, model.ErrNotFound) {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to get project invite info")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	nsInfo, err := h.namespaces.GetNamespaceInviteInfo(r.Context(), code)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "invite not found")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to get namespace invite info")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, inviteInfoResponseUnified{
		Type:                 "namespace",
		Role:                 nsInfo.Role,
		NamespaceSlug:        nsInfo.NamespaceSlug,
		NamespaceDisplayName: nsInfo.NamespaceDisplayName,
		Expired:              nsInfo.Expired,
		Full:                 nsInfo.Full,
	})
}

// acceptInviteResponseUnified describes the accepted invite.
type acceptInviteResponseUnified struct {
	Type string `json:"type"` // "project" or "namespace"
	// Project fields (populated when type == "project")
	Project              *projectResponse `json:"project,omitempty"`
	ProjectNamespaceSlug string           `json:"project_namespace_slug,omitempty"`
	// Namespace fields (populated when type == "namespace")
	NamespaceSlug        string `json:"namespace_slug,omitempty"`
	NamespaceDisplayName string `json:"namespace_display_name,omitempty"`

	RoleNotApplied bool   `json:"role_not_applied,omitempty"`
	ExistingRole   string `json:"existing_role,omitempty"`
	InviteRole     string `json:"invite_role,omitempty"`
}

// AcceptInvite handles POST /api/v1/invites/{code}/accept. Tries project invites
// first, then falls back to namespace invites.
func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	code := chi.URLParam(r, "code")

	// Try project invite first (read-only check before mutation)
	if _, err := h.projects.GetInviteInfo(r.Context(), code); err == nil {
		result, err := h.projects.AcceptInvite(r.Context(), info, code)
		if err != nil {
			handleProjectError(w, r, err, "failed to accept invite")
			return
		}
		proj := toProjectResponse(result.Project)
		resp := acceptInviteResponseUnified{
			Type:                 "project",
			Project:              &proj,
			ProjectNamespaceSlug: result.NamespaceSlug,
		}
		if result.RoleNotApplied {
			resp.RoleNotApplied = true
			resp.ExistingRole = result.ExistingRole
			resp.InviteRole = result.InviteRole
		}
		writeData(w, http.StatusOK, resp)
		return
	} else if !errors.Is(err, model.ErrNotFound) {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to look up project invite")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	// Fall back to namespace invite
	result, err := h.namespaces.AcceptNamespaceInvite(r.Context(), info, code)
	if err != nil {
		handleNamespaceError(w, r, err, "failed to accept namespace invite")
		return
	}

	resp := acceptInviteResponseUnified{
		Type:                 "namespace",
		NamespaceSlug:        result.Namespace.Slug,
		NamespaceDisplayName: result.Namespace.DisplayName,
	}
	if result.RoleNotApplied {
		resp.RoleNotApplied = true
		resp.ExistingRole = result.ExistingRole
		resp.InviteRole = result.InviteRole
	}
	writeData(w, http.StatusOK, resp)
}
