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
	SearchByVector(ctx context.Context, vector []float32, filter *model.SearchFilter, access model.SearchAccess) ([]model.SearchResult, error)
}

// SearchWorkItemRepository is the minimal interface for FTS search operations.
type SearchWorkItemRepository interface {
	SearchFTS(ctx context.Context, query string, access model.SearchAccess, limit int) ([]model.SearchResult, error)
}

// SearchEntityFTSRepository is the interface for FTS search on non-work-item entities (teams, queues, milestones).
type SearchEntityFTSRepository interface {
	SearchFTS(ctx context.Context, query string, fullProjectIDs []uuid.UUID, limit int) ([]model.SearchResult, error)
}

// SearchMemberRepository is the minimal interface for listing a user's project memberships.
type SearchMemberRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.ProjectMemberWithProject, error)
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
	teams      SearchEntityFTSRepository
	queues     SearchEntityFTSRepository
	milestones SearchEntityFTSRepository
	members    SearchMemberRepository
	settings   SearchSettingsRepository
}

// NewSearchService creates a new SearchService.
func NewSearchService(
	embedding *EmbeddingService,
	embeddings SearchEmbeddingRepository,
	workItems SearchWorkItemRepository,
	teams SearchEntityFTSRepository,
	queues SearchEntityFTSRepository,
	milestones SearchEntityFTSRepository,
	members SearchMemberRepository,
	settings SearchSettingsRepository,
) *SearchService {
	return &SearchService{
		embedding:  embedding,
		embeddings: embeddings,
		workItems:  workItems,
		teams:      teams,
		queues:     queues,
		milestones: milestones,
		members:    members,
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
// Queries work items, teams, queues, and milestones and merges results.
func (s *SearchService) SearchFTS(ctx context.Context, info *model.AuthInfo, filter *model.SearchFilter) ([]model.SearchResult, error) {
	if len(filter.Query) > model.MaxSearchQueryLen {
		return nil, fmt.Errorf("search query exceeds %d characters: %w", model.MaxSearchQueryLen, model.ErrValidation)
	}
	access, err := s.resolveAccess(ctx, info.UserID, filter.ProjectIDs)
	if err != nil {
		return nil, err
	}

	// Determine which entity types to search. If no filter, search all.
	typeSet := make(map[string]bool, len(filter.EntityTypes))
	for _, et := range filter.EntityTypes {
		typeSet[et] = true
	}
	searchAll := len(typeSet) == 0

	var results []model.SearchResult

	// Work items (always included unless explicitly filtered out)
	if searchAll || typeSet[model.EntityTypeWorkItem] {
		wiResults, err := s.workItems.SearchFTS(ctx, filter.Query, access, filter.Limit)
		if err != nil {
			return nil, err
		}
		results = append(results, wiResults...)
	}

	// Teams, queues, milestones — only visible to full-access members (not customers)
	entityRepos := []struct {
		entityType string
		repo       SearchEntityFTSRepository
	}{
		{model.EntityTypeTeam, s.teams},
		{model.EntityTypeQueue, s.queues},
		{model.EntityTypeMilestone, s.milestones},
	}

	for _, er := range entityRepos {
		if !searchAll && !typeSet[er.entityType] {
			continue
		}
		entityResults, err := er.repo.SearchFTS(ctx, filter.Query, access.FullProjectIDs, filter.Limit)
		if err != nil {
			return nil, err
		}
		results = append(results, entityResults...)
	}

	return results, nil
}

// SearchSemantic performs a semantic (vector) search with RBAC filtering.
func (s *SearchService) SearchSemantic(ctx context.Context, info *model.AuthInfo, filter *model.SearchFilter) ([]model.SearchResult, error) {
	if len(filter.Query) > model.MaxSearchQueryLen {
		return nil, fmt.Errorf("search query exceeds %d characters: %w", model.MaxSearchQueryLen, model.ErrValidation)
	}
	access, err := s.resolveAccess(ctx, info.UserID, filter.ProjectIDs)
	if err != nil {
		return nil, err
	}

	vector, err := s.embedding.Embed(ctx, filter.Query)
	if err != nil {
		return nil, err
	}

	results, err := s.embeddings.SearchByVector(ctx, vector, filter, access)
	if err != nil {
		return nil, fmt.Errorf("searching embeddings: %w", err)
	}
	return results, nil
}

// SemanticEnabled returns whether semantic search is enabled and available.
func (s *SearchService) SemanticEnabled(ctx context.Context) bool {
	return s.isSemanticEnabled(ctx)
}

// resolveAccess returns the user's SearchAccess (full-access project IDs,
// customer-role project IDs, and user ID used for customer-scoped filtering).
// Optional filterProjectIDs intersect the user's reachable projects.
func (s *SearchService) resolveAccess(ctx context.Context, userID uuid.UUID, filterProjectIDs []uuid.UUID) (model.SearchAccess, error) {
	memberships, err := s.members.ListByUser(ctx, userID)
	if err != nil {
		return model.SearchAccess{}, fmt.Errorf("listing user memberships: %w", err)
	}

	var allowed map[uuid.UUID]bool
	if len(filterProjectIDs) > 0 {
		allowed = make(map[uuid.UUID]bool, len(filterProjectIDs))
		for _, pid := range filterProjectIDs {
			allowed[pid] = true
		}
	}

	access := model.SearchAccess{UserID: userID}
	for _, m := range memberships {
		if allowed != nil && !allowed[m.ProjectID] {
			continue
		}
		if m.Role == model.ProjectRoleCustomer {
			access.CustomerProjectIDs = append(access.CustomerProjectIDs, m.ProjectID)
		} else {
			access.FullProjectIDs = append(access.FullProjectIDs, m.ProjectID)
		}
	}
	return access, nil
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
