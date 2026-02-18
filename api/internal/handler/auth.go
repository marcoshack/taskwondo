package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/model"
	"github.com/marcoshack/trackforge/internal/service"
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
		"token": token,
		"user":  toUserResponse(user),
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
