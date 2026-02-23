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

// --- Mock repositories for project service ---

type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
	byKey    map[string]*model.Project
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[uuid.UUID]*model.Project),
		byKey:    make(map[string]*model.Project),
	}
}

func (m *mockProjectRepo) Create(_ context.Context, project *model.Project) error {
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	m.projects[project.ID] = project
	m.byKey[project.Key] = project
	return nil
}

func (m *mockProjectRepo) GetByKey(_ context.Context, key string) (*model.Project, error) {
	p, ok := m.byKey[key]
	if !ok || p.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok || p.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]model.Project, error) {
	var result []model.Project
	for _, p := range m.projects {
		if p.DeletedAt == nil {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockProjectRepo) ListAll(_ context.Context) ([]model.Project, error) {
	var result []model.Project
	for _, p := range m.projects {
		if p.DeletedAt == nil {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockProjectRepo) Update(_ context.Context, project *model.Project) error {
	existing, ok := m.projects[project.ID]
	if !ok || existing.DeletedAt != nil {
		return model.ErrNotFound
	}
	if existing.Key != project.Key {
		delete(m.byKey, existing.Key)
	}
	now := time.Now()
	project.UpdatedAt = now
	m.projects[project.ID] = project
	m.byKey[project.Key] = project
	return nil
}

func (m *mockProjectRepo) Delete(_ context.Context, id uuid.UUID) error {
	p, ok := m.projects[id]
	if !ok || p.DeletedAt != nil {
		return model.ErrNotFound
	}
	now := time.Now()
	p.DeletedAt = &now
	return nil
}

func (m *mockProjectRepo) GetSummaries(_ context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectSummary, error) {
	result := make(map[uuid.UUID]model.ProjectSummary, len(projectIDs))
	for _, id := range projectIDs {
		result[id] = model.ProjectSummary{}
	}
	return result, nil
}

func (m *mockProjectRepo) CountByOwner(_ context.Context, _ uuid.UUID) (int, error) {
	count := 0
	for _, p := range m.projects {
		if p.DeletedAt == nil {
			count++
		}
	}
	return count, nil
}

type mockProjectMemberRepo struct {
	members map[string]*model.ProjectMember
}

func newMockProjectMemberRepo() *mockProjectMemberRepo {
	return &mockProjectMemberRepo{
		members: make(map[string]*model.ProjectMember),
	}
}

func pmKey(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockProjectMemberRepo) Add(_ context.Context, member *model.ProjectMember) error {
	key := pmKey(member.ProjectID, member.UserID)
	if _, exists := m.members[key]; exists {
		return model.ErrAlreadyExists
	}
	member.CreatedAt = time.Now()
	m.members[key] = member
	return nil
}

func (m *mockProjectMemberRepo) GetByProjectAndUser(_ context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error) {
	key := pmKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return member, nil
}

func (m *mockProjectMemberRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectMemberWithUser, error) {
	var result []model.ProjectMemberWithUser
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, model.ProjectMemberWithUser{
				ProjectMember: *member,
				Email:         "user@example.com",
				DisplayName:   "Test User",
			})
		}
	}
	return result, nil
}

func (m *mockProjectMemberRepo) UpdateRole(_ context.Context, projectID, userID uuid.UUID, role string) error {
	key := pmKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return model.ErrNotFound
	}
	member.Role = role
	return nil
}

func (m *mockProjectMemberRepo) Remove(_ context.Context, projectID, userID uuid.UUID) error {
	key := pmKey(projectID, userID)
	if _, ok := m.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.members, key)
	return nil
}

func (m *mockProjectMemberRepo) CountByRole(_ context.Context, projectID uuid.UUID, role string) (int, error) {
	count := 0
	for _, member := range m.members {
		if member.ProjectID == projectID && member.Role == role {
			count++
		}
	}
	return count, nil
}

type mockProjectInviteRepo struct {
	invites map[uuid.UUID]*model.ProjectInvite
	byCode  map[string]*model.ProjectInvite
}

func newMockProjectInviteRepo() *mockProjectInviteRepo {
	return &mockProjectInviteRepo{
		invites: make(map[uuid.UUID]*model.ProjectInvite),
		byCode:  make(map[string]*model.ProjectInvite),
	}
}

func (m *mockProjectInviteRepo) Create(_ context.Context, invite *model.ProjectInvite) error {
	invite.CreatedAt = time.Now()
	m.invites[invite.ID] = invite
	m.byCode[invite.Code] = invite
	return nil
}

func (m *mockProjectInviteRepo) GetByCode(_ context.Context, code string) (*model.ProjectInvite, error) {
	inv, ok := m.byCode[code]
	if !ok {
		return nil, model.ErrNotFound
	}
	return inv, nil
}

