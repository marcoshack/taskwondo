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

func milestoneTestSetup(t *testing.T) (*MilestoneHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	milestoneRepo := newMockMilestoneRepo()
	svc := service.NewMilestoneService(milestoneRepo, projectRepo, memberRepo)
	h := NewMilestoneHandler(svc)

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

func TestMilestoneHandler_Create(t *testing.T) {
	h, info, projectKey := milestoneTestSetup(t)

	body := `{"name":"v1.0","due_date":"2026-06-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/milestones", bytes.NewBufferString(body))
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
	if data["name"] != "v1.0" {
		t.Fatalf("expected name 'v1.0', got %v", data["name"])
	}
	if data["status"] != "open" {
		t.Fatalf("expected status 'open', got %v", data["status"])
	}
	if data["due_date"] != "2026-06-01" {
		t.Fatalf("expected due_date '2026-06-01', got %v", data["due_date"])
	}
}

func TestMilestoneHandler_Create_InvalidDate(t *testing.T) {
	h, info, projectKey := milestoneTestSetup(t)

	body := `{"name":"v1.0","due_date":"not-a-date"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/milestones", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMilestoneHandler_List(t *testing.T) {
	h, info, projectKey := milestoneTestSetup(t)

	// Create two milestones
	for _, name := range []string{"v1.0", "v2.0"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/milestones", bytes.NewBufferString(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("projectKey", projectKey)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
		w := httptest.NewRecorder()
		h.Create(w, req)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectKey+"/milestones", nil)
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
	if len(data) != 2 {
		t.Fatalf("expected 2 milestones, got %d", len(data))
	}
}

func TestMilestoneHandler_Get(t *testing.T) {
	h, info, projectKey := milestoneTestSetup(t)

	// Create
	body := `{"name":"v1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/milestones", bytes.NewBufferString(body))
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
	milestoneID := createdData["id"].(string)

	// Get
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectKey+"/milestones/"+milestoneID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("milestoneId", milestoneID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMilestoneHandler_Update(t *testing.T) {
	h, info, projectKey := milestoneTestSetup(t)

	// Create
	body := `{"name":"v1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/milestones", bytes.NewBufferString(body))
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
	milestoneID := createdData["id"].(string)

	// Update
	updateBody := `{"name":"v1.1","status":"closed"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/projects/"+projectKey+"/milestones/"+milestoneID, bytes.NewBufferString(updateBody))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("milestoneId", milestoneID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updateResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	var updatedData map[string]interface{}
	json.Unmarshal(updateResp["data"], &updatedData)
	if updatedData["name"] != "v1.1" {
		t.Fatalf("expected name 'v1.1', got %v", updatedData["name"])
	}
	if updatedData["status"] != "closed" {
		t.Fatalf("expected status 'closed', got %v", updatedData["status"])
	}
}

func TestMilestoneHandler_Delete(t *testing.T) {
	h, info, projectKey := milestoneTestSetup(t)

	// Create
	body := `{"name":"v1.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/milestones", bytes.NewBufferString(body))
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
	milestoneID := createdData["id"].(string)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectKey+"/milestones/"+milestoneID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("milestoneId", milestoneID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMilestoneHandler_Unauthenticated(t *testing.T) {
	h, _, projectKey := milestoneTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectKey+"/milestones", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
