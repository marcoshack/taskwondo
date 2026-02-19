package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/marcoshack/trackforge/internal/model"
)

// --- Mock repositories ---

type mockUserRepo struct {
	users map[string]*model.User // keyed by email
	byID  map[uuid.UUID]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*model.User),
		byID:  make(map[uuid.UUID]*model.User),
	}
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	m.users[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

func (m *mockUserRepo) UpdateLastLogin(_ context.Context, id uuid.UUID) error {
	if u, ok := m.byID[id]; ok {
		now := time.Now()
		u.LastLoginAt = &now
	}
	return nil
}

func (m *mockUserRepo) UpdateAvatarURL(_ context.Context, id uuid.UUID, avatarURL string) error {
	if u, ok := m.byID[id]; ok {
		u.AvatarURL = &avatarURL
	}
	return nil
}

func (m *mockUserRepo) Search(_ context.Context, query string) ([]model.User, error) {
	var result []model.User
	q := strings.ToLower(query)
	for _, u := range m.byID {
		if u.IsActive && (strings.Contains(strings.ToLower(u.Email), q) || strings.Contains(strings.ToLower(u.DisplayName), q)) {
			result = append(result, *u)
		}
	}
	return result, nil
}

type mockAPIKeyRepo struct {
	keys   map[string]*model.APIKey // keyed by key_hash
	byUser map[uuid.UUID][]model.APIKey
	byID   map[uuid.UUID]*model.APIKey
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{
		keys:   make(map[string]*model.APIKey),
		byUser: make(map[uuid.UUID][]model.APIKey),
		byID:   make(map[uuid.UUID]*model.APIKey),
	}
}

func (m *mockAPIKeyRepo) Create(_ context.Context, key *model.APIKey) error {
	m.keys[key.KeyHash] = key
	m.byUser[key.UserID] = append(m.byUser[key.UserID], *key)
	m.byID[key.ID] = key
	return nil
}

func (m *mockAPIKeyRepo) GetByKeyHash(_ context.Context, keyHash string) (*model.APIKey, error) {
	k, ok := m.keys[keyHash]
	if !ok {
		return nil, model.ErrNotFound
	}
	return k, nil
}

func (m *mockAPIKeyRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.APIKey, error) {
	return m.byUser[userID], nil
}

func (m *mockAPIKeyRepo) Delete(_ context.Context, id, userID uuid.UUID) error {
	k, ok := m.byID[id]
	if !ok || k.UserID != userID {
		return model.ErrNotFound
	}
	delete(m.keys, k.KeyHash)
	delete(m.byID, id)
	keys := m.byUser[userID]
	for i, key := range keys {
		if key.ID == id {
			m.byUser[userID] = append(keys[:i], keys[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockAPIKeyRepo) UpdateLastUsed(_ context.Context, id uuid.UUID) error {
	return nil
}

type mockOAuthAccountRepo struct {
	accounts map[string]*model.OAuthAccount // keyed by "provider:provider_user_id"
	byUser   map[uuid.UUID][]model.OAuthAccount
}

func newMockOAuthAccountRepo() *mockOAuthAccountRepo {
	return &mockOAuthAccountRepo{
		accounts: make(map[string]*model.OAuthAccount),
		byUser:   make(map[uuid.UUID][]model.OAuthAccount),
	}
}

func (m *mockOAuthAccountRepo) GetByProviderUser(_ context.Context, provider, providerUserID string) (*model.OAuthAccount, error) {
	key := provider + ":" + providerUserID
	a, ok := m.accounts[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return a, nil
}

func (m *mockOAuthAccountRepo) Create(_ context.Context, account *model.OAuthAccount) error {
	key := account.Provider + ":" + account.ProviderUserID
	m.accounts[key] = account
	m.byUser[account.UserID] = append(m.byUser[account.UserID], *account)
	return nil
}

func (m *mockOAuthAccountRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.OAuthAccount, error) {
	return m.byUser[userID], nil
}

func (m *mockOAuthAccountRepo) Delete(_ context.Context, id, userID uuid.UUID) error {
	for key, a := range m.accounts {
		if a.ID == id && a.UserID == userID {
			delete(m.accounts, key)
			accounts := m.byUser[userID]
			for i, acc := range accounts {
				if acc.ID == id {
					m.byUser[userID] = append(accounts[:i], accounts[i+1:]...)
					break
				}
			}
			return nil
		}
	}
	return model.ErrNotFound
}

// --- Tests ---

func newTestAuthService() (*AuthService, *mockUserRepo, *mockAPIKeyRepo) {
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour, "", "", "")
	return svc, userRepo, apiKeyRepo
}

func newTestAuthServiceWithOAuth() (*AuthService, *mockUserRepo, *mockOAuthAccountRepo) {
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour,
		"test-client-id", "test-client-secret", "http://localhost:5173/auth/discord/callback")
	return svc, userRepo, oauthRepo
}

func createTestUser(t *testing.T, repo *mockUserRepo, email, password, role string) *model.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		DisplayName:  "Test User",
		PasswordHash: string(hash),
		GlobalRole:   role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[email] = user
	repo.byID[user.ID] = user
	return user
}

func TestLogin_Success(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	token, user, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", user.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	_, _, err := svc.Login(context.Background(), "test@example.com", "wrongpassword")
	if err != model.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, _, _ := newTestAuthService()

	_, _, err := svc.Login(context.Background(), "nobody@example.com", "password")
	if err != model.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_DisabledAccount(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)
	user.IsActive = false

	_, _, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != model.ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestValidateJWT_Success(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleAdmin)

	token, _, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	info, err := svc.ValidateJWT(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, info.UserID)
	}
	if info.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", info.Email)
	}
	if info.GlobalRole != model.RoleAdmin {
		t.Fatalf("expected role admin, got %s", info.GlobalRole)
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	svc, _, _ := newTestAuthService()

	_, err := svc.ValidateJWT("invalid-token")
	if err != model.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	svc1, userRepo, _ := newTestAuthService()
	createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	token, _, _ := svc1.Login(context.Background(), "test@example.com", "password123")

	// Different service with different secret
	svc2 := NewAuthService(newMockUserRepo(), newMockAPIKeyRepo(), newMockOAuthAccountRepo(), "different-secret-at-least-32!!!", 24*time.Hour, "", "", "")

	_, err := svc2.ValidateJWT(token)
	if err != model.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestCreateAndValidateAPIKey(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	apiKey, fullKey, err := svc.CreateAPIKey(context.Background(), user.ID, "Test Key", []string{"read"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if apiKey.Name != "Test Key" {
		t.Fatalf("expected name 'Test Key', got %s", apiKey.Name)
	}
	if fullKey[:4] != "tfk_" {
		t.Fatalf("expected key to start with 'tfk_', got %s", fullKey[:4])
	}
	if apiKey.KeyPrefix != fullKey[:8] {
		t.Fatalf("expected key_prefix %s, got %s", fullKey[:8], apiKey.KeyPrefix)
	}

	// Validate the API key
	info, err := svc.ValidateAPIKey(context.Background(), fullKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, info.UserID)
	}
}

func TestValidateAPIKey_Expired(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	expired := time.Now().Add(-1 * time.Hour)
	_, fullKey, err := svc.CreateAPIKey(context.Background(), user.ID, "Expired Key", nil, &expired)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ValidateAPIKey(context.Background(), fullKey)
	if err != model.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	svc, _, _ := newTestAuthService()

	_, err := svc.ValidateAPIKey(context.Background(), "tfk_invalid_key_12345")
	if err != model.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestSeedAdminUser(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()

	err := svc.SeedAdminUser(context.Background(), "admin@example.com", "adminpass")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	user, err := userRepo.GetByEmail(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("expected admin user to exist, got %v", err)
	}
	if user.GlobalRole != model.RoleAdmin {
		t.Fatalf("expected admin role, got %s", user.GlobalRole)
	}

	// Calling again should be idempotent
	err = svc.SeedAdminUser(context.Background(), "admin@example.com", "adminpass")
	if err != nil {
		t.Fatalf("expected no error on second seed, got %v", err)
	}
}

func TestRefresh(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	info := &model.AuthInfo{
		UserID:     user.ID,
		Email:      user.Email,
		GlobalRole: user.GlobalRole,
	}

	token, err := svc.Refresh(context.Background(), info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Validate the refreshed token
	newInfo, err := svc.ValidateJWT(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newInfo.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, newInfo.UserID)
	}
}

func TestListAndDeleteAPIKeys(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	// Create two keys
	key1, _, _ := svc.CreateAPIKey(context.Background(), user.ID, "Key 1", nil, nil)
	svc.CreateAPIKey(context.Background(), user.ID, "Key 2", nil, nil)

	keys, err := svc.ListAPIKeys(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	// Delete one
	err = svc.DeleteAPIKey(context.Background(), key1.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	keys, _ = svc.ListAPIKeys(context.Background(), user.ID)
	if len(keys) != 1 {
		t.Fatalf("expected 1 key after delete, got %d", len(keys))
	}
}

// --- OAuth state tests ---

func TestOAuthState_GenerateAndValidate(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	state, err := svc.generateOAuthState()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	err = svc.validateOAuthState(state)
	if err != nil {
		t.Fatalf("expected valid state, got %v", err)
	}
}

func TestOAuthState_InvalidSignature(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	// Create a service with different secret
	svc2 := NewAuthService(newMockUserRepo(), newMockAPIKeyRepo(), newMockOAuthAccountRepo(),
		"different-secret-at-least-32!!!", 24*time.Hour, "", "", "")

	state, _ := svc2.generateOAuthState()

	err := svc.validateOAuthState(state)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestOAuthState_InvalidFormat(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	err := svc.validateOAuthState("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid state format")
	}
}

func TestDiscordOAuthURL(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	authURL, err := svc.DiscordOAuthURL()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(authURL, "discord.com/oauth2/authorize") {
		t.Fatalf("expected discord authorize URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "client_id=test-client-id") {
		t.Fatalf("expected client_id in URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "scope=identify+email") {
		t.Fatalf("expected scope in URL, got %s", authURL)
	}
}

func TestFindOrCreateOAuthUser_NewUser(t *testing.T) {
	svc, userRepo, oauthRepo := newTestAuthServiceWithOAuth()

	email := "discord@example.com"
	verified := true
	globalName := "DiscordUser"
	avatar := "abc123"
	discord := &model.DiscordUser{
		ID:         "123456789",
		Username:   "discorduser",
		GlobalName: &globalName,
		Avatar:     &avatar,
		Email:      &email,
		Verified:   &verified,
	}

	user, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, discord)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != email {
		t.Fatalf("expected email %s, got %s", email, user.Email)
	}
	if user.DisplayName != "DiscordUser" {
		t.Fatalf("expected display name 'DiscordUser', got %s", user.DisplayName)
	}
	if user.GlobalRole != model.RoleUser {
		t.Fatalf("expected role user, got %s", user.GlobalRole)
	}

	// Verify user was created in repo
	_, err = userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("expected user in repo, got %v", err)
	}

	// Verify OAuth account was linked
	accounts, _ := oauthRepo.ListByUserID(context.Background(), user.ID)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 oauth account, got %d", len(accounts))
	}
	if accounts[0].ProviderUserID != "123456789" {
		t.Fatalf("expected provider user id 123456789, got %s", accounts[0].ProviderUserID)
	}
}

func TestFindOrCreateOAuthUser_ExistingLink(t *testing.T) {
	svc, userRepo, oauthRepo := newTestAuthServiceWithOAuth()

	// Create existing user and link
	user := &model.User{
		ID:          uuid.New(),
		Email:       "existing@example.com",
		DisplayName: "Existing User",
		GlobalRole:  model.RoleUser,
		IsActive:    true,
	}
	userRepo.Create(context.Background(), user)

	oauthRepo.Create(context.Background(), &model.OAuthAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       model.OAuthProviderDiscord,
		ProviderUserID: "123456789",
	})

	discord := &model.DiscordUser{
		ID:       "123456789",
		Username: "discorduser",
	}

	found, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, discord)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != user.ID {
		t.Fatalf("expected existing user ID %s, got %s", user.ID, found.ID)
	}
}

func TestFindOrCreateOAuthUser_EmailMatch(t *testing.T) {
	svc, userRepo, oauthRepo := newTestAuthServiceWithOAuth()

	// Create existing user with matching email
	existingUser := createTestUser(t, userRepo, "shared@example.com", "password123", model.RoleUser)

	email := "shared@example.com"
	verified := true
	discord := &model.DiscordUser{
		ID:       "987654321",
		Username: "newdiscord",
		Email:    &email,
		Verified: &verified,
	}

	user, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, discord)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != existingUser.ID {
		t.Fatalf("expected existing user ID %s, got %s", existingUser.ID, user.ID)
	}

	// Verify OAuth account was linked to existing user
	accounts, _ := oauthRepo.ListByUserID(context.Background(), existingUser.ID)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 oauth account, got %d", len(accounts))
	}
}

func TestFindOrCreateOAuthUser_DisabledExistingLink(t *testing.T) {
	svc, userRepo, oauthRepo := newTestAuthServiceWithOAuth()

	user := &model.User{
		ID:          uuid.New(),
		Email:       "disabled@example.com",
		DisplayName: "Disabled User",
		GlobalRole:  model.RoleUser,
		IsActive:    false,
	}
	userRepo.Create(context.Background(), user)

	oauthRepo.Create(context.Background(), &model.OAuthAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       model.OAuthProviderDiscord,
		ProviderUserID: "111222333",
	})

	discord := &model.DiscordUser{
		ID:       "111222333",
		Username: "disableduser",
	}

	_, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, discord)
	if err != model.ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}
