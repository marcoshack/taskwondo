package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// QueueRepository defines persistence operations for queues.
type QueueRepository interface {
	Create(ctx context.Context, q *model.Queue) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Queue, error)
	GetPublicByProject(ctx context.Context, projectID uuid.UUID) (*model.Queue, error)
	List(ctx context.Context, projectID uuid.UUID) ([]model.Queue, error)
	CountPublicByProject(ctx context.Context, projectID uuid.UUID) (int, error)
	Update(ctx context.Context, q *model.Queue) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueueCategoryRepository defines persistence operations for queue categories.
type QueueCategoryRepository interface {
	Create(ctx context.Context, cat *model.QueueCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.QueueCategory, error)
	ListByQueue(ctx context.Context, queueID uuid.UUID) ([]model.QueueCategory, error)
	Update(ctx context.Context, cat *model.QueueCategory) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueueTeamRepository defines persistence operations for queue-team assignments.
type QueueTeamRepository interface {
	Assign(ctx context.Context, queueID, teamID uuid.UUID) error
	Unassign(ctx context.Context, queueID, teamID uuid.UUID) error
	ListTeamsByQueue(ctx context.Context, queueID uuid.UUID) ([]model.Team, error)
	ListQueuesByTeam(ctx context.Context, teamID uuid.UUID) ([]model.Queue, error)
}

// CreateQueueInput holds the input for creating a queue.
type CreateQueueInput struct {
	Name              string
	Description       *string
	QueueType         string
	IsPublic          bool
	DefaultPriority   string
	DefaultAssigneeID *uuid.UUID
	WorkflowID        *uuid.UUID
}

// UpdateQueueInput holds the input for updating a queue.
type UpdateQueueInput struct {
	Name              *string
	Description       *string
	ClearDescription  bool
	QueueType         *string
	IsPublic          *bool
	DefaultPriority   *string
	DefaultAssigneeID *uuid.UUID
	ClearDefaultAssignee bool
	WorkflowID        *uuid.UUID
	ClearWorkflow     bool
}

// CreateCategoryInput holds the input for creating a queue category.
type CreateCategoryInput struct {
	Name        string
	Description *string
	Position    int
}

// UpdateCategoryInput holds the input for updating a queue category.
type UpdateCategoryInput struct {
	Name             *string
	Description      *string
	ClearDescription bool
	Position         *int
}

// QueueService handles queue business logic and authorization.
type QueueService struct {
	queues     QueueRepository
	categories QueueCategoryRepository
	queueTeams QueueTeamRepository
	projects   ProjectRepository
	members    ProjectMemberRepository
	publisher  EventPublisher
	embedCache *FeatureFlagCache
}

// SetPublisher configures the event publisher for embed events.
func (s *QueueService) SetPublisher(p EventPublisher) {
	s.publisher = p
}

// SetEmbedCache configures the feature flag cache for semantic search indexing.
func (s *QueueService) SetEmbedCache(cache *FeatureFlagCache) {
	s.embedCache = cache
}

// NewQueueService creates a new QueueService.
func NewQueueService(queues QueueRepository, categories QueueCategoryRepository, queueTeams QueueTeamRepository, projects ProjectRepository, members ProjectMemberRepository) *QueueService {
	return &QueueService{
		queues:     queues,
		categories: categories,
		queueTeams: queueTeams,
		projects:   projects,
		members:    members,
	}
}

// Create creates a new queue in the given project.
func (s *QueueService) Create(ctx context.Context, info *model.AuthInfo, projectKey string, input CreateQueueInput) (*model.Queue, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("queue name is required: %w", model.ErrValidation)
	}
	if !isValidQueueType(input.QueueType) {
		return nil, fmt.Errorf("invalid queue type %q: %w", input.QueueType, model.ErrValidation)
	}
	if input.DefaultPriority == "" {
		input.DefaultPriority = model.PriorityMedium
	}
	if !isValidPriority(input.DefaultPriority) {
		return nil, fmt.Errorf("invalid default priority %q: %w", input.DefaultPriority, model.ErrValidation)
	}

	// Enforce only one public queue per project
	if input.IsPublic {
		count, err := s.queues.CountPublicByProject(ctx, project.ID)
		if err != nil {
			return nil, fmt.Errorf("checking public queues: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("project already has a public queue: %w", model.ErrValidation)
		}
	}

	q := &model.Queue{
		ID:                uuid.New(),
		ProjectID:         project.ID,
		Name:              strings.TrimSpace(input.Name),
		Description:       input.Description,
		QueueType:         input.QueueType,
		IsPublic:          input.IsPublic,
		DefaultPriority:   input.DefaultPriority,
		DefaultAssigneeID: input.DefaultAssigneeID,
		WorkflowID:        input.WorkflowID,
	}

	if err := s.queues.Create(ctx, q); err != nil {
		return nil, fmt.Errorf("creating queue: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("queue_id", q.ID.String()).
		Str("project_key", projectKey).
		Str("name", q.Name).
		Msg("queue created")

	// Publish embed.index event
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeQueue, q.ID, &project.ID)

	return s.queues.GetByID(ctx, q.ID)
}

// Get returns a queue by ID.
func (s *QueueService) Get(ctx context.Context, info *model.AuthInfo, projectKey string, queueID uuid.UUID) (*model.Queue, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return nil, err
	}

	if q.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return q, nil
}

// GetPublicQueue returns the public queue for a project.
func (s *QueueService) GetPublicQueue(ctx context.Context, projectKey string) (*model.Queue, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	return s.queues.GetPublicByProject(ctx, project.ID)
}

// List returns all queues for a project.
func (s *QueueService) List(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.Queue, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.queues.List(ctx, project.ID)
}

// Update modifies a queue.
func (s *QueueService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, queueID uuid.UUID, input UpdateQueueInput) (*model.Queue, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return nil, err
	}

	if q.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("queue name cannot be empty: %w", model.ErrValidation)
		}
		q.Name = name
	}

	if input.ClearDescription {
		q.Description = nil
	} else if input.Description != nil {
		q.Description = input.Description
	}

	if input.QueueType != nil {
		if !isValidQueueType(*input.QueueType) {
			return nil, fmt.Errorf("invalid queue type %q: %w", *input.QueueType, model.ErrValidation)
		}
		q.QueueType = *input.QueueType
	}

	if input.IsPublic != nil && *input.IsPublic && !q.IsPublic {
		// Enforce only one public queue per project
		count, err := s.queues.CountPublicByProject(ctx, project.ID)
		if err != nil {
			return nil, fmt.Errorf("checking public queues: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("project already has a public queue: %w", model.ErrValidation)
		}
		q.IsPublic = true
	} else if input.IsPublic != nil {
		q.IsPublic = *input.IsPublic
	}

	if input.DefaultPriority != nil {
		if !isValidPriority(*input.DefaultPriority) {
			return nil, fmt.Errorf("invalid default priority %q: %w", *input.DefaultPriority, model.ErrValidation)
		}
		q.DefaultPriority = *input.DefaultPriority
	}

	if input.ClearDefaultAssignee {
		q.DefaultAssigneeID = nil
	} else if input.DefaultAssigneeID != nil {
		q.DefaultAssigneeID = input.DefaultAssigneeID
	}

	if input.ClearWorkflow {
		q.WorkflowID = nil
	} else if input.WorkflowID != nil {
		q.WorkflowID = input.WorkflowID
	}

	if err := s.queues.Update(ctx, q); err != nil {
		return nil, fmt.Errorf("updating queue: %w", err)
	}

	// Reindex queue embedding
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeQueue, q.ID, &project.ID)

	return s.queues.GetByID(ctx, q.ID)
}

