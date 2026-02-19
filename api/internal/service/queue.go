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
	List(ctx context.Context, projectID uuid.UUID) ([]model.Queue, error)
	Update(ctx context.Context, q *model.Queue) error
	Delete(ctx context.Context, id uuid.UUID) error
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

// QueueService handles queue business logic and authorization.
type QueueService struct {
	queues   QueueRepository
	projects ProjectRepository
	members  ProjectMemberRepository
}

// NewQueueService creates a new QueueService.
func NewQueueService(queues QueueRepository, projects ProjectRepository, members ProjectMemberRepository) *QueueService {
	return &QueueService{
		queues:   queues,
		projects: projects,
		members:  members,
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

	if input.IsPublic != nil {
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

	return s.queues.Delete(ctx, queueID)
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
