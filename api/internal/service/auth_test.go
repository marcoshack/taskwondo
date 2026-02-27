package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/marcoshack/taskwondo/internal/model"
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

func (m *mockUserRepo) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string, forceChange bool) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.PasswordHash = hash
	u.ForcePasswordChange = forceChange
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
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour, nil)
	return svc, userRepo, apiKeyRepo
}

func newTestAuthServiceWithOAuth() (*AuthService, *mockUserRepo, *mockOAuthAccountRepo) {
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	discord := NewDiscordProvider("test-client-id", "test-client-secret", "http://localhost:5173/auth/discord/callback", nil)
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour,
		[]OAuthProvider{discord})
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
	svc2 := NewAuthService(newMockUserRepo(), newMockAPIKeyRepo(), newMockOAuthAccountRepo(), "different-secret-at-least-32!!!", 24*time.Hour, nil)

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
	if fullKey[:4] != "twk_" {
		t.Fatalf("expected key to start with 'twk_', got %s", fullKey[:4])
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

	_, err := svc.ValidateAPIKey(context.Background(), "twk_invalid_key_12345")
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
		"different-secret-at-least-32!!!", 24*time.Hour, nil)

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

func TestOAuthURL_Discord(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	authURL, err := svc.OAuthURL(context.Background(), "discord")
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

func TestOAuthURL_NotConfigured(t *testing.T) {
	svc, _, _ := newTestAuthService()
	_, err := svc.OAuthURL(context.Background(), "discord")
	if err == nil {
		t.Fatal("expected error when discord is not configured")
	}
}

func TestEnabledProviders(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTestAuthService()
	providers := svc.EnabledProviders(ctx)
	// No OAuth providers, but email_login defaults to true, email_registration defaults to false
	if providers["discord"] {
		t.Fatal("expected discord to not be enabled")
	}
	if !providers["email_login"] {
		t.Fatal("expected email_login to default to true")
	}
	if providers["email_registration"] {
		t.Fatal("expected email_registration to default to false")
	}

	svc2, _, _ := newTestAuthServiceWithOAuth()
	providers2 := svc2.EnabledProviders(ctx)
	if !providers2["discord"] {
		t.Fatal("expected discord to be enabled")
	}
}

func TestFindOrCreateOAuthUser_NewUser(t *testing.T) {
	svc, userRepo, oauthRepo := newTestAuthServiceWithOAuth()

	info := model.OAuthUserInfo{
		ProviderUserID: "123456789",
		Email:          "discord@example.com",
		EmailVerified:  true,
		DisplayName:    "DiscordUser",
		AvatarURL:      "https://cdn.discordapp.com/avatars/123456789/abc123.png",
		Username:       "discorduser",
		RawAvatar:      "abc123",
	}

	user, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "discord@example.com" {
		t.Fatalf("expected email discord@example.com, got %s", user.Email)
	}
	if user.DisplayName != "DiscordUser" {
		t.Fatalf("expected display name 'DiscordUser', got %s", user.DisplayName)
	}
	if user.GlobalRole != model.RoleUser {
		t.Fatalf("expected role user, got %s", user.GlobalRole)
	}

	// Verify user was created in repo
	_, err = userRepo.GetByEmail(context.Background(), "discord@example.com")
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

	info := model.OAuthUserInfo{
		ProviderUserID: "123456789",
		Username:       "discorduser",
		AvatarURL:      "https://cdn.discordapp.com/avatars/123456789/default.png",
	}

	found, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, info)
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

	info := model.OAuthUserInfo{
		ProviderUserID: "987654321",
		Email:          "shared@example.com",
		EmailVerified:  true,
		Username:       "newdiscord",
	}

	user, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, info)
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

	info := model.OAuthUserInfo{
		ProviderUserID: "111222333",
		Username:       "disableduser",
	}

	_, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, info)
	if err != model.ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestGetUser_Success(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "user@example.com", "password123", model.RoleUser)

	result, err := svc.GetUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", result.Email)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	svc, _, _ := newTestAuthService()

	_, err := svc.GetUser(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestSearchUsers_Success(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	createTestUser(t, userRepo, "alice@example.com", "pass", model.RoleUser)
	createTestUser(t, userRepo, "bob@example.com", "pass", model.RoleUser)

	results, err := svc.SearchUsers(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", results[0].Email)
	}
}

func TestSearchUsers_TooShort(t *testing.T) {
	svc, _, _ := newTestAuthService()

	results, err := svc.SearchUsers(context.Background(), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil for short query, got %v", results)
	}
}

func TestSearchUsers_NoMatch(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	createTestUser(t, userRepo, "alice@example.com", "pass", model.RoleUser)

	results, err := svc.SearchUsers(context.Background(), "zzzz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestRefresh_DisabledAccount(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "disabled@example.com", "pass", model.RoleUser)
	user.IsActive = false

	info := &model.AuthInfo{UserID: user.ID, Email: user.Email, GlobalRole: model.RoleUser}
	_, err := svc.Refresh(context.Background(), info)
	if err != model.ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestRefresh_UserNotFound(t *testing.T) {
	svc, _, _ := newTestAuthService()

	info := &model.AuthInfo{UserID: uuid.New(), Email: "gone@example.com", GlobalRole: model.RoleUser}
	_, err := svc.Refresh(context.Background(), info)
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestOAuthCallback_NotConfigured(t *testing.T) {
	svc, _, _ := newTestAuthService() // no providers
	_, _, err := svc.OAuthCallback(context.Background(), "discord", "code", "state")
	if err == nil {
		t.Fatal("expected error when discord is not configured")
	}
}

func TestOAuthCallback_InvalidState(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()
	_, _, err := svc.OAuthCallback(context.Background(), "discord", "code", "invalid-state")
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestOAuthCallback_DiscordFullSuccess(t *testing.T) {
	svc, _, oauthRepo := newTestAuthServiceWithOAuth()

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/token":
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
			})
		case "/api/v10/users/@me":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "999888777",
				"username": "testuser",
				"email":    "discord@example.com",
				"verified": true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer discordServer.Close()

	// Override the Discord provider's HTTP client and base URL for testing
	dp := svc.providers["discord"].(*DiscordProvider)
	dp.httpClient = discordServer.Client()
	dp.baseURL = discordServer.URL

	state, err := svc.generateOAuthState()
	if err != nil {
		t.Fatal(err)
	}

	token, user, err := svc.OAuthCallback(context.Background(), "discord", "valid-code", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if user.Email != "discord@example.com" {
		t.Fatalf("expected email discord@example.com, got %s", user.Email)
	}

	accounts, _ := oauthRepo.ListByUserID(context.Background(), user.ID)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 oauth account, got %d", len(accounts))
	}
}

func TestOAuthCallback_TokenExchangeFails(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer discordServer.Close()

	dp := svc.providers["discord"].(*DiscordProvider)
	dp.httpClient = discordServer.Client()
	dp.baseURL = discordServer.URL

	state, _ := svc.generateOAuthState()
	_, _, err := svc.OAuthCallback(context.Background(), "discord", "bad-code", state)
	if err == nil {
		t.Fatal("expected error for failed token exchange")
	}
}

func TestOAuthCallback_UserFetchFails(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/token":
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
			})
		case "/api/v10/users/@me":
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer discordServer.Close()

	dp := svc.providers["discord"].(*DiscordProvider)
	dp.httpClient = discordServer.Client()
	dp.baseURL = discordServer.URL

	state, _ := svc.generateOAuthState()
	_, _, err := svc.OAuthCallback(context.Background(), "discord", "valid-code", state)
	if err == nil {
		t.Fatal("expected error for failed user fetch")
	}
}

func TestFindOrCreateOAuthUser_NoEmail(t *testing.T) {
	svc, userRepo, _ := newTestAuthServiceWithOAuth()

	info := model.OAuthUserInfo{
		ProviderUserID: "555666777",
		Username:       "noemailuser",
		DisplayName:    "noemailuser",
	}

	user, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedEmail := "discord_555666777@oauth.taskwondo.local"
	if user.Email != expectedEmail {
		t.Fatalf("expected email %s, got %s", expectedEmail, user.Email)
	}

	_, err = userRepo.GetByEmail(context.Background(), expectedEmail)
	if err != nil {
		t.Fatal("expected user to be created in repo")
	}
}

func TestFindOrCreateOAuthUser_DisabledEmailMatch(t *testing.T) {
	svc, userRepo, _ := newTestAuthServiceWithOAuth()

	user := &model.User{
		ID:          uuid.New(),
		Email:       "disabled@example.com",
		DisplayName: "Disabled",
		GlobalRole:  model.RoleUser,
		IsActive:    false,
	}
	userRepo.Create(context.Background(), user)

	info := model.OAuthUserInfo{
		ProviderUserID: "444555666",
		Email:          "disabled@example.com",
		EmailVerified:  true,
		Username:       "disabledmatch",
	}

	_, err := svc.findOrCreateOAuthUser(context.Background(), model.OAuthProviderDiscord, info)
	if err != model.ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}

// --- ChangePassword tests ---

func TestChangePassword_Success(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "oldpassword", model.RoleUser)
	user.ForcePasswordChange = true

	err := svc.ChangePassword(context.Background(), user.ID, "oldpassword", "newpassword123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), user.ID)
	if updated.ForcePasswordChange {
		t.Fatal("expected ForcePasswordChange to be false after change")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("newpassword123")); err != nil {
		t.Fatal("expected new password to be valid")
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "correctpassword", model.RoleUser)

	err := svc.ChangePassword(context.Background(), user.ID, "wrongpassword", "newpassword123")
	if err != model.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestChangePassword_TooShort(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "oldpassword", model.RoleUser)

	err := svc.ChangePassword(context.Background(), user.ID, "oldpassword", "short")
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestLogin_ReturnsForcePasswordChange(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)
	user.ForcePasswordChange = true

	token, returnedUser, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !returnedUser.ForcePasswordChange {
		t.Fatal("expected ForcePasswordChange to be true in returned user")
	}

	info, err := svc.ValidateJWT(token)
	if err != nil {
		t.Fatalf("expected valid JWT, got %v", err)
	}
	if !info.ForcePasswordChange {
		t.Fatal("expected ForcePasswordChange in JWT claims")
	}
}

func TestLogin_NoPasswordHash(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := &model.User{
		ID:          uuid.New(),
		Email:       "oauth-only@example.com",
		DisplayName: "OAuth User",
		GlobalRole:  model.RoleUser,
		IsActive:    true,
	}
	userRepo.Create(context.Background(), user)

	_, _, err := svc.Login(context.Background(), "oauth-only@example.com", "anypassword")
	if err != model.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateAPIKey_DisabledUser(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "disabled@example.com", "pass", model.RoleUser)
	_, fullKey, _ := svc.CreateAPIKey(context.Background(), user.ID, "Key", nil, nil)
	user.IsActive = false

	_, err := svc.ValidateAPIKey(context.Background(), fullKey)
	if err != model.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

// --- Google OAuth tests ---

func TestOAuthURL_Google(t *testing.T) {
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	google := NewGoogleProvider("google-client-id", "google-secret", "http://localhost:3000/auth/google/callback", nil)
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour,
		[]OAuthProvider{google})

	authURL, err := svc.OAuthURL(context.Background(), "google")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(authURL, "accounts.google.com/o/oauth2/v2/auth") {
		t.Fatalf("expected google authorize URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "client_id=google-client-id") {
		t.Fatalf("expected client_id in URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "scope=openid+email+profile") {
		t.Fatalf("expected scope in URL, got %s", authURL)
	}
}

func TestOAuthCallback_GoogleFullSuccess(t *testing.T) {
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()

	googleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-google-token",
				"token_type":   "Bearer",
			})
		case "/oauth2/v2/userinfo":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             "google-user-123",
				"email":          "user@gmail.com",
				"verified_email": true,
				"name":           "Google User",
				"picture":        "https://lh3.googleusercontent.com/photo.jpg",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer googleServer.Close()

	google := NewGoogleProvider("google-client-id", "google-secret", "http://localhost:3000/auth/google/callback", googleServer.Client())
	google.tokenURL = googleServer.URL
	google.apiBaseURL = googleServer.URL

	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour,
		[]OAuthProvider{google})

	state, err := svc.generateOAuthState()
	if err != nil {
		t.Fatal(err)
	}

	token, user, err := svc.OAuthCallback(context.Background(), "google", "valid-code", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if user.Email != "user@gmail.com" {
		t.Fatalf("expected email user@gmail.com, got %s", user.Email)
	}
	if user.DisplayName != "Google User" {
		t.Fatalf("expected display name 'Google User', got %s", user.DisplayName)
	}

	accounts, _ := oauthRepo.ListByUserID(context.Background(), user.ID)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 oauth account, got %d", len(accounts))
	}
	if accounts[0].Provider != model.OAuthProviderGoogle {
		t.Fatalf("expected provider google, got %s", accounts[0].Provider)
	}
}

