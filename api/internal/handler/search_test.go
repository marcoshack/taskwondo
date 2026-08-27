package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

type mockSearchEmbeddingRepo struct {
	results    []model.SearchResult
	lastFilter *model.SearchFilter
	lastAccess model.SearchAccess
}

func (m *mockSearchEmbeddingRepo) SearchByVector(_ context.Context, _ []float32, filter *model.SearchFilter, access model.SearchAccess) ([]model.SearchResult, error) {
	m.lastFilter = filter
	m.lastAccess = access
	return m.results, nil
}

type mockSearchWorkItemRepo struct {
	results    []model.SearchResult
	err        error
	lastAccess model.SearchAccess
}

func (m *mockSearchWorkItemRepo) SearchFTS(_ context.Context, _ string, access model.SearchAccess, _ int) ([]model.SearchResult, error) {
	m.lastAccess = access
	return m.results, m.err
}

type mockSearchEntityFTSRepo struct {
	results        []model.SearchResult
	lastProjectIDs []uuid.UUID
}

func (m *mockSearchEntityFTSRepo) SearchFTS(_ context.Context, _ string, projectIDs []uuid.UUID, _ int) ([]model.SearchResult, error) {
	m.lastProjectIDs = projectIDs
	return m.results, nil
}

type mockSearchMemberRepo struct {
	memberships []model.ProjectMemberWithProject
}

func (m *mockSearchMemberRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]model.ProjectMemberWithProject, error) {
	if m.memberships == nil {
		return []model.ProjectMemberWithProject{
			{ProjectMember: model.ProjectMember{ProjectID: uuid.New(), Role: model.ProjectRoleOwner}},
		}, nil
	}
	return m.memberships, nil
}

type mockSearchSettingsRepo struct {
	settings map[string]*model.SystemSetting
}

func (m *mockSearchSettingsRepo) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}

func newTestSearchHandler(workItemResults []model.SearchResult, semanticEnabled bool) *SearchHandler {
	settings := map[string]*model.SystemSetting{}
	if semanticEnabled {
		settings[model.SettingFeatureSemanticSearch] = &model.SystemSetting{
			Key:   model.SettingFeatureSemanticSearch,
			Value: []byte("true"),
		}
	}
	svc := service.NewSearchService(
		&service.EmbeddingService{},
		&mockSearchEmbeddingRepo{},
		&mockSearchWorkItemRepo{results: workItemResults},
		&mockSearchEntityFTSRepo{}, &mockSearchEntityFTSRepo{}, &mockSearchEntityFTSRepo{},
		&mockSearchMemberRepo{},
		&mockSearchSettingsRepo{settings: settings},
	)
	return NewSearchHandler(svc)
}

