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

	h := NewAuthHandler(authSvc)
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
