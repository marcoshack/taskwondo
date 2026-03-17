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

// EscalationRepository defines persistence operations for escalation lists.
type EscalationRepository interface {
	Create(ctx context.Context, el *model.EscalationList) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.EscalationList, error)
	List(ctx context.Context, projectID uuid.UUID) ([]model.EscalationList, error)
	Update(ctx context.Context, el *model.EscalationList) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListMappings(ctx context.Context, projectID uuid.UUID) ([]model.TypeEscalationMapping, error)
	UpsertMapping(ctx context.Context, m *model.TypeEscalationMapping) error
	DeleteMapping(ctx context.Context, projectID uuid.UUID, workItemType string) error
}

// EscalationUserRepository defines user lookup needed for escalation validation.
type EscalationUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// CreateEscalationListInput holds the input for creating an escalation list.
type CreateEscalationListInput struct {
	Name   string
	Levels []EscalationLevelInput
}

// EscalationLevelInput holds the input for a single escalation level.
type EscalationLevelInput struct {
	ThresholdPct int
	UserIDs      []uuid.UUID
}

// EscalationService handles escalation list business logic and authorization.
type EscalationService struct {
	escalations EscalationRepository
	projects    ProjectRepository
	members     ProjectMemberRepository
	users       EscalationUserRepository
}

// NewEscalationService creates a new EscalationService.
func NewEscalationService(
	escalations EscalationRepository,
	projects ProjectRepository,
	members ProjectMemberRepository,
	users EscalationUserRepository,
) *EscalationService {
	return &EscalationService{
		escalations: escalations,
		projects:    projects,
		members:     members,
		users:       users,
	}
}

// Create creates a new escalation list in the given project.
func (s *EscalationService) Create(ctx context.Context, info *model.AuthInfo, projectKey string, input CreateEscalationListInput) (*model.EscalationList, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if err := s.validateInput(ctx, input); err != nil {
		return nil, err
	}

	el := &model.EscalationList{
		ID:        uuid.Must(uuid.NewV7()),
		ProjectID: project.ID,
		Name:      strings.TrimSpace(input.Name),
		Levels:    buildLevels(input.Levels),
	}

	if err := s.escalations.Create(ctx, el); err != nil {
		return nil, fmt.Errorf("creating escalation list: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("escalation_list_id", el.ID.String()).
		Str("project_key", projectKey).
		Str("name", el.Name).
		Msg("escalation list created")

	return s.escalations.GetByID(ctx, el.ID)
}

// Get returns an escalation list by ID.
func (s *EscalationService) Get(ctx context.Context, info *model.AuthInfo, projectKey string, listID uuid.UUID) (*model.EscalationList, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	el, err := s.escalations.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}

	if el.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return el, nil
}

// List returns all escalation lists for a project.
func (s *EscalationService) List(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.EscalationList, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.escalations.List(ctx, project.ID)
}

// Update modifies an escalation list (full replace of levels).
func (s *EscalationService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, listID uuid.UUID, input CreateEscalationListInput) (*model.EscalationList, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	existing, err := s.escalations.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if existing.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if err := s.validateInput(ctx, input); err != nil {
		return nil, err
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Levels = buildLevels(input.Levels)

	if err := s.escalations.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("updating escalation list: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("escalation_list_id", listID.String()).
		Str("project_key", projectKey).
		Msg("escalation list updated")

	return s.escalations.GetByID(ctx, listID)
}

// Delete removes an escalation list.
func (s *EscalationService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, listID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	existing, err := s.escalations.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if existing.ProjectID != project.ID {
		return model.ErrNotFound
	}

	return s.escalations.Delete(ctx, listID)
}

// ListMappings returns all type-escalation-list mappings for a project.
func (s *EscalationService) ListMappings(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.TypeEscalationMapping, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.escalations.ListMappings(ctx, project.ID)
}

// UpdateMapping sets or updates the escalation list for a work item type.
func (s *EscalationService) UpdateMapping(ctx context.Context, info *model.AuthInfo, projectKey string, workItemType string, escalationListID uuid.UUID) (*model.TypeEscalationMapping, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if strings.TrimSpace(workItemType) == "" {
		return nil, fmt.Errorf("work item type is required: %w", model.ErrValidation)
	}

	// Verify escalation list exists and belongs to this project
	el, err := s.escalations.GetByID(ctx, escalationListID)
	if err != nil {
		return nil, err
	}
	if el.ProjectID != project.ID {
		return nil, fmt.Errorf("escalation list not found in project: %w", model.ErrNotFound)
	}

	m := &model.TypeEscalationMapping{
		ProjectID:        project.ID,
		WorkItemType:     workItemType,
		EscalationListID: escalationListID,
	}

	if err := s.escalations.UpsertMapping(ctx, m); err != nil {
		return nil, fmt.Errorf("updating escalation mapping: %w", err)
	}

	return m, nil
}

// DeleteMapping removes the escalation list assignment for a work item type.
func (s *EscalationService) DeleteMapping(ctx context.Context, info *model.AuthInfo, projectKey string, workItemType string) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	return s.escalations.DeleteMapping(ctx, project.ID, workItemType)
}

// validateInput validates the create/update input.
func (s *EscalationService) validateInput(ctx context.Context, input CreateEscalationListInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("escalation list name is required: %w", model.ErrValidation)
	}

	if len(input.Levels) == 0 {
		return fmt.Errorf("at least one escalation level is required: %w", model.ErrValidation)
	}

	thresholds := make(map[int]bool)
	for _, lv := range input.Levels {
		if lv.ThresholdPct <= 0 {
			return fmt.Errorf("threshold percentage must be greater than 0: %w", model.ErrValidation)
		}
		if thresholds[lv.ThresholdPct] {
			return fmt.Errorf("duplicate threshold percentage %d: %w", lv.ThresholdPct, model.ErrValidation)
		}
		thresholds[lv.ThresholdPct] = true

		if len(lv.UserIDs) == 0 {
			return fmt.Errorf("each level must have at least one user: %w", model.ErrValidation)
		}

		// Verify each user exists
		for _, uid := range lv.UserIDs {
			if _, err := s.users.GetByID(ctx, uid); err != nil {
				return fmt.Errorf("user %s not found: %w", uid, model.ErrValidation)
			}
		}
	}

	return nil
}

// buildLevels converts input levels to model levels with new UUIDs.
func buildLevels(inputs []EscalationLevelInput) []model.EscalationLevel {
	levels := make([]model.EscalationLevel, len(inputs))
	for i, input := range inputs {
		users := make([]model.EscalationLevelUser, len(input.UserIDs))
		for j, uid := range input.UserIDs {
			users[j] = model.EscalationLevelUser{UserID: uid}
		}
		levels[i] = model.EscalationLevel{
			ID:           uuid.Must(uuid.NewV7()),
			ThresholdPct: input.ThresholdPct,
			Position:     i,
			Users:        users,
		}
	}
	return levels
}

func (s *EscalationService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *EscalationService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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
