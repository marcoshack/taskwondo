package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// --- Mock repos implementing service interfaces ---

type mockUserRepo struct {
	users map[string]*model.User
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

func (m *mockUserRepo) UpdateDisplayName(_ context.Context, id uuid.UUID, displayName string) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.DisplayName = displayName
	return nil
}

func (m *mockUserRepo) UpdateAvatarURL(_ context.Context, id uuid.UUID, avatarURL string) error {
	if u, ok := m.byID[id]; ok {
		if avatarURL == "" {
			u.AvatarURL = nil
		} else {
			u.AvatarURL = &avatarURL
		}
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

func (m *mockUserRepo) Search(_ context.Context, _ uuid.UUID, query string) ([]model.User, error) {
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
	keys   map[string]*model.APIKey
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
	if key.UserID != nil {
		m.byUser[*key.UserID] = append(m.byUser[*key.UserID], *key)
	}
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

func (m *mockAPIKeyRepo) ListByType(_ context.Context, keyType string) ([]model.APIKey, error) {
	var result []model.APIKey
	for _, k := range m.byID {
		if k.Type == keyType {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *mockAPIKeyRepo) Delete(_ context.Context, id, userID uuid.UUID) error {
	k, ok := m.byID[id]
	if !ok || k.UserID == nil || *k.UserID != userID {
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

func (m *mockAPIKeyRepo) DeleteByID(_ context.Context, id uuid.UUID) error {
	k, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	delete(m.keys, k.KeyHash)
	delete(m.byID, id)
	return nil
}

func (m *mockAPIKeyRepo) UpdateName(_ context.Context, id, userID uuid.UUID, name string) error {
	k, ok := m.byID[id]
	if !ok || k.UserID == nil || *k.UserID != userID {
		return model.ErrNotFound
	}
	k.Name = name
	return nil
}

func (m *mockAPIKeyRepo) UpdateNameByID(_ context.Context, id uuid.UUID, name string) error {
	k, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	k.Name = name
	return nil
}

func (m *mockAPIKeyRepo) UpdateLastUsed(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockOAuthAccountRepo struct {
	accounts map[string]*model.OAuthAccount
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
			return nil
		}
	}
	return model.ErrNotFound
}

// --- Test setup ---

func testSetup(t *testing.T) (*AuthHandler, *service.AuthService, string) {
	t.Helper()

	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	discord := service.NewDiscordProvider("test-client-id", "test-client-secret", "http://localhost:5173/auth/discord/callback", nil)
	authSvc := service.NewAuthService(userRepo, apiKeyRepo, oauthRepo,
		"test-secret-that-is-at-least-32!", 1*time.Hour,
		[]service.OAuthProvider{discord})

	if err := authSvc.SeedAdminUser(context.Background(), "admin@test.com", "adminpass"); err != nil {
		t.Fatal(err)
	}

	token, _, err := authSvc.Login(context.Background(), "admin@test.com", "adminpass")
	if err != nil {
		t.Fatal(err)
	}

	h := NewAuthHandler(authSvc, nil)
	return h, authSvc, token
}

// --- Tests ---

func TestLoginHandler_Success(t *testing.T) {
	h, _, _ := testSetup(t)

	body := `{"email":"admin@test.com","password":"adminpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data in response")
	}
	if data["token"] == nil || data["token"] == "" {
		t.Fatal("expected token in response")
	}
	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user in response")
	}
	if user["email"] != "admin@test.com" {
		t.Fatalf("expected email admin@test.com, got %v", user["email"])
	}
	if user["global_role"] != "admin" {
		t.Fatalf("expected role admin, got %v", user["global_role"])
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	h, _, _ := testSetup(t)

	body := `{"email":"admin@test.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	h, _, _ := testSetup(t)

	body := `{"email":"admin@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMeHandler(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["email"] != "admin@test.com" {
		t.Fatalf("expected email admin@test.com, got %v", data["email"])
	}
}

func TestRefreshHandler(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["token"] == nil || data["token"] == "" {
		t.Fatal("expected new token in response")
	}
}

func TestLogoutHandler(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestCreateAPIKey_Success(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	body := `{"name":"CI Key","permissions":["read"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/api-keys", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["key"] == nil || data["key"] == "" {
		t.Fatal("expected full key in create response")
	}
	if data["name"] != "CI Key" {
		t.Fatalf("expected name 'CI Key', got %v", data["name"])
	}
}

func TestListAPIKeys(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)

	// Create a key first
	authSvc.CreateAPIKey(context.Background(), info.UserID, "Test Key", nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/api-keys", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ListAPIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	keys := resp["data"].([]interface{})
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}

func TestDeleteAPIKey(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)

	apiKey, _, _ := authSvc.CreateAPIKey(context.Background(), info.UserID, "To Delete", nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/api-keys/"+apiKey.ID.String(), nil)
	// Set chi route context with URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", apiKey.ID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.DeleteAPIKey(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAPIKey_MissingName(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	body := `{"permissions":["read"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/api-keys", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMeHandler_Unauthenticated(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	h.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- OAuth handler tests (generic) ---

func TestOAuthAuthHandler_ReturnsURL(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/discord", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "discord")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.OAuthAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	url, ok := data["url"].(string)
	if !ok || url == "" {
		t.Fatal("expected url in response")
	}
	if !strings.Contains(url, "discord.com/oauth2/authorize") {
		t.Fatalf("expected discord authorize URL, got %s", url)
	}
}

func TestOAuthAuthHandler_NotConfigured(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "google")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.OAuthAuth(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthCallbackHandler_MissingFields(t *testing.T) {
	h, _, _ := testSetup(t)

	body := `{"code":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/callback", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "discord")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.OAuthCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthCallbackHandler_InvalidBody(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/callback", bytes.NewBufferString("not json"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "discord")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.OAuthCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ChangePassword handler tests ---

func TestChangePassword_Handler_200(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"old_password":"adminpass","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["token"] == nil || data["token"] == "" {
		t.Fatal("expected new token in response")
	}
}

func TestChangePassword_Handler_401_WrongPassword(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"old_password":"wrongpassword","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ChangePassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_Handler_ReturnsForcePasswordChange(t *testing.T) {
	h, _, _ := testSetup(t)

	body := `{"email":"admin@test.com","password":"adminpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["force_password_change"] != false {
		t.Fatalf("expected force_password_change=false for seeded admin, got %v", data["force_password_change"])
	}
}

func TestRenameAPIKey_Success(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	apiKey, _, _ := authSvc.CreateAPIKey(context.Background(), info.UserID, "Old Name", nil, nil)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/api-keys/"+apiKey.ID.String(), bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", apiKey.ID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameAPIKey(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameAPIKey_EmptyName(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	apiKey, _, _ := authSvc.CreateAPIKey(context.Background(), info.UserID, "Some Key", nil, nil)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/api-keys/"+apiKey.ID.String(), bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", apiKey.ID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameAPIKey_NotFound(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)

	body := `{"name":"New Name"}`
	fakeID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/api-keys/"+fakeID.String(), bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", fakeID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameAPIKey(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameAPIKey_InvalidKeyID(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/api-keys/not-a-uuid", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", "not-a-uuid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAPIKey_InvalidPermission(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	body := `{"name":"Bad Key","permissions":["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/api-keys", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid permission, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAPIKey_ExpiredDate(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	body := `{"name":"Expired Key","expires_at":"` + past + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/api-keys", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for past expiration, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAPIKey_WithExpiration(t *testing.T) {
	h, authSvc, token := testSetup(t)

	info, _ := authSvc.ValidateJWT(token)
	future := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	body := `{"name":"Future Key","expires_at":"` + future + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/api-keys", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Email registration handler tests ---

type handlerMockEmailVerifRepo struct {
	tokens map[string]*model.EmailVerificationToken
}

func newHandlerMockEmailVerifRepo() *handlerMockEmailVerifRepo {
	return &handlerMockEmailVerifRepo{tokens: make(map[string]*model.EmailVerificationToken)}
}

func (m *handlerMockEmailVerifRepo) Create(_ context.Context, token *model.EmailVerificationToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *handlerMockEmailVerifRepo) GetByTokenHash(_ context.Context, tokenHash string) (*model.EmailVerificationToken, error) {
	t, ok := m.tokens[tokenHash]
	if !ok || t.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrNotFound
	}
	return t, nil
}

func (m *handlerMockEmailVerifRepo) DeleteByTokenHash(_ context.Context, tokenHash string) error {
	delete(m.tokens, tokenHash)
	return nil
}

func (m *handlerMockEmailVerifRepo) DeleteByEmail(_ context.Context, email string) error {
	for k, t := range m.tokens {
		if t.Email == email {
			delete(m.tokens, k)
		}
	}
	return nil
}

func (m *handlerMockEmailVerifRepo) DeleteExpired(_ context.Context) (int64, error) {
	var count int64
	for k, t := range m.tokens {
		if t.ExpiresAt.Before(time.Now()) {
			delete(m.tokens, k)
			count++
		}
	}
	return count, nil
}

type handlerMockSettingsReader struct {
	settings map[string]*model.SystemSetting
}

func newHandlerMockSettingsReader() *handlerMockSettingsReader {
	return &handlerMockSettingsReader{settings: make(map[string]*model.SystemSetting)}
}

func (m *handlerMockSettingsReader) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	s, ok := m.settings[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *handlerMockSettingsReader) setBool(key string, val bool) {
	raw, _ := json.Marshal(val)
	m.settings[key] = &model.SystemSetting{Key: key, Value: raw}
}

type handlerMockEmailSender struct {
	sent []string
}

func (m *handlerMockEmailSender) Send(_ context.Context, to, subject, body string) error {
	m.sent = append(m.sent, to)
	return nil
}

func testSetupWithEmail(t *testing.T) (*AuthHandler, *service.AuthService, *handlerMockSettingsReader) {
	t.Helper()

	userRepo := newMockUserRepo()
	apiKeyRepo := newMockAPIKeyRepo()
	oauthRepo := newMockOAuthAccountRepo()
	authSvc := service.NewAuthService(userRepo, apiKeyRepo, oauthRepo,
		"test-secret-that-is-at-least-32!", 1*time.Hour, nil)

	verifRepo := newHandlerMockEmailVerifRepo()
	settings := newHandlerMockSettingsReader()
	sender := &handlerMockEmailSender{}
	authSvc.SetEmailVerification(verifRepo, settings, sender, "http://localhost:5173")

	h := NewAuthHandler(authSvc, nil)
	return h, authSvc, settings
}

func TestRegisterHandler_Success(t *testing.T) {
	h, _, settings := testSetupWithEmail(t)
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	body := `{"email":"new@example.com","display_name":"New User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_Disabled(t *testing.T) {
	h, _, _ := testSetupWithEmail(t)
	// Registration disabled by default

	body := `{"email":"new@example.com","display_name":"New User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	h, _, settings := testSetupWithEmail(t)
	settings.setBool(model.SettingAuthEmailRegistrationEnabled, true)

	body := `{"email":"new@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVerifyEmailHandler_MissingFields(t *testing.T) {
	h, _, _ := testSetupWithEmail(t)

	body := `{"token":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.VerifyEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVerifyEmailHandler_InvalidToken(t *testing.T) {
	h, _, _ := testSetupWithEmail(t)

	body := `{"token":"nonexistent","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.VerifyEmail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Profile handler tests ---

func TestUpdateProfileHandler_Success(t *testing.T) {
	h, authSvc, token := testSetup(t)
	_ = authSvc

	body := `{"display_name":"Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	info, _ := authSvc.ValidateJWT(token)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["display_name"] != "Updated Name" {
		t.Fatalf("expected 'Updated Name', got '%v'", data["display_name"])
	}
}

func TestUpdateProfileHandler_EmptyName(t *testing.T) {
	h, authSvc, token := testSetup(t)

	body := `{"display_name":""}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	info, _ := authSvc.ValidateJWT(token)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAvatarHandler_Success(t *testing.T) {
	h, authSvc, token := testSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/avatar", nil)
	info, _ := authSvc.ValidateJWT(token)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.DeleteAvatar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetUserAvatarHandler_NotFound(t *testing.T) {
	h, _, _ := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+uuid.New().String()+"/avatar", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetUserAvatar(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- System API Key handler tests ---

func TestAuthHandler_CreateSystemAPIKey(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"name":"CI Pipeline","permissions":["metrics:r"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateSystemAPIKey(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["key"] == nil || data["key"] == "" {
		t.Fatal("expected full key in create response")
	}
	key := data["key"].(string)
	if !strings.HasPrefix(key, "twks_") {
		t.Fatalf("expected key to start with 'twks_', got %s", key[:5])
	}
	if data["name"] != "CI Pipeline" {
		t.Fatalf("expected name 'CI Pipeline', got %v", data["name"])
	}
	if data["created_by"] == nil {
		t.Fatal("expected created_by in response")
	}
}

func TestAuthHandler_CreateSystemAPIKey_MissingName(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"permissions":["metrics:r"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateSystemAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_CreateSystemAPIKey_InvalidPermission(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"name":"Bad Key","permissions":["badresource:x"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateSystemAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid permission, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_CreateSystemAPIKey_EmptyPermissions(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"name":"No Perms","permissions":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateSystemAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty permissions, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_ListSystemAPIKeys(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	// Create two system keys
	authSvc.CreateSystemAPIKey(context.Background(), info.UserID, "Key A", []string{"metrics:r"}, nil)
	authSvc.CreateSystemAPIKey(context.Background(), info.UserID, "Key B", []string{"items:rw"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-keys", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ListSystemAPIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	keys := resp["data"].([]interface{})
	if len(keys) != 2 {
		t.Fatalf("expected 2 system keys, got %d", len(keys))
	}
}

func TestAuthHandler_ListSystemAPIKeys_Empty(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-keys", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ListSystemAPIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	keys := resp["data"].([]interface{})
	if len(keys) != 0 {
		t.Fatalf("expected 0 system keys, got %d", len(keys))
	}
}

func TestAuthHandler_RenameSystemAPIKey(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	apiKey, _, _ := authSvc.CreateSystemAPIKey(context.Background(), info.UserID, "Old Name", []string{"metrics:r"}, nil)

	body := `{"name":"New System Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/"+apiKey.ID.String(), bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", apiKey.ID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameSystemAPIKey(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RenameSystemAPIKey_EmptyName(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	apiKey, _, _ := authSvc.CreateSystemAPIKey(context.Background(), info.UserID, "Some Key", []string{"metrics:r"}, nil)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/"+apiKey.ID.String(), bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", apiKey.ID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameSystemAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RenameSystemAPIKey_NotFound(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	fakeID := uuid.New()
	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/"+fakeID.String(), bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", fakeID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameSystemAPIKey(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RenameSystemAPIKey_InvalidKeyID(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/api-keys/not-a-uuid", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", "not-a-uuid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RenameSystemAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_DeleteSystemAPIKey(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	apiKey, _, _ := authSvc.CreateSystemAPIKey(context.Background(), info.UserID, "To Delete", []string{"metrics:r"}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/"+apiKey.ID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", apiKey.ID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.DeleteSystemAPIKey(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_DeleteSystemAPIKey_NotFound(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	fakeID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/"+fakeID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", fakeID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.DeleteSystemAPIKey(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_DeleteSystemAPIKey_InvalidKeyID(t *testing.T) {
	h, authSvc, token := testSetup(t)
	info, _ := authSvc.ValidateJWT(token)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/api-keys/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyId", "not-a-uuid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.DeleteSystemAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

