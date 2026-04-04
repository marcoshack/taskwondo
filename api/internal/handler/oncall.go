package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// OncallHandler handles on-call rotation endpoints.
type OncallHandler struct {
	oncall *service.OncallService
}

// NewOncallHandler creates a new OncallHandler.
func NewOncallHandler(oncall *service.OncallService) *OncallHandler {
	return &OncallHandler{oncall: oncall}
}

// --- Request DTOs ---

type createOncallRotationRequest struct {
	PeriodDays   int      `json:"period_days"`
	RotationTime string   `json:"rotation_time"`
	Timezone     string   `json:"timezone"`
	StartDate    string   `json:"start_date"`
	MemberIDs    []string `json:"member_ids"`
}

type updateOncallRotationRequest struct {
	PeriodDays   *int     `json:"period_days,omitempty"`
	RotationTime *string  `json:"rotation_time,omitempty"`
	Timezone     *string  `json:"timezone,omitempty"`
	StartDate    *string  `json:"start_date,omitempty"`
	MemberIDs    []string `json:"member_ids,omitempty"`
}

// --- Response DTOs ---

type oncallRotationResponse struct {
	ID              uuid.UUID                    `json:"id"`
	TeamID          uuid.UUID                    `json:"team_id"`
	PeriodDays      int                          `json:"period_days"`
	RotationTime    string                       `json:"rotation_time"`
	Timezone        string                       `json:"timezone"`
	StartDate       string                       `json:"start_date"`
	CurrentUserID   *uuid.UUID                   `json:"current_user_id"`
	CurrentPosition int                          `json:"current_position"`
	NextRotationAt  *time.Time                   `json:"next_rotation_at"`
	Members         []oncallRotationMemberResp   `json:"members"`
	CreatedAt       time.Time                    `json:"created_at"`
	UpdatedAt       time.Time                    `json:"updated_at"`
}

type oncallRotationMemberResp struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Position    int       `json:"position"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
}

type oncallHistoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	RotationID  uuid.UUID  `json:"rotation_id"`
	UserID      uuid.UUID  `json:"user_id"`
	DisplayName string     `json:"display_name"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type oncallHistoryListResponse struct {
	Data   []oncallHistoryResponse `json:"data"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

func toOncallRotationResponse(r *service.OncallRotationResult) oncallRotationResponse {
	members := make([]oncallRotationMemberResp, len(r.Members))
	for i, m := range r.Members {
		members[i] = oncallRotationMemberResp{
			ID:          m.ID,
			UserID:      m.UserID,
			Position:    m.Position,
			Email:       m.Email,
			DisplayName: m.DisplayName,
			AvatarURL:   m.AvatarURL,
		}
	}
	return oncallRotationResponse{
		ID:              r.ID,
		TeamID:          r.TeamID,
		PeriodDays:      r.PeriodDays,
		RotationTime:    r.RotationTime,
		Timezone:        r.Timezone,
		StartDate:       r.StartDate,
		CurrentUserID:   r.CurrentUserID,
		CurrentPosition: r.CurrentPosition,
		NextRotationAt:  r.NextRotationAt,
		Members:         members,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func toOncallHistoryResponse(h *model.OncallRotationHistoryWithUser) oncallHistoryResponse {
	return oncallHistoryResponse{
		ID:          h.ID,
		RotationID:  h.RotationID,
		UserID:      h.UserID,
		DisplayName: h.DisplayName,
		AvatarURL:   h.AvatarURL,
		StartedAt:   h.StartedAt,
		EndedAt:     h.EndedAt,
		CreatedAt:   h.CreatedAt,
	}
}

// --- Handlers ---

// Get handles GET /teams/{teamId}/oncall
func (h *OncallHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid team ID")
		return
	}

	result, err := h.oncall.GetRotation(r.Context(), info, projectKey, teamID)
	if err != nil {
		handleOncallError(w, r, err, "failed to get oncall rotation")
		return
	}

	writeData(w, http.StatusOK, toOncallRotationResponse(result))
}

// Create handles POST /teams/{teamId}/oncall
func (h *OncallHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid team ID")
		return
	}

	var req createOncallRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	memberIDs, err := parseUUIDs(req.MemberIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member_ids")
		return
	}

	input := service.CreateOncallRotationInput{
		PeriodDays:   req.PeriodDays,
		RotationTime: req.RotationTime,
		Timezone:     req.Timezone,
		StartDate:    req.StartDate,
		MemberIDs:    memberIDs,
	}

	result, err := h.oncall.CreateRotation(r.Context(), info, projectKey, teamID, input)
	if err != nil {
		handleOncallError(w, r, err, "failed to create oncall rotation")
		return
	}

	writeData(w, http.StatusCreated, toOncallRotationResponse(result))
}

// Update handles PATCH /teams/{teamId}/oncall
func (h *OncallHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid team ID")
		return
	}

	var req updateOncallRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	var memberIDs []uuid.UUID
	if req.MemberIDs != nil {
		memberIDs, err = parseUUIDs(req.MemberIDs)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member_ids")
			return
		}
	}

	input := service.UpdateOncallRotationInput{
		PeriodDays:   req.PeriodDays,
		RotationTime: req.RotationTime,
		Timezone:     req.Timezone,
		StartDate:    req.StartDate,
		MemberIDs:    memberIDs,
	}

	result, err := h.oncall.UpdateRotation(r.Context(), info, projectKey, teamID, input)
	if err != nil {
		handleOncallError(w, r, err, "failed to update oncall rotation")
		return
	}

	writeData(w, http.StatusOK, toOncallRotationResponse(result))
}

// Delete handles DELETE /teams/{teamId}/oncall
func (h *OncallHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid team ID")
		return
	}

	if err := h.oncall.DeleteRotation(r.Context(), info, projectKey, teamID); err != nil {
		handleOncallError(w, r, err, "failed to delete oncall rotation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListHistory handles GET /teams/{teamId}/oncall/history
func (h *OncallHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid team ID")
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	history, total, err := h.oncall.ListHistory(r.Context(), info, projectKey, teamID, limit, offset)
	if err != nil {
		handleOncallError(w, r, err, "failed to list oncall history")
		return
	}

	resp := make([]oncallHistoryResponse, len(history))
	for i := range history {
		resp[i] = toOncallHistoryResponse(&history[i])
	}

	writeJSON(w, http.StatusOK, oncallHistoryListResponse{
		Data:   resp,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// --- Helpers ---

func parseUUIDs(strs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(strs))
	for i, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func handleOncallError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, model.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	case errors.Is(err, model.ErrValidation):
		writeErrorFromService(w, http.StatusBadRequest, "VALIDATION_ERROR", err)
	case errors.Is(err, model.ErrAlreadyExists) || errors.Is(err, model.ErrConflict):
		writeErrorFromService(w, http.StatusConflict, "CONFLICT", err)
	default:
		log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
