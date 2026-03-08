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

func queueTestSetup(t *testing.T) (*QueueHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	queueRepo := newMockQueueRepo()
	svc := service.NewQueueService(queueRepo, projectRepo, memberRepo)
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
