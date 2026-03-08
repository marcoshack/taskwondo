package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// --- mock ---

type mockSavedSearchRepo struct {
	items map[uuid.UUID]*model.SavedSearch
}

func newMockSavedSearchRepo() *mockSavedSearchRepo {
	return &mockSavedSearchRepo{items: make(map[uuid.UUID]*model.SavedSearch)}
}

func (m *mockSavedSearchRepo) Create(_ context.Context, s *model.SavedSearch) error {
	m.items[s.ID] = s
	return nil
}

func (m *mockSavedSearchRepo) GetByID(_ context.Context, id uuid.UUID) (*model.SavedSearch, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *mockSavedSearchRepo) ListByProjectAndUser(_ context.Context, projectID, userID uuid.UUID) ([]model.SavedSearch, error) {
	var result []model.SavedSearch
	for _, s := range m.items {
		if s.ProjectID == projectID && (s.UserID == nil || *s.UserID == userID) {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSavedSearchRepo) Update(_ context.Context, s *model.SavedSearch) error {
	if _, ok := m.items[s.ID]; !ok {
		return model.ErrNotFound
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockSavedSearchRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

// --- helpers ---

func savedSearchTestSetup(t *testing.T) (*SavedSearchHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	ssRepo := newMockSavedSearchRepo()
	svc := service.NewSavedSearchService(ssRepo, projectRepo, memberRepo)
	h := NewSavedSearchHandler(svc)

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

func ssRequest(method, url, body string, projectKey string, info *model.AuthInfo, extraParams ...string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	for i := 0; i+1 < len(extraParams); i += 2 {
		rctx.URLParams.Add(extraParams[i], extraParams[i+1])
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	return req, httptest.NewRecorder()
}

// --- Tests ---

func TestSavedSearchHandler_Create(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	body := `{"name":"My bugs","filters":{"type":["bug"]},"view_mode":"list","shared":false}`
	req, w := ssRequest(http.MethodPost, "/api/v1/default/projects/"+pk+"/saved-searches", body, pk, info)
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]any
	json.Unmarshal(resp["data"], &data)
	if data["name"] != "My bugs" {
		t.Fatalf("expected name 'My bugs', got %v", data["name"])
	}
	if data["scope"] != "user" {
		t.Fatalf("expected scope 'user', got %v", data["scope"])
	}
}

func TestSavedSearchHandler_Create_MissingName(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	body := `{"name":"","filters":{},"view_mode":"list"}`
	req, w := ssRequest(http.MethodPost, "/api/v1/default/projects/"+pk+"/saved-searches", body, pk, info)
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSavedSearchHandler_Create_InvalidBody(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	req, w := ssRequest(http.MethodPost, "/api/v1/default/projects/"+pk+"/saved-searches", "{invalid}", pk, info)
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSavedSearchHandler_List(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	// Create a search first
	body := `{"name":"All bugs","filters":{"type":["bug"]},"view_mode":"list"}`
	req, w := ssRequest(http.MethodPost, "/api/v1/default/projects/"+pk+"/saved-searches", body, pk, info)
	h.Create(w, req)

	// List
	req, w = ssRequest(http.MethodGet, "/api/v1/default/projects/"+pk+"/saved-searches", "", pk, info)
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 saved search, got %d", len(data))
	}
}

func TestSavedSearchHandler_Update(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	// Create
	body := `{"name":"Old name","filters":{},"view_mode":"list"}`
	req, w := ssRequest(http.MethodPost, "/api/v1/default/projects/"+pk+"/saved-searches", body, pk, info)
	h.Create(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]any
	json.Unmarshal(createResp["data"], &createdData)
	searchID := createdData["id"].(string)

	// Update
	body = `{"name":"New name"}`
	req, w = ssRequest(http.MethodPatch, "/api/v1/default/projects/"+pk+"/saved-searches/"+searchID, body, pk, info, "searchId", searchID)
	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updateResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	var updatedData map[string]any
	json.Unmarshal(updateResp["data"], &updatedData)
	if updatedData["name"] != "New name" {
		t.Fatalf("expected name 'New name', got %v", updatedData["name"])
	}
}

func TestSavedSearchHandler_Update_InvalidID(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	body := `{"name":"New name"}`
	req, w := ssRequest(http.MethodPatch, "/api/v1/default/projects/"+pk+"/saved-searches/not-a-uuid", body, pk, info, "searchId", "not-a-uuid")
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSavedSearchHandler_Delete(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	// Create
	body := `{"name":"To delete","filters":{},"view_mode":"list"}`
	req, w := ssRequest(http.MethodPost, "/api/v1/default/projects/"+pk+"/saved-searches", body, pk, info)
	h.Create(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]any
	json.Unmarshal(createResp["data"], &createdData)
	searchID := createdData["id"].(string)

	// Delete
	req, w = ssRequest(http.MethodDelete, "/api/v1/default/projects/"+pk+"/saved-searches/"+searchID, "", pk, info, "searchId", searchID)
	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSavedSearchHandler_Delete_NotFound(t *testing.T) {
	h, info, pk := savedSearchTestSetup(t)

	fakeID := uuid.New().String()
	req, w := ssRequest(http.MethodDelete, "/api/v1/default/projects/"+pk+"/saved-searches/"+fakeID, "", pk, info, "searchId", fakeID)
	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSavedSearchHandler_Unauthorized(t *testing.T) {
	h, _, pk := savedSearchTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+pk+"/saved-searches", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", pk)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	// No auth info
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
