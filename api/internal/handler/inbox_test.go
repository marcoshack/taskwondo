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

// --- Mock InboxRepository for handler tests ---

type mockHandlerInboxRepo struct {
	items      map[uuid.UUID]*model.InboxItem
	projectIDs map[uuid.UUID]uuid.UUID
}

func newMockHandlerInboxRepo() *mockHandlerInboxRepo {
	return &mockHandlerInboxRepo{
		items:      make(map[uuid.UUID]*model.InboxItem),
		projectIDs: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockHandlerInboxRepo) Add(_ context.Context, item *model.InboxItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockHandlerInboxRepo) Remove(_ context.Context, userID, workItemID uuid.UUID) error {
	for id, item := range m.items {
		if item.UserID == userID && item.WorkItemID == workItemID {
			delete(m.items, id)
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *mockHandlerInboxRepo) RemoveByID(_ context.Context, id, userID uuid.UUID) error {
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return model.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockHandlerInboxRepo) List(_ context.Context, userID uuid.UUID, _ bool, _ string, _ []string, _ *uuid.UUID, limit int) (*model.InboxItemList, error) {
	var items []model.InboxItemWithWorkItem
	for _, item := range m.items {
		if item.UserID == userID {
			items = append(items, model.InboxItemWithWorkItem{
				InboxItem:      *item,
				DisplayID:      "TEST-1",
				Title:          "Test Item",
				Type:           model.WorkItemTypeTask,
				Status:         "open",
				StatusCategory: model.CategoryTodo,
				Priority:       model.PriorityMedium,
				ProjectKey:     "TEST",
				ProjectName:    "Test Project",
			})
		}
	}
	return &model.InboxItemList{
		Items:   items,
		HasMore: false,
		Total:   len(items),
	}, nil
}

func (m *mockHandlerInboxRepo) CountByUser(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, item := range m.items {
		if item.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockHandlerInboxRepo) CountAllByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return m.CountByUser(ctx, userID)
}

func (m *mockHandlerInboxRepo) UpdatePosition(_ context.Context, id, userID uuid.UUID, position int) error {
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return model.ErrNotFound
	}
	item.Position = position
	return nil
}

func (m *mockHandlerInboxRepo) MaxPosition(_ context.Context, userID uuid.UUID) (int, error) {
	maxPos := 0
	for _, item := range m.items {
		if item.UserID == userID && item.Position > maxPos {
			maxPos = item.Position
		}
	}
	return maxPos, nil
}

func (m *mockHandlerInboxRepo) RemoveCompleted(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockHandlerInboxRepo) Exists(_ context.Context, userID, workItemID uuid.UUID) (bool, error) {
	for _, item := range m.items {
		if item.UserID == userID && item.WorkItemID == workItemID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockHandlerInboxRepo) GetWorkItemProjectID(_ context.Context, workItemID uuid.UUID) (uuid.UUID, error) {
	pid, ok := m.projectIDs[workItemID]
	if !ok {
		return uuid.Nil, model.ErrNotFound
	}
	return pid, nil
}

func (m *mockHandlerInboxRepo) RemoveByProjectID(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// --- Test setup ---

func inboxTestSetup(t *testing.T) (*InboxHandler, *model.AuthInfo, *mockHandlerInboxRepo) {
	t.Helper()

	inboxRepo := newMockHandlerInboxRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := service.NewInboxService(inboxRepo, memberRepo)
	h := NewInboxHandler(svc, nil)

	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}

	// Set up a project and member
	projectID := uuid.New()
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: projectID,
		UserID:    info.UserID,
		Role:      model.ProjectRoleMember,
	})

	// Register a work item in that project
	workItemID := uuid.New()
	inboxRepo.projectIDs[workItemID] = projectID

	return h, info, inboxRepo
}

func authRequest(req *http.Request, info *model.AuthInfo) *http.Request {
	return req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
}

// --- Tests ---

func TestInboxHandler_Add(t *testing.T) {
	h, info, inboxRepo := inboxTestSetup(t)

	// Get a valid work item ID
	var workItemID string
	for wiID := range inboxRepo.projectIDs {
		workItemID = wiID.String()
		break
	}

	body := `{"work_item_id":"` + workItemID + `"}`
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w := httptest.NewRecorder()

	h.Add(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInboxHandler_Add_InvalidBody(t *testing.T) {
	h, info, _ := inboxTestSetup(t)

	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString("{invalid}")), info)
	w := httptest.NewRecorder()

	h.Add(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInboxHandler_Add_InvalidUUID(t *testing.T) {
	h, info, _ := inboxTestSetup(t)

	body := `{"work_item_id":"not-a-uuid"}`
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w := httptest.NewRecorder()

	h.Add(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInboxHandler_Add_Duplicate(t *testing.T) {
	h, info, inboxRepo := inboxTestSetup(t)

	var workItemID string
	for wiID := range inboxRepo.projectIDs {
		workItemID = wiID.String()
		break
	}

	body := `{"work_item_id":"` + workItemID + `"}`

	// First add
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w := httptest.NewRecorder()
	h.Add(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first add: expected 201, got %d", w.Code)
	}

	// Duplicate add
	req = authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w = httptest.NewRecorder()
	h.Add(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate add: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInboxHandler_List(t *testing.T) {
	h, info, inboxRepo := inboxTestSetup(t)

	// Add an item first
	var workItemID string
	for wiID := range inboxRepo.projectIDs {
		workItemID = wiID.String()
		break
	}
	body := `{"work_item_id":"` + workItemID + `"}`
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w := httptest.NewRecorder()
	h.Add(w, req)

	// List
	req = authRequest(httptest.NewRequest(http.MethodGet, "/api/v1/user/inbox", nil), info)
	w = httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data inboxListResponse
	json.Unmarshal(resp["data"], &data)
	if data.Total != 1 {
		t.Fatalf("expected 1 item, got %d", data.Total)
	}
}

func TestInboxHandler_Remove(t *testing.T) {
	h, info, inboxRepo := inboxTestSetup(t)

	// Add an item
	var workItemID string
	for wiID := range inboxRepo.projectIDs {
		workItemID = wiID.String()
		break
	}
	body := `{"work_item_id":"` + workItemID + `"}`
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w := httptest.NewRecorder()
	h.Add(w, req)

	// Find the inbox item ID
	var inboxItemID string
	for id := range inboxRepo.items {
		inboxItemID = id.String()
		break
	}

	// Remove
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/user/inbox/"+inboxItemID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("inboxItemId", inboxItemID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = authRequest(req, info)
	w = httptest.NewRecorder()
	h.Remove(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInboxHandler_Remove_NotFound(t *testing.T) {
	h, info, _ := inboxTestSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/inbox/"+uuid.New().String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("inboxItemId", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = authRequest(req, info)
	w := httptest.NewRecorder()
	h.Remove(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestInboxHandler_Reorder(t *testing.T) {
	h, info, inboxRepo := inboxTestSetup(t)

	// Add an item
	var workItemID string
	for wiID := range inboxRepo.projectIDs {
		workItemID = wiID.String()
		break
	}
	body := `{"work_item_id":"` + workItemID + `"}`
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/v1/user/inbox", bytes.NewBufferString(body)), info)
	w := httptest.NewRecorder()
	h.Add(w, req)

	var inboxItemID string
	for id := range inboxRepo.items {
		inboxItemID = id.String()
		break
	}

	// Reorder
	body = `{"position":500}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/user/inbox/"+inboxItemID, bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("inboxItemId", inboxItemID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = authRequest(req, info)
	w = httptest.NewRecorder()
	h.Reorder(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInboxHandler_Count(t *testing.T) {
	h, info, _ := inboxTestSetup(t)

	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/v1/user/inbox/count", nil), info)
	w := httptest.NewRecorder()
	h.Count(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data inboxCountResponse
	json.Unmarshal(resp["data"], &data)
	if data.Count != 0 {
		t.Fatalf("expected count 0, got %d", data.Count)
	}
}

func TestInboxHandler_ClearCompleted(t *testing.T) {
	h, info, _ := inboxTestSetup(t)

	req := authRequest(httptest.NewRequest(http.MethodDelete, "/api/v1/user/inbox/completed", nil), info)
	w := httptest.NewRecorder()
	h.ClearCompleted(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data clearCompletedResponse
	json.Unmarshal(resp["data"], &data)
	if data.Removed != 0 {
		t.Fatalf("expected 0 removed, got %d", data.Removed)
	}
}