func (m *mockProjectInviteRepo) GetByID(_ context.Context, id uuid.UUID) (*model.ProjectInvite, error) {
	inv, ok := m.invites[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return inv, nil
}

func (m *mockProjectInviteRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectInvite, error) {
	var result []model.ProjectInvite
	for _, inv := range m.invites {
		if inv.ProjectID == projectID {
			result = append(result, *inv)
		}
	}
	return result, nil
}

func (m *mockProjectInviteRepo) IncrementUseCount(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invites[id]
	if !ok {
		return model.ErrNotFound
	}
	if inv.MaxUses > 0 && inv.UseCount >= inv.MaxUses {
		return model.ErrNotFound
	}
	inv.UseCount++
	return nil
}

func (m *mockProjectInviteRepo) Delete(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invites[id]
	if !ok {
		return model.ErrNotFound
	}
	delete(m.byCode, inv.Code)
	delete(m.invites, id)
	return nil
}

func (m *mockProjectInviteRepo) DeleteByProject(_ context.Context, projectID uuid.UUID) error {
	for id, inv := range m.invites {
		if inv.ProjectID == projectID {
			delete(m.byCode, inv.Code)
			delete(m.invites, id)
		}
	}
	return nil
}

// --- Test setup ---

func projectTestSetup(t *testing.T) (*ProjectHandler, *model.AuthInfo) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	userRepo := newMockUserRepo()
	workflowRepo := newMockWorkflowRepo()
	typeWorkflowRepo := newMockTypeWorkflowRepo()
	inviteRepo := newMockProjectInviteRepo()
	projectSvc := service.NewProjectService(projectRepo, memberRepo, userRepo, workflowRepo, typeWorkflowRepo, newMockSystemSettingRepo(), inviteRepo)
	h := NewProjectHandler(projectSvc, "http://localhost:3000")

	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "owner@test.com",
		GlobalRole: model.RoleUser,
	}

	// Register user in repo so AddMember can look them up
	userRepo.Create(context.Background(), &model.User{
		ID:          info.UserID,
		Email:       info.Email,
		DisplayName: "Owner",
		IsActive:    true,
	})

	return h, info
}

func createTestProject(t *testing.T, h *ProjectHandler, info *model.AuthInfo) {
	t.Helper()
	body := `{"name":"Test Project","key":"TEST","description":"A test project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201 creating project, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Tests ---

func TestCreateProject_Handler_Success(t *testing.T) {
	h, info := projectTestSetup(t)

	body := `{"name":"Infrastructure","key":"INFRA","description":"Infra management"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["key"] != "INFRA" {
		t.Fatalf("expected key 'INFRA', got %v", data["key"])
	}
	if data["name"] != "Infrastructure" {
		t.Fatalf("expected name 'Infrastructure', got %v", data["name"])
	}
}

func TestCreateProject_Handler_MissingName(t *testing.T) {
	h, info := projectTestSetup(t)

	body := `{"key":"INFRA"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateProject_Handler_MissingKey(t *testing.T) {
	h, info := projectTestSetup(t)

	body := `{"name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateProject_Handler_InvalidKey(t *testing.T) {
	h, info := projectTestSetup(t)

	body := `{"name":"Test","key":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListProjects_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 project, got %d", len(data))
	}
}

func TestGetProject_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["key"] != "TEST" {
		t.Fatalf("expected key 'TEST', got %v", data["key"])
	}
}

func TestGetProject_Handler_NotFound(t *testing.T) {
	h, info := projectTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/NOPE", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "NOPE")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteProject_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/TEST", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListMembers_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/members", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListMembers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 member (owner), got %d", len(data))
	}
	totalCount := int(resp["total_count"].(float64))
	if totalCount != 1 {
		t.Fatalf("expected total_count 1, got %d", totalCount)
	}
}

func TestUpdateProject_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	body := `{"name":"Updated Project"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/TEST", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["name"] != "Updated Project" {
		t.Fatalf("expected name 'Updated Project', got %v", data["name"])
	}
}

func TestCreateProject_Handler_Unauthenticated(t *testing.T) {
	h, _ := projectTestSetup(t)

	body := `{"name":"Test","key":"TEST"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAddMember_Handler_InvalidRole(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	body := `{"user_id":"` + uuid.New().String() + `","role":"superadmin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/members", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.AddMember(w, req)

	// Invalid role should cause a bad request (validation error via ErrConflict)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddMember_Handler_MissingUserID(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	body := `{"role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/members", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.AddMember(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Type Workflow Handler Tests ---

func TestListTypeWorkflows_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/type-workflows", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListTypeWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	// data should be an array (may be empty if no workflows seeded)
	if _, ok := resp["data"]; !ok {
		t.Fatal("expected 'data' field in response")
	}
}

func TestListTypeWorkflows_Handler_Unauthenticated(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/type-workflows", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListTypeWorkflows(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateTypeWorkflow_Handler(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	wfID := uuid.New()
	body := `{"workflow_id":"` + wfID.String() + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/TEST/type-workflows/task", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	rctx.URLParams.Add("type", "task")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateTypeWorkflow(w, req)

	// The workflow doesn't exist in the mock, so this should fail with a service error
	// mapped to 404 (workflow not found wraps ErrNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent workflow, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTypeWorkflow_Handler_MissingWorkflowID(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/TEST/type-workflows/task", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	rctx.URLParams.Add("type", "task")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateTypeWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTypeWorkflow_Handler_InvalidWorkflowID(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	body := `{"workflow_id":"not-a-uuid"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/TEST/type-workflows/task", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	rctx.URLParams.Add("type", "task")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateTypeWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTypeWorkflow_Handler_Unauthenticated(t *testing.T) {
	h, info := projectTestSetup(t)
	createTestProject(t, h, info)

	wfID := uuid.New()
	body := `{"workflow_id":"` + wfID.String() + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/TEST/type-workflows/task", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	rctx.URLParams.Add("type", "task")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.UpdateTypeWorkflow(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