// Delete removes a queue.
func (s *QueueService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, queueID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return err
	}

	if q.ProjectID != project.ID {
		return model.ErrNotFound
	}

	if err := s.queues.Delete(ctx, queueID); err != nil {
		return err
	}

	// Delete queue embedding
	publishEmbedDelete(ctx, s.publisher, s.embedCache, model.EntityTypeQueue, queueID)

	return nil
}

// CreateCategory creates a new category within a queue.
func (s *QueueService) CreateCategory(ctx context.Context, info *model.AuthInfo, projectKey string, queueID uuid.UUID, input CreateCategoryInput) (*model.QueueCategory, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("category name is required: %w", model.ErrValidation)
	}

	cat := &model.QueueCategory{
		ID:          uuid.New(),
		QueueID:     queueID,
		Name:        strings.TrimSpace(input.Name),
		Description: input.Description,
		Position:    input.Position,
	}

	if err := s.categories.Create(ctx, cat); err != nil {
		return nil, fmt.Errorf("creating queue category: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("category_id", cat.ID.String()).
		Str("queue_id", queueID.String()).
		Str("name", cat.Name).
		Msg("queue category created")

	return s.categories.GetByID(ctx, cat.ID)
}

// ListCategories returns all categories for a queue.
func (s *QueueService) ListCategories(ctx context.Context, info *model.AuthInfo, projectKey string, queueID uuid.UUID) ([]model.QueueCategory, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return s.categories.ListByQueue(ctx, queueID)
}

// UpdateCategory modifies a queue category.
func (s *QueueService) UpdateCategory(ctx context.Context, info *model.AuthInfo, projectKey string, queueID, categoryID uuid.UUID, input UpdateCategoryInput) (*model.QueueCategory, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	cat, err := s.categories.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if cat.QueueID != queueID {
		return nil, model.ErrNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("category name cannot be empty: %w", model.ErrValidation)
		}
		cat.Name = name
	}

	if input.ClearDescription {
		cat.Description = nil
	} else if input.Description != nil {
		cat.Description = input.Description
	}

	if input.Position != nil {
		cat.Position = *input.Position
	}

	if err := s.categories.Update(ctx, cat); err != nil {
		return nil, fmt.Errorf("updating queue category: %w", err)
	}

	return s.categories.GetByID(ctx, cat.ID)
}

