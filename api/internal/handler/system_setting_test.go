package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/email"
	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// --- Mock repository ---

type mockSystemSettingRepo struct {
	settings map[string]*model.SystemSetting
}

func newMockSystemSettingRepo() *mockSystemSettingRepo {
	return &mockSystemSettingRepo{settings: make(map[string]*model.SystemSetting)}
}

func (m *mockSystemSettingRepo) Upsert(_ context.Context, s *model.SystemSetting) error {
	m.settings[s.Key] = &model.SystemSetting{
		Key:       s.Key,
		Value:     s.Value,
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *mockSystemSettingRepo) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	s, ok := m.settings[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *mockSystemSettingRepo) List(_ context.Context) ([]model.SystemSetting, error) {
	var result []model.SystemSetting
	for _, s := range m.settings {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockSystemSettingRepo) Delete(_ context.Context, key string) error {
	if _, ok := m.settings[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.settings, key)
	return nil
}

// --- Test setup ---

func testEncryptor(t *testing.T) *crypto.Encryptor {
	t.Helper()
	key, err := crypto.DeriveKey("test-secret-key-that-is-long-enough-32chars")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func systemSettingTestSetup(t *testing.T) (*SystemSettingHandler, *mockSystemSettingRepo) {
	t.Helper()
	repo := newMockSystemSettingRepo()
	svc := service.NewSystemSettingService(repo)
	enc := testEncryptor(t)
	sender := email.NewSender(enc, repo)
	h := NewSystemSettingHandler(svc, enc, sender)
	return h, repo
}

func sysAdminCtx() context.Context {
	return model.ContextWithAuthInfo(context.Background(), &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "admin@test.com",
		GlobalRole: model.RoleAdmin,
	})
}

func sysUserCtx() context.Context {
	return model.ContextWithAuthInfo(context.Background(), &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	})
}

// --- Tests ---

func TestSystemSettingListHandler(t *testing.T) {
	h, repo := systemSettingTestSetup(t)
	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(data))
	}
}

