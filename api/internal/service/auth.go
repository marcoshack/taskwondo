package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"

	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/storage"
)

// UserRepository defines the persistence operations the auth service needs for users.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	UpdateDisplayName(ctx context.Context, id uuid.UUID, displayName string) error
	UpdateAvatarURL(ctx context.Context, id uuid.UUID, avatarURL string) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, forceChange bool) error
	Search(ctx context.Context, query string) ([]model.User, error)
}

// APIKeyRepository defines the persistence operations the auth service needs for API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// OAuthAccountRepository defines the persistence operations for OAuth accounts.
type OAuthAccountRepository interface {
	GetByProviderUser(ctx context.Context, provider, providerUserID string) (*model.OAuthAccount, error)
	Create(ctx context.Context, account *model.OAuthAccount) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.OAuthAccount, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// EmailVerificationRepo defines persistence operations for email verification tokens.
type EmailVerificationRepo interface {
	Create(ctx context.Context, token *model.EmailVerificationToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.EmailVerificationToken, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByEmail(ctx context.Context, email string) error
}

// AuthSettingsReader loads system settings by key for auth decisions.
type AuthSettingsReader interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
}

// EmailSender sends emails.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// Claims are the JWT token claims.
type Claims struct {
	jwt.RegisteredClaims
	Email               string `json:"email"`
	GlobalRole          string `json:"role"`
	ForcePasswordChange bool   `json:"force_password_change,omitempty"`
}

const oauthStateExpiry = 10 * time.Minute

const emailVerificationTokenExpiry = 24 * time.Hour

// AuthService handles authentication and authorization.
type AuthService struct {
	users              UserRepository
	apiKeys            APIKeyRepository
	oauthAccounts      OAuthAccountRepository
	emailVerifications EmailVerificationRepo
	settings           AuthSettingsReader
	emailSender        EmailSender
	encryptor          *crypto.Encryptor
	storage            storage.Storage
	baseURL            string
	jwtSecret          []byte
	jwtExpiry          time.Duration
	providers          map[string]OAuthProvider // static providers from env vars (fallback)
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	users UserRepository,
	apiKeys APIKeyRepository,
	oauthAccounts OAuthAccountRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
	providers []OAuthProvider,
) *AuthService {
	pm := make(map[string]OAuthProvider, len(providers))
	for _, p := range providers {
		pm[p.Name()] = p
	}
	return &AuthService{
		users:         users,
		apiKeys:       apiKeys,
		oauthAccounts: oauthAccounts,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiry:     jwtExpiry,
		providers:     pm,
	}
}

// SetEmailVerification configures the email verification dependencies.
func (s *AuthService) SetEmailVerification(repo EmailVerificationRepo, settings AuthSettingsReader, sender EmailSender, baseURL string) {
	s.emailVerifications = repo
	s.settings = settings
	s.emailSender = sender
	s.baseURL = baseURL
}

// SetStorage configures the storage backend for avatar uploads.
func (s *AuthService) SetStorage(store storage.Storage) {
	s.storage = store
}

// SetEncryptor configures the encryptor used to decrypt OAuth client secrets from DB.
func (s *AuthService) SetEncryptor(enc *crypto.Encryptor) {
	s.encryptor = enc
}

// getProvider returns an OAuthProvider for the given name, preferring DB config over static.
// Returns nil if the provider is not configured anywhere.
func (s *AuthService) getProvider(ctx context.Context, name string) OAuthProvider {
	// Try DB config first
	if s.settings != nil && s.encryptor != nil {
		settingKey := model.OAuthConfigSettingKey(name)
		if settingKey != "" {
			setting, err := s.settings.Get(ctx, settingKey)
			if err == nil {
				var cfg model.OAuthProviderConfig
				if err := json.Unmarshal(setting.Value, &cfg); err == nil && cfg.ClientID != "" {
					// Decrypt the client secret
					secret, err := s.encryptor.Decrypt(cfg.ClientSecret)
					if err != nil {
						log.Ctx(ctx).Error().Err(err).Str("provider", name).Msg("failed to decrypt oauth client secret, falling back to static provider")
					} else {
						redirectURI := s.baseURL + "/auth/" + name + "/callback"
						switch name {
						case model.OAuthProviderDiscord:
							return NewDiscordProvider(cfg.ClientID, secret, redirectURI, nil)
						case model.OAuthProviderGoogle:
							return NewGoogleProvider(cfg.ClientID, secret, redirectURI, nil)
						case model.OAuthProviderGitHub:
							return NewGitHubProvider(cfg.ClientID, secret, redirectURI, nil)
						case model.OAuthProviderMicrosoft:
							return NewMicrosoftProvider(cfg.ClientID, secret, redirectURI, nil)
						}
					}
				}
			}
		}
	}

	// Fall back to static provider from env vars
	if p, ok := s.providers[name]; ok {
		return p
	}
	return nil
}

// Login verifies credentials and returns a JWT token and user.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *model.User, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return "", nil, model.ErrInvalidCredentials
		}
		return "", nil, fmt.Errorf("looking up user: %w", err)
	}

	if !user.IsActive {
		return "", nil, model.ErrAccountDisabled
	}

	if user.PasswordHash == "" {
		return "", nil, model.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, model.ErrInvalidCredentials
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to update last login")
	}

	return token, user, nil
}

