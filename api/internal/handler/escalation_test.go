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

// --- Mock escalation repository for handler tests ---

type mockEscalationRepo struct {
	lists    map[uuid.UUID]*model.EscalationList
	mappings map[string]*model.TypeEscalationMapping
}

func newMockEscalationRepo() *mockEscalationRepo {
	return &mockEscalationRepo{
		lists:    make(map[uuid.UUID]*model.EscalationList),
		mappings: make(map[string]*model.TypeEscalationMapping),
	}
}

func (m *mockEscalationRepo) Create(_ context.Context, el *model.EscalationList) error {
	now := time.Now()
	el.CreatedAt = now
	el.UpdatedAt = now
	for i := range el.Levels {
		el.Levels[i].CreatedAt = now
	}
	m.lists[el.ID] = el
	return nil
}

func (m *mockEscalationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.EscalationList, error) {
	el, ok := m.lists[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return el, nil
}

func (m *mockEscalationRepo) List(_ context.Context, projectID uuid.UUID) ([]model.EscalationList, error) {
	var result []model.EscalationList
	for _, el := range m.lists {
		if el.ProjectID == projectID {
			result = append(result, *el)
		}
	}
	return result, nil
}

func (m *mockEscalationRepo) Update(_ context.Context, el *model.EscalationList) error {
	if _, ok := m.lists[el.ID]; !ok {
		return model.ErrNotFound
	}
	el.UpdatedAt = time.Now()
	for i := range el.Levels {
		el.Levels[i].CreatedAt = time.Now()
	}
	m.lists[el.ID] = el
	return nil
}

func (m *mockEscalationRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.lists[id]; !ok {
		return model.ErrNotFound
	}
	for key, mapping := range m.mappings {
		if mapping.EscalationListID == id {
			delete(m.mappings, key)
		}
	}
	delete(m.lists, id)
	return nil
}

func (m *mockEscalationRepo) ListMappings(_ context.Context, projectID uuid.UUID) ([]model.TypeEscalationMapping, error) {
	var result []model.TypeEscalationMapping
	for _, mapping := range m.mappings {
		if mapping.ProjectID == projectID {
			result = append(result, *mapping)
		}
	}
	return result, nil
}

func (m *mockEscalationRepo) UpsertMapping(_ context.Context, mapping *model.TypeEscalationMapping) error {
	key := mapping.ProjectID.String() + ":" + mapping.WorkItemType
	m.mappings[key] = mapping
	return nil
}

func (m *mockEscalationRepo) DeleteMapping(_ context.Context, projectID uuid.UUID, workItemType string) error {
	key := projectID.String() + ":" + workItemType
	if _, ok := m.mappings[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.mappings, key)
	return nil
}

// --- Mock user repository for handler escalation tests ---

type mockEscalationUserRepo struct {
	users map[uuid.UUID]*model.User
}

func newMockEscalationUserRepo() *mockEscalationUserRepo {
	return &mockEscalationUserRepo{users: make(map[uuid.UUID]*model.User)}
}

func (m *mockEscalationUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockEscalationUserRepo) addUser(id uuid.UUID) {
	m.users[id] = &model.User{
		ID:          id,
		Email:       id.String() + "@test.com",
		DisplayName: "User " + id.String()[:8],
		GlobalRole:  model.RoleUser,
		IsActive:    true,
	}
}

// --- Test setup ---

func escalationTestSetup(t *testing.T) (*EscalationHandler, *model.AuthInfo, string, uuid.UUID) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	escRepo := newMockEscalationRepo()
	userRepo := newMockEscalationUserRepo()
	svc := service.NewEscalationService(escRepo, projectRepo, memberRepo, userRepo)
	h := NewEscalationHandler(svc)

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

	// Add a test user for levels
	testUserID := uuid.New()
	userRepo.addUser(testUserID)

	return h, info, "TEST", testUserID
}

func createTestEscalationList(t *testing.T, h *EscalationHandler, info *model.AuthInfo, projectKey string, userID uuid.UUID) string {
	t.Helper()
	body := `{"name":"Test Escalation","levels":[{"threshold_pct":75,"user_ids":["` + userID.String() + `"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/escalation-lists", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	return data["id"].(string)
}

// --- Tests ---

func TestEscalationHandler_Create(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	body := `{"name":"Critical","levels":[{"threshold_pct":75,"user_ids":["` + userID.String() + `"]},{"threshold_pct":100,"user_ids":["` + userID.String() + `"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/escalation-lists", bytes.NewBufferString(body))
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
	if data["name"] != "Critical" {
		t.Fatalf("expected name 'Critical', got %v", data["name"])
	}
}

func TestEscalationHandler_Create_InvalidBody(t *testing.T) {
	h, info, projectKey, _ := escalationTestSetup(t)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/escalation-lists", bytes.NewBufferString(body))
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

func TestEscalationHandler_List(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	createTestEscalationList(t, h, info, projectKey, userID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/escalation-lists", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 list, got %d", len(data))
	}
}

func TestEscalationHandler_Get(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	listID := createTestEscalationList(t, h, info, projectKey, userID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/escalation-lists/"+listID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("listId", listID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEscalationHandler_Update(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	listID := createTestEscalationList(t, h, info, projectKey, userID)

	body := `{"name":"Updated","levels":[{"threshold_pct":50,"user_ids":["` + userID.String() + `"]}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/default/projects/"+projectKey+"/escalation-lists/"+listID, bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("listId", listID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if data["name"] != "Updated" {
		t.Fatalf("expected name 'Updated', got %v", data["name"])
	}
}

func TestEscalationHandler_Delete(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	listID := createTestEscalationList(t, h, info, projectKey, userID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/default/projects/"+projectKey+"/escalation-lists/"+listID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("listId", listID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEscalationHandler_Unauthenticated(t *testing.T) {
	h, _, projectKey, _ := escalationTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/escalation-lists", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestEscalationHandler_Get_InvalidID(t *testing.T) {
	h, info, projectKey, _ := escalationTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/escalation-lists/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("listId", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEscalationHandler_ListMappings(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	listID := createTestEscalationList(t, h, info, projectKey, userID)

	// Set a mapping
	body := `{"escalation_list_id":"` + listID + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/default/projects/"+projectKey+"/escalation-lists/mappings/bug", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("type", "bug")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.UpdateMapping(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for update mapping, got %d: %s", w.Code, w.Body.String())
	}

	// List mappings
	req = httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/escalation-lists/mappings", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()
	h.ListMappings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(data))
	}
}

func TestEscalationHandler_DeleteMapping(t *testing.T) {
	h, info, projectKey, userID := escalationTestSetup(t)

	listID := createTestEscalationList(t, h, info, projectKey, userID)

	// Set a mapping first
	body := `{"escalation_list_id":"` + listID + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/default/projects/"+projectKey+"/escalation-lists/mappings/bug", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("type", "bug")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.UpdateMapping(w, req)

	// Delete the mapping
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/default/projects/"+projectKey+"/escalation-lists/mappings/bug", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("type", "bug")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.DeleteMapping(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
