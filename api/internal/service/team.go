package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// TeamRepository defines persistence operations for teams.
type TeamRepository interface {
	Create(ctx context.Context, t *model.Team) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error)
	List(ctx context.Context, projectID uuid.UUID) ([]model.Team, error)
	Update(ctx context.Context, t *model.Team) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error)
	RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
	ListMembers(ctx context.Context, teamID uuid.UUID) ([]model.TeamMemberWithUser, error)
}

// CreateTeamInput holds the input for creating a team.
type CreateTeamInput struct {
	Name        string
	Description *string
}

// UpdateTeamInput holds the input for updating a team.
type UpdateTeamInput struct {
	Name             *string
	Description      *string
	ClearDescription bool
}

// TeamService handles team business logic and authorization.
type TeamService struct {
	teams    TeamRepository
	projects ProjectRepository
	members  ProjectMemberRepository
}

// NewTeamService creates a new TeamService.
func NewTeamService(teams TeamRepository, projects ProjectRepository, members ProjectMemberRepository) *TeamService {
	return &TeamService{
		teams:    teams,
		projects: projects,
		members:  members,
	}
}

// Create creates a new team in the given project.
func (s *TeamService) Create(ctx context.Context, info *model.AuthInfo, projectKey string, input CreateTeamInput) (*model.Team, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("team name is required: %w", model.ErrValidation)
	}

	t := &model.Team{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		Name:        strings.TrimSpace(input.Name),
		Description: input.Description,
	}

	if err := s.teams.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("creating team: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("team_id", t.ID.String()).
		Str("project_key", projectKey).
		Str("name", t.Name).
		Msg("team created")

	return s.teams.GetByID(ctx, t.ID)
}

// Get returns a team by ID.
func (s *TeamService) Get(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID) (*model.Team, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	t, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if t.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return t, nil
}

// List returns all teams for a project.
func (s *TeamService) List(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.Team, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.teams.List(ctx, project.ID)
}

// Update modifies a team.
func (s *TeamService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, input UpdateTeamInput) (*model.Team, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	t, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if t.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("team name cannot be empty: %w", model.ErrValidation)
		}
		t.Name = name
	}

	if input.ClearDescription {
		t.Description = nil
	} else if input.Description != nil {
		t.Description = input.Description
	}

	if err := s.teams.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("updating team: %w", err)
	}

	return s.teams.GetByID(ctx, t.ID)
}

// Delete removes a team.
func (s *TeamService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	t, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	if t.ProjectID != project.ID {
		return model.ErrNotFound
	}

	return s.teams.Delete(ctx, teamID)
}

// ListMembers returns all members of a team with user details.
func (s *TeamService) ListMembers(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID) ([]model.TeamMemberWithUser, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	t, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if t.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	return s.teams.ListMembers(ctx, teamID)
}

// AddMember adds a user to a team.
func (s *TeamService) AddMember(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, userID uuid.UUID) (*model.TeamMember, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	t, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if t.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	// Prevent customers from being added to teams
	projectMember, err := s.members.GetByProjectAndUser(ctx, project.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("user must be a project member: %w", model.ErrValidation)
	}
	if projectMember.Role == model.ProjectRoleCustomer {
		return nil, fmt.Errorf("customers cannot be added to teams: %w", model.ErrValidation)
	}

	member, err := s.teams.AddMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("adding team member: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("team_id", teamID.String()).
		Str("user_id", userID.String()).
		Msg("team member added")

	return member, nil
}

// RemoveMember removes a user from a team.
func (s *TeamService) RemoveMember(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, userID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	t, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	if t.ProjectID != project.ID {
		return model.ErrNotFound
	}

	if err := s.teams.RemoveMember(ctx, teamID, userID); err != nil {
		return fmt.Errorf("removing team member: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("team_id", teamID.String()).
		Str("user_id", userID.String()).
		Msg("team member removed")

	return nil
}

func (s *TeamService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *TeamService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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
