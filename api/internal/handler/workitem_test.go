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

func (m *mockWorkItemEventRepo) ListByWorkItemFiltered(_ context.Context, workItemID uuid.UUID, visibility string) ([]model.WorkItemEventWithActor, error) {
	events := m.events[workItemID]
	var result []model.WorkItemEventWithActor
	for _, e := range events {
		if visibility != "" && e.Visibility != visibility {
			continue
		}
		result = append(result, model.WorkItemEventWithActor{WorkItemEvent: e})
	}
	return result, nil
}

// --- Mock comment repository ---

type mockCommentRepo struct {
	comments map[uuid.UUID]*model.Comment
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{comments: make(map[uuid.UUID]*model.Comment)}
}

func (m *mockCommentRepo) Create(_ context.Context, comment *model.Comment) error {
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	m.comments[comment.ID] = comment
	return nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Comment, error) {
	c, ok := m.comments[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return c, nil
}

func (m *mockCommentRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID, visibility string) ([]model.Comment, error) {
	var result []model.Comment
	for _, c := range m.comments {
		if c.WorkItemID != workItemID {
			continue
		}
		if visibility != "" && c.Visibility != visibility {
			continue
		}
		result = append(result, *c)
	}
	return result, nil
}

func (m *mockCommentRepo) Update(_ context.Context, comment *model.Comment) error {
	existing, ok := m.comments[comment.ID]
	if !ok {
		return model.ErrNotFound
	}
	existing.Body = comment.Body
	existing.UpdatedAt = time.Now()
	return nil
}

func (m *mockCommentRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.comments[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.comments, id)
	return nil
}

// --- Mock relation repository ---

type mockRelationRepo struct {
	relations map[uuid.UUID]*model.WorkItemRelation
}

func newMockRelationRepo() *mockRelationRepo {
	return &mockRelationRepo{relations: make(map[uuid.UUID]*model.WorkItemRelation)}
}

func (m *mockRelationRepo) Create(_ context.Context, rel *model.WorkItemRelation) error {
	rel.CreatedAt = time.Now()
	m.relations[rel.ID] = rel
	return nil
}

func (m *mockRelationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.WorkItemRelation, error) {
	rel, ok := m.relations[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return rel, nil
}

func (m *mockRelationRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID) ([]model.WorkItemRelation, error) {
	var result []model.WorkItemRelation
	for _, rel := range m.relations {
		if rel.SourceID == workItemID || rel.TargetID == workItemID {
			result = append(result, *rel)
		}
	}
	return result, nil
}

func (m *mockRelationRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.relations[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.relations, id)
	return nil
}

// --- Test setup ---

func workItemTestSetup(t *testing.T) (*WorkItemHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	itemRepo := newMockWorkItemRepo()
	eventRepo := newMockWorkItemEventRepo()
	commentRepo := newMockCommentRepo()
	relationRepo := newMockRelationRepo()

	svc := service.NewWorkItemService(itemRepo, eventRepo, commentRepo, relationRepo, projectRepo, memberRepo)
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

// --- Comment Handler Tests ---

func TestCreateComment_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	body := `{"body":"This is a comment","visibility":"internal"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/comments", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateComment(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["body"] != "This is a comment" {
		t.Fatalf("expected body 'This is a comment', got %v", data["body"])
	}
}

func TestCreateComment_Handler_EmptyBody(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	body := `{"body":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/comments", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateComment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListComments_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	// Create 2 comments
	for i := 0; i < 2; i++ {
		body := fmt.Sprintf(`{"body":"Comment %d"}`, i)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/comments", bytes.NewBufferString(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("projectKey", projectKey)
		rctx.URLParams.Add("itemNumber", "1")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = model.ContextWithAuthInfo(ctx, info)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		h.CreateComment(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/1/comments", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListComments(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(data))
	}
}

func TestUpdateComment_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	// Create a comment
	createBody := `{"body":"Original"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/comments", bytes.NewBufferString(createBody))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.CreateComment(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	commentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Update the comment
	updateBody := `{"body":"Updated"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/projects/TEST/items/1/comments/"+commentID, bytes.NewBufferString(updateBody))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	rctx.URLParams.Add("commentId", commentID)
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	h.UpdateComment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updateResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	data := updateResp["data"].(map[string]interface{})
	if data["body"] != "Updated" {
		t.Fatalf("expected body 'Updated', got %v", data["body"])
	}
}

func TestDeleteComment_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	// Create a comment
	createBody := `{"body":"To delete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/comments", bytes.NewBufferString(createBody))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.CreateComment(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	commentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Delete it
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/TEST/items/1/comments/"+commentID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	rctx.URLParams.Add("commentId", commentID)
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	h.DeleteComment(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Relation Handler Tests ---

func TestCreateRelation_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey) // item 1
	createTestWorkItem(t, h, info, projectKey) // item 2

	body := `{"target_display_id":"TEST-2","relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/relations", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateRelation(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["relation_type"] != "blocks" {
		t.Fatalf("expected relation_type 'blocks', got %v", data["relation_type"])
	}
	if data["source_display_id"] != "TEST-1" {
		t.Fatalf("expected source_display_id 'TEST-1', got %v", data["source_display_id"])
	}
	if data["target_display_id"] != "TEST-2" {
		t.Fatalf("expected target_display_id 'TEST-2', got %v", data["target_display_id"])
	}
}

func TestCreateRelation_Handler_MissingFields(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey)

	body := `{"relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/relations", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateRelation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListRelations_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey) // item 1
	createTestWorkItem(t, h, info, projectKey) // item 2

	// Create a relation
	body := `{"target_display_id":"TEST-2","relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/relations", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.CreateRelation(w, req)

	// List relations
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/1/relations", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	h.ListRelations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(data))
	}
}

func TestDeleteRelation_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey) // item 1
	createTestWorkItem(t, h, info, projectKey) // item 2

	// Create a relation
	body := `{"target_display_id":"TEST-2","relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/items/1/relations", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.CreateRelation(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	relationID := createResp["data"].(map[string]interface{})["id"].(string)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/TEST/items/1/relations/"+relationID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	rctx.URLParams.Add("relationId", relationID)
	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	h.DeleteRelation(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Event Handler Tests ---

func TestListEvents_Handler_Success(t *testing.T) {
	h, info, projectKey := workItemTestSetup(t)
	createTestWorkItem(t, h, info, projectKey) // creates a "created" event

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/1/events", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("itemNumber", "1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) < 1 {
		t.Fatal("expected at least 1 event")
	}
}

func TestListEvents_Handler_Unauthenticated(t *testing.T) {
	h, _, _ := workItemTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/items/1/events", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", "TEST")
	rctx.URLParams.Add("itemNumber", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.ListEvents(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
