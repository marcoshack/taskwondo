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

func systemSettingTestSetup(t *testing.T) (*SystemSettingHandler, *mockSystemSettingRepo) {
	t.Helper()
	repo := newMockSystemSettingRepo()
	svc := service.NewSystemSettingService(repo)
	h := NewSystemSettingHandler(svc)
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
