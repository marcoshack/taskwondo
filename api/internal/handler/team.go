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

// TeamHandler handles team endpoints.
type TeamHandler struct {
	teams *service.TeamService
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(teams *service.TeamService) *TeamHandler {
	return &TeamHandler{teams: teams}
}

// --- Request DTOs ---

type createTeamRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type updateTeamRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type addTeamMemberRequest struct {
	UserID string `json:"user_id"`
}

// --- Response DTOs ---

type teamResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type teamMemberResponse struct {
	ID          uuid.UUID `json:"id"`
	TeamID      uuid.UUID `json:"team_id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func toTeamResponse(t *model.Team) teamResponse {
	return teamResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Name:        t.Name,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toTeamMemberResponse(m *model.TeamMemberWithUser) teamMemberResponse {
	return teamMemberResponse{
		ID:          m.ID,
		TeamID:      m.TeamID,
		UserID:      m.UserID,
		Email:       m.Email,
		DisplayName: m.DisplayName,
		AvatarURL:   m.AvatarURL,
		CreatedAt:   m.CreatedAt,
	}
}

// --- Handlers ---

// List handles GET /api/v1/projects/{projectKey}/teams
func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	teams, err := h.teams.List(r.Context(), info, projectKey)
	if err != nil {
		handleTeamError(w, r, err, "failed to list teams")
		return
	}

	resp := make([]teamResponse, len(teams))
	for i := range teams {
		resp[i] = toTeamResponse(&teams[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/projects/{projectKey}/teams
func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	input := service.CreateTeamInput{
		Name:        req.Name,
		Description: req.Description,
	}

	t, err := h.teams.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleTeamError(w, r, err, "failed to create team")
		return
	}

	writeData(w, http.StatusCreated, toTeamResponse(t))
}

// Get handles GET /api/v1/projects/{projectKey}/teams/{teamId}
func (h *TeamHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	t, err := h.teams.Get(r.Context(), info, projectKey, teamID)
	if err != nil {
		handleTeamError(w, r, err, "failed to get team")
		return
	}

	writeData(w, http.StatusOK, toTeamResponse(t))
}

// Update handles PATCH /api/v1/projects/{projectKey}/teams/{teamId}
func (h *TeamHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	// Decode to raw map for explicit null detection
	raw := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	var input service.UpdateTeamInput

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

	t, err := h.teams.Update(r.Context(), info, projectKey, teamID, input)
	if err != nil {
		handleTeamError(w, r, err, "failed to update team")
		return
	}

	writeData(w, http.StatusOK, toTeamResponse(t))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/teams/{teamId}
func (h *TeamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	if err := h.teams.Delete(r.Context(), info, projectKey, teamID); err != nil {
		handleTeamError(w, r, err, "failed to delete team")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/projects/{projectKey}/teams/{teamId}/members
func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	members, err := h.teams.ListMembers(r.Context(), info, projectKey, teamID)
	if err != nil {
		handleTeamError(w, r, err, "failed to list team members")
		return
	}

	resp := make([]teamMemberResponse, len(members))
	for i := range members {
		resp[i] = toTeamMemberResponse(&members[i])
	}

	writeData(w, http.StatusOK, resp)
}

// AddMember handles POST /api/v1/projects/{projectKey}/teams/{teamId}/members
func (h *TeamHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	var req addTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid user_id")
		return
	}

	member, err := h.teams.AddMember(r.Context(), info, projectKey, teamID, userID)
	if err != nil {
		handleTeamError(w, r, err, "failed to add team member")
		return
	}

	writeData(w, http.StatusCreated, member)
}

// RemoveMember handles DELETE /api/v1/projects/{projectKey}/teams/{teamId}/members/{userId}
func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	userID, ok := parseUUIDParam(w, r, "userId", "invalid user ID")
	if !ok {
		return
	}

	if err := h.teams.RemoveMember(r.Context(), info, projectKey, teamID, userID); err != nil {
		handleTeamError(w, r, err, "failed to remove team member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleTeamError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "team not found")
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
