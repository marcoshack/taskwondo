package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
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

type mockSearchProjectRepo struct{}

func (m *mockSearchProjectRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]model.Project, error) {
	return nil, nil
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

func TestSearchHandler_MissingQuery(t *testing.T) {
	svc := service.NewSearchService(
		&service.EmbeddingService{},
		&mockSearchEmbeddingRepo{},
		&mockSearchProjectRepo{},
		&mockSearchSettingsRepo{settings: map[string]*model.SystemSetting{}},
	)
	h := NewSearchHandler(svc)

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

func TestSearchHandler_FeatureDisabled(t *testing.T) {
	svc := service.NewSearchService(
		&service.EmbeddingService{},
		&mockSearchEmbeddingRepo{},
		&mockSearchProjectRepo{},
		&mockSearchSettingsRepo{settings: map[string]*model.SystemSetting{}},
	)
	h := NewSearchHandler(svc)

	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	req = req.WithContext(model.ContextWithAuthInfo(req.Context(), &model.AuthInfo{
		UserID:     uuid.New(),
		GlobalRole: model.RoleUser,
	}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (feature disabled), got %d", w.Code)
	}
}

func TestSearchHandler_Unauthenticated(t *testing.T) {
	svc := service.NewSearchService(
		&service.EmbeddingService{},
		&mockSearchEmbeddingRepo{},
		&mockSearchProjectRepo{},
		&mockSearchSettingsRepo{settings: map[string]*model.SystemSetting{}},
	)
	h := NewSearchHandler(svc)

	r := chi.NewRouter()
	r.Get("/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	// No auth info in context
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