// DeleteCategory removes a queue category.
func (s *QueueService) DeleteCategory(ctx context.Context, info *model.AuthInfo, projectKey string, queueID, categoryID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return err
	}
	if q.ProjectID != project.ID {
		return model.ErrNotFound
	}

	cat, err := s.categories.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if cat.QueueID != queueID {
		return model.ErrNotFound
	}

	return s.categories.Delete(ctx, categoryID)
}

// AssignTeam assigns a team to a queue.
func (s *QueueService) AssignTeam(ctx context.Context, info *model.AuthInfo, projectKey string, queueID, teamID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return err
	}
	if q.ProjectID != project.ID {
		return model.ErrNotFound
	}

	if err := s.queueTeams.Assign(ctx, queueID, teamID); err != nil {
		return fmt.Errorf("assigning team to queue: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("queue_id", queueID.String()).
		Str("team_id", teamID.String()).
		Msg("team assigned to queue")

	return nil
}

// UnassignTeam removes a team from a queue.
func (s *QueueService) UnassignTeam(ctx context.Context, info *model.AuthInfo, projectKey string, queueID, teamID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return err
	}
	if q.ProjectID != project.ID {
		return model.ErrNotFound
	}

	return s.queueTeams.Unassign(ctx, queueID, teamID)
}

// ListQueueTeams returns all teams assigned to a queue.
func (s *QueueService) ListQueueTeams(ctx context.Context, info *model.AuthInfo, projectKey string, queueID uuid.UUID) ([]model.Team, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	q, err := s.queues.GetByID(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return s.queueTeams.ListTeamsByQueue(ctx, queueID)
}

func (s *QueueService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *QueueService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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
	for _, role := range allowedRoles {
		if member.Role == role {
			return nil
		}
	}
	return model.ErrForbidden
}

func isValidQueueType(t string) bool {
	switch t {
	case model.QueueTypeSupport, model.QueueTypeAlerts, model.QueueTypeFeedback, model.QueueTypeGeneral:
		return true
	}
	return false
}
