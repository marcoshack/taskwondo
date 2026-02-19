package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// UserSettingHandler handles user setting endpoints.
type UserSettingHandler struct {
	settings *service.UserSettingService
}

// NewUserSettingHandler creates a new UserSettingHandler.
func NewUserSettingHandler(settings *service.UserSettingService) *UserSettingHandler {
	return &UserSettingHandler{settings: settings}
}

type setUserSettingRequest struct {
	Value json.RawMessage `json:"value"`
}

type userSettingResponse struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func toUserSettingResponse(s *model.UserSetting) userSettingResponse {
	return userSettingResponse{
		Key:   s.Key,
		Value: s.Value,
	}
}

// List handles GET /api/v1/projects/{projectKey}/user-settings
func (h *UserSettingHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	settings, err := h.settings.List(r.Context(), info, projectKey)
	if err != nil {
		handleUserSettingError(w, r, err, "failed to list user settings")
		return
	}

	resp := make([]userSettingResponse, len(settings))
	for i := range settings {
		resp[i] = toUserSettingResponse(&settings[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/projects/{projectKey}/user-settings/{key}
func (h *UserSettingHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	key := chi.URLParam(r, "key")

	setting, err := h.settings.Get(r.Context(), info, projectKey, key)
	if err != nil {
		handleUserSettingError(w, r, err, "failed to get user setting")
		return
	}

	writeData(w, http.StatusOK, toUserSettingResponse(setting))
}

// Set handles PUT /api/v1/projects/{projectKey}/user-settings/{key}
func (h *UserSettingHandler) Set(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	key := chi.URLParam(r, "key")

	var req setUserSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if len(req.Value) == 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "value is required")
		return
	}

	setting, err := h.settings.Set(r.Context(), info, projectKey, key, req.Value)
	if err != nil {
		handleUserSettingError(w, r, err, "failed to set user setting")
		return
	}

	writeData(w, http.StatusOK, toUserSettingResponse(setting))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/user-settings/{key}
func (h *UserSettingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	key := chi.URLParam(r, "key")

	if err := h.settings.Delete(r.Context(), info, projectKey, key); err != nil {
		handleUserSettingError(w, r, err, "failed to delete user setting")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListGlobal handles GET /api/v1/user/preferences
func (h *UserSettingHandler) ListGlobal(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	settings, err := h.settings.ListGlobal(r.Context(), info)
	if err != nil {
		handleUserSettingError(w, r, err, "failed to list global preferences")
		return
	}

	resp := make([]userSettingResponse, len(settings))
	for i := range settings {
		resp[i] = toUserSettingResponse(&settings[i])
	}

	writeData(w, http.StatusOK, resp)
}

// GetGlobal handles GET /api/v1/user/preferences/{key}
func (h *UserSettingHandler) GetGlobal(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	key := chi.URLParam(r, "key")

	setting, err := h.settings.GetGlobal(r.Context(), info, key)
	if err != nil {
		handleUserSettingError(w, r, err, "failed to get global preference")
		return
	}

	writeData(w, http.StatusOK, toUserSettingResponse(setting))
}

// SetGlobal handles PUT /api/v1/user/preferences/{key}
func (h *UserSettingHandler) SetGlobal(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	key := chi.URLParam(r, "key")

	var req setUserSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if len(req.Value) == 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "value is required")
		return
	}

	setting, err := h.settings.SetGlobal(r.Context(), info, key, req.Value)
	if err != nil {
		handleUserSettingError(w, r, err, "failed to set global preference")
		return
	}

	writeData(w, http.StatusOK, toUserSettingResponse(setting))
}

// DeleteGlobal handles DELETE /api/v1/user/preferences/{key}
func (h *UserSettingHandler) DeleteGlobal(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	key := chi.URLParam(r, "key")

	if err := h.settings.DeleteGlobal(r.Context(), info, key); err != nil {
		handleUserSettingError(w, r, err, "failed to delete global preference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleUserSettingError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "setting not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
