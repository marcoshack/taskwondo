package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/marcoshack/trackforge/internal/model"
)

// UserRepository defines the persistence operations the auth service needs for users.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// APIKeyRepository defines the persistence operations the auth service needs for API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// Claims are the JWT token claims.
type Claims struct {
	jwt.RegisteredClaims
	Email      string `json:"email"`
	GlobalRole string `json:"role"`
}

// AuthService handles authentication and authorization.
type AuthService struct {
	users     UserRepository
	apiKeys   APIKeyRepository
	jwtSecret []byte
	jwtExpiry time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(users UserRepository, apiKeys APIKeyRepository, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		users:     users,
		apiKeys:   apiKeys,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
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
		UserID:     userID,
		Email:      claims.Email,
		GlobalRole: claims.GlobalRole,
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

	fullKey := "tfk_" + hex.EncodeToString(raw)
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

func (s *AuthService) generateJWT(user *model.User) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
			Issuer:    "trackforge",
		},
		Email:      user.Email,
		GlobalRole: user.GlobalRole,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// HashAPIKey computes the SHA-256 hash of an API key for storage/lookup.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
