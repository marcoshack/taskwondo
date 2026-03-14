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

type mockSearchEmbeddingRepo struct{}

func (m *mockSearchEmbeddingRepo) SearchByVector(_ context.Context, _ []float32, _ *model.SearchFilter, _ []uuid.UUID) ([]model.SearchResult, error) {
	return nil, nil
}

type mockSearchWorkItemRepo struct {
	results []model.SearchResult
	err     error
}

func (m *mockSearchWorkItemRepo) SearchFTS(_ context.Context, _ string, _ []uuid.UUID, _ int) ([]model.SearchResult, error) {
	return m.results, m.err
}

type mockSearchProjectRepo struct{}

func (m *mockSearchProjectRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]model.Project, error) {
	return []model.Project{{ID: uuid.New()}}, nil
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
		&mockSearchProjectRepo{},
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
