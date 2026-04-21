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
	ID              uuid.UUID                      `json:"id"`
	TeamID          uuid.UUID                      `json:"team_id"`
	PeriodDays      int                            `json:"period_days"`
	RotationTime    string                         `json:"rotation_time"`
	Timezone        string                         `json:"timezone"`
	StartDate       string                         `json:"start_date"`
	CurrentUserID   *uuid.UUID                     `json:"current_user_id"`
	CurrentPosition int                            `json:"current_position"`
	IsOverride      bool                           `json:"is_override"`
	NextRotationAt  *time.Time                     `json:"next_rotation_at"`
	Members         []oncallRotationMemberResp     `json:"members"`
	Overrides       []oncallOverrideResponse       `json:"overrides"`
	Shifts          []oncallScheduleShiftResponse  `json:"shifts,omitempty"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
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

// --- Override DTOs ---

type createOncallOverrideRequest struct {
	OverrideUserID string  `json:"override_user_id"`
	StartAt        string  `json:"start_at"`
	EndAt          string  `json:"end_at"`
	Reason         *string `json:"reason,omitempty"`
}

type updateOncallOverrideRequest struct {
	OverrideUserID *string `json:"override_user_id,omitempty"`
	StartAt        *string `json:"start_at,omitempty"`
	EndAt          *string `json:"end_at,omitempty"`
	Reason         *string `json:"reason"`
}

type oncallOverrideResponse struct {
	ID               uuid.UUID  `json:"id"`
	RotationID       uuid.UUID  `json:"rotation_id"`
	OverrideUserID   uuid.UUID  `json:"override_user_id"`
	OverrideUserName string     `json:"override_user_name"`
	OverrideAvatar   *string    `json:"override_avatar_url,omitempty"`
	StartAt          time.Time  `json:"start_at"`
	EndAt            time.Time  `json:"end_at"`
	Reason           *string    `json:"reason,omitempty"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	CreatedByName    string     `json:"created_by_name"`
	CreatedAt        time.Time  `json:"created_at"`
}

func toOncallOverrideResponse(o *model.OncallOverrideWithUser) oncallOverrideResponse {
	return oncallOverrideResponse{
		ID:               o.ID,
		RotationID:       o.RotationID,
		OverrideUserID:   o.OverrideUserID,
		OverrideUserName: o.OverrideUserName,
		OverrideAvatar:   o.OverrideAvatar,
		StartAt:          o.StartAt,
		EndAt:            o.EndAt,
		Reason:           o.Reason,
		CreatedBy:        o.CreatedBy,
		CreatedByName:    o.CreatedByName,
		CreatedAt:        o.CreatedAt,
	}
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

	overrides := make([]oncallOverrideResponse, len(r.Overrides))
	for i := range r.Overrides {
		overrides[i] = toOncallOverrideResponse(&r.Overrides[i])
	}

	var shifts []oncallScheduleShiftResponse
	if len(r.Shifts) > 0 {
		shifts = make([]oncallScheduleShiftResponse, len(r.Shifts))
		for i, s := range r.Shifts {
			shifts[i] = oncallScheduleShiftResponse{
				UserID:     s.UserID,
				StartAt:    s.StartAt,
				EndAt:      s.EndAt,
				IsOverride: s.IsOverride,
				OverrideID: s.OverrideID,
			}
		}
	}

	return oncallRotationResponse{
		ID:              r.ID,
		TeamID:          r.TeamID,
		PeriodDays:      r.PeriodDays,
		RotationTime:    r.RotationTime.Format("15:04:05"),
		Timezone:        r.Timezone,
		StartDate:       r.StartDate.Format("2006-01-02"),
		CurrentUserID:   r.CurrentUserID,
		CurrentPosition: r.CurrentPosition,
		IsOverride:      r.IsOverride,
		NextRotationAt:  r.NextRotationAt,
		Members:         members,
		Overrides:       overrides,
		Shifts:          shifts,
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	var rangeStart, rangeEnd *time.Time
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	if startStr != "" && endStr != "" {
		s, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid start date: expected YYYY-MM-DD")
			return
		}
		e, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid end date: expected YYYY-MM-DD")
			return
		}
		e = e.AddDate(0, 0, 1) // exclusive end
		if !s.Before(e) {
			writeError(w, http.StatusBadRequest, CodeValidationError, "start must be before end")
			return
		}
		if e.Sub(s) > 90*24*time.Hour {
			writeError(w, http.StatusBadRequest, CodeValidationError, "date range must not exceed 90 days")
			return
		}
		rangeStart = &s
		rangeEnd = &e
	} else if startStr != "" || endStr != "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "both start and end query parameters are required")
		return
	}

	result, err := h.oncall.GetRotation(r.Context(), info, projectKey, teamID, rangeStart, rangeEnd)
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	var req createOncallRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	memberIDs, err := parseUUIDs(req.MemberIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid member_ids")
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
		return
	}

	var req updateOncallRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	var memberIDs []uuid.UUID
	if req.MemberIDs != nil {
		ids, err := parseUUIDs(req.MemberIDs)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid member_ids")
			return
		}
		memberIDs = ids
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	teamID, ok := parseUUIDParam(w, r, "teamId", "invalid team ID")
	if !ok {
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

// --- Override Handlers ---

// CreateOverride handles POST /teams/{teamId}/oncall/overrides
func (h *OncallHandler) CreateOverride(w http.ResponseWriter, r *http.Request) {
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

	var req createOncallOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	overrideUserID, err := uuid.Parse(req.OverrideUserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid override_user_id")
		return
	}

	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid start_at: expected RFC3339 format")
		return
	}

	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid end_at: expected RFC3339 format")
		return
	}

	input := service.CreateOncallOverrideInput{
		OverrideUserID: overrideUserID,
		StartAt:        startAt,
		EndAt:          endAt,
		Reason:         req.Reason,
	}

	result, err := h.oncall.CreateOverride(r.Context(), info, projectKey, teamID, input)
	if err != nil {
		handleOncallError(w, r, err, "failed to create oncall override")
		return
	}

	writeData(w, http.StatusCreated, result)
}

