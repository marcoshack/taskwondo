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

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// --- Mock oncall rotation repository ---

type mockOncallRotationRepo struct {
	rotations map[uuid.UUID]*model.OncallRotation
	byTeam    map[uuid.UUID]*model.OncallRotation
	members   map[uuid.UUID][]model.OncallRotationMember
	history   map[uuid.UUID][]model.OncallRotationHistory
}

func newMockOncallRotationRepo() *mockOncallRotationRepo {
	return &mockOncallRotationRepo{
		rotations: make(map[uuid.UUID]*model.OncallRotation),
		byTeam:    make(map[uuid.UUID]*model.OncallRotation),
		members:   make(map[uuid.UUID][]model.OncallRotationMember),
		history:   make(map[uuid.UUID][]model.OncallRotationHistory),
	}
}

func (m *mockOncallRotationRepo) Create(_ context.Context, rot *model.OncallRotation) (*model.OncallRotation, error) {
	now := time.Now()
	rot.CreatedAt = now
	rot.UpdatedAt = now
	m.rotations[rot.ID] = rot
	m.byTeam[rot.TeamID] = rot
	return rot, nil
}

func (m *mockOncallRotationRepo) GetByTeamID(_ context.Context, teamID uuid.UUID) (*model.OncallRotation, error) {
	rot, ok := m.byTeam[teamID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return rot, nil
}

func (m *mockOncallRotationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.OncallRotation, error) {
	rot, ok := m.rotations[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return rot, nil
}

func (m *mockOncallRotationRepo) Update(_ context.Context, rot *model.OncallRotation) (*model.OncallRotation, error) {
	if _, ok := m.rotations[rot.ID]; !ok {
		return nil, model.ErrNotFound
	}
	rot.UpdatedAt = time.Now()
	m.rotations[rot.ID] = rot
	m.byTeam[rot.TeamID] = rot
	return rot, nil
}

func (m *mockOncallRotationRepo) Delete(_ context.Context, teamID uuid.UUID) error {
	rot, ok := m.byTeam[teamID]
	if !ok {
		return model.ErrNotFound
	}
	delete(m.rotations, rot.ID)
	delete(m.byTeam, teamID)
	return nil
}

func (m *mockOncallRotationRepo) SetMembers(_ context.Context, rotationID uuid.UUID, members []model.OncallRotationMember) error {
	m.members[rotationID] = members
	return nil
}

func (m *mockOncallRotationRepo) ListMembers(_ context.Context, rotationID uuid.UUID) ([]model.OncallRotationMemberWithUser, error) {
	members := m.members[rotationID]
	result := make([]model.OncallRotationMemberWithUser, len(members))
	for i, member := range members {
		result[i] = model.OncallRotationMemberWithUser{
			OncallRotationMember: member,
			Email:                "user@example.com",
			DisplayName:          "Test User",
		}
	}
	return result, nil
}

func (m *mockOncallRotationRepo) CreateHistory(_ context.Context, h *model.OncallRotationHistory) error {
	m.history[h.RotationID] = append(m.history[h.RotationID], *h)
	return nil
}

func (m *mockOncallRotationRepo) EndCurrentHistory(_ context.Context, rotationID uuid.UUID) error {
	entries := m.history[rotationID]
	for i := range entries {
		if entries[i].EndedAt == nil {
			now := time.Now()
			entries[i].EndedAt = &now
		}
	}
	m.history[rotationID] = entries
	return nil
}

func (m *mockOncallRotationRepo) ListHistory(_ context.Context, rotationID uuid.UUID, limit, offset int) ([]model.OncallRotationHistoryWithUser, int, error) {
	entries := m.history[rotationID]
	total := len(entries)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	result := make([]model.OncallRotationHistoryWithUser, 0, end-offset)
	for _, h := range entries[offset:end] {
		result = append(result, model.OncallRotationHistoryWithUser{
			OncallRotationHistory: h,
			DisplayName:           "Test User",
		})
	}
	return result, total, nil
}

func (m *mockOncallRotationRepo) ListDueRotations(_ context.Context) ([]model.OncallRotation, error) {
	return nil, nil
}

// --- Test helpers ---

func oncallTestSetup(t *testing.T) (*OncallHandler, *model.AuthInfo, string, uuid.UUID, []uuid.UUID) {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	teamRepo := newMockTeamRepo()
	oncallRepo := newMockOncallRotationRepo()
	svc := service.NewOncallService(oncallRepo, teamRepo, projectRepo, memberRepo)
	h := NewOncallHandler(svc)

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

	team := &model.Team{ID: uuid.New(), ProjectID: project.ID, Name: "Engineering"}
	teamRepo.Create(context.Background(), team)

	user1 := info.UserID
	user2 := uuid.New()
	teamRepo.AddMember(context.Background(), team.ID, user1)
	teamRepo.AddMember(context.Background(), team.ID, user2)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    user2,
		Role:      model.ProjectRoleMember,
	})

	return h, info, "TEST", team.ID, []uuid.UUID{user1, user2}
}

func oncallCreateRequest(t *testing.T, h *OncallHandler, info *model.AuthInfo, projectKey string, teamID uuid.UUID, memberIDs []uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	ids := make([]string, len(memberIDs))
	for i, id := range memberIDs {
		ids[i] = id.String()
	}
	bodyMap := map[string]interface{}{
		"period_days":   7,
		"rotation_time": "12:00:00",
		"timezone":      "UTC",
		"start_date":    "2026-04-01",
		"member_ids":    ids,
	}
	bodyBytes, _ := json.Marshal(bodyMap)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)
	return w
}

// --- Tests ---

func TestOncallHandler_Create(t *testing.T) {
	h, info, projectKey, teamID, memberIDs := oncallTestSetup(t)

	w := oncallCreateRequest(t, h, info, projectKey, teamID, memberIDs)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if data["period_days"] != float64(7) {
		t.Fatalf("expected period_days 7, got %v", data["period_days"])
	}
}

func TestOncallHandler_Create_InvalidBody(t *testing.T) {
	h, info, projectKey, teamID, _ := oncallTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid}"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOncallHandler_Get(t *testing.T) {
	h, info, projectKey, teamID, memberIDs := oncallTestSetup(t)

	// Create first
	oncallCreateRequest(t, h, info, projectKey, teamID, memberIDs)

	// Then get
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOncallHandler_Get_NotFound(t *testing.T) {
	h, info, projectKey, teamID, _ := oncallTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestOncallHandler_Update(t *testing.T) {
	h, info, projectKey, teamID, memberIDs := oncallTestSetup(t)

	// Create first
	oncallCreateRequest(t, h, info, projectKey, teamID, memberIDs)

	// Then update
	updateBody := `{"period_days": 14}`
	req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(updateBody))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
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
	if data["period_days"] != float64(14) {
		t.Fatalf("expected period_days 14, got %v", data["period_days"])
	}
}

func TestOncallHandler_Delete(t *testing.T) {
	h, info, projectKey, teamID, memberIDs := oncallTestSetup(t)

	// Create first
	oncallCreateRequest(t, h, info, projectKey, teamID, memberIDs)

	// Then delete
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOncallHandler_ListHistory(t *testing.T) {
	h, info, projectKey, teamID, memberIDs := oncallTestSetup(t)

	// Create first
	oncallCreateRequest(t, h, info, projectKey, teamID, memberIDs)

	// List history
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?limit=10&offset=0"), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", teamID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.ListHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oncallHistoryListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
}

func TestOncallHandler_InvalidTeamID(t *testing.T) {
	h, info, projectKey, _, _ := oncallTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("teamId", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
