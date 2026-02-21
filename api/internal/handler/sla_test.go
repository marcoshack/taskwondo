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

// --- In-memory SLA repo for handler tests ---

type inMemorySLARepo struct {
	targets map[uuid.UUID]*model.SLAStatusTarget
	elapsed map[string]*model.SLAElapsed
}

func newInMemorySLARepo() *inMemorySLARepo {
	return &inMemorySLARepo{
		targets: make(map[uuid.UUID]*model.SLAStatusTarget),
		elapsed: make(map[string]*model.SLAElapsed),
	}
}

func (m *inMemorySLARepo) ListTargetsByProject(_ context.Context, projectID uuid.UUID) ([]model.SLAStatusTarget, error) {
	var result []model.SLAStatusTarget
	for _, t := range m.targets {
		if t.ProjectID == projectID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *inMemorySLARepo) ListTargetsByProjectAndType(_ context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) ([]model.SLAStatusTarget, error) {
	var result []model.SLAStatusTarget
	for _, t := range m.targets {
		if t.ProjectID == projectID && t.WorkItemType == workItemType && t.WorkflowID == workflowID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *inMemorySLARepo) GetTarget(_ context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID, statusName string) (*model.SLAStatusTarget, error) {
	for _, t := range m.targets {
		if t.ProjectID == projectID && t.WorkItemType == workItemType && t.WorkflowID == workflowID && t.StatusName == statusName {
			return t, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *inMemorySLARepo) BulkUpsertTargets(_ context.Context, targets []model.SLAStatusTarget) ([]model.SLAStatusTarget, error) {
	now := time.Now()
	result := make([]model.SLAStatusTarget, len(targets))
	for i, t := range targets {
		t.CreatedAt = now
		t.UpdatedAt = now
		m.targets[t.ID] = &t
		result[i] = t
	}
	return result, nil
}

func (m *inMemorySLARepo) DeleteTarget(_ context.Context, id uuid.UUID) error {
	if _, ok := m.targets[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.targets, id)
	return nil
}

func (m *inMemorySLARepo) DeleteTargetsByTypeAndWorkflow(_ context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) error {
	for id, t := range m.targets {
		if t.ProjectID == projectID && t.WorkItemType == workItemType && t.WorkflowID == workflowID {
			delete(m.targets, id)
		}
	}
	return nil
}

func (m *inMemorySLARepo) InitElapsedOnCreate(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *inMemorySLARepo) UpsertElapsedOnEnter(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *inMemorySLARepo) UpdateElapsedOnLeave(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *inMemorySLARepo) GetElapsed(_ context.Context, _ uuid.UUID, _ string) (*model.SLAElapsed, error) {
	return nil, model.ErrNotFound
}
func (m *inMemorySLARepo) ListElapsedByWorkItemIDs(_ context.Context, _ []uuid.UUID) ([]model.SLAElapsed, error) {
	return nil, nil
}

func slaTestSetup(t *testing.T) (*SLAHandler, *model.AuthInfo, string, *model.Workflow) {
	t.Helper()

	slaRepo := newInMemorySLARepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	workflowRepo := newMockWorkflowRepo()
	svc := service.NewSLAService(slaRepo, projectRepo, memberRepo, workflowRepo)
	h := NewSLAHandler(svc)

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

	// Create a workflow
	wf := &model.Workflow{
		ID:   uuid.New(),
		Name: "Test Workflow",
		Statuses: []model.WorkflowStatus{
			{Name: "Open", Category: model.CategoryTodo},
			{Name: "In Progress", Category: model.CategoryInProgress},
			{Name: "Done", Category: model.CategoryDone},
		},
	}
	workflowRepo.Create(context.Background(), wf)

	return h, info, "TEST", wf
}

func TestSLAHandler_BulkUpsert(t *testing.T) {
	h, info, projectKey, wf := slaTestSetup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"work_item_type": "task",
		"workflow_id":    wf.ID.String(),
		"targets": []map[string]interface{}{
			{"status_name": "Open", "target_seconds": 3600, "calendar_mode": "24x7"},
			{"status_name": "In Progress", "target_seconds": 7200, "calendar_mode": "business_hours"},
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectKey+"/sla-targets", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.BulkUpsert(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if len(data) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(data))
	}
}

func TestSLAHandler_List(t *testing.T) {
	h, info, projectKey, wf := slaTestSetup(t)

	// First create some targets
	body, _ := json.Marshal(map[string]interface{}{
		"work_item_type": "task",
		"workflow_id":    wf.ID.String(),
		"targets": []map[string]interface{}{
			{"status_name": "Open", "target_seconds": 3600, "calendar_mode": "24x7"},
		},
	})

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectKey+"/sla-targets", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(putReq.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	putReq = putReq.WithContext(ctx)
	w := httptest.NewRecorder()
	h.BulkUpsert(w, putReq)
	if w.Code != http.StatusOK {
		t.Fatalf("setup: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Now list
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectKey+"/sla-targets", nil)
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("projectKey", projectKey)
	ctx2 := context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx2)
	ctx2 = model.ContextWithAuthInfo(ctx2, info)
	getReq = getReq.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	h.List(w2, getReq)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w2.Body.Bytes(), &resp)
	var data []map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 target, got %d", len(data))
	}
}

func TestSLAHandler_Delete(t *testing.T) {
	h, info, projectKey, wf := slaTestSetup(t)

	// Create a target
	body, _ := json.Marshal(map[string]interface{}{
		"work_item_type": "task",
		"workflow_id":    wf.ID.String(),
		"targets": []map[string]interface{}{
			{"status_name": "Open", "target_seconds": 3600, "calendar_mode": "24x7"},
		},
	})

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectKey+"/sla-targets", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(putReq.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	putReq = putReq.WithContext(ctx)
	w := httptest.NewRecorder()
	h.BulkUpsert(w, putReq)

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	targetID := data[0]["id"].(string)

	// Delete it
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectKey+"/sla-targets/"+targetID, nil)
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("projectKey", projectKey)
	rctx2.URLParams.Add("targetId", targetID)
	ctx2 := context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx2)
	ctx2 = model.ContextWithAuthInfo(ctx2, info)
	delReq = delReq.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	h.Delete(w2, delReq)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestSLAHandler_BulkUpsert_TerminalStatus(t *testing.T) {
	h, info, projectKey, wf := slaTestSetup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"work_item_type": "task",
		"workflow_id":    wf.ID.String(),
		"targets": []map[string]interface{}{
			{"status_name": "Done", "target_seconds": 3600, "calendar_mode": "24x7"},
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectKey+"/sla-targets", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.BulkUpsert(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for terminal status, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSLAHandler_BulkUpsert_Unauthorized(t *testing.T) {
	h, _, projectKey, wf := slaTestSetup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"work_item_type": "task",
		"workflow_id":    wf.ID.String(),
		"targets": []map[string]interface{}{
			{"status_name": "Open", "target_seconds": 3600, "calendar_mode": "24x7"},
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectKey+"/sla-targets", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	// No auth info
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.BulkUpsert(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSLAHandler_BulkUpsert_InvalidWorkflowID(t *testing.T) {
	h, info, projectKey, _ := slaTestSetup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"work_item_type": "task",
		"workflow_id":    "not-a-uuid",
		"targets": []map[string]interface{}{
			{"status_name": "Open", "target_seconds": 3600, "calendar_mode": "24x7"},
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectKey+"/sla-targets", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = model.ContextWithAuthInfo(ctx, info)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.BulkUpsert(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