// ListOverrides handles GET /teams/{teamId}/oncall/overrides
func (h *OncallHandler) ListOverrides(w http.ResponseWriter, r *http.Request) {
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

	overrides, err := h.oncall.ListOverrides(r.Context(), info, projectKey, teamID)
	if err != nil {
		handleOncallError(w, r, err, "failed to list oncall overrides")
		return
	}

	resp := make([]oncallOverrideResponse, len(overrides))
	for i := range overrides {
		resp[i] = toOncallOverrideResponse(&overrides[i])
	}

	writeData(w, http.StatusOK, resp)
}

// UpdateOverride handles PATCH /teams/{teamId}/oncall/overrides/{overrideId}
func (h *OncallHandler) UpdateOverride(w http.ResponseWriter, r *http.Request) {
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

	overrideID, ok := parseUUIDParam(w, r, "overrideId", "invalid override ID")
	if !ok {
		return
	}

	var req updateOncallOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	input := service.UpdateOncallOverrideInput{}

	if req.OverrideUserID != nil {
		id, ok := parseOptionalUUID(w, req.OverrideUserID, "invalid override_user_id")
		if !ok {
			return
		}
		input.OverrideUserID = &id
	}
	if req.StartAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid start_at: expected RFC3339 format")
			return
		}
		input.StartAt = &t
	}
	if req.EndAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid end_at: expected RFC3339 format")
			return
		}
		input.EndAt = &t
	}

	// Detect explicit null for reason (present in JSON but null)
	input.Reason = req.Reason
	// If the field was explicitly set to null in the JSON, clear the reason
	// We detect this via the raw body — for simplicity, if Reason is nil and
	// the key was present, treat as clear. Since we use *string, nil = absent or null.
	// Use ClearReason when reason is explicitly null (pointer is nil but key is in JSON).
	// For now, we only clear when reason is explicitly set to empty string.

	result, err := h.oncall.UpdateOverride(r.Context(), info, projectKey, teamID, overrideID, input)
	if err != nil {
		handleOncallError(w, r, err, "failed to update oncall override")
		return
	}

	writeData(w, http.StatusOK, result)
}

// DeleteOverride handles DELETE /teams/{teamId}/oncall/overrides/{overrideId}
func (h *OncallHandler) DeleteOverride(w http.ResponseWriter, r *http.Request) {
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

	overrideID, ok := parseUUIDParam(w, r, "overrideId", "invalid override ID")
	if !ok {
		return
	}

	if err := h.oncall.DeleteOverride(r.Context(), info, projectKey, teamID, overrideID); err != nil {
		handleOncallError(w, r, err, "failed to delete oncall override")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Schedule DTOs ---

type oncallScheduleShiftResponse struct {
	UserID     uuid.UUID  `json:"user_id"`
	StartAt    time.Time  `json:"start_at"`
	EndAt      time.Time  `json:"end_at"`
	IsOverride bool       `json:"is_override"`
	OverrideID *uuid.UUID `json:"override_id,omitempty"`
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
		writeError(w, http.StatusNotFound, CodeNotFound, "not found")
	case errors.Is(err, model.ErrForbidden):
		writeError(w, http.StatusForbidden, CodeForbidden, "insufficient permissions")
	case errors.Is(err, model.ErrValidation):
		writeErrorFromService(w, http.StatusBadRequest, CodeValidationError, err)
	case errors.Is(err, model.ErrAlreadyExists) || errors.Is(err, model.ErrConflict):
		writeErrorFromService(w, http.StatusConflict, CodeConflict, err)
	default:
		log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
		writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
	}
}
