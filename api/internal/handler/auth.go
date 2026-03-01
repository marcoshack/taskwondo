package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// InviteAcceptor accepts a project invite on behalf of a user.
type InviteAcceptor interface {
	AcceptInvite(ctx context.Context, info *model.AuthInfo, code string) (*service.AcceptInviteResult, error)
}

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	auth    *service.AuthService
	invites InviteAcceptor
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(auth *service.AuthService, invites InviteAcceptor) *AuthHandler {
	return &AuthHandler{auth: auth, invites: invites}
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
	AvatarURL   *string   `json:"avatar_url,omitempty"`
}

func toUserResponse(u *model.User) userResponse {
	resp := userResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		GlobalRole:  u.GlobalRole,
	}
	resp.AvatarURL = avatarURL(u.AvatarURL, u.ID, u.UpdatedAt.Unix())
	return resp
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

// AuthProviders returns which auth providers are enabled.
func (h *AuthHandler) AuthProviders(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, h.auth.EnabledProviders(r.Context()))
}

// OAuth handlers (generic for all providers)

type oauthCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// OAuthAuth returns the OAuth authorization URL for the given provider.
func (h *AuthHandler) OAuthAuth(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	authURL, err := h.auth.OAuthURL(r.Context(), provider)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", provider+" oauth is not configured")
		return
	}

	writeData(w, http.StatusOK, map[string]string{
		"url": authURL,
	})
}

// OAuthCallback exchanges the authorization code and logs in or registers the user.
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	var req oauthCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Code == "" || req.State == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "code and state are required")
		return
	}

	token, user, err := h.auth.OAuthCallback(r.Context(), provider, req.Code, req.State)
	if err != nil {
		if errors.Is(err, model.ErrAccountDisabled) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "account is disabled")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg(provider + " oauth callback failed")
		writeError(w, http.StatusUnauthorized, "OAUTH_ERROR", provider+" authentication failed")
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
		if t.Before(time.Now()) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "expires_at must be in the future")
			return
		}
		expiresAt = &t
	}

	if req.Permissions == nil {
		req.Permissions = []string{}
	}

	apiKey, fullKey, err := h.auth.CreateAPIKey(r.Context(), info.UserID, req.Name, req.Permissions, expiresAt)
	if err != nil {
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
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

// Registration handlers

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	InviteCode  string `json:"invite_code,omitempty"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Email == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and display_name are required")
		return
	}

	if err := h.auth.RequestRegistration(r.Context(), req.Email, req.DisplayName, req.InviteCode); err != nil {
		if errors.Is(err, model.ErrForbidden) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "email registration is disabled")
			return
		}
		if errors.Is(err, model.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "CONFLICT", "a user with this email already exists")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("registration failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, map[string]string{
		"message": "verification email sent",
	})
}

type verifyEmailRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// VerifyEmail handles POST /api/v1/auth/verify-email.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Token == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "token and password are required")
		return
	}

	result, err := h.auth.VerifyEmailAndCreateUser(r.Context(), req.Token, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "invalid or expired verification token")
			return
		}
		if errors.Is(err, model.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "CONFLICT", "a user with this email already exists")
			return
		}
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if errors.Is(err, model.ErrForbidden) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "email registration is not configured")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("email verification failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	resp := map[string]interface{}{
		"token": result.Token,
		"user":  toUserResponse(result.User),
	}

	// Auto-accept project invite if one was stored with the verification token
	if result.InviteCode != "" && h.invites != nil {
		authInfo := &model.AuthInfo{UserID: result.User.ID, GlobalRole: result.User.GlobalRole}
		inviteResult, err := h.invites.AcceptInvite(r.Context(), authInfo, result.InviteCode)
		if err != nil {
			log.Ctx(r.Context()).Warn().Err(err).Str("invite_code", result.InviteCode).Msg("failed to auto-accept invite after registration")
		} else {
			resp["project_key"] = inviteResult.Project.Key
		}
	}

	writeData(w, http.StatusOK, resp)
}

// Profile endpoints

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

// UpdateProfile handles PATCH /api/v1/user/profile.
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	user, err := h.auth.UpdateProfile(r.Context(), info.UserID, req.DisplayName)
	if err != nil {
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to update profile")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, toUserResponse(user))
}

// UploadAvatar handles POST /api/v1/user/avatar.
func (h *AuthHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "only JPEG and PNG files are allowed")
		return
	}

	if header.Size > 2<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "VALIDATION_ERROR", "file must be under 2 MB")
		return
	}

	user, err := h.auth.UploadAvatar(r.Context(), info.UserID, file, header.Size, contentType)
	if err != nil {
		if errors.Is(err, model.ErrValidation) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to upload avatar")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, toUserResponse(user))
}

// DeleteAvatar handles DELETE /api/v1/user/avatar.
func (h *AuthHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	user, err := h.auth.DeleteAvatar(r.Context(), info.UserID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to delete avatar")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeData(w, http.StatusOK, toUserResponse(user))
}

// GetUserAvatar handles GET /api/v1/users/{userId}/avatar.
func (h *AuthHandler) GetUserAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user ID")
		return
	}

	size := r.URL.Query().Get("size")
	reader, contentType, err := h.auth.GetAvatarFile(r.Context(), userID, size)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "avatar not found")
			return
		}
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to get avatar")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	io.Copy(w, reader)
}
