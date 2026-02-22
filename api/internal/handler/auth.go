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

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	GlobalRole  string    `json:"global_role"`
}

func toUserResponse(u *model.User) userResponse {
	return userResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		GlobalRole:  u.GlobalRole,
	}
}

// Login authenticates a user with email and password.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required")
		return
	}

	token, user, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
			return
		}
		if errors.Is(err, model.ErrAccountDisabled) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "account is disabled")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("login failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, map[string]interface{}{
		"token":                 token,
		"user":                  toUserResponse(user),
		"force_password_change": user.ForcePasswordChange,
	})
}

// Refresh issues a new JWT token for an authenticated user.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	token, err := h.auth.Refresh(r.Context(), info)
	if err != nil {
		if errors.Is(err, model.ErrAccountDisabled) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "account is disabled")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("token refresh failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, map[string]interface{}{
		"token": token,
	})
}

// Me returns the authenticated user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	user, err := h.auth.GetUser(r.Context(), info.UserID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to get user")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, toUserResponse(user))
}

// Logout is a no-op for stateless JWT auth. The client should discard the token.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// AuthProviders returns which OAuth providers are enabled.
func (h *AuthHandler) AuthProviders(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, map[string]interface{}{
		"discord": h.auth.DiscordEnabled(),
	})
}

// Discord OAuth handlers

type discordCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// DiscordAuth returns the Discord OAuth authorization URL.
func (h *AuthHandler) DiscordAuth(w http.ResponseWriter, r *http.Request) {
	if !h.auth.DiscordEnabled() {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "discord oauth is not configured")
		return
	}

	authURL, err := h.auth.DiscordOAuthURL()
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to generate discord oauth url")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, map[string]string{
		"url": authURL,
	})
}

// DiscordCallback exchanges the authorization code and logs in or registers the user.
func (h *AuthHandler) DiscordCallback(w http.ResponseWriter, r *http.Request) {
	var req discordCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Code == "" || req.State == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "code and state are required")
		return
	}

	token, user, err := h.auth.DiscordCallback(r.Context(), req.Code, req.State)
	if err != nil {
		if errors.Is(err, model.ErrAccountDisabled) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "account is disabled")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("discord oauth callback failed")
		writeError(w, http.StatusUnauthorized, "OAUTH_ERROR", "discord authentication failed")
		return
	}

	writeData(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  toUserResponse(user),
	})
}

// API Key handlers

type createAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	ExpiresAt   *string  `json:"expires_at,omitempty"`
}

type apiKeyResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func toAPIKeyResponse(k *model.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:          k.ID,
		Name:        k.Name,
		KeyPrefix:   k.KeyPrefix,
		Permissions: k.Permissions,
		LastUsedAt:  k.LastUsedAt,
		ExpiresAt:   k.ExpiresAt,
		CreatedAt:   k.CreatedAt,
	}
}

// ListAPIKeys returns all API keys for the authenticated user.
func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	keys, err := h.auth.ListAPIKeys(r.Context(), info.UserID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list api keys")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]apiKeyResponse, len(keys))
	for i := range keys {
		resp[i] = toAPIKeyResponse(&keys[i])
	}

	writeData(w, http.StatusOK, resp)
}

// CreateAPIKey creates a new API key for the authenticated user.
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "expires_at must be in RFC3339 format")
			return
		}
		expiresAt = &t
	}

	if req.Permissions == nil {
		req.Permissions = []string{}
	}

	apiKey, fullKey, err := h.auth.CreateAPIKey(r.Context(), info.UserID, req.Name, req.Permissions, expiresAt)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to create api key")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusCreated, map[string]interface{}{
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"key":         fullKey,
		"key_prefix":  apiKey.KeyPrefix,
		"permissions": apiKey.Permissions,
		"expires_at":  apiKey.ExpiresAt,
		"created_at":  apiKey.CreatedAt,
	})
}

// SearchUsers handles GET /api/v1/users/search?q=...
func (h *AuthHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeData(w, http.StatusOK, []userResponse{})
		return
	}

	users, err := h.auth.SearchUsers(r.Context(), query)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to search users")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := make([]userResponse, len(users))
	for i := range users {
		resp[i] = toUserResponse(&users[i])
	}
	writeData(w, http.StatusOK, resp)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword handles POST /api/v1/auth/change-password.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "old_password and new_password are required")
		return
	}

	if err := h.auth.ChangePassword(r.Context(), info.UserID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid current password")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to change password")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	// Get updated user and generate new token without force_password_change
	user, err := h.auth.GetUser(r.Context(), info.UserID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to get user after password change")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	token, err := h.auth.Refresh(r.Context(), &model.AuthInfo{
		UserID:     user.ID,
		Email:      user.Email,
		GlobalRole: user.GlobalRole,
	})
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to generate token after password change")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, map[string]interface{}{
		"token": token,
	})
}

// DeleteAPIKey deletes an API key by ID.
func (h *AuthHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	keyID, err := uuid.Parse(chi.URLParam(r, "keyId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid key ID")
		return
	}

	if err := h.auth.DeleteAPIKey(r.Context(), keyID, info.UserID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "api key not found")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to delete api key")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
