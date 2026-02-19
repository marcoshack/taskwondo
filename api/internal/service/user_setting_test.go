package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock user setting repository ---

type mockUserSettingRepo struct {
	settings map[string]*model.UserSetting // key: "userID:projectID:key"
}

func newMockUserSettingRepo() *mockUserSettingRepo {
	return &mockUserSettingRepo{settings: make(map[string]*model.UserSetting)}
}

func settingKey(userID uuid.UUID, projectID *uuid.UUID, key string) string {
	pid := "global"
	if projectID != nil {
		pid = projectID.String()
	}
	return userID.String() + ":" + pid + ":" + key
}

func (m *mockUserSettingRepo) Upsert(_ context.Context, s *model.UserSetting) error {
	s.UpdatedAt = time.Now()
	m.settings[settingKey(s.UserID, s.ProjectID, s.Key)] = s
	return nil
}

func (m *mockUserSettingRepo) Get(_ context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) (*model.UserSetting, error) {
	s, ok := m.settings[settingKey(userID, projectID, key)]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *mockUserSettingRepo) ListByProject(_ context.Context, userID uuid.UUID, projectID uuid.UUID) ([]model.UserSetting, error) {
	var result []model.UserSetting
	for _, s := range m.settings {
		if s.UserID == userID && s.ProjectID != nil && *s.ProjectID == projectID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockUserSettingRepo) ListGlobal(_ context.Context, userID uuid.UUID) ([]model.UserSetting, error) {
	var result []model.UserSetting
	for _, s := range m.settings {
		if s.UserID == userID && s.ProjectID == nil {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockUserSettingRepo) Delete(_ context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) error {
	k := settingKey(userID, projectID, key)
	if _, ok := m.settings[k]; !ok {
		return model.ErrNotFound
	}
	delete(m.settings, k)
	return nil
}

// --- Helpers ---

func newTestUserSettingService() (*UserSettingService, *mockUserSettingRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	settingRepo := newMockUserSettingRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewUserSettingService(settingRepo, projectRepo, memberRepo)
	return svc, settingRepo, projectRepo, memberRepo
}

// --- Tests ---

func TestUserSetting_Set_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestUserSettingService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	val := json.RawMessage(`"dark"`)
	setting, err := svc.Set(context.Background(), info, "TEST", "theme", val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setting.Key != "theme" {
		t.Fatalf("expected key 'theme', got %s", setting.Key)
	}
	if string(setting.Value) != `"dark"` {
		t.Fatalf("expected value '\"dark\"', got %s", string(setting.Value))
	}
}

func TestUserSetting_Set_NotMember(t *testing.T) {
	svc, _, projectRepo, _ := newTestUserSettingService()
	info := userAuthInfo()
	// Create project but don't add user as member
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	val := json.RawMessage(`"dark"`)
	_, err := svc.Set(context.Background(), info, "TEST", "theme", val)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserSetting_Set_AdminBypass(t *testing.T) {
	svc, _, projectRepo, _ := newTestUserSettingService()
	info := adminAuthInfo()
	// Create project but don't add admin as member — admin should bypass
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	val := json.RawMessage(`true`)
	setting, err := svc.Set(context.Background(), info, "TEST", "notifications", val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setting.Key != "notifications" {
		t.Fatalf("expected key 'notifications', got %s", setting.Key)
	}
}

func TestUserSetting_Set_ProjectNotFound(t *testing.T) {
	svc, _, _, _ := newTestUserSettingService()
	info := userAuthInfo()

	val := json.RawMessage(`"value"`)
	_, err := svc.Set(context.Background(), info, "NOPE", "key", val)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserSetting_Get_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestUserSettingService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	// Set first, then get
	val := json.RawMessage(`42`)
	_, err := svc.Set(context.Background(), info, "TEST", "font_size", val)
	if err != nil {
		t.Fatal(err)
	}

	setting, err := svc.Get(context.Background(), info, "TEST", "font_size")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(setting.Value) != `42` {
		t.Fatalf("expected value '42', got %s", string(setting.Value))
	}
}

func TestUserSetting_Get_NotFound(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestUserSettingService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Get(context.Background(), info, "TEST", "nonexistent")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserSetting_List_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestUserSettingService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	// Set two settings
	svc.Set(context.Background(), info, "TEST", "theme", json.RawMessage(`"dark"`))
	svc.Set(context.Background(), info, "TEST", "font_size", json.RawMessage(`14`))

	settings, err := svc.List(context.Background(), info, "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settings) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(settings))
	}
}

func TestUserSetting_Delete_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestUserSettingService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	svc.Set(context.Background(), info, "TEST", "theme", json.RawMessage(`"dark"`))

	err := svc.Delete(context.Background(), info, "TEST", "theme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deleted
	_, err = svc.Get(context.Background(), info, "TEST", "theme")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestUserSetting_Delete_NotMember(t *testing.T) {
	svc, _, projectRepo, _ := newTestUserSettingService()
	info := userAuthInfo()
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	err := svc.Delete(context.Background(), info, "TEST", "theme")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserSetting_SetGlobal_Success(t *testing.T) {
	svc, _, _, _ := newTestUserSettingService()
	info := userAuthInfo()

	val := json.RawMessage(`"en"`)
	setting, err := svc.SetGlobal(context.Background(), info, "language", val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setting.Key != "language" {
		t.Fatalf("expected key 'language', got %s", setting.Key)
	}
	if setting.ProjectID != nil {
		t.Fatal("expected nil ProjectID for global setting")
	}
}

func TestUserSetting_GetGlobal_Success(t *testing.T) {
	svc, _, _, _ := newTestUserSettingService()
	info := userAuthInfo()

	svc.SetGlobal(context.Background(), info, "language", json.RawMessage(`"pt"`))

	setting, err := svc.GetGlobal(context.Background(), info, "language")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(setting.Value) != `"pt"` {
		t.Fatalf("expected value '\"pt\"', got %s", string(setting.Value))
	}
}

func TestUserSetting_GetGlobal_NotFound(t *testing.T) {
	svc, _, _, _ := newTestUserSettingService()
	info := userAuthInfo()

	_, err := svc.GetGlobal(context.Background(), info, "nonexistent")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserSetting_ListGlobal_Success(t *testing.T) {
	svc, _, _, _ := newTestUserSettingService()
	info := userAuthInfo()

	svc.SetGlobal(context.Background(), info, "language", json.RawMessage(`"en"`))
	svc.SetGlobal(context.Background(), info, "timezone", json.RawMessage(`"UTC"`))

	settings, err := svc.ListGlobal(context.Background(), info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settings) != 2 {
		t.Fatalf("expected 2 global settings, got %d", len(settings))
	}
}

func TestUserSetting_DeleteGlobal_Success(t *testing.T) {
	svc, _, _, _ := newTestUserSettingService()
	info := userAuthInfo()

	svc.SetGlobal(context.Background(), info, "language", json.RawMessage(`"en"`))

	err := svc.DeleteGlobal(context.Background(), info, "language")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetGlobal(context.Background(), info, "language")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