// Refresh generates a new JWT from a valid existing one.
func (s *AuthService) Refresh(ctx context.Context, info *model.AuthInfo) (string, error) {
	user, err := s.users.GetByID(ctx, info.UserID)
	if err != nil {
		return "", fmt.Errorf("looking up user: %w", err)
	}

	if !user.IsActive {
		return "", model.ErrAccountDisabled
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}

	return token, nil
}

// GetUser returns a user by ID.
func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return user, nil
}

// SearchUsers finds active users matching a query string (by email or display name).
func (s *AuthService) SearchUsers(ctx context.Context, query string) ([]model.User, error) {
	if len(query) < 2 {
		return nil, nil
	}
	users, err := s.users.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	return users, nil
}

// ValidateJWT parses and validates a JWT token string.
func (s *AuthService) ValidateJWT(tokenString string) (*model.AuthInfo, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, model.ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, model.ErrUnauthorized
	}

	return &model.AuthInfo{
		UserID:              userID,
		Email:               claims.Email,
		GlobalRole:          claims.GlobalRole,
		ForcePasswordChange: claims.ForcePasswordChange,
	}, nil
}

// ValidateAPIKey validates an API key and returns auth info.
func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (*model.AuthInfo, error) {
	keyHash := HashAPIKey(key)

	apiKey, err := s.apiKeys.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, model.ErrUnauthorized
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, model.ErrUnauthorized
	}

	if !user.IsActive {
		return nil, model.ErrUnauthorized
	}

	if err := s.apiKeys.UpdateLastUsed(ctx, apiKey.ID); err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("api_key_id", apiKey.ID.String()).Msg("failed to update api key last used")
	}

	return &model.AuthInfo{
		UserID:      user.ID,
		Email:       user.Email,
		GlobalRole:  user.GlobalRole,
		Permissions: apiKey.Permissions,
	}, nil
}

// CreateAPIKey generates a new API key for a user.
func (s *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, name string, permissions []string, expiresAt *time.Time) (*model.APIKey, string, error) {
	for _, p := range permissions {
		if !model.ValidPermissions[p] {
			return nil, "", fmt.Errorf("%w: invalid permission: %s", model.ErrValidation, p)
		}
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generating random bytes: %w", err)
	}

	fullKey := "twk_" + hex.EncodeToString(raw)
	keyPrefix := fullKey[:8]
	keyHash := HashAPIKey(fullKey)

	apiKey := &model.APIKey{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Permissions: permissions,
		ExpiresAt:   expiresAt,
	}

	if err := s.apiKeys.Create(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("creating api key: %w", err)
	}

	return apiKey, fullKey, nil
}

