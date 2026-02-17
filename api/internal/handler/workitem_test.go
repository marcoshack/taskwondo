package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
	"github.com/marcoshack/trackforge/internal/service"
)

// --- Mock work item repository ---

type mockWorkItemRepo struct {
	items        map[uuid.UUID]*model.WorkItem
	byProjectNum map[string]*model.WorkItem
	counters     map[uuid.UUID]int
}

func newMockWorkItemRepo() *mockWorkItemRepo {
	return &mockWorkItemRepo{
		items:        make(map[uuid.UUID]*model.WorkItem),
		byProjectNum: make(map[string]*model.WorkItem),
		counters:     make(map[uuid.UUID]int),
	}
}

func wiKey(projectID uuid.UUID, itemNumber int) string {
	return fmt.Sprintf("%s:%d", projectID, itemNumber)
}

func (m *mockWorkItemRepo) Create(_ context.Context, item *model.WorkItem) error {
	m.counters[item.ProjectID]++
	item.ItemNumber = m.counters[item.ProjectID]
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	m.items[item.ID] = item
	m.byProjectNum[wiKey(item.ProjectID, item.ItemNumber)] = item
	return nil
}

func (m *mockWorkItemRepo) GetByProjectAndNumber(_ context.Context, projectID uuid.UUID, itemNumber int) (*model.WorkItem, error) {
	key := wiKey(projectID, itemNumber)
	item, ok := m.byProjectNum[key]
	if !ok || item.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return item, nil
}

func (m *mockWorkItemRepo) List(_ context.Context, projectID uuid.UUID, filter *model.WorkItemFilter) (*model.WorkItemList, error) {
	var matched []model.WorkItem
	for _, item := range m.items {
		if item.ProjectID != projectID || item.DeletedAt != nil {
			continue
		}
		if len(filter.Types) > 0 && !strContains(filter.Types, item.Type) {
			continue
		}
		if len(filter.Statuses) > 0 && !strContains(filter.Statuses, item.Status) {
			continue
		}
		matched = append(matched, *item)
	}

	total := len(matched)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	hasMore := len(matched) > limit
	if hasMore {
		matched = matched[:limit]
	}

	var cursor string
	if len(matched) > 0 {
		cursor = matched[len(matched)-1].ID.String()
	}

	return &model.WorkItemList{
		Items:   matched,
		Cursor:  cursor,
		HasMore: hasMore,
		Total:   total,
	}, nil
}

func (m *mockWorkItemRepo) Update(_ context.Context, item *model.WorkItem) error {
	existing, ok := m.items[item.ID]
	if !ok || existing.DeletedAt != nil {
		return model.ErrNotFound
	}
	now := time.Now()
	item.UpdatedAt = now
	m.items[item.ID] = item
	m.byProjectNum[wiKey(item.ProjectID, item.ItemNumber)] = item
	return nil
}

func (m *mockWorkItemRepo) Delete(_ context.Context, id uuid.UUID) error {
	item, ok := m.items[id]
	if !ok || item.DeletedAt != nil {
		return model.ErrNotFound
	}
	now := time.Now()
	item.DeletedAt = &now
	return nil
}

func strContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- Mock work item event repository ---

type mockWorkItemEventRepo struct {
	events map[uuid.UUID][]model.WorkItemEvent
}

func newMockWorkItemEventRepo() *mockWorkItemEventRepo {
	return &mockWorkItemEventRepo{
		events: make(map[uuid.UUID][]model.WorkItemEvent),
	}
}

func (m *mockWorkItemEventRepo) Create(_ context.Context, event *model.WorkItemEvent) error {
	event.CreatedAt = time.Now()
	m.events[event.WorkItemID] = append(m.events[event.WorkItemID], *event)
	return nil
}

func (m *mockWorkItemEventRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID) ([]model.WorkItemEvent, error) {
	return m.events[workItemID], nil
}

// --- Test setup ---

func workItemTestSetup(t *testing.T) (*WorkItemHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	itemRepo := newMockWorkItemRepo()
	eventRepo := newMockWorkItemEventRepo()

	svc := service.NewWorkItemService(itemRepo, eventRepo, projectRepo, memberRepo)
	h := NewWorkItemHandler(svc)

	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}

	// Create a project and add the user as owner
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

func createTestWorkItem(t *testing.T, h *WorkItemHandler, info *model.AuthInfo, projectKey string) map[string]interface{} {
	t.Helper()
	body := `{"type":"task","title":"Test item","priority":"medium"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201 creating work item, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["data"].(map[string]interface{})
}

// --- Tests ---

func TestCreateWorkItem_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	body := `{"type":"task","title":"Upgrade PostgreSQL","priority":"high","labels":["database"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})

	if data["title"] != "Upgrade PostgreSQL" {
		t.Fatalf("expected title 'Upgrade PostgreSQL', got %v", data["title"])
	}
	if data["type"] != "task" {
		t.Fatalf("expected type 'task', got %v", data["type"])
	}
	if data["display_id"] != "TEST-1" {
		t.Fatalf("expected display_id 'TEST-1', got %v", data["display_id"])
	}
	if data["priority"] != "high" {
		t.Fatalf("expected priority 'high', got %v", data["priority"])
	}
}

func TestCreateWorkItem_Handler_MissingTitle(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	body := `{"type":"task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateWorkItem_Handler_MissingType(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	body := `{"title":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateWorkItem_Handler_InvalidBody(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	body := `not json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateWorkItem_Handler_Unauthenticated(t *testing.T) {
	h, _, _ := workItemTestSetup(t)

	body := `{"type":"task","title":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListWorkItems_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)
	createTestWorkItem(t, h, info, projectKey)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(data))
	}

	meta := resp["meta"].(map[string]interface{})
	if meta["total"].(float64) != 2 {
		t.Fatalf("expected total 2, got %v", meta["total"])
	}
}

func TestListWorkItems_Handler_WithFilters(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	// Create a task
	createTestWorkItem(t, h, info, projectKey)

	// Create a bug
	body := `{"type":"bug","title":"A bug"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Create(w, req)

	// List filtering by type=bug
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items?type=bug", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 bug, got %d items", len(data))
	}
}

func TestGetWorkItem_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
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
	if data["display_id"] != "TEST-1" {
		t.Fatalf("expected display_id 'TEST-1', got %v", data["display_id"])
	}
}

func TestGetWorkItem_Handler_NotFound(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "999")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetWorkItem_Handler_InvalidItemNumber(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "abc")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateWorkItem_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	body := `{"title":"Updated title","status":"in_progress"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/TEST/items/1", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
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
	if data["title"] != "Updated title" {
		t.Fatalf("expected title 'Updated title', got %v", data["title"])
	}
	if data["status"] != "in_progress" {
		t.Fatalf("expected status 'in_progress', got %v", data["status"])
	}
}

func TestDeleteWorkItem_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/TEST/items/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteWorkItem_Handler_NotFound(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/TEST/items/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "999")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
