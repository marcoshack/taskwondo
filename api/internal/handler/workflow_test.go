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

	"github.com/marcoshack/trackforge/internal/model"
	"github.com/marcoshack/trackforge/internal/service"
)

func workflowTestSetup(t *testing.T) (*WorkflowHandler, *service.WorkflowService, *model.AuthInfo) {
	t.Helper()
	repo := newMockWorkflowRepo()
	svc := service.NewWorkflowService(repo)
	h := NewWorkflowHandler(svc)

	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}

	return h, svc, info
}

func TestWorkflowHandler_List(t *testing.T) {
	h, svc, info := workflowTestSetup(t)

	// Seed workflows so there's data
	if err := svc.SeedDefaultWorkflows(context.Background()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)

	var workflows []json.RawMessage
	json.Unmarshal(resp["data"], &workflows)
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}
}

func TestWorkflowHandler_Create(t *testing.T) {
	h, _, info := workflowTestSetup(t)

	body := `{
		"name": "Custom WF",
		"statuses": [
			{"name": "open", "display_name": "Open", "category": "todo", "position": 0},
			{"name": "done", "display_name": "Done", "category": "done", "position": 1}
		],
		"transitions": [
			{"from_status": "open", "to_status": "done", "name": "Complete"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkflowHandler_Create_MissingName(t *testing.T) {
	h, _, info := workflowTestSetup(t)

	body := `{
		"name": "",
		"statuses": [
			{"name": "open", "display_name": "Open", "category": "todo", "position": 0}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkflowHandler_Get(t *testing.T) {
	h, svc, info := workflowTestSetup(t)

	created, err := svc.Create(context.Background(), service.CreateWorkflowInput{
		Name: "Test WF",
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "done"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Get("/api/v1/workflows/{workflowId}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+created.ID.String(), nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkflowHandler_Get_NotFound(t *testing.T) {
	h, _, info := workflowTestSetup(t)

	r := chi.NewRouter()
	r.Get("/api/v1/workflows/{workflowId}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+uuid.New().String(), nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkflowHandler_Update(t *testing.T) {
	h, svc, info := workflowTestSetup(t)

	created, err := svc.Create(context.Background(), service.CreateWorkflowInput{
		Name: "Old Name",
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"name": "New Name"}`
	r := chi.NewRouter()
	r.Patch("/api/v1/workflows/{workflowId}", h.Update)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workflows/"+created.ID.String(), bytes.NewBufferString(body))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)

	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if data["name"] != "New Name" {
		t.Fatalf("expected name 'New Name', got %v", data["name"])
	}
}

func TestWorkflowHandler_ListTransitions(t *testing.T) {
	h, svc, info := workflowTestSetup(t)

	created, err := svc.Create(context.Background(), service.CreateWorkflowInput{
		Name: "Test",
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "wip", DisplayName: "WIP", Category: model.CategoryInProgress, Position: 1},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 2},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "wip"},
			{FromStatus: "wip", ToStatus: "done"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Get("/api/v1/workflows/{workflowId}/transitions", h.ListTransitions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+created.ID.String()+"/transitions", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)

	var data map[string][]json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data["open"]) != 1 {
		t.Fatalf("expected 1 transition from 'open', got %d", len(data["open"]))
	}
	if len(data["wip"]) != 1 {
		t.Fatalf("expected 1 transition from 'wip', got %d", len(data["wip"]))
	}
}

func TestWorkflowHandler_Unauthenticated(t *testing.T) {
	h, _, _ := workflowTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	// No auth info in context
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
