package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// --- Mock repos for admin service ---

type mockAdminUserRepo struct {
	byID    map[uuid.UUID]*model.User
	byEmail map[string]*model.User
}

func newMockAdminUserRepo() *mockAdminUserRepo {
	return &mockAdminUserRepo{
		byID:    make(map[uuid.UUID]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (m *mockAdminUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockAdminUserRepo) ListAll(_ context.Context) ([]model.User, error) {
	var result []model.User
	for _, u := range m.byID {
		result = append(result, *u)
	}
	return result, nil
}

func (m *mockAdminUserRepo) UpdateGlobalRole(_ context.Context, id uuid.UUID, role string) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.GlobalRole = role
	return nil
}

func (m *mockAdminUserRepo) UpdateIsActive(_ context.Context, id uuid.UUID, isActive bool) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.IsActive = isActive
	return nil
}

func (m *mockAdminUserRepo) CountByRole(_ context.Context, role string) (int, error) {
	count := 0
	for _, u := range m.byID {
		if u.GlobalRole == role && u.IsActive {
			count++
		}
	}
	return count, nil
}

func (m *mockAdminUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockAdminUserRepo) Create(_ context.Context, user *model.User) error {
	m.byID[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockAdminUserRepo) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string, forceChange bool) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.PasswordHash = hash
	u.ForcePasswordChange = forceChange
	return nil
}

type mockAdminProjectRepo struct {
	byID map[uuid.UUID]*model.Project
}

func newMockAdminProjectRepo() *mockAdminProjectRepo {
	return &mockAdminProjectRepo{byID: make(map[uuid.UUID]*model.Project)}
}

func (m *mockAdminProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return p, nil
}

type mockAdminMemberRepo struct {
	members map[string]*model.ProjectMember
}

func newMockAdminMemberRepo() *mockAdminMemberRepo {
	return &mockAdminMemberRepo{members: make(map[string]*model.ProjectMember)}
}

func adminMemberKey(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockAdminMemberRepo) Add(_ context.Context, member *model.ProjectMember) error {
	key := adminMemberKey(member.ProjectID, member.UserID)
	if _, exists := m.members[key]; exists {
		return model.ErrAlreadyExists
	}
	member.CreatedAt = time.Now()
	m.members[key] = member
	return nil
}

func (m *mockAdminMemberRepo) GetByProjectAndUser(_ context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error) {
	key := adminMemberKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return member, nil
}

func (m *mockAdminMemberRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]model.ProjectMemberWithProject, error) {
	var result []model.ProjectMemberWithProject
	for _, member := range m.members {
		if member.UserID == userID {
			ownerCount := 0
			for _, m2 := range m.members {
				if m2.ProjectID == member.ProjectID && m2.Role == "owner" {
					ownerCount++
				}
			}
			result = append(result, model.ProjectMemberWithProject{
				ProjectMember: *member,
				ProjectName:   "Test Project",
				ProjectKey:    "TEST",
				OwnerCount:    ownerCount,
			})
		}
	}
	return result, nil
}

func (m *mockAdminMemberRepo) CountByRole(_ context.Context, projectID uuid.UUID, role string) (int, error) {
	count := 0
	for _, member := range m.members {
		if member.ProjectID == projectID && member.Role == role {
			count++
		}
	}
	return count, nil
}

func (m *mockAdminMemberRepo) UpdateRole(_ context.Context, projectID, userID uuid.UUID, role string) error {
	key := adminMemberKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return model.ErrNotFound
	}
	member.Role = role
	return nil
}

func (m *mockAdminMemberRepo) Remove(_ context.Context, projectID, userID uuid.UUID) error {
	key := adminMemberKey(projectID, userID)
	if _, ok := m.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.members, key)
	return nil
}

// --- Test setup ---