func TestSearchHandler_MissingQuery(t *testing.T) {
	h := newTestSearchHandler(nil, false)
	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     uuid.New(),
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSearchHandler_Unauthenticated(t *testing.T) {
	h := newTestSearchHandler(nil, false)
	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSearchHandler_JSON_FTSOnly(t *testing.T) {
	itemID := uuid.New()
	projectID := uuid.New()
	num := 42
	results := []model.SearchResult{
		{EntityType: "work_item", EntityID: itemID, ProjectID: &projectID, Score: 0, Content: "[task] Fix login", ProjectKey: "TF", ItemNumber: &num},
	}
	h := newTestSearchHandler(results, false)
	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=login", nil)
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     uuid.New(),
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Query    string          `json:"query"`
			FTS      ftsSection      `json:"fts"`
			Semantic semanticSection `json:"semantic"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Data.FTS.Total != 1 {
		t.Errorf("expected 1 FTS result, got %d", resp.Data.FTS.Total)
	}
	if resp.Data.Semantic.Available {
		t.Errorf("expected semantic.available=false")
	}
	if resp.Data.Semantic.Status != "complete" {
		t.Errorf("expected semantic.status=complete, got %s", resp.Data.Semantic.Status)
	}
}

func TestSearchHandler_SSE_FTSOnly(t *testing.T) {
	itemID := uuid.New()
	projectID := uuid.New()
	num := 1
	results := []model.SearchResult{
		{EntityType: "work_item", EntityID: itemID, ProjectID: &projectID, Score: 0, Content: "[bug] Crash", ProjectKey: "TF", ItemNumber: &num},
	}
	h := newTestSearchHandler(results, false)
	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=crash", nil)
	req.Header.Set("Accept", "text/event-stream")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     uuid.New(),
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %s", contentType)
	}

	// Parse SSE events
	events := parseSSEResponse(t, w.Body.String())
	if len(events) < 2 {
		t.Fatalf("expected at least 2 SSE events (fts, done), got %d", len(events))
	}

	// First event should be fts
	if events[0].name != "fts" {
		t.Errorf("expected first event 'fts', got '%s'", events[0].name)
	}

	var ftsPayload ftsEventPayload
	if err := json.Unmarshal([]byte(events[0].data), &ftsPayload); err != nil {
		t.Fatalf("decoding fts event: %v", err)
	}
	if ftsPayload.FTS.Total != 1 {
		t.Errorf("expected 1 FTS result, got %d", ftsPayload.FTS.Total)
	}
	if ftsPayload.Semantic.Available {
		t.Errorf("expected semantic.available=false")
	}
	if ftsPayload.Semantic.Status != "complete" {
		t.Errorf("expected semantic.status=complete, got %s", ftsPayload.Semantic.Status)
	}

	// Last event should be done
	lastEvent := events[len(events)-1]
	if lastEvent.name != "done" {
		t.Errorf("expected last event 'done', got '%s'", lastEvent.name)
	}
}

func TestSearchHandler_JSON_StatusFields(t *testing.T) {
	itemID := uuid.New()
	projectID := uuid.New()
	num := 10
	results := []model.SearchResult{
		{
			EntityType:     "work_item",
			EntityID:       itemID,
			ProjectID:      &projectID,
			Score:          0,
			Content:        "[task] Completed task",
			ProjectKey:     "TF",
			ItemNumber:     &num,
			Status:         "done",
			StatusCategory: "done",
		},
	}
	h := newTestSearchHandler(results, false)
	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=completed", nil)
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     uuid.New(),
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			FTS struct {
				Results []struct {
					Status         string `json:"status"`
					StatusCategory string `json:"status_category"`
				} `json:"results"`
			} `json:"fts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp.Data.FTS.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Data.FTS.Results))
	}
	if resp.Data.FTS.Results[0].Status != "done" {
		t.Errorf("expected status 'done', got %q", resp.Data.FTS.Results[0].Status)
	}
	if resp.Data.FTS.Results[0].StatusCategory != "done" {
		t.Errorf("expected status_category 'done', got %q", resp.Data.FTS.Results[0].StatusCategory)
	}
}

type sseEvent struct {
	name string
	data string
}

func parseSSEResponse(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	var currentEvent sseEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent.name = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && currentEvent.name != "" {
			events = append(events, currentEvent)
			currentEvent = sseEvent{}
		}
	}
	return events
}

// --- project scoping (TF-432) ---

// scopedSearchMocks bundles the repository mocks a project-scope test needs to
// assert on.
type scopedSearchMocks struct {
	workItems  *mockSearchWorkItemRepo
	teams      *mockSearchEntityFTSRepo
	embeddings *mockSearchEmbeddingRepo
}

// newScopedSearchHandler wires a SearchHandler whose caller holds the given
// memberships, returning the mocks so the test can inspect what the service
// asked each repository for.
func newScopedSearchHandler(memberships []model.ProjectMemberWithProject, embedBaseURL string) (*SearchHandler, *scopedSearchMocks) {
	mocks := &scopedSearchMocks{
		workItems:  &mockSearchWorkItemRepo{},
		teams:      &mockSearchEntityFTSRepo{},
		embeddings: &mockSearchEmbeddingRepo{},
	}
	settings := map[string]*model.SystemSetting{}
	embedding := &service.EmbeddingService{}
	if embedBaseURL != "" {
		embedding = service.NewEmbeddingService(embedBaseURL, "test-model")
		settings[model.SettingFeatureSemanticSearch] = &model.SystemSetting{
			Key:   model.SettingFeatureSemanticSearch,
			Value: []byte("true"),
		}
	}
	svc := service.NewSearchService(
		embedding,
		mocks.embeddings,
		mocks.workItems,
		mocks.teams, &mockSearchEntityFTSRepo{}, &mockSearchEntityFTSRepo{},
		&mockSearchMemberRepo{memberships: memberships},
		&mockSearchSettingsRepo{settings: settings},
	)
	return NewSearchHandler(svc), mocks
}

