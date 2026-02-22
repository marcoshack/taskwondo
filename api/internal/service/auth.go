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
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/marcoshack/taskwondo/internal/model"
)

// UserRepository defines the persistence operations the auth service needs for users.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
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

// Claims are the JWT token claims.
type Claims struct {
	jwt.RegisteredClaims
	Email               string `json:"email"`
	GlobalRole          string `json:"role"`
	ForcePasswordChange bool   `json:"force_password_change,omitempty"`
}

const oauthStateExpiry = 10 * time.Minute

// AuthService handles authentication and authorization.
type AuthService struct {
	users              UserRepository
	apiKeys            APIKeyRepository
	oauthAccounts      OAuthAccountRepository
	jwtSecret          []byte
	jwtExpiry          time.Duration
	discordClientID    string
	discordSecret      string
	discordRedirectURI string
	httpClient         *http.Client
	discordBaseURL     string
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	users UserRepository,
	apiKeys APIKeyRepository,
	oauthAccounts OAuthAccountRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
	discordClientID, discordSecret, discordRedirectURI string,
) *AuthService {
	return &AuthService{
		users:              users,
		apiKeys:            apiKeys,
		oauthAccounts:      oauthAccounts,
		jwtSecret:          []byte(jwtSecret),
		jwtExpiry:          jwtExpiry,
		discordClientID:    discordClientID,
		discordSecret:      discordSecret,
		discordRedirectURI: discordRedirectURI,
		httpClient:         http.DefaultClient,
		discordBaseURL:     "https://discord.com",
	}
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
		UserID:     user.ID,
		Email:      user.Email,
		GlobalRole: user.GlobalRole,
	}, nil
}

// CreateAPIKey generates a new API key for a user.
func (s *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, name string, permissions []string, expiresAt *time.Time) (*model.APIKey, string, error) {
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

// DiscordEnabled returns true if Discord OAuth is configured.
func (s *AuthService) DiscordEnabled() bool {
	return s.discordClientID != "" && s.discordSecret != "" && s.discordRedirectURI != ""
}

// DiscordOAuthURL generates the Discord authorization URL with a signed state parameter.
func (s *AuthService) DiscordOAuthURL() (string, error) {
	if !s.DiscordEnabled() {
		return "", fmt.Errorf("discord oauth is not configured")
	}

	state, err := s.generateOAuthState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}

	params := url.Values{
		"client_id":     {s.discordClientID},
		"redirect_uri":  {s.discordRedirectURI},
		"response_type": {"code"},
		"scope":         {"identify email"},
		"state":         {state},
	}

	return "https://discord.com/oauth2/authorize?" + params.Encode(), nil
}

// DiscordCallback exchanges the authorization code for tokens, fetches the Discord user
// profile, and either logs in or creates a new user account.
func (s *AuthService) DiscordCallback(ctx context.Context, code, state string) (string, *model.User, error) {
	if !s.DiscordEnabled() {
		return "", nil, fmt.Errorf("discord oauth is not configured")
	}

	if err := s.validateOAuthState(state); err != nil {
		return "", nil, fmt.Errorf("invalid state: %w", err)
	}

	accessToken, err := s.exchangeDiscordCode(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("exchanging code: %w", err)
	}

	discordUser, err := s.fetchDiscordUser(ctx, accessToken)
	if err != nil {
		return "", nil, fmt.Errorf("fetching discord user: %w", err)
	}

	user, err := s.findOrCreateOAuthUser(ctx, model.OAuthProviderDiscord, discordUser)
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

func (s *AuthService) exchangeDiscordCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {s.discordClientID},
		"client_secret": {s.discordSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.discordRedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.discordBaseURL+"/api/oauth2/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discord token error: status=%d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

func (s *AuthService) fetchDiscordUser(ctx context.Context, accessToken string) (*model.DiscordUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.discordBaseURL+"/api/v10/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user error: status=%d", resp.StatusCode)
	}

	var discordUser model.DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return nil, fmt.Errorf("decoding discord user: %w", err)
	}

	return &discordUser, nil
}

func (s *AuthService) findOrCreateOAuthUser(ctx context.Context, provider string, discord *model.DiscordUser) (*model.User, error) {
	// Case 1: OAuth account already linked — log in existing user.
	existing, err := s.oauthAccounts.GetByProviderUser(ctx, provider, discord.ID)
	if err == nil {
		user, err := s.users.GetByID(ctx, existing.UserID)
		if err != nil {
			return nil, fmt.Errorf("getting linked user: %w", err)
		}
		if !user.IsActive {
			return nil, model.ErrAccountDisabled
		}
		avatarURL := discord.AvatarURL()
		if err := s.users.UpdateAvatarURL(ctx, user.ID, avatarURL); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to update avatar")
		} else {
			user.AvatarURL = &avatarURL
		}
		return user, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, fmt.Errorf("looking up oauth account: %w", err)
	}

	// Case 2: User with same verified email exists — link and log in.
	var user *model.User
	if discord.Email != nil && *discord.Email != "" && discord.Verified != nil && *discord.Verified {
		user, err = s.users.GetByEmail(ctx, *discord.Email)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("looking up user by email: %w", err)
		}
	}

	// Case 3: Create new user.
	if user == nil {
		email := ""
		if discord.Email != nil {
			email = *discord.Email
		}
		if email == "" {
			email = "discord_" + discord.ID + "@oauth.taskwondo.local"
		}

		avatarURL := discord.AvatarURL()
		user = &model.User{
			ID:          uuid.New(),
			Email:       email,
			DisplayName: discord.DisplayName(),
			GlobalRole:  model.RoleUser,
			AvatarURL:   &avatarURL,
			IsActive:    true,
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("creating oauth user: %w", err)
		}
		log.Ctx(ctx).Info().
			Str("provider", provider).
			Str("provider_user_id", discord.ID).
			Str("email", email).
			Msg("created new user via oauth")
	} else {
		if !user.IsActive {
			return nil, model.ErrAccountDisabled
		}
		if user.AvatarURL == nil {
			avatarURL := discord.AvatarURL()
			if err := s.users.UpdateAvatarURL(ctx, user.ID, avatarURL); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to update avatar")
			} else {
				user.AvatarURL = &avatarURL
			}
		}
	}

	// Link the OAuth account.
	oauthAccount := &model.OAuthAccount{
		ID:               uuid.New(),
		UserID:           user.ID,
		Provider:         provider,
		ProviderUserID:   discord.ID,
		ProviderEmail:    ptrStr(discord.Email),
		ProviderUsername: discord.Username,
		ProviderAvatar:   ptrStr(discord.Avatar),
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

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
