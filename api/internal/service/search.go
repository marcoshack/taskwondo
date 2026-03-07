package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// SearchEmbeddingRepository is the minimal interface for semantic search operations.
type SearchEmbeddingRepository interface {
	SearchByVector(ctx context.Context, vector []float32, filter *model.SearchFilter, projectIDs []uuid.UUID) ([]model.SearchResult, error)
}

// SearchWorkItemRepository is the minimal interface for FTS search operations.
type SearchWorkItemRepository interface {
	SearchFTS(ctx context.Context, query string, projectIDs []uuid.UUID, limit int) ([]model.SearchResult, error)
}

// SearchProjectRepository is the minimal interface for listing user's projects.
type SearchProjectRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Project, error)
}

// SearchSettingsRepository is the minimal interface for checking feature flags.
type SearchSettingsRepository interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
}

// SearchService handles RBAC-filtered search (both FTS and semantic).
type SearchService struct {
	embedding  *EmbeddingService
	embeddings SearchEmbeddingRepository
	workItems  SearchWorkItemRepository
	projects   SearchProjectRepository
	settings   SearchSettingsRepository
}

// NewSearchService creates a new SearchService.
func NewSearchService(
	embedding *EmbeddingService,
	embeddings SearchEmbeddingRepository,
	workItems SearchWorkItemRepository,
	projects SearchProjectRepository,
	settings SearchSettingsRepository,
) *SearchService {
	return &SearchService{
		embedding:  embedding,
		embeddings: embeddings,
		workItems:  workItems,
		projects:   projects,
		settings:   settings,
	}
}

// Search performs a semantic search with RBAC filtering (legacy method for backward compatibility).
func (s *SearchService) Search(ctx context.Context, info *model.AuthInfo, filter *model.SearchFilter) ([]model.SearchResult, error) {
	if !s.isSemanticEnabled(ctx) {
		return nil, model.ErrFeatureDisabled
	}
	return s.SearchSemantic(ctx, info, filter)
}

// SearchFTS performs a cross-project full-text search with RBAC filtering.
func (s *SearchService) SearchFTS(ctx context.Context, info *model.AuthInfo, filter *model.SearchFilter) ([]model.SearchResult, error) {
	projectIDs, err := s.resolveProjectIDs(ctx, info.UserID, filter.ProjectIDs)
	if err != nil {
		return nil, err
	}
	return s.workItems.SearchFTS(ctx, filter.Query, projectIDs, filter.Limit)
}

// SearchSemantic performs a semantic (vector) search with RBAC filtering.
func (s *SearchService) SearchSemantic(ctx context.Context, info *model.AuthInfo, filter *model.SearchFilter) ([]model.SearchResult, error) {
	projectIDs, err := s.resolveProjectIDs(ctx, info.UserID, filter.ProjectIDs)
	if err != nil {
		return nil, err
	}

	vector, err := s.embedding.Embed(ctx, filter.Query)
	if err != nil {
		return nil, err
	}

	results, err := s.embeddings.SearchByVector(ctx, vector, filter, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("searching embeddings: %w", err)
	}
	return results, nil
}

// SemanticEnabled returns whether semantic search is enabled and available.
func (s *SearchService) SemanticEnabled(ctx context.Context) bool {
	return s.isSemanticEnabled(ctx)
}

// resolveProjectIDs returns the user's accessible project IDs, optionally intersected with filter.ProjectIDs.
func (s *SearchService) resolveProjectIDs(ctx context.Context, userID uuid.UUID, filterProjectIDs []uuid.UUID) ([]uuid.UUID, error) {
	userProjects, err := s.projects.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user projects: %w", err)
	}

	projectIDs := make([]uuid.UUID, len(userProjects))
	for i, p := range userProjects {
		projectIDs[i] = p.ID
	}

	if len(filterProjectIDs) > 0 {
		allowed := make(map[uuid.UUID]bool)
		for _, pid := range projectIDs {
			allowed[pid] = true
		}
		var filtered []uuid.UUID
		for _, pid := range filterProjectIDs {
			if allowed[pid] {
				filtered = append(filtered, pid)
			}
		}
		projectIDs = filtered
	}

	return projectIDs, nil
}

func (s *SearchService) isSemanticEnabled(ctx context.Context) bool {
	setting, err := s.settings.Get(ctx, model.SettingFeatureSemanticSearch)
	if err != nil {
		return false
	}
	var enabled bool
	if err := json.Unmarshal(setting.Value, &enabled); err != nil {
		return false
	}
	return enabled
}