func scopedMembership(projectID uuid.UUID, key, role string) model.ProjectMemberWithProject {
	return model.ProjectMemberWithProject{
		ProjectMember: model.ProjectMember{ProjectID: projectID, Role: role},
		ProjectKey:    key,
	}
}

// doSearch issues an authenticated GET against the search endpoint.
func doSearch(h *SearchHandler, userID uuid.UUID, target string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     userID,
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// GET /search?q=…&project=TF must narrow every project-scoped source to TF.
func TestSearchHandler_ProjectScope(t *testing.T) {
	tf := uuid.New()
	other := uuid.New()
	userID := uuid.New()

	h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(tf, "TF", model.ProjectRoleMember),
		scopedMembership(other, "OTHER", model.ProjectRoleOwner),
	}, "")

	w := doSearch(h, userID, "/search?q=login&project=TF")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if got := mocks.workItems.lastAccess.FullProjectIDs; len(got) != 1 || got[0] != tf {
		t.Errorf("expected work item search scoped to %s, got %v", tf, got)
	}
	if got := mocks.teams.lastProjectIDs; len(got) != 1 || got[0] != tf {
		t.Errorf("expected team search scoped to %s, got %v", tf, got)
	}
}

// A lowercase key still resolves — the palette hands over whatever was typed.
func TestSearchHandler_ProjectScopeCaseInsensitive(t *testing.T) {
	tf := uuid.New()
	h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(tf, "TF", model.ProjectRoleMember),
	}, "")

	w := doSearch(h, uuid.New(), "/search?q=login&project=tf")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := mocks.workItems.lastAccess.FullProjectIDs; len(got) != 1 || got[0] != tf {
		t.Errorf("expected work item search scoped to %s, got %v", tf, got)
	}
}

// Omitting `project` must preserve the pre-TF-432 behaviour exactly.
func TestSearchHandler_NoProjectScopeSearchesEverything(t *testing.T) {
	tf := uuid.New()
	other := uuid.New()

	h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(tf, "TF", model.ProjectRoleMember),
		scopedMembership(other, "OTHER", model.ProjectRoleOwner),
	}, "")

	w := doSearch(h, uuid.New(), "/search?q=login")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := mocks.workItems.lastAccess.FullProjectIDs; len(got) != 2 {
		t.Errorf("expected both projects searched, got %v", got)
	}
	if got := mocks.teams.lastProjectIDs; len(got) != 2 {
		t.Errorf("expected both projects searched for teams, got %v", got)
	}
}

// An empty `project=` is treated as "no scope" rather than an error, so a
// client that always appends the parameter keeps working.
func TestSearchHandler_EmptyProjectParamIsUnscoped(t *testing.T) {
	tf := uuid.New()
	other := uuid.New()

	h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(tf, "TF", model.ProjectRoleMember),
		scopedMembership(other, "OTHER", model.ProjectRoleOwner),
	}, "")

	w := doSearch(h, uuid.New(), "/search?q=login&project=")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := mocks.workItems.lastAccess.FullProjectIDs; len(got) != 2 {
		t.Errorf("expected both projects searched, got %v", got)
	}
}

// A project the caller is not a member of and a project that does not exist
// must produce byte-identical responses, and must never run a global search.
func TestSearchHandler_ProjectScopeForbidden(t *testing.T) {
	tf := uuid.New()

	for _, key := range []string{"SECRET", "DOESNOTEXIST"} {
		t.Run(key, func(t *testing.T) {
			h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
				scopedMembership(tf, "TF", model.ProjectRoleMember),
			}, "")

			w := doSearch(h, uuid.New(), "/search?q=login&project="+key)
			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
			}

			var resp struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}
			if resp.Error.Code != CodeForbidden {
				t.Errorf("expected code %s, got %s", CodeForbidden, resp.Error.Code)
			}
			if resp.Error.Message != "project not found or not accessible" {
				t.Errorf("unexpected message %q", resp.Error.Message)
			}
			// No fallback to a global search.
			if mocks.workItems.lastAccess.HasAny() {
				t.Errorf("expected no search to run, got access %+v", mocks.workItems.lastAccess)
			}
		})
	}
}

