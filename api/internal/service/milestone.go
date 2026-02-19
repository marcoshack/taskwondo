package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// MilestoneRepository defines persistence operations for milestones.
type MilestoneRepository interface {
	Create(ctx context.Context, m *model.Milestone) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Milestone, error)
	GetByIDWithProgress(ctx context.Context, id uuid.UUID) (*model.MilestoneWithProgress, error)
	List(ctx context.Context, projectID uuid.UUID) ([]model.Milestone, error)
	ListWithProgress(ctx context.Context, projectID uuid.UUID) ([]model.MilestoneWithProgress, error)
	Update(ctx context.Context, m *model.Milestone) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreateMilestoneInput holds the input for creating a milestone.
type CreateMilestoneInput struct {
	Name        string
	Description *string
	DueDate     *time.Time
}

// UpdateMilestoneInput holds the input for updating a milestone.
type UpdateMilestoneInput struct {
	Name             *string
	Description      *string
	ClearDescription bool
	DueDate          *time.Time
	ClearDueDate     bool
	Status           *string
}

// MilestoneService handles milestone business logic and authorization.
type MilestoneService struct {
	milestones MilestoneRepository
	projects   ProjectRepository
	members    ProjectMemberRepository
}

// NewMilestoneService creates a new MilestoneService.
func NewMilestoneService(milestones MilestoneRepository, projects ProjectRepository, members ProjectMemberRepository) *MilestoneService {
	return &MilestoneService{
		milestones: milestones,
		projects:   projects,
		members:    members,
	}
}

// Create creates a new milestone in the given project.
func (s *MilestoneService) Create(ctx context.Context, info *model.AuthInfo, projectKey string, input CreateMilestoneInput) (*model.MilestoneWithProgress, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("milestone name is required: %w", model.ErrValidation)
	}

	m := &model.Milestone{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		Name:        strings.TrimSpace(input.Name),
		Description: input.Description,
		DueDate:     input.DueDate,
		Status:      model.MilestoneStatusOpen,
	}

	if err := s.milestones.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("creating milestone: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("milestone_id", m.ID.String()).
		Str("project_key", projectKey).
		Str("name", m.Name).
		Msg("milestone created")

	return s.milestones.GetByIDWithProgress(ctx, m.ID)
}

// Get returns a milestone with progress by ID.
func (s *MilestoneService) Get(ctx context.Context, info *model.AuthInfo, projectKey string, milestoneID uuid.UUID) (*model.MilestoneWithProgress, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	mp, err := s.milestones.GetByIDWithProgress(ctx, milestoneID)
	if err != nil {
		return nil, err
	}

	if mp.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return mp, nil
}

// List returns all milestones with progress for a project.
func (s *MilestoneService) List(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.MilestoneWithProgress, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.milestones.ListWithProgress(ctx, project.ID)
}

// Update modifies a milestone.
func (s *MilestoneService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, milestoneID uuid.UUID, input UpdateMilestoneInput) (*model.MilestoneWithProgress, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	m, err := s.milestones.GetByID(ctx, milestoneID)
	if err != nil {
		return nil, err
	}

	if m.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("milestone name cannot be empty: %w", model.ErrValidation)
		}
		m.Name = name
	}

	if input.ClearDescription {
		m.Description = nil
	} else if input.Description != nil {
		m.Description = input.Description
	}

	if input.ClearDueDate {
		m.DueDate = nil
	} else if input.DueDate != nil {
		m.DueDate = input.DueDate
	}

	if input.Status != nil {
		if !isValidMilestoneStatus(*input.Status) {
			return nil, fmt.Errorf("invalid milestone status %q: %w", *input.Status, model.ErrValidation)
		}
		m.Status = *input.Status
	}

	if err := s.milestones.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("updating milestone: %w", err)
	}

	return s.milestones.GetByIDWithProgress(ctx, m.ID)
}

// Delete removes a milestone.
func (s *MilestoneService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, milestoneID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	m, err := s.milestones.GetByID(ctx, milestoneID)
	if err != nil {
		return err
	}

	if m.ProjectID != project.ID {
		return model.ErrNotFound
	}

	return s.milestones.Delete(ctx, milestoneID)
}

func (s *MilestoneService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *MilestoneService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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

func isValidMilestoneStatus(s string) bool {
	switch s {
	case model.MilestoneStatusOpen, model.MilestoneStatusClosed:
		return true
	}
	return false
}