func TestSystemSettingListHandler_Forbidden(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	req = req.WithContext(sysUserCtx())
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemSettingGetHandler(t *testing.T) {
	h, repo := systemSettingTestSetup(t)
	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/brand_name", nil)
	ctx := sysAdminCtx()
	ctx = withChiParam(ctx, "key", "brand_name")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["key"] != "brand_name" {
		t.Fatalf("expected key brand_name, got %v", data["key"])
	}
}

func TestSystemSettingGetHandler_NotFound(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/nonexistent", nil)
	ctx := sysAdminCtx()
	ctx = withChiParam(ctx, "key", "nonexistent")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemSettingSetHandler(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	body := `{"value":"MyBrand"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/brand_name", bytes.NewBufferString(body))
	ctx := sysAdminCtx()
	ctx = withChiParam(ctx, "key", "brand_name")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Set(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["key"] != "brand_name" {
		t.Fatalf("expected key brand_name, got %v", data["key"])
	}
}

func TestSystemSettingSetHandler_Forbidden(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	body := `{"value":"MyBrand"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/brand_name", bytes.NewBufferString(body))
	ctx := sysUserCtx()
	ctx = withChiParam(ctx, "key", "brand_name")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Set(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemSettingSetHandler_EmptyValue(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/brand_name", bytes.NewBufferString(body))
	ctx := sysAdminCtx()
	ctx = withChiParam(ctx, "key", "brand_name")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Set(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemSettingDeleteHandler(t *testing.T) {
	h, repo := systemSettingTestSetup(t)
	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/settings/brand_name", nil)
	ctx := sysAdminCtx()
	ctx = withChiParam(ctx, "key", "brand_name")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemSettingDeleteHandler_NotFound(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/settings/nonexistent", nil)
	ctx := sysAdminCtx()
	ctx = withChiParam(ctx, "key", "nonexistent")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemSettingGetPublicHandler(t *testing.T) {
	h, repo := systemSettingTestSetup(t)
	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil)
	w := httptest.NewRecorder()

	h.GetPublic(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["brand_name"] != "MyBrand" {
		t.Fatalf("expected brand_name MyBrand, got %v", data["brand_name"])
	}
}

func TestSystemSettingGetPublicHandler_NoAuth(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	// No auth context - should still work
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil)
	w := httptest.NewRecorder()

	h.GetPublic(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- SMTP handler tests ---

func TestGetSMTP_DefaultWhenNotConfigured(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/smtp_config", nil)
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.GetSMTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.SMTPConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.SMTPPort != 587 {
		t.Errorf("expected default smtp_port 587, got %d", resp.Data.SMTPPort)
	}
	if resp.Data.IMAPPort != 993 {
		t.Errorf("expected default imap_port 993, got %d", resp.Data.IMAPPort)
	}
	if resp.Data.Encryption != model.SMTPEncryptionSTARTTLS {
		t.Errorf("expected default encryption starttls, got %s", resp.Data.Encryption)
	}
}

func TestSetSMTP_SaveAndMaskPassword(t *testing.T) {
	h, repo := systemSettingTestSetup(t)

	cfg := model.SMTPConfig{
		Enabled:     true,
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		Username:    "user@example.com",
		Password:    "secret123",
		Encryption:  "starttls",
		FromAddress: "noreply@example.com",
		FromName:    "Test",
	}
	body, _ := json.Marshal(cfg)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/smtp_config", bytes.NewBuffer(body))
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.SetSMTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Response should have masked password
	var resp struct {
		Data model.SMTPConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Password != model.PasswordMask {
		t.Errorf("expected masked password in response, got %q", resp.Data.Password)
	}

	// Stored value should have encrypted password (not plaintext)
	stored := repo.settings[model.SettingSMTPConfig]
	if stored == nil {
		t.Fatal("smtp_config not stored in repo")
	}
	var storedCfg model.SMTPConfig
	json.Unmarshal(stored.Value, &storedCfg)
	if storedCfg.Password == "secret123" {
		t.Error("password should be encrypted in storage, not plaintext")
	}
	if storedCfg.Password == "" {
		t.Error("encrypted password should not be empty")
	}
}

func TestSetSMTP_PreservesExistingPassword(t *testing.T) {
	h, repo := systemSettingTestSetup(t)

	// First save with real password
	enc := testEncryptor(t)
	encrypted, _ := enc.Encrypt("original-pass")
	existing := model.SMTPConfig{
		Enabled:     true,
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		Username:    "user@example.com",
		Password:    encrypted,
		Encryption:  "starttls",
		FromAddress: "noreply@example.com",
	}
	raw, _ := json.Marshal(existing)
	repo.settings[model.SettingSMTPConfig] = &model.SystemSetting{Key: model.SettingSMTPConfig, Value: raw, UpdatedAt: time.Now()}

	// Update with masked password (should preserve original)
	cfg := model.SMTPConfig{
		Enabled:     true,
		SMTPHost:    "smtp.new.com",
		SMTPPort:    587,
		Username:    "user@example.com",
		Password:    model.PasswordMask,
		Encryption:  "starttls",
		FromAddress: "noreply@example.com",
	}
	body, _ := json.Marshal(cfg)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/smtp_config", bytes.NewBuffer(body))
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.SetSMTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify password was preserved
	stored := repo.settings[model.SettingSMTPConfig]
	var storedCfg model.SMTPConfig
	json.Unmarshal(stored.Value, &storedCfg)
	if storedCfg.Password != encrypted {
		t.Error("password should be preserved when masked value is sent")
	}
	// Verify host was updated
	if storedCfg.SMTPHost != "smtp.new.com" {
		t.Errorf("expected updated host smtp.new.com, got %s", storedCfg.SMTPHost)
	}
}

func TestSetSMTP_ValidationError(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	// Enabled but missing required fields
	cfg := model.SMTPConfig{
		Enabled: true,
		// Missing smtp_host, username, from_address, etc.
		SMTPPort:   587,
		Encryption: "starttls",
	}
	body, _ := json.Marshal(cfg)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/smtp_config", bytes.NewBuffer(body))
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.SetSMTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetSMTP_DisabledSkipsValidation(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	// Disabled - should save even with empty fields
	cfg := model.SMTPConfig{Enabled: false}
	body, _ := json.Marshal(cfg)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/smtp_config", bytes.NewBuffer(body))
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.SetSMTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSMTP_MasksPassword(t *testing.T) {
	h, repo := systemSettingTestSetup(t)

	enc := testEncryptor(t)
	encrypted, _ := enc.Encrypt("secret-pass")
	cfg := model.SMTPConfig{
		Enabled:     true,
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		Username:    "user",
		Password:    encrypted,
		Encryption:  "starttls",
		FromAddress: "test@example.com",
	}
	raw, _ := json.Marshal(cfg)
	repo.settings[model.SettingSMTPConfig] = &model.SystemSetting{Key: model.SettingSMTPConfig, Value: raw, UpdatedAt: time.Now()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/smtp_config", nil)
	req = req.WithContext(sysAdminCtx())
	w := httptest.NewRecorder()

	h.GetSMTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.SMTPConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Password != model.PasswordMask {
		t.Errorf("expected masked password, got %q", resp.Data.Password)
	}
	if resp.Data.SMTPHost != "smtp.example.com" {
		t.Errorf("expected smtp host, got %q", resp.Data.SMTPHost)
	}
}

func TestTestSMTP_Forbidden(t *testing.T) {
	h, _ := systemSettingTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/smtp_config/test", nil)
	req = req.WithContext(sysUserCtx())
	w := httptest.NewRecorder()

	h.TestSMTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