// ListAPIKeys returns all API keys for a user.
func (s *AuthService) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error) {
	return s.apiKeys.ListByUserID(ctx, userID)
}

// DeleteAPIKey deletes an API key, scoped to the owning user.
func (s *AuthService) DeleteAPIKey(ctx context.Context, id, userID uuid.UUID) error {
	return s.apiKeys.Delete(ctx, id, userID)
}

// SeedAdminUser creates an admin user if one doesn't already exist with the given email.
func (s *AuthService) SeedAdminUser(ctx context.Context, email, password string) error {
	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		log.Ctx(ctx).Debug().Str("email", email).Msg("admin user already exists")
		return nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("checking admin user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		DisplayName:  "Admin",
		PasswordHash: string(hash),
		GlobalRole:   model.RoleAdmin,
		IsActive:     true,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return fmt.Errorf("creating admin user: %w", err)
	}

	log.Ctx(ctx).Info().Str("email", email).Msg("admin user created")
	return nil
}

// EnabledProviders returns the names of all enabled auth providers.
// OAuth providers must be both configured (DB or env vars) and enabled in settings.
// When a setting doesn't exist: OAuth defaults to enabled (backward compat),
// email login defaults to enabled, email registration defaults to disabled.
func (s *AuthService) EnabledProviders(ctx context.Context) map[string]bool {
	result := make(map[string]bool, 4)

	// Check each known OAuth provider — configured via DB or static env vars
	for _, name := range []string{model.OAuthProviderDiscord, model.OAuthProviderGoogle, model.OAuthProviderGitHub, model.OAuthProviderMicrosoft} {
		if s.isOAuthConfigured(ctx, name) {
			settingKey := ""
			switch name {
			case model.OAuthProviderDiscord:
				settingKey = model.SettingAuthDiscordEnabled
			case model.OAuthProviderGoogle:
				settingKey = model.SettingAuthGoogleEnabled
			case model.OAuthProviderGitHub:
				settingKey = model.SettingAuthGitHubEnabled
			case model.OAuthProviderMicrosoft:
				settingKey = model.SettingAuthMicrosoftEnabled
			}
			if settingKey != "" {
				result[name] = s.getBoolSetting(ctx, settingKey, true)
			} else {
				result[name] = true
			}
		}
	}

	// Email login: default true (backward compat for existing password users)
	result["email_login"] = s.getBoolSetting(ctx, model.SettingAuthEmailLoginEnabled, true)

	// Email registration: default false (opt-in)
	result["email_registration"] = s.getBoolSetting(ctx, model.SettingAuthEmailRegistrationEnabled, false)

	return result
}

// isOAuthConfigured checks if an OAuth provider has config in DB or static providers.
func (s *AuthService) isOAuthConfigured(ctx context.Context, name string) bool {
	// Check DB config
	if s.settings != nil {
		settingKey := model.OAuthConfigSettingKey(name)
		if settingKey != "" {
			setting, err := s.settings.Get(ctx, settingKey)
			if err == nil {
				var cfg model.OAuthProviderConfig
				if err := json.Unmarshal(setting.Value, &cfg); err == nil && cfg.ClientID != "" {
					return true
				}
			}
		}
	}
	// Check static providers
	_, ok := s.providers[name]
	return ok
}

// getBoolSetting reads a boolean system setting, returning defaultVal if not found.
func (s *AuthService) getBoolSetting(ctx context.Context, key string, defaultVal bool) bool {
	if s.settings == nil {
		return defaultVal
	}
	setting, err := s.settings.Get(ctx, key)
	if err != nil {
		return defaultVal
	}
	var val bool
	if err := json.Unmarshal(setting.Value, &val); err != nil {
		return defaultVal
	}
	return val
}