// The scope is rejected before the SSE stream opens, so the client sees a plain
// JSON error rather than a half-open event stream.
func TestSearchHandler_ProjectScopeForbiddenSSE(t *testing.T) {
	h, _ := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(uuid.New(), "TF", model.ProjectRoleMember),
	}, "")

	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=login&project=SECRET", nil)
	req.Header.Set("Accept", "text/event-stream")
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     uuid.New(),
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected a JSON error, got Content-Type %q", ct)
	}
	if strings.Contains(w.Body.String(), "event:") {
		t.Errorf("expected no SSE events, got %q", w.Body.String())
	}
}

// A customer-role caller scoping to their own customer project keeps the
// customer restriction — scoping must not widen access.
func TestSearchHandler_ProjectScopeCustomerRole(t *testing.T) {
	customerProject := uuid.New()
	userID := uuid.New()

	h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(customerProject, "CUST", model.ProjectRoleCustomer),
		scopedMembership(uuid.New(), "OWN", model.ProjectRoleOwner),
	}, "")

	w := doSearch(h, userID, "/search?q=login&project=CUST")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	access := mocks.workItems.lastAccess
	if len(access.FullProjectIDs) != 0 {
		t.Errorf("expected no full-access projects, got %v", access.FullProjectIDs)
	}
	if len(access.CustomerProjectIDs) != 1 || access.CustomerProjectIDs[0] != customerProject {
		t.Errorf("expected customer project %s, got %v", customerProject, access.CustomerProjectIDs)
	}
	if access.UserID != userID {
		t.Errorf("expected user id %s, got %s", userID, access.UserID)
	}
	// Teams are full-access only, so the scoped customer search gets nothing.
	if len(mocks.teams.lastProjectIDs) != 0 {
		t.Errorf("expected no full-access projects for team search, got %v", mocks.teams.lastProjectIDs)
	}
}

// The semantic path carries the scope on the filter while the RBAC access set
// stays the caller's full reach, which is what keeps `project` results global.
func TestSearchHandler_ProjectScopeSemanticCarveOut(t *testing.T) {
	embedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"embeddings":[[0.1,0.2,0.3]]}`))
	}))
	defer embedSrv.Close()

	tf := uuid.New()
	other := uuid.New()

	h, mocks := newScopedSearchHandler([]model.ProjectMemberWithProject{
		scopedMembership(tf, "TF", model.ProjectRoleMember),
		scopedMembership(other, "OTHER", model.ProjectRoleOwner),
	}, embedSrv.URL)

	// A project hit from a *different* project must survive a TF-scoped search.
	mocks.embeddings.results = []model.SearchResult{
		{EntityType: model.EntityTypeProject, EntityID: uuid.New(), ProjectID: &other, ProjectKey: "OTHER", Content: "Project: Other"},
	}

	w := doSearch(h, uuid.New(), "/search?q=other&project=TF")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if mocks.embeddings.lastFilter == nil || mocks.embeddings.lastFilter.ScopeProjectID == nil {
		t.Fatal("expected the project scope to reach the embedding repository")
	}
	if *mocks.embeddings.lastFilter.ScopeProjectID != tf {
		t.Errorf("expected scope %s, got %s", tf, *mocks.embeddings.lastFilter.ScopeProjectID)
	}
	if len(mocks.embeddings.lastAccess.FullProjectIDs) != 2 {
		t.Errorf("expected semantic access to keep both projects, got %v", mocks.embeddings.lastAccess.FullProjectIDs)
	}

	var resp struct {
		Data struct {
			Semantic struct {
				Results []model.SearchResult `json:"results"`
			} `json:"semantic"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data.Semantic.Results) != 1 || resp.Data.Semantic.Results[0].ProjectKey != "OTHER" {
		t.Errorf("expected the out-of-scope project hit to survive, got %+v", resp.Data.Semantic.Results)
	}
}
