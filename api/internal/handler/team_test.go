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

// --- Mock team repository ---

type mockTeamRepo struct {
	teams   map[uuid.UUID]*model.Team
	members map[string]*model.TeamMember // key: teamID:userID
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{
		teams:   make(map[uuid.UUID]*model.Team),
		members: make(map[string]*model.TeamMember),
	}
}

func teamMemberKey(teamID, userID uuid.UUID) string {
	return teamID.String() + ":" + userID.String()
}

func (m *mockTeamRepo) Create(_ context.Context, t *model.Team) error {
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	m.teams[t.ID] = t
	return nil
}

func (m *mockTeamRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t, nil
}

func (m *mockTeamRepo) List(_ context.Context, projectID uuid.UUID) ([]model.Team, error) {
	var result []model.Team
	for _, t := range m.teams {
		if t.ProjectID == projectID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTeamRepo) Update(_ context.Context, t *model.Team) error {
	if _, ok := m.teams[t.ID]; !ok {
		return model.ErrNotFound
	}
	t.UpdatedAt = time.Now()
	m.teams[t.ID] = t
	return nil
}

func (m *mockTeamRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.teams[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.teams, id)
	return nil
}

func (m *mockTeamRepo) AddMember(_ context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	key := teamMemberKey(teamID, userID)
	if _, exists := m.members[key]; exists {
		return nil, model.ErrAlreadyExists
	}
	member := &model.TeamMember{
		ID:        uuid.New(),
		TeamID:    teamID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	m.members[key] = member
	return member, nil
}

func (m *mockTeamRepo) RemoveMember(_ context.Context, teamID, userID uuid.UUID) error {
	key := teamMemberKey(teamID, userID)
	if _, ok := m.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.members, key)
	return nil
}

func (m *mockTeamRepo) ListMembers(_ context.Context, teamID uuid.UUID) ([]model.TeamMemberWithUser, error) {
	var result []model.TeamMemberWithUser
	for _, member := range m.members {
		if member.TeamID == teamID {
			result = append(result, model.TeamMemberWithUser{
				TeamMember:  *member,
				Email:       "user@example.com",
				DisplayName: "Test User",
			})
		}
	}
	return result, nil
}

// --- Test helpers ---

func teamTestSetup(t *testing.T) (*TeamHandler, *model.AuthInfo, string) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	teamRepo := newMockTeamRepo()
	svc := service.NewTeamService(teamRepo, projectRepo, memberRepo)
	h := NewTeamHandler(svc)

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

// --- Tests ---

func TestTeamHandler_Create(t *testing.T) {
	h, info, projectKey := teamTestSetup(t)

	body := `{"name":"Engineering"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/teams", bytes.NewBufferString(body))
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
	if data["name"] != "Engineering" {
		t.Fatalf("expected name 'Engineering', got %v", data["name"])
	}
}

func TestTeamHandler_Create_InvalidBody(t *testing.T) {
	h, info, projectKey := teamTestSetup(t)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/teams", bytes.NewBufferString(body))
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

func TestTeamHandler_List(t *testing.T) {
	h, info, projectKey := teamTestSetup(t)

	// Create a team first
	body := `{"name":"Design"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/teams", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()
	h.Create(w, req)

	// List teams
	req = httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/teams", nil)
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
		t.Fatalf("expected 1 team, got %d", len(data))
	}
}

func TestTeamHandler_Get(t *testing.T) {
	h, info, projectKey := teamTestSetup(t)

	// Create a team
	body := `{"name":"Platform"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/teams", bytes.NewBufferString(body))
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
	teamID := createdData["id"].(string)

	// Get team
	req = httptest.NewRequest(http.MethodGet, "/api/v1/default/projects/"+projectKey+"/teams/"+teamID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTeamHandler_Update(t *testing.T) {
	h, info, projectKey := teamTestSetup(t)

	// Create a team
	body := `{"name":"Original"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/teams", bytes.NewBufferString(body))
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
	teamID := createdData["id"].(string)

	// Update team
	updateBody := `{"name":"Updated"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/default/projects/"+projectKey+"/teams/"+teamID, bytes.NewBufferString(updateBody))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

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

func TestTeamHandler_Delete(t *testing.T) {
	h, info, projectKey := teamTestSetup(t)

	// Create a team
	body := `{"name":"ToDelete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects/"+projectKey+"/teams", bytes.NewBufferString(body))
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
	teamID := createdData["id"].(string)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/default/projects/"+projectKey+"/teams/"+teamID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w = httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
