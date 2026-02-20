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

// SystemSettingHandler handles system setting endpoints.
type SystemSettingHandler struct {
	settings *service.SystemSettingService
}

// NewSystemSettingHandler creates a new SystemSettingHandler.
func NewSystemSettingHandler(settings *service.SystemSettingService) *SystemSettingHandler {
	return &SystemSettingHandler{settings: settings}
}

type systemSettingResponse struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func toSystemSettingResponse(s *model.SystemSetting) systemSettingResponse {
	return systemSettingResponse{
		Key:   s.Key,
		Value: s.Value,
	}
}

// List handles GET /api/v1/admin/settings
func (h *SystemSettingHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	settings, err := h.settings.List(r.Context(), info)
	if err != nil {
		handleSystemSettingError(w, r, err, "failed to list system settings")
		return
	}

	resp := make([]systemSettingResponse, len(settings))
	for i := range settings {
		resp[i] = toSystemSettingResponse(&settings[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/admin/settings/{key}
func (h *SystemSettingHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	key := chi.URLParam(r, "key")

	setting, err := h.settings.Get(r.Context(), info, key)
	if err != nil {
		handleSystemSettingError(w, r, err, "failed to get system setting")
		return
	}

	writeData(w, http.StatusOK, toSystemSettingResponse(setting))
}

// Set handles PUT /api/v1/admin/settings/{key}
func (h *SystemSettingHandler) Set(w http.ResponseWriter, r *http.Request) {
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

	setting, err := h.settings.Set(r.Context(), info, key, req.Value)
	if err != nil {
		handleSystemSettingError(w, r, err, "failed to set system setting")
		return
	}

	writeData(w, http.StatusOK, toSystemSettingResponse(setting))
}

// Delete handles DELETE /api/v1/admin/settings/{key}
func (h *SystemSettingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	key := chi.URLParam(r, "key")

	if err := h.settings.Delete(r.Context(), info, key); err != nil {
		handleSystemSettingError(w, r, err, "failed to delete system setting")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPublic handles GET /api/v1/settings/public
func (h *SystemSettingHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.GetPublic(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to get public settings")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, settings)
}

func handleSystemSettingError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
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
