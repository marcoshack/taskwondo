package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/email"
	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// SystemSettingHandler handles system setting endpoints.
type SystemSettingHandler struct {
	settings  *service.SystemSettingService
	encryptor *crypto.Encryptor
	email     *email.Sender
}

// NewSystemSettingHandler creates a new SystemSettingHandler.
func NewSystemSettingHandler(settings *service.SystemSettingService, encryptor *crypto.Encryptor, emailSender *email.Sender) *SystemSettingHandler {
	return &SystemSettingHandler{settings: settings, encryptor: encryptor, email: emailSender}
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

// GetSMTP handles GET /api/v1/admin/settings/smtp_config
func (h *SystemSettingHandler) GetSMTP(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	setting, err := h.settings.Get(r.Context(), info, model.SettingSMTPConfig)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			// Return empty default config
			writeData(w, http.StatusOK, &model.SMTPConfig{
				SMTPPort:   587,
				IMAPPort:   993,
				Encryption: model.SMTPEncryptionSTARTTLS,
			})
			return
		}
		handleSystemSettingError(w, r, err, "failed to get smtp config")
		return
	}

	var cfg model.SMTPConfig
	if err := json.Unmarshal(setting.Value, &cfg); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to parse smtp config")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	// Mask the password
	if cfg.Password != "" {
		cfg.Password = model.PasswordMask
	}

	writeData(w, http.StatusOK, cfg)
}

// SetSMTP handles PUT /api/v1/admin/settings/smtp_config
func (h *SystemSettingHandler) SetSMTP(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var cfg model.SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := cfg.Validate(); err != nil {
		handleSystemSettingError(w, r, err, "smtp config validation failed")
		return
	}

	// Handle password: if masked or empty, preserve the existing encrypted password
	if cfg.Password == "" || cfg.Password == model.PasswordMask {
		existing, err := h.settings.Get(r.Context(), info, model.SettingSMTPConfig)
		if err == nil {
			var existingCfg model.SMTPConfig
			if err := json.Unmarshal(existing.Value, &existingCfg); err == nil {
				cfg.Password = existingCfg.Password
			}
		}
		// If no existing config, password stays empty
	} else {
		// Encrypt the new password
		encrypted, err := h.encryptor.Encrypt(cfg.Password)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to encrypt smtp password")
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}
		cfg.Password = encrypted
	}

	value, err := json.Marshal(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	if _, err := h.settings.Set(r.Context(), info, model.SettingSMTPConfig, value); err != nil {
		handleSystemSettingError(w, r, err, "failed to save smtp config")
		return
	}

	// Return the config with masked password
	if cfg.Password != "" {
		cfg.Password = model.PasswordMask
	}
	writeData(w, http.StatusOK, cfg)
}

// TestSMTP handles POST /api/v1/admin/settings/smtp_config/test
func (h *SystemSettingHandler) TestSMTP(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	if info.GlobalRole != model.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}

	if err := h.email.SendTest(r.Context(), info.Email); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("smtp test email failed")
		writeError(w, http.StatusBadRequest, "SMTP_ERROR", err.Error())
		return
	}

	writeData(w, http.StatusOK, map[string]string{"message": "test email sent successfully"})
}

// validOAuthProviders is the set of allowed OAuth provider names.
var validOAuthProviders = map[string]bool{
	model.OAuthProviderDiscord: true,
	model.OAuthProviderGoogle:  true,
	model.OAuthProviderGitHub:  true,
}

// GetOAuthConfig handles GET /api/v1/admin/settings/oauth_config/{provider}
func (h *SystemSettingHandler) GetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	provider := chi.URLParam(r, "provider")
	if !validOAuthProviders[provider] {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid oauth provider")
		return
	}

	settingKey := model.OAuthConfigSettingKey(provider)
	setting, err := h.settings.Get(r.Context(), info, settingKey)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeData(w, http.StatusOK, &model.OAuthProviderConfig{})
			return
		}
		handleSystemSettingError(w, r, err, "failed to get oauth config")
		return
	}

	var cfg model.OAuthProviderConfig
	if err := json.Unmarshal(setting.Value, &cfg); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to parse oauth config")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	// Mask the client secret
	if cfg.ClientSecret != "" {
		cfg.ClientSecret = model.PasswordMask
	}

	writeData(w, http.StatusOK, cfg)
}

// SetOAuthConfig handles PUT /api/v1/admin/settings/oauth_config/{provider}
func (h *SystemSettingHandler) SetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	provider := chi.URLParam(r, "provider")
	if !validOAuthProviders[provider] {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid oauth provider")
		return
	}

	var cfg model.OAuthProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := cfg.Validate(); err != nil {
		handleSystemSettingError(w, r, err, "oauth config validation failed")
		return
	}

	settingKey := model.OAuthConfigSettingKey(provider)

	// Handle client secret: if masked or empty, preserve the existing encrypted value
	if cfg.ClientSecret == "" || cfg.ClientSecret == model.PasswordMask {
		existing, err := h.settings.Get(r.Context(), info, settingKey)
		if err == nil {
			var existingCfg model.OAuthProviderConfig
			if err := json.Unmarshal(existing.Value, &existingCfg); err == nil {
				cfg.ClientSecret = existingCfg.ClientSecret
			}
		}
	} else {
		// Encrypt the new client secret
		encrypted, err := h.encryptor.Encrypt(cfg.ClientSecret)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to encrypt oauth client secret")
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}
		cfg.ClientSecret = encrypted
	}

	value, err := json.Marshal(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	if _, err := h.settings.Set(r.Context(), info, settingKey, value); err != nil {
		handleSystemSettingError(w, r, err, "failed to save oauth config")
		return
	}

	// Return the config with masked client secret
	if cfg.ClientSecret != "" {
		cfg.ClientSecret = model.PasswordMask
	}
	writeData(w, http.StatusOK, cfg)
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
