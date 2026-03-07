package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- mock embedding repo ---

type mockSearchEmbedding struct {
	results []model.SearchResult
	err     error
}

func (m *mockSearchEmbedding) SearchByVector(_ context.Context, _ []float32, _ *model.SearchFilter, _ []uuid.UUID) ([]model.SearchResult, error) {
	return m.results, m.err
}

// --- mock work item repo ---

type mockSearchWorkItems struct {
	results []model.SearchResult
	err     error
}

func (m *mockSearchWorkItems) SearchFTS(_ context.Context, _ string, _ []uuid.UUID, _ int) ([]model.SearchResult, error) {
	return m.results, m.err
}

// --- mock search projects ---

type mockSearchProjects struct {
	projects []model.Project
	err      error
}

func (m *mockSearchProjects) ListByUser(_ context.Context, _ uuid.UUID) ([]model.Project, error) {
	return m.projects, m.err
}

// --- mock search settings ---

type mockSearchSettings struct {
	settings map[string]*model.SystemSetting
}

func (m *mockSearchSettings) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}

func TestSearchService_FeatureDisabled(t *testing.T) {
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		&mockSearchProjects{},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	_, err := svc.Search(context.Background(), info, &model.SearchFilter{Query: "test"})
	if err != model.ErrFeatureDisabled {
		t.Fatalf("expected ErrFeatureDisabled, got %v", err)
	}
}

func TestSearchService_SemanticEnabled(t *testing.T) {
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		&mockSearchProjects{
			projects: []model.Project{{ID: uuid.New()}},
		},
		&mockSearchSettings{
			settings: map[string]*model.SystemSetting{
				model.SettingFeatureSemanticSearch: {Key: model.SettingFeatureSemanticSearch, Value: []byte("true")},
			},
		},
	)

	enabled := svc.SemanticEnabled(context.Background())
	if !enabled {
		t.Fatal("expected SemanticEnabled to return true")
	}
}

func TestSearchService_SemanticDisabled(t *testing.T) {
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		&mockSearchProjects{},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	enabled := svc.SemanticEnabled(context.Background())
	if enabled {
		t.Fatal("expected SemanticEnabled to return false")
	}
}

func TestSearchService_SearchFTS(t *testing.T) {
	projectID := uuid.New()
	itemID := uuid.New()
	num := 42

	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{
			results: []model.SearchResult{
				{EntityType: "work_item", EntityID: itemID, ProjectID: &projectID, Content: "[task] Fix login", ProjectKey: "TF", ItemNumber: &num},
			},
		},
		&mockSearchProjects{
			projects: []model.Project{{ID: projectID}},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	results, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{Query: "fix login", Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].EntityID != itemID {
		t.Errorf("expected entity ID %s, got %s", itemID, results[0].EntityID)
	}
}

func TestSearchService_ProjectIDFiltering(t *testing.T) {
	allowedProject := uuid.New()
	disallowedProject := uuid.New()

	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{results: []model.SearchResult{}},
		&mockSearchWorkItems{results: []model.SearchResult{}},
		&mockSearchProjects{
			projects: []model.Project{{ID: allowedProject}},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	filter := &model.SearchFilter{
		Query:      "test",
		ProjectIDs: []uuid.UUID{allowedProject, disallowedProject},
	}

	results, err := svc.SearchFTS(context.Background(), info, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}