func adminTestSetup(t *testing.T) (*AdminHandler, *mockAdminUserRepo, *mockAdminProjectRepo, *mockAdminMemberRepo) {
	t.Helper()
	userRepo := newMockAdminUserRepo()
	projectRepo := newMockAdminProjectRepo()
	memberRepo := newMockAdminMemberRepo()
	adminSvc := service.NewAdminService(userRepo, projectRepo, memberRepo)
	h := NewAdminHandler(adminSvc)
	return h, userRepo, projectRepo, memberRepo
}

func addTestUser(repo *mockAdminUserRepo, role string, active bool) *model.User {
	u := &model.User{
		ID:          uuid.New(),
		Email:       uuid.New().String()[:8] + "@test.com",
		DisplayName: "Test User",
		GlobalRole:  role,
		IsActive:    active,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.byID[u.ID] = u
	repo.byEmail[u.Email] = u
	return u
}

func addTestProject(repo *mockAdminProjectRepo, key, name string) *model.Project {
	p := &model.Project{
		ID:        uuid.New(),
		Key:       key,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.byID[p.ID] = p
	return p
}

func adminCtx(userID uuid.UUID) context.Context {
	return model.ContextWithAuthInfo(context.Background(), &model.AuthInfo{
		UserID:     userID,
		Email:      "admin@test.com",
		GlobalRole: model.RoleAdmin,
	})
}

func withChiParam(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

func withChiParams(ctx context.Context, params map[string]string) context.Context {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

// --- Tests ---

func TestAdminListUsersHandler(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	addTestUser(userRepo, model.RoleUser, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req = req.WithContext(adminCtx(admin.ID))
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(data))
	}
}

func TestAdminUpdateUserHandler_ChangeRole(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	target := addTestUser(userRepo, model.RoleUser, true)

	body := `{"global_role":"admin"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+target.ID.String(), bytes.NewBufferString(body))
	ctx := adminCtx(admin.ID)
	ctx = withChiParam(ctx, "userId", target.ID.String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["global_role"] != "admin" {
		t.Fatalf("expected role admin, got %v", data["global_role"])
	}
}

func TestAdminUpdateUserHandler_InvalidUserId(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)

	body := `{"global_role":"admin"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/invalid", bytes.NewBufferString(body))
	ctx := adminCtx(admin.ID)
	ctx = withChiParam(ctx, "userId", "invalid")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminUpdateUserHandler_NoFields(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	target := addTestUser(userRepo, model.RoleUser, true)

	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+target.ID.String(), bytes.NewBufferString(body))
	ctx := adminCtx(admin.ID)
	ctx = withChiParam(ctx, "userId", target.ID.String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminListUserProjectsHandler(t *testing.T) {
	h, userRepo, projectRepo, memberRepo := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	user := addTestUser(userRepo, model.RoleUser, true)
	proj := addTestProject(projectRepo, "TEST", "Test Project")

	memberRepo.members[adminMemberKey(proj.ID, user.ID)] = &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: proj.ID,
		UserID:    user.ID,
		Role:      model.ProjectRoleMember,
		CreatedAt: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+user.ID.String()+"/projects", nil)
	ctx := adminCtx(admin.ID)
	ctx = withChiParam(ctx, "userId", user.ID.String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListUserProjects(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 project, got %d", len(data))
	}
}

func TestAdminAddUserToProjectHandler(t *testing.T) {
	h, userRepo, projectRepo, _ := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	user := addTestUser(userRepo, model.RoleUser, true)
	proj := addTestProject(projectRepo, "TEST", "Test Project")

	body, _ := json.Marshal(map[string]string{
		"project_id": proj.ID.String(),
		"role":       "member",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/projects", bytes.NewBuffer(body))
	ctx := adminCtx(admin.ID)
	ctx = withChiParam(ctx, "userId", user.ID.String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.AddUserToProject(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminAddUserToProjectHandler_Duplicate(t *testing.T) {
	h, userRepo, projectRepo, memberRepo := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	user := addTestUser(userRepo, model.RoleUser, true)
	proj := addTestProject(projectRepo, "TEST", "Test Project")

	memberRepo.members[adminMemberKey(proj.ID, user.ID)] = &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: proj.ID,
		UserID:    user.ID,
		Role:      model.ProjectRoleMember,
		CreatedAt: time.Now(),
	}

	body, _ := json.Marshal(map[string]string{
		"project_id": proj.ID.String(),
		"role":       "member",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/projects", bytes.NewBuffer(body))
	ctx := adminCtx(admin.ID)
	ctx = withChiParam(ctx, "userId", user.ID.String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.AddUserToProject(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminRemoveUserFromProjectHandler(t *testing.T) {
	h, userRepo, projectRepo, memberRepo := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	user := addTestUser(userRepo, model.RoleUser, true)
	proj := addTestProject(projectRepo, "TEST", "Test Project")

	memberRepo.members[adminMemberKey(proj.ID, user.ID)] = &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: proj.ID,
		UserID:    user.ID,
		Role:      model.ProjectRoleMember,
		CreatedAt: time.Now(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+user.ID.String()+"/projects/"+proj.ID.String(), nil)
	ctx := adminCtx(admin.ID)
	ctx = withChiParams(ctx, map[string]string{
		"userId":    user.ID.String(),
		"projectId": proj.ID.String(),
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RemoveUserFromProject(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminRemoveUserFromProjectHandler_NotFound(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)
	admin := addTestUser(userRepo, model.RoleAdmin, true)
	user := addTestUser(userRepo, model.RoleUser, true)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+user.ID.String()+"/projects/"+uuid.New().String(), nil)
	ctx := adminCtx(admin.ID)
	ctx = withChiParams(ctx, map[string]string{
		"userId":    user.ID.String(),
		"projectId": uuid.New().String(),
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.RemoveUserFromProject(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- CreateUser handler tests ---

func TestCreateUser_Handler_201(t *testing.T) {
	h, _, _, _ := adminTestSetup(t)

	body := `{"email":"new@test.com","display_name":"New User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	caller := addTestUser(newMockAdminUserRepo(), model.RoleAdmin, true)
	req = req.WithContext(adminCtx(caller.ID))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["temporary_password"] == nil || data["temporary_password"] == "" {
		t.Fatal("expected temporary_password in response")
	}
	user := data["user"].(map[string]interface{})
	if user["email"] != "new@test.com" {
		t.Fatalf("expected email new@test.com, got %v", user["email"])
	}
	if user["force_password_change"] != true {
		t.Fatal("expected force_password_change to be true")
	}
}

func TestCreateUser_Handler_409_EmailExists(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)

	existing := addTestUser(userRepo, model.RoleUser, true)

	body := `{"email":"` + existing.Email + `","display_name":"Duplicate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	caller := addTestUser(userRepo, model.RoleAdmin, true)
	req = req.WithContext(adminCtx(caller.ID))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateUser_Handler_400_MissingFields(t *testing.T) {
	h, _, _, _ := adminTestSetup(t)

	body := `{"email":"","display_name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(adminCtx(uuid.New()))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ResetUserPassword handler tests ---

func TestResetUserPassword_Handler_200(t *testing.T) {
	h, userRepo, _, _ := adminTestSetup(t)
	user := addTestUser(userRepo, model.RoleUser, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/reset-password", nil)
	ctx := adminCtx(uuid.New())
	ctx = withChiParam(ctx, "userId", user.ID.String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ResetUserPassword(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["temporary_password"] == nil || data["temporary_password"] == "" {
		t.Fatal("expected temporary_password in response")
	}
}

func TestResetUserPassword_Handler_404(t *testing.T) {
	h, _, _, _ := adminTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+uuid.New().String()+"/reset-password", nil)
	ctx := adminCtx(uuid.New())
	ctx = withChiParam(ctx, "userId", uuid.New().String())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ResetUserPassword(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
