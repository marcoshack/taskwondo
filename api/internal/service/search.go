package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// SearchEmbeddingRepository is the minimal interface for search operations.
type SearchEmbeddingRepository interface {
	SearchByVector(ctx context.Context, vector []float32, filter *model.SearchFilter, projectIDs []uuid.UUID) ([]model.SearchResult, error)
}

// SearchProjectRepository is the minimal interface for listing user's projects.
type SearchProjectRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Project, error)
}

// SearchSettingsRepository is the minimal interface for checking feature flags.
type SearchSettingsRepository interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
}

// SearchService handles RBAC-filtered semantic search.
type SearchService struct {
	embedding  *EmbeddingService
	embeddings SearchEmbeddingRepository
	projects   SearchProjectRepository
	settings   SearchSettingsRepository
}

// NewSearchService creates a new SearchService.
func NewSearchService(
	embedding *EmbeddingService,
	embeddings SearchEmbeddingRepository,
	projects SearchProjectRepository,
	settings SearchSettingsRepository,
) *SearchService {
	return &SearchService{
		embedding:  embedding,
		embeddings: embeddings,
		projects:   projects,
		settings:   settings,
	}
}

// Search performs a semantic search with RBAC filtering.
func (s *SearchService) Search(ctx context.Context, info *model.AuthInfo, filter *model.SearchFilter) ([]model.SearchResult, error) {
	// Check feature flag
	if !s.isEnabled(ctx) {
		return nil, model.ErrFeatureDisabled
	}

	// Resolve user's project memberships for RBAC
	userProjects, err := s.projects.ListByUser(ctx, info.UserID)
	if err != nil {
		return nil, fmt.Errorf("listing user projects: %w", err)
	}

	projectIDs := make([]uuid.UUID, len(userProjects))
	for i, p := range userProjects {
		projectIDs[i] = p.ID
	}

	// If caller specified project IDs, intersect with user's projects
	if len(filter.ProjectIDs) > 0 {
		allowed := make(map[uuid.UUID]bool)
		for _, pid := range projectIDs {
			allowed[pid] = true
		}
		var filtered []uuid.UUID
		for _, pid := range filter.ProjectIDs {
			if allowed[pid] {
				filtered = append(filtered, pid)
			}
		}
		projectIDs = filtered
	}

	// Embed the query text
	vector, err := s.embedding.Embed(ctx, filter.Query)
	if err != nil {
		return nil, err
	}

	// Search by vector with RBAC filter
	results, err := s.embeddings.SearchByVector(ctx, vector, filter, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("searching embeddings: %w", err)
	}

	return results, nil
}

func (s *SearchService) isEnabled(ctx context.Context) bool {
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