// RequestRegistration creates a verification token and sends a verification email.
// If inviteCode is non-empty, it is stored with the token so the invite can be
// auto-accepted when the user verifies their email (even from a different device).
func (s *AuthService) RequestRegistration(ctx context.Context, email, displayName, inviteCode string) error {
	if s.emailVerifications == nil || s.emailSender == nil || s.settings == nil {
		return fmt.Errorf("%w: email registration is not configured", model.ErrForbidden)
	}

	// Check registration is enabled
	if !s.getBoolSetting(ctx, model.SettingAuthEmailRegistrationEnabled, false) {
		return fmt.Errorf("%w: email registration is disabled", model.ErrForbidden)
	}

	// Validate inputs
	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("%w: valid email is required", model.ErrValidation)
	}
	if displayName == "" {
		return fmt.Errorf("%w: display name is required", model.ErrValidation)
	}
	if len(displayName) > 100 {
		return fmt.Errorf("%w: display name must be 100 characters or fewer", model.ErrValidation)
	}

	// Check user doesn't already exist
	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return fmt.Errorf("%w: a user with this email already exists", model.ErrAlreadyExists)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("checking existing user: %w", err)
	}

	// Clean up any existing tokens for this email
	if err := s.emailVerifications.DeleteByEmail(ctx, email); err != nil {
		return fmt.Errorf("cleaning up old tokens: %w", err)
	}

	// Generate random token
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return fmt.Errorf("generating random token: %w", err)
	}
	rawToken := hex.EncodeToString(rawBytes)
	tokenHash := hashToken(rawToken)

	token := &model.EmailVerificationToken{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: displayName,
		TokenHash:   tokenHash,
		InviteCode:  inviteCode,
		ExpiresAt:   time.Now().Add(emailVerificationTokenExpiry),
	}
	if err := s.emailVerifications.Create(ctx, token); err != nil {
		return fmt.Errorf("storing verification token: %w", err)
	}

	// Build verification URL and send email
	verifyURL := strings.TrimRight(s.baseURL, "/") + "/verify-email?token=" + rawToken
	htmlBody := verificationEmailHTML(displayName, verifyURL)

	if err := s.emailSender.Send(ctx, email, "Verify your email", htmlBody); err != nil {
		log.Ctx(ctx).Error().Err(err).Str("email", email).Msg("failed to send verification email")
		return fmt.Errorf("sending verification email: %w", err)
	}

	log.Ctx(ctx).Info().Str("email", email).Msg("verification email sent")
	return nil
}

// VerifyEmailResult contains the result of a successful email verification.
type VerifyEmailResult struct {
	Token      string
	User       *model.User
	InviteCode string // non-empty if a project invite should be auto-accepted
}

