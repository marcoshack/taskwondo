package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
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

// --- Test helpers ---

func newTestSystemSettingService() (*SystemSettingService, *mockSystemSettingRepo) {
	repo := newMockSystemSettingRepo()
	svc := NewSystemSettingService(repo)
	return svc, repo
}

func adminInfo() *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "admin@test.com",
		GlobalRole: model.RoleAdmin,
	}
}

func userInfo() *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}
}

// --- Tests ---

func TestSystemSettingSet(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	setting, err := svc.Set(ctx, adminInfo(), "brand_name", json.RawMessage(`"MyBrand"`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if setting.Key != "brand_name" {
		t.Fatalf("expected key brand_name, got %s", setting.Key)
	}
	if string(setting.Value) != `"MyBrand"` {
		t.Fatalf("expected value \"MyBrand\", got %s", string(setting.Value))
	}
}

func TestSystemSettingSet_NonAdmin(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	_, err := svc.Set(ctx, userInfo(), "brand_name", json.RawMessage(`"MyBrand"`))
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSystemSettingSet_NilAuth(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	_, err := svc.Set(ctx, nil, "brand_name", json.RawMessage(`"MyBrand"`))
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSystemSettingGet(t *testing.T) {
	svc, repo := newTestSystemSettingService()
	ctx := context.Background()

	repo.settings["brand_name"] = &model.SystemSetting{
		Key:       "brand_name",
		Value:     json.RawMessage(`"MyBrand"`),
		UpdatedAt: time.Now(),
	}

	setting, err := svc.Get(ctx, adminInfo(), "brand_name")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(setting.Value) != `"MyBrand"` {
		t.Fatalf("expected value \"MyBrand\", got %s", string(setting.Value))
	}
}

func TestSystemSettingGet_NotFound(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	_, err := svc.Get(ctx, adminInfo(), "nonexistent")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSystemSettingGet_NonAdmin(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	_, err := svc.Get(ctx, userInfo(), "brand_name")
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSystemSettingList(t *testing.T) {
	svc, repo := newTestSystemSettingService()
	ctx := context.Background()

	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}
	repo.settings["other"] = &model.SystemSetting{Key: "other", Value: json.RawMessage(`"val"`), UpdatedAt: time.Now()}

	settings, err := svc.List(ctx, adminInfo())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(settings) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(settings))
	}
}

func TestSystemSettingList_NonAdmin(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	_, err := svc.List(ctx, userInfo())
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSystemSettingDelete(t *testing.T) {
	svc, repo := newTestSystemSettingService()
	ctx := context.Background()

	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}

	err := svc.Delete(ctx, adminInfo(), "brand_name")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.settings) != 0 {
		t.Fatal("expected settings to be empty")
	}
}

func TestSystemSettingDelete_NotFound(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	err := svc.Delete(ctx, adminInfo(), "nonexistent")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSystemSettingDelete_NonAdmin(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	err := svc.Delete(ctx, userInfo(), "brand_name")
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSystemSettingGetPublic(t *testing.T) {
	svc, repo := newTestSystemSettingService()
	ctx := context.Background()

	repo.settings["brand_name"] = &model.SystemSetting{Key: "brand_name", Value: json.RawMessage(`"MyBrand"`), UpdatedAt: time.Now()}
	repo.settings["secret"] = &model.SystemSetting{Key: "secret", Value: json.RawMessage(`"hidden"`), UpdatedAt: time.Now()}

	result, err := svc.GetPublic(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 public setting, got %d", len(result))
	}

	if string(result["brand_name"]) != `"MyBrand"` {
		t.Fatalf("expected brand_name to be \"MyBrand\", got %s", string(result["brand_name"]))
	}

	if _, ok := result["secret"]; ok {
		t.Fatal("secret should not be in public settings")
	}
}

func TestSystemSettingGetPublic_Empty(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()

	result, err := svc.GetPublic(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 public settings, got %d", len(result))
	}
}

func TestSystemSettingUpsert(t *testing.T) {
	svc, _ := newTestSystemSettingService()
	ctx := context.Background()
	info := adminInfo()

	// Set initial value
	_, err := svc.Set(ctx, info, "brand_name", json.RawMessage(`"Brand1"`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Update value
	setting, err := svc.Set(ctx, info, "brand_name", json.RawMessage(`"Brand2"`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(setting.Value) != `"Brand2"` {
		t.Fatalf("expected value \"Brand2\", got %s", string(setting.Value))
	}
}