func TestEnabledProviders_Multiple(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	discord := NewDiscordProvider("dc-id", "dc-secret", "http://localhost/discord", nil)
	google := NewGoogleProvider("g-id", "g-secret", "http://localhost/google", nil)
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour,
		[]OAuthProvider{discord, google})

	providers := svc.EnabledProviders(ctx)
	if !providers["discord"] {
		t.Fatal("expected discord to be enabled")
	}
	if !providers["google"] {
		t.Fatal("expected google to be enabled")
	}
	// 2 OAuth + email_login + email_registration = 4
	if len(providers) != 4 {
		t.Fatalf("expected 4 providers, got %d: %v", len(providers), providers)
	}
}

func TestCreateAPIKey_InvalidPermission(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	_, _, err := svc.CreateAPIKey(context.Background(), user.ID, "Bad Key", []string{"admin"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid permission")
	}
	if !strings.Contains(err.Error(), "invalid permission") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestCreateAPIKey_ValidPermissions(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	apiKey, _, err := svc.CreateAPIKey(context.Background(), user.ID, "RW Key", []string{"read", "write"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(apiKey.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(apiKey.Permissions))
	}
}

func TestValidateAPIKey_PermissionsPassedThrough(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	_, fullKey, err := svc.CreateAPIKey(context.Background(), user.ID, "Read Key", []string{"read"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	info, err := svc.ValidateAPIKey(context.Background(), fullKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(info.Permissions) != 1 || info.Permissions[0] != "read" {
		t.Fatalf("expected permissions [read], got %v", info.Permissions)
	}
}

func TestValidateAPIKey_EmptyPermissions(t *testing.T) {
	svc, userRepo, _ := newTestAuthService()
	user := createTestUser(t, userRepo, "test@example.com", "password123", model.RoleUser)

	_, fullKey, err := svc.CreateAPIKey(context.Background(), user.ID, "Full Key", []string{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	info, err := svc.ValidateAPIKey(context.Background(), fullKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(info.Permissions) != 0 {
		t.Fatalf("expected empty permissions, got %v", info.Permissions)
	}
}

// --- Email verification mocks ---

type mockEmailVerificationRepo struct {
	tokens map[string]*model.EmailVerificationToken // keyed by token_hash
}

func newMockEmailVerificationRepo() *mockEmailVerificationRepo {
	return &mockEmailVerificationRepo{tokens: make(map[string]*model.EmailVerificationToken)}
}

func (m *mockEmailVerificationRepo) Create(_ context.Context, token *model.EmailVerificationToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockEmailVerificationRepo) GetByTokenHash(_ context.Context, tokenHash string) (*model.EmailVerificationToken, error) {
	t, ok := m.tokens[tokenHash]
	if !ok {
		return nil, model.ErrNotFound
	}
	if t.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrNotFound
	}
	return t, nil
}

func (m *mockEmailVerificationRepo) DeleteByTokenHash(_ context.Context, tokenHash string) error {
	delete(m.tokens, tokenHash)
	return nil
}

func (m *mockEmailVerificationRepo) DeleteByEmail(_ context.Context, email string) error {
	for k, t := range m.tokens {
		if t.Email == email {
			delete(m.tokens, k)
		}
	}
	return nil
}

type mockSettingsReader struct {
	settings map[string]*model.SystemSetting
}

func newMockSettingsReader() *mockSettingsReader {
	return &mockSettingsReader{settings: make(map[string]*model.SystemSetting)}
}

func (m *mockSettingsReader) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	s, ok := m.settings[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *mockSettingsReader) setBool(key string, val bool) {
	raw, _ := json.Marshal(val)
	m.settings[key] = &model.SystemSetting{Key: key, Value: raw}
}

type mockEmailSender struct {
	sent []struct{ to, subject, body string }
}

func (m *mockEmailSender) Send(_ context.Context, to, subject, body string) error {
	m.sent = append(m.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

func newTestAuthServiceWithEmail() (*AuthService, *mockUserRepo, *mockEmailVerificationRepo, *mockSettingsReader, *mockEmailSender) {
	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	svc := NewAuthService(userRepo, apiKeyRepo, oauthRepo, "test-secret-at-least-32-chars!!", 24*time.Hour, nil)

	emailVerifRepo := newMockEmailVerificationRepo()
	settings := newMockSettingsReader()
	sender := &mockEmailSender{}
	svc.SetEmailVerification(emailVerifRepo, settings, sender, "http://localhost:5173")

	return svc, userRepo, emailVerifRepo, settings, sender
}

func TestRequestRegistration_Success(t *testing.T) {
	svc, _, _, settings, sender := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	err := svc.RequestRegistration(context.Background(), "new@example.com", "New User", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(sender.sent))
	}
	if sender.sent[0].to != "new@example.com" {
		t.Fatalf("expected email to new@example.com, got %s", sender.sent[0].to)
	}
	if !strings.Contains(sender.sent[0].body, "verify-email?token=") {
		t.Fatal("expected verification URL in email body")
	}
}

func TestRequestRegistration_Disabled(t *testing.T) {
	svc, _, _, _, _ := newTestAuthServiceWithEmail()
	// Registration is disabled by default

	err := svc.RequestRegistration(context.Background(), "new@example.com", "New User", "")
	if err == nil {
		t.Fatal("expected error when registration is disabled")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected 'disabled' in error, got: %v", err)
	}
}

func TestRequestRegistration_DuplicateEmail(t *testing.T) {
	svc, userRepo, _, settings, _ := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	// Create existing user
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	existing := &model.User{
		ID:           uuid.New(),
		Email:        "existing@example.com",
		DisplayName:  "Existing",
		PasswordHash: string(hash),
		GlobalRole:   model.RoleUser,
		IsActive:     true,
	}
	userRepo.users[existing.Email] = existing
	userRepo.byID[existing.ID] = existing

	err := svc.RequestRegistration(context.Background(), "existing@example.com", "New User", "")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestRequestRegistration_InvalidEmail(t *testing.T) {
	svc, _, _, settings, _ := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	err := svc.RequestRegistration(context.Background(), "invalid", "New User", "")
	if err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestVerifyEmailAndCreateUser_Success(t *testing.T) {
	svc, userRepo, _, settings, _ := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	// First request registration to create a token
	sender := svc.emailSender.(*mockEmailSender)
	err := svc.RequestRegistration(context.Background(), "verify@example.com", "Verify User", "")
	if err != nil {
		t.Fatalf("request registration failed: %v", err)
	}

	// Extract token from email body
	body := sender.sent[0].body
	idx := strings.Index(body, "verify-email?token=")
	if idx == -1 {
		t.Fatal("token not found in email")
	}
	tokenStart := idx + len("verify-email?token=")
	// Find end of token (next quote or end of string)
	tokenEnd := strings.IndexAny(body[tokenStart:], "\"' ")
	rawToken := body[tokenStart : tokenStart+tokenEnd]

	// Verify
	result, err := svc.VerifyEmailAndCreateUser(context.Background(), rawToken, "SecurePass123!")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected JWT token")
	}
	if result.User.Email != "verify@example.com" {
		t.Fatalf("expected email verify@example.com, got %s", result.User.Email)
	}
	if result.User.DisplayName != "Verify User" {
		t.Fatalf("expected display name 'Verify User', got %s", result.User.DisplayName)
	}

	// User should exist in repo
	if _, err := userRepo.GetByEmail(context.Background(), "verify@example.com"); err != nil {
		t.Fatalf("expected user in repo, got: %v", err)
	}
}

func TestRequestRegistration_WithInviteCode(t *testing.T) {
	svc, _, emailVerifRepo, settings, _ := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	err := svc.RequestRegistration(context.Background(), "invite@example.com", "Invite User", "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the invite code was stored in the token
	for _, tok := range emailVerifRepo.tokens {
		if tok.Email == "invite@example.com" {
			if tok.InviteCode != "abc123" {
				t.Fatalf("expected invite code 'abc123', got '%s'", tok.InviteCode)
			}
			return
		}
	}
	t.Fatal("token not found in repo")
}

func TestVerifyEmailAndCreateUser_WithInviteCode(t *testing.T) {
	svc, _, _, settings, _ := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	sender := svc.emailSender.(*mockEmailSender)
	err := svc.RequestRegistration(context.Background(), "inviteverify@example.com", "Invite Verify", "invcode42")
	if err != nil {
		t.Fatalf("request registration failed: %v", err)
	}

	// Extract token from email
	body := sender.sent[0].body
	idx := strings.Index(body, "verify-email?token=")
	tokenStart := idx + len("verify-email?token=")
	tokenEnd := strings.IndexAny(body[tokenStart:], "\"' ")
	rawToken := body[tokenStart : tokenStart+tokenEnd]

	result, err := svc.VerifyEmailAndCreateUser(context.Background(), rawToken, "SecurePass123!")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.InviteCode != "invcode42" {
		t.Fatalf("expected invite code 'invcode42', got '%s'", result.InviteCode)
	}
}

func TestVerifyEmailAndCreateUser_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthServiceWithEmail()

	_, err := svc.VerifyEmailAndCreateUser(context.Background(), "nonexistent-token", "password123")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestVerifyEmailAndCreateUser_WeakPassword(t *testing.T) {
	svc, _, emailVerifRepo, settings, _ := newTestAuthServiceWithEmail()
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	// Manually create a token
	tokenHash := hashToken("test-token")
	emailVerifRepo.tokens[tokenHash] = &model.EmailVerificationToken{
		ID:          uuid.New(),
		Email:       "weak@example.com",
		DisplayName: "Weak Pass",
		TokenHash:   tokenHash,
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	_, err := svc.VerifyEmailAndCreateUser(context.Background(), "test-token", "short")
	if err == nil {
		t.Fatal("expected error for weak password")
	}
	if !strings.Contains(err.Error(), "8 characters") {
		t.Fatalf("expected password length error, got: %v", err)
	}
}

func TestEnabledProviders_WithSettings(t *testing.T) {
	ctx := context.Background()
	svc, _, _, settings, _ := newTestAuthServiceWithEmail()

	// Default: email_login true, email_registration false
	providers := svc.EnabledProviders(ctx)
	if !providers["email_login"] {
		t.Fatal("email_login should default to true")
	}
	if providers["email_registration"] {
		t.Fatal("email_registration should default to false")
	}

	// Enable registration
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)
	providers = svc.EnabledProviders(ctx)
	if !providers["email_registration"] {
		t.Fatal("email_registration should be enabled")
	}

	// Disable email login
	settings.setBool(model.SettingAuthEmailLoginEnabled, false)
	providers = svc.EnabledProviders(ctx)
	if providers["email_login"] {
		t.Fatal("email_login should be disabled")
	}
}
