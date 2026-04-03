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

func queueTestSetup(t *testing.T) (*QueueHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	queueRepo := newMockQueueRepo()
	svc := service.NewQueueService(queueRepo, nil, nil, projectRepo, memberRepo)
	h := NewQueueHandler(svc)

	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}

	project := &model.Project{ID: uuid.New(), Name: "Test Project", Key: "TEST"}
	projectRepo.Create(context.Background(), project)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    info.UserID,
		Role:      model.ProjectRoleOwner,
	})

	return h, info, "TEST"
}

func TestQueueHandler_Create(t *testing.T) {
	h, info, projectKey := queueTestSetup(t)

	body := `{"name":"Support","queue_type":"support","default_priority":"medium"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if data["name"] != "Support" {
		t.Fatalf("expected name 'Support', got %v", data["name"])
	}
}

func TestQueueHandler_Create_InvalidBody(t *testing.T) {
	h, info, projectKey := queueTestSetup(t)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestQueueHandler_List(t *testing.T) {
	h, info, projectKey := queueTestSetup(t)

	// Create a queue first
	body := `{"name":"Alerts","queue_type":"alerts","default_priority":"high"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.Create(w, req)

	// List queues
	req = httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/queues", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 queue, got %d", len(data))
	}
}

func TestQueueHandler_Get(t *testing.T) {
	h, info, projectKey := queueTestSetup(t)

	// Create a queue
	body := `{"name":"Feedback","queue_type":"feedback"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	queueID := createdData["id"].(string)

	// Get queue
	req = httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQueueHandler_Delete(t *testing.T) {
	h, info, projectKey := queueTestSetup(t)

	// Create a queue
	body := `{"name":"ToDelete","queue_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	queueID := createdData["id"].(string)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQueueHandler_Unauthenticated(t *testing.T) {
	h, _, projectKey := queueTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/queues", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- Mock repos for category/team handler tests ---

type mockQueueCategoryRepo struct {
	categories map[uuid.UUID]*model.QueueCategory
}

func newMockQueueCategoryRepo() *mockQueueCategoryRepo {
	return &mockQueueCategoryRepo{categories: make(map[uuid.UUID]*model.QueueCategory)}
}

func (m *mockQueueCategoryRepo) Create(_ context.Context, cat *model.QueueCategory) error {
	now := time.Now()
	cat.CreatedAt = now
	cat.UpdatedAt = now
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockQueueCategoryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.QueueCategory, error) {
	cat, ok := m.categories[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return cat, nil
}

func (m *mockQueueCategoryRepo) ListByQueue(_ context.Context, queueID uuid.UUID) ([]model.QueueCategory, error) {
	var result []model.QueueCategory
	for _, cat := range m.categories {
		if cat.QueueID == queueID {
			result = append(result, *cat)
		}
	}
	return result, nil
}

func (m *mockQueueCategoryRepo) Update(_ context.Context, cat *model.QueueCategory) error {
	if _, ok := m.categories[cat.ID]; !ok {
		return model.ErrNotFound
	}
	cat.UpdatedAt = time.Now()
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockQueueCategoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.categories[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.categories, id)
	return nil
}

type mockQueueTeamRepo struct {
	assignments map[string]bool
	teams       map[uuid.UUID]*model.Team
}

func newMockQueueTeamRepo() *mockQueueTeamRepo {
	return &mockQueueTeamRepo{
		assignments: make(map[string]bool),
		teams:       make(map[uuid.UUID]*model.Team),
	}
}

func (m *mockQueueTeamRepo) Assign(_ context.Context, queueID, teamID uuid.UUID) error {
	m.assignments[queueID.String()+":"+teamID.String()] = true
	return nil
}

func (m *mockQueueTeamRepo) Unassign(_ context.Context, queueID, teamID uuid.UUID) error {
	key := queueID.String() + ":" + teamID.String()
	if !m.assignments[key] {
		return model.ErrNotFound
	}
	delete(m.assignments, key)
	return nil
}

func (m *mockQueueTeamRepo) ListTeamsByQueue(_ context.Context, queueID uuid.UUID) ([]model.Team, error) {
	var result []model.Team
	for key := range m.assignments {
		qID := key[:36]
		tID := key[37:]
		if qID == queueID.String() {
			teamID, _ := uuid.Parse(tID)
			if team, ok := m.teams[teamID]; ok {
				result = append(result, *team)
			}
		}
	}
	return result, nil
}

func (m *mockQueueTeamRepo) ListQueuesByTeam(_ context.Context, _ uuid.UUID) ([]model.Queue, error) {
	return nil, nil
}

func categoryTestSetup(t *testing.T) (*QueueHandler, *model.AuthInfo, string, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	queueRepo := newMockQueueRepo()
	categoryRepo := newMockQueueCategoryRepo()
	queueTeamRepo := newMockQueueTeamRepo()
	svc := service.NewQueueService(queueRepo, categoryRepo, queueTeamRepo, projectRepo, memberRepo)
	h := NewQueueHandler(svc)

	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}

	project := &model.Project{ID: uuid.New(), Name: "Test Project", Key: "TEST"}
	projectRepo.Create(context.Background(), project)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    info.UserID,
		Role:      model.ProjectRoleOwner,
	})

	// Create a queue
	q := &model.Queue{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Test Queue",
		QueueType: model.QueueTypeGeneral,
	}
	queueRepo.Create(context.Background(), q)

	return h, info, "TEST", q.ID.String()
}

func TestQueueHandler_CreateCategory(t *testing.T) {
	h, info, projectKey, queueID := categoryTestSetup(t)

	body := `{"name":"Bug Reports","position":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/categories", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.CreateCategory(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if data["name"] != "Bug Reports" {
		t.Fatalf("expected name 'Bug Reports', got %v", data["name"])
	}
}

func TestQueueHandler_ListCategories(t *testing.T) {
	h, info, projectKey, queueID := categoryTestSetup(t)

	// Create a category first
	body := `{"name":"Feature Requests","position":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/categories", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.CreateCategory(w, req)

	// List categories
	req = httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/categories", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.ListCategories(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 category, got %d", len(data))
	}
}

func TestQueueHandler_DeleteCategory(t *testing.T) {
	h, info, projectKey, queueID := categoryTestSetup(t)

	// Create a category first
	body := `{"name":"ToDelete","position":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/categories", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.CreateCategory(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	categoryID := createdData["id"].(string)

	// Delete category
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/categories/"+categoryID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	rctx.URLParams.Add("categoryId", categoryID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.DeleteCategory(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQueueHandler_AssignQueueTeam(t *testing.T) {
	h, info, projectKey, queueID := categoryTestSetup(t)

	teamID := uuid.New().String()
	body := `{"team_id":"` + teamID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/teams", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.AssignQueueTeam(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQueueHandler_ListQueueTeams(t *testing.T) {
	h, info, projectKey, queueID := categoryTestSetup(t)

	// List teams (should be empty)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/queues/"+queueID+"/teams", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("queueId", queueID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ListQueueTeams(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
