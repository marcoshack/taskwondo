package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// SavedSearchRepository defines persistence operations for saved searches.
type SavedSearchRepository interface {
	Create(ctx context.Context, s *model.SavedSearch) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.SavedSearch, error)
	ListByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) ([]model.SavedSearch, error)
	Update(ctx context.Context, s *model.SavedSearch) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreateSavedSearchInput holds the input for creating a saved search.
type CreateSavedSearchInput struct {
	Name     string
	Filters  model.SavedSearchFilters
	ViewMode string
	Shared   bool
}

// UpdateSavedSearchInput holds the input for updating a saved search.
type UpdateSavedSearchInput struct {
	Name     *string
	Filters  *model.SavedSearchFilters
	ViewMode *string
	Position *int
}

// SavedSearchService handles saved search business logic and authorization.
type SavedSearchService struct {
	savedSearches SavedSearchRepository
	projects      ProjectRepository
	members       ProjectMemberRepository
}

// NewSavedSearchService creates a new SavedSearchService.
func NewSavedSearchService(savedSearches SavedSearchRepository, projects ProjectRepository, members ProjectMemberRepository) *SavedSearchService {
	return &SavedSearchService{
		savedSearches: savedSearches,
		projects:      projects,
		members:       members,
	}
}

// Create creates a new saved search in the given project.
func (s *SavedSearchService) Create(ctx context.Context, info *model.AuthInfo, projectKey string, input CreateSavedSearchInput) (*model.SavedSearch, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	if input.Shared {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("saved search name is required: %w", model.ErrValidation)
	}
	if !isValidViewMode(input.ViewMode) {
		return nil, fmt.Errorf("invalid view mode %q: %w", input.ViewMode, model.ErrValidation)
	}

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      strings.TrimSpace(input.Name),
		Filters:   input.Filters,
		ViewMode:  input.ViewMode,
	}

	if !input.Shared {
		ss.UserID = &info.UserID
	}

	if err := s.savedSearches.Create(ctx, ss); err != nil {
		return nil, fmt.Errorf("creating saved search: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("saved_search_id", ss.ID.String()).
		Str("project_key", projectKey).
		Str("name", ss.Name).
		Str("scope", ss.Scope()).
		Msg("saved search created")

	return s.savedSearches.GetByID(ctx, ss.ID)
}

// List returns all saved searches visible to the user in the given project.
func (s *SavedSearchService) List(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.SavedSearch, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.savedSearches.ListByProjectAndUser(ctx, project.ID, info.UserID)
}

// Update modifies a saved search.
func (s *SavedSearchService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, searchID uuid.UUID, input UpdateSavedSearchInput) (*model.SavedSearch, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	ss, err := s.savedSearches.GetByID(ctx, searchID)
	if err != nil {
		return nil, err
	}

	if ss.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if err := s.authorizeModify(ctx, info, project.ID, ss); err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("saved search name cannot be empty: %w", model.ErrValidation)
		}
		ss.Name = name
	}
	if input.Filters != nil {
		ss.Filters = *input.Filters
	}
	if input.ViewMode != nil {
		if !isValidViewMode(*input.ViewMode) {
			return nil, fmt.Errorf("invalid view mode %q: %w", *input.ViewMode, model.ErrValidation)
		}
		ss.ViewMode = *input.ViewMode
	}
	if input.Position != nil {
		if *input.Position < 0 {
			return nil, fmt.Errorf("position must be non-negative: %w", model.ErrValidation)
		}
		ss.Position = *input.Position
	}

	if err := s.savedSearches.Update(ctx, ss); err != nil {
		return nil, fmt.Errorf("updating saved search: %w", err)
	}

	return s.savedSearches.GetByID(ctx, ss.ID)
}

// Delete removes a saved search.
func (s *SavedSearchService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, searchID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	ss, err := s.savedSearches.GetByID(ctx, searchID)
	if err != nil {
		return err
	}

	if ss.ProjectID != project.ID {
		return model.ErrNotFound
	}

	if err := s.authorizeModify(ctx, info, project.ID, ss); err != nil {
		return err
	}

	return s.savedSearches.Delete(ctx, searchID)
}

// authorizeModify checks if the user can modify the saved search.
// Personal searches: only the owner can modify.
// Shared searches: only project owner/admin can modify.
func (s *SavedSearchService) authorizeModify(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, ss *model.SavedSearch) error {
	if ss.UserID != nil {
		// Personal search — only the owner
		if *ss.UserID != info.UserID {
			return model.ErrForbidden
		}
		return nil
	}
	// Shared search — require owner/admin
	return s.requireRole(ctx, info, projectID, model.ProjectRoleOwner, model.ProjectRoleAdmin)
}

func (s *SavedSearchService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	_, err := s.members.GetByProjectAndUser(ctx, projectID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking membership: %w", err)
	}
	return nil
}

func (s *SavedSearchService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	member, err := s.members.GetByProjectAndUser(ctx, projectID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking membership: %w", err)
	}
	if slices.Contains(allowedRoles, member.Role) {
		return nil
	}
	return model.ErrForbidden
}

func isValidViewMode(vm string) bool {
	switch vm {
	case "list", "board":
		return true
	}
	return false
}