// VerifyEmailAndCreateUser validates a token, creates the user, and returns a JWT.
func (s *AuthService) VerifyEmailAndCreateUser(ctx context.Context, rawToken, password string) (*VerifyEmailResult, error) {
	if s.emailVerifications == nil {
		return nil, fmt.Errorf("%w: email registration is not configured", model.ErrForbidden)
	}

	tokenHash := hashToken(rawToken)
	verification, err := s.emailVerifications.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("%w: invalid or expired verification token", model.ErrNotFound)
		}
		return nil, fmt.Errorf("looking up verification token: %w", err)
	}

	// Validate password
	if len(password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", model.ErrValidation)
	}
	if len(password) > 72 {
		return nil, fmt.Errorf("%w: password must be 72 characters or fewer", model.ErrValidation)
	}

	// Check email not already taken (race condition guard)
	_, err = s.users.GetByEmail(ctx, verification.Email)
	if err == nil {
		// Clean up the token
		_ = s.emailVerifications.DeleteByTokenHash(ctx, tokenHash)
		return nil, fmt.Errorf("%w: a user with this email already exists", model.ErrAlreadyExists)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}

	// Hash password and create user
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &model.User{
		ID:           uuid.New(),
		Email:        verification.Email,
		DisplayName:  verification.DisplayName,
		PasswordHash: string(hash),
		GlobalRole:   model.RoleUser,
		IsActive:     true,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Delete the used token
	if err := s.emailVerifications.DeleteByTokenHash(ctx, tokenHash); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to delete used verification token")
	}

	// Generate JWT
	jwtToken, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to update last login")
	}

	log.Ctx(ctx).Info().Str("email", user.Email).Msg("user created via email verification")
	return &VerifyEmailResult{
		Token:      jwtToken,
		User:       user,
		InviteCode: verification.InviteCode,
	}, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func verificationEmailHTML(displayName, verifyURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f9fafb;">
<div style="max-width: 480px; margin: 0 auto; background: #fff; border-radius: 8px; padding: 32px; border: 1px solid #e5e7eb;">
<h2 style="margin: 0 0 16px;">Verify your email</h2>
<p>Hi %s,</p>
<p>Click the button below to verify your email address and set your password:</p>
<p style="text-align: center; margin: 24px 0;">
<a href="%s" style="display: inline-block; padding: 12px 24px; background: #4f46e5; color: #fff; text-decoration: none; border-radius: 6px; font-weight: 600;">Verify email</a>
</p>
<p style="color: #6b7280; font-size: 14px;">This link expires in 24 hours. If you didn't request this, you can safely ignore this email.</p>
</div>
</body>
</html>`, html.EscapeString(displayName), verifyURL)
}

// OAuthURL generates the authorization URL for the given provider.
func (s *AuthService) OAuthURL(ctx context.Context, providerName string) (string, error) {
	provider := s.getProvider(ctx, providerName)
	if provider == nil {
		return "", fmt.Errorf("oauth provider %q is not configured", providerName)
	}

	state, err := s.generateOAuthState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}

	return provider.AuthURL(state), nil
}

// OAuthCallback validates state, exchanges the code via the provider, and finds or creates a user.
func (s *AuthService) OAuthCallback(ctx context.Context, providerName, code, state string) (string, *model.User, error) {
	provider := s.getProvider(ctx, providerName)
	if provider == nil {
		return "", nil, fmt.Errorf("oauth provider %q is not configured", providerName)
	}

	if err := s.validateOAuthState(state); err != nil {
		return "", nil, fmt.Errorf("invalid state: %w", err)
	}

	userInfo, err := provider.ExchangeCode(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("exchanging code: %w", err)
	}

	user, err := s.findOrCreateOAuthUser(ctx, providerName, userInfo)
	if err != nil {
		return "", nil, err
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to update last login")
	}

	return token, user, nil
}

func (s *AuthService) findOrCreateOAuthUser(ctx context.Context, provider string, info model.OAuthUserInfo) (*model.User, error) {
	// Case 1: OAuth account already linked — log in existing user.
	existing, err := s.oauthAccounts.GetByProviderUser(ctx, provider, info.ProviderUserID)
	if err == nil {
		user, err := s.users.GetByID(ctx, existing.UserID)
		if err != nil {
			return nil, fmt.Errorf("getting linked user: %w", err)
		}
		if !user.IsActive {
			return nil, model.ErrAccountDisabled
		}
		if info.AvatarURL != "" && !hasUploadedAvatar(user) {
			if err := s.users.UpdateAvatarURL(ctx, user.ID, info.AvatarURL); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to update avatar")
			} else {
				user.AvatarURL = &info.AvatarURL
			}
		}
		return user, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, fmt.Errorf("looking up oauth account: %w", err)
	}

	// Case 2: User with same verified email exists — link and log in.
	var user *model.User
	if info.Email != "" && info.EmailVerified {
		user, err = s.users.GetByEmail(ctx, info.Email)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("looking up user by email: %w", err)
		}
	}

	// Case 3: Create new user.
	if user == nil {
		email := info.Email
		if email == "" {
			email = provider + "_" + info.ProviderUserID + "@oauth.taskwondo.local"
		}

		user = &model.User{
			ID:          uuid.New(),
			Email:       email,
			DisplayName: info.DisplayName,
			GlobalRole:  model.RoleUser,
			IsActive:    true,
		}
		if info.AvatarURL != "" {
			user.AvatarURL = &info.AvatarURL
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("creating oauth user: %w", err)
		}
		log.Ctx(ctx).Info().
			Str("provider", provider).
			Str("provider_user_id", info.ProviderUserID).
			Str("email", email).
			Msg("created new user via oauth")
	} else {
		if !user.IsActive {
			return nil, model.ErrAccountDisabled
		}
		if info.AvatarURL != "" && !hasUploadedAvatar(user) {
			if err := s.users.UpdateAvatarURL(ctx, user.ID, info.AvatarURL); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to update avatar")
			} else {
				user.AvatarURL = &info.AvatarURL
			}
		}
	}

	// Link the OAuth account.
	oauthAccount := &model.OAuthAccount{
		ID:               uuid.New(),
		UserID:           user.ID,
		Provider:         provider,
		ProviderUserID:   info.ProviderUserID,
		ProviderEmail:    info.Email,
		ProviderUsername: info.Username,
		ProviderAvatar:   info.RawAvatar,
	}
	if err := s.oauthAccounts.Create(ctx, oauthAccount); err != nil {
		return nil, fmt.Errorf("linking oauth account: %w", err)
	}

	return user, nil
}

func (s *AuthService) generateOAuthState() (string, error) {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte(ts))
	sig := mac.Sum(nil)
	raw := ts + "." + hex.EncodeToString(sig)
	return base64.URLEncoding.EncodeToString([]byte(raw)), nil
}

func (s *AuthService) validateOAuthState(state string) error {
	raw, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return fmt.Errorf("decoding state: %w", err)
	}

	parts := strings.SplitN(string(raw), ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("malformed state")
	}

	ts, sigHex := parts[0], parts[1]
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}

	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte(ts))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return fmt.Errorf("invalid signature")
	}

	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing timestamp: %w", err)
	}
	if time.Since(time.Unix(unix, 0)) > oauthStateExpiry {
		return fmt.Errorf("state expired")
	}

	return nil
}

func (s *AuthService) generateJWT(user *model.User) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
			Issuer:    "taskwondo",
		},
		Email:               user.Email,
		GlobalRole:          user.GlobalRole,
		ForcePasswordChange: user.ForcePasswordChange,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ChangePassword validates the old password and sets a new one, clearing force_password_change.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("looking up user: %w", err)
	}

	if user.PasswordHash == "" {
		return model.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return model.ErrInvalidCredentials
	}

	if len(newPassword) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters", model.ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.users.UpdatePasswordHash(ctx, userID, string(hash), false); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	return nil
}

// HashAPIKey computes the SHA-256 hash of an API key for storage/lookup.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

const maxAvatarSize = 2 << 20 // 2 MB
const thumbSize = 128
const thumbSizeLarge = 256

// UpdateProfile updates the user's display name.
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, displayName string) (*model.User, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, fmt.Errorf("%w: display name is required", model.ErrValidation)
	}
	if len(displayName) > 100 {
		return nil, fmt.Errorf("%w: display name must be 100 characters or fewer", model.ErrValidation)
	}

	if err := s.users.UpdateDisplayName(ctx, userID, displayName); err != nil {
		return nil, fmt.Errorf("updating display name: %w", err)
	}

	return s.users.GetByID(ctx, userID)
}

// UploadAvatar processes and stores a user's avatar image.
// It generates a 128x128 thumbnail and stores both the original and thumbnail.
func (s *AuthService) UploadAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, size int64, contentType string) (*model.User, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("%w: storage is not configured", model.ErrValidation)
	}

	if size > maxAvatarSize {
		return nil, fmt.Errorf("%w: file must be under 2 MB", model.ErrValidation)
	}

	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, fmt.Errorf("%w: only JPEG and PNG files are allowed", model.ErrValidation)
	}

	// Decode the image
	var src image.Image
	var err error
	switch contentType {
	case "image/jpeg":
		src, err = jpeg.Decode(file)
	case "image/png":
		src, err = png.Decode(file)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: invalid image file", model.ErrValidation)
	}

	// Generate thumbnails (center-crop to square, then resize)
	for _, ts := range []struct {
		size int
		key  string
	}{
		{thumbSize, fmt.Sprintf("avatars/%s/thumb.jpg", userID)},
		{thumbSizeLarge, fmt.Sprintf("avatars/%s/large.jpg", userID)},
	} {
		thumb := generateThumbnail(src, ts.size)
		var buf strings.Builder
		if err := jpeg.Encode(io.Writer(&buf), thumb, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encoding thumbnail (%d): %w", ts.size, err)
		}
		if _, err := s.storage.Put(ctx, ts.key, strings.NewReader(buf.String()), int64(buf.Len()), "image/jpeg"); err != nil {
			return nil, fmt.Errorf("storing thumbnail (%d): %w", ts.size, err)
		}
	}

	// Update avatar_url in DB to the thumb storage key
	if err := s.users.UpdateAvatarURL(ctx, userID, fmt.Sprintf("avatars/%s/thumb.jpg", userID)); err != nil {
		return nil, fmt.Errorf("updating avatar url: %w", err)
	}

	return s.users.GetByID(ctx, userID)
}

// DeleteAvatar removes the user's avatar from storage and clears avatar_url.
func (s *AuthService) DeleteAvatar(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		// Delete both sizes from storage (ignore errors — may be an OAuth URL)
		for _, key := range []string{
			fmt.Sprintf("avatars/%s/thumb.jpg", userID),
			fmt.Sprintf("avatars/%s/large.jpg", userID),
		} {
			if err := s.storage.Delete(ctx, key); err != nil {
				log.Ctx(ctx).Warn().Err(err).Str("key", key).Msg("failed to delete avatar from storage")
			}
		}
	}

	// Clear avatar_url in DB
	if err := s.users.UpdateAvatarURL(ctx, userID, ""); err != nil {
		return nil, fmt.Errorf("clearing avatar url: %w", err)
	}

	return s.users.GetByID(ctx, userID)
}

// GetAvatarFile retrieves a user's avatar thumbnail from storage.
// When size is "large", it returns the 256px variant (falls back to the default 128px thumb).
func (s *AuthService) GetAvatarFile(ctx context.Context, userID uuid.UUID, size string) (io.ReadCloser, string, error) {
	if s.storage == nil {
		return nil, "", model.ErrNotFound
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("getting user: %w", err)
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		return nil, "", model.ErrNotFound
	}

	// If it's an external URL (OAuth avatar), return not found — client should use the URL directly
	if strings.HasPrefix(*user.AvatarURL, "http") {
		return nil, "", model.ErrNotFound
	}

	key := *user.AvatarURL
	if size == "large" {
		largeKey := fmt.Sprintf("avatars/%s/large.jpg", userID)
		if probe, _, err := s.storage.Get(ctx, largeKey); err == nil {
			probe.Close()
			key = largeKey
		}
	}

	reader, info, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, "", fmt.Errorf("getting avatar from storage: %w", err)
	}

	return reader, info.ContentType, nil
}

// generateThumbnail center-crops the source image to a square, then resizes to the given dimension.
func generateThumbnail(src image.Image, size int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Center-crop to square
	var cropRect image.Rectangle
	if w > h {
		offset := (w - h) / 2
		cropRect = image.Rect(bounds.Min.X+offset, bounds.Min.Y, bounds.Min.X+offset+h, bounds.Max.Y)
	} else {
		offset := (h - w) / 2
		cropRect = image.Rect(bounds.Min.X, bounds.Min.Y+offset, bounds.Max.X, bounds.Min.Y+offset+w)
	}

	// Create the cropped sub-image
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	var cropped image.Image
	if si, ok := src.(subImager); ok {
		cropped = si.SubImage(cropRect)
	} else {
		cropped = src // fallback — use full image
	}

	// Resize to target size using high-quality CatmullRom interpolation
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)
	return dst
}

// hasUploadedAvatar returns true when the user has a custom-uploaded avatar
// (stored in our object storage) as opposed to an OAuth provider URL or no avatar.
func hasUploadedAvatar(u *model.User) bool {
	return u.AvatarURL != nil && *u.AvatarURL != "" && !strings.HasPrefix(*u.AvatarURL, "http")
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
