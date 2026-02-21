package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

var projectKeyRegexp = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,4}$`)

// ProjectRepository defines the persistence operations for projects.
type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) error
	GetByKey(ctx context.Context, key string) (*model.Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Project, error)
	ListAll(ctx context.Context) ([]model.Project, error)
	GetSummaries(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectSummary, error)
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProjectMemberRepository defines the persistence operations for project members.
type ProjectMemberRepository interface {
	Add(ctx context.Context, member *model.ProjectMember) error
	GetByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectMemberWithUser, error)
	UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role string) error
	Remove(ctx context.Context, projectID, userID uuid.UUID) error
	CountByRole(ctx context.Context, projectID uuid.UUID, role string) (int, error)
}

// ProjectTypeWorkflowRepository defines persistence operations for project type-workflow mappings.
type ProjectTypeWorkflowRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectTypeWorkflow, error)
	GetByProjectAndType(ctx context.Context, projectID uuid.UUID, workItemType string) (*model.ProjectTypeWorkflow, error)
	Upsert(ctx context.Context, mapping *model.ProjectTypeWorkflow) error
}

// ProjectService handles project business logic and authorization.
type ProjectService struct {
	projects      ProjectRepository
	members       ProjectMemberRepository
	users         UserRepository
	workflows     WorkflowRepository
	typeWorkflows ProjectTypeWorkflowRepository
}

// NewProjectService creates a new ProjectService.
func NewProjectService(projects ProjectRepository, members ProjectMemberRepository, users UserRepository, workflows WorkflowRepository, typeWorkflows ProjectTypeWorkflowRepository) *ProjectService {
	return &ProjectService{
		projects:      projects,
		members:       members,
		users:         users,
		workflows:     workflows,
		typeWorkflows: typeWorkflows,
	}
}

// Create creates a new project and adds the creator as owner.
func (s *ProjectService) Create(ctx context.Context, info *model.AuthInfo, name, key string, description *string, defaultWorkflowID *uuid.UUID) (*model.Project, error) {
	if !projectKeyRegexp.MatchString(key) {
		return nil, fmt.Errorf("project key must be 2-5 uppercase alphanumeric characters starting with a letter: %w", model.ErrConflict)
	}

	// Check for duplicate key
	existing, err := s.projects.GetByKey(ctx, key)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("project key %q already in use: %w", key, model.ErrAlreadyExists)
	}
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("checking project key: %w", err)
	}

	// Fetch all workflows for default assignment and type-workflow seeding
	workflows, err := s.workflows.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}

	// Auto-assign the first default workflow if none specified
	if defaultWorkflowID == nil {
		for i := range workflows {
			if workflows[i].IsDefault {
				defaultWorkflowID = &workflows[i].ID
				break
			}
		}
	}

	project := &model.Project{
		ID:                uuid.New(),
		Name:              name,
		Key:               key,
		Description:       description,
		DefaultWorkflowID: defaultWorkflowID,
	}

	if err := s.projects.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	// Add creator as owner
	member := &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    info.UserID,
		Role:      model.ProjectRoleOwner,
	}
	if err := s.members.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("adding project owner: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_id", project.ID.String()).
		Str("project_key", project.Key).
		Str("user_id", info.UserID.String()).
		Msg("project created")

	// Auto-populate type-workflow mappings
	s.seedTypeWorkflows(ctx, project.ID, workflows)

	// Re-fetch to get timestamps set by the database
	created, err := s.projects.GetByID(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching created project: %w", err)
	}

	return created, nil
}

// Get returns a project by key, checking membership.
func (s *ProjectService) Get(ctx context.Context, info *model.AuthInfo, projectKey string) (*model.Project, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return project, nil
}

// List returns projects visible to the authenticated user.
func (s *ProjectService) List(ctx context.Context, info *model.AuthInfo) ([]model.Project, error) {
	if info.GlobalRole == model.RoleAdmin {
		return s.projects.ListAll(ctx)
	}
	return s.projects.ListByUser(ctx, info.UserID)
}

// ListWithSummary returns projects with aggregate counts.
func (s *ProjectService) ListWithSummary(ctx context.Context, info *model.AuthInfo) ([]model.ProjectWithSummary, error) {
	projects, err := s.List(ctx, info)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(projects))
	for i := range projects {
		ids[i] = projects[i].ID
	}

	summaries, err := s.projects.GetSummaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetching project summaries: %w", err)
	}

	result := make([]model.ProjectWithSummary, len(projects))
	for i := range projects {
		result[i] = model.ProjectWithSummary{
			Project:        projects[i],
			ProjectSummary: summaries[projects[i].ID],
		}
	}
	return result, nil
}

// Update modifies a project. Requires owner or admin role.
func (s *ProjectService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, name, key *string, description *string, clearDescription bool, defaultWorkflowID *uuid.UUID, allowedComplexityValues []int, clearAllowedComplexityValues bool, businessHours *model.BusinessHoursConfig, clearBusinessHours bool) (*model.Project, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if name != nil {
		project.Name = *name
	}
	if key != nil {
		if !projectKeyRegexp.MatchString(*key) {
			return nil, fmt.Errorf("project key must be 2-5 uppercase alphanumeric characters starting with a letter: %w", model.ErrConflict)
		}
		// Check for duplicate key if changing
		if *key != project.Key {
			existing, err := s.projects.GetByKey(ctx, *key)
			if err == nil && existing != nil {
				return nil, fmt.Errorf("project key %q already in use: %w", *key, model.ErrAlreadyExists)
			}
			if err != nil && err != model.ErrNotFound {
				return nil, fmt.Errorf("checking project key: %w", err)
			}
		}
		project.Key = *key
	}
	if description != nil {
		project.Description = description
	}
	if clearDescription {
		project.Description = nil
	}
	if defaultWorkflowID != nil {
		project.DefaultWorkflowID = defaultWorkflowID
	}
	if clearAllowedComplexityValues {
		project.AllowedComplexityValues = []int{}
	} else if allowedComplexityValues != nil {
		// Validate: all values must be positive, no duplicates
		seen := make(map[int]bool, len(allowedComplexityValues))
		for _, v := range allowedComplexityValues {
			if v <= 0 {
				return nil, fmt.Errorf("allowed complexity values must be positive integers: %w", model.ErrValidation)
			}
			if seen[v] {
				return nil, fmt.Errorf("duplicate complexity value %d: %w", v, model.ErrValidation)
			}
			seen[v] = true
		}
		project.AllowedComplexityValues = allowedComplexityValues
	}

	if clearBusinessHours {
		project.BusinessHours = nil
	} else if businessHours != nil {
		project.BusinessHours = businessHours
	}

	if err := s.projects.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	return s.projects.GetByKey(ctx, project.Key)
}

// Delete soft-deletes a project. Requires owner role.
func (s *ProjectService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner); err != nil {
		return err
	}

	if err := s.projects.Delete(ctx, project.ID); err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_id", project.ID.String()).
		Str("project_key", project.Key).
		Str("user_id", info.UserID.String()).
		Msg("project deleted")

	return nil
}

// AddMember adds a user to a project. Requires owner or admin role.
func (s *ProjectService) AddMember(ctx context.Context, info *model.AuthInfo, projectKey string, userID uuid.UUID, role string) (*model.ProjectMember, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if !isValidProjectRole(role) {
		return nil, fmt.Errorf("invalid project role %q: %w", role, model.ErrConflict)
	}

	// Only owners can add other owners
	if role == model.ProjectRoleOwner {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner); err != nil {
			return nil, model.ErrForbidden
		}
	}

	// Verify user exists
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	// Check if already a member
	existing, err := s.members.GetByProjectAndUser(ctx, project.ID, userID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("user is already a member of this project: %w", model.ErrAlreadyExists)
	}
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("checking membership: %w", err)
	}

	member := &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    userID,
		Role:      role,
	}

	if err := s.members.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("adding member: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("user_id", userID.String()).
		Str("role", role).
		Msg("member added to project")

	return member, nil
}

// ListMembers returns all members of a project. Requires membership.
func (s *ProjectService) ListMembers(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.ProjectMemberWithUser, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.members.ListByProject(ctx, project.ID)
}

// UpdateMemberRole changes a member's role. Requires owner or admin role.
func (s *ProjectService) UpdateMemberRole(ctx context.Context, info *model.AuthInfo, projectKey string, userID uuid.UUID, role string) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	if !isValidProjectRole(role) {
		return fmt.Errorf("invalid project role %q: %w", role, model.ErrConflict)
	}

	// Get target member to check current role
	target, err := s.members.GetByProjectAndUser(ctx, project.ID, userID)
	if err != nil {
		return err
	}

	// Only owners can promote to or demote from owner
	if role == model.ProjectRoleOwner || target.Role == model.ProjectRoleOwner {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner); err != nil {
			return model.ErrForbidden
		}
	}

	// Protect last owner
	if target.Role == model.ProjectRoleOwner && role != model.ProjectRoleOwner {
		count, err := s.members.CountByRole(ctx, project.ID, model.ProjectRoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return fmt.Errorf("cannot demote the last owner: %w", model.ErrConflict)
		}
	}

	if err := s.members.UpdateRole(ctx, project.ID, userID, role); err != nil {
		return fmt.Errorf("updating member role: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("user_id", userID.String()).
		Str("new_role", role).
		Msg("member role updated")

	return nil
}

// RemoveMember removes a user from a project. Requires owner or admin role.
func (s *ProjectService) RemoveMember(ctx context.Context, info *model.AuthInfo, projectKey string, userID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	// Get target member to check role
	target, err := s.members.GetByProjectAndUser(ctx, project.ID, userID)
	if err != nil {
		return err
	}

	// Only owners can remove owners
	if target.Role == model.ProjectRoleOwner {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner); err != nil {
			return model.ErrForbidden
		}

		// Protect last owner
		count, err := s.members.CountByRole(ctx, project.ID, model.ProjectRoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return fmt.Errorf("cannot remove the last owner: %w", model.ErrConflict)
		}
	}

	if err := s.members.Remove(ctx, project.ID, userID); err != nil {
		return fmt.Errorf("removing member: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("user_id", userID.String()).
		Msg("member removed from project")

	return nil
}

// GetTypeWorkflows returns all type-workflow mappings for a project. Requires membership.
func (s *ProjectService) GetTypeWorkflows(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.ProjectTypeWorkflow, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.typeWorkflows.ListByProject(ctx, project.ID)
}

// UpdateTypeWorkflow updates the workflow for a specific work item type in a project. Requires owner or admin role.
func (s *ProjectService) UpdateTypeWorkflow(ctx context.Context, info *model.AuthInfo, projectKey string, workItemType string, workflowID uuid.UUID) (*model.ProjectTypeWorkflow, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	// Validate work item type
	if !isValidWorkItemType(workItemType) {
		return nil, fmt.Errorf("invalid work item type %q: %w", workItemType, model.ErrValidation)
	}

	// Verify workflow exists
	if _, err := s.workflows.GetByID(ctx, workflowID); err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	mapping := &model.ProjectTypeWorkflow{
		ID:           uuid.New(),
		ProjectID:    project.ID,
		WorkItemType: workItemType,
		WorkflowID:   workflowID,
	}

	if err := s.typeWorkflows.Upsert(ctx, mapping); err != nil {
		return nil, fmt.Errorf("upserting type workflow: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("work_item_type", workItemType).
		Str("workflow_id", workflowID.String()).
		Msg("type workflow mapping updated")

	return mapping, nil
}

// SeedExistingProjectTypeWorkflows backfills type-workflow mappings for projects
// created before the per-type workflow feature. Projects that already have mappings
// are skipped. This is idempotent and safe to call on every startup.
func (s *ProjectService) SeedExistingProjectTypeWorkflows(ctx context.Context) error {
	workflows, err := s.workflows.List(ctx)
	if err != nil {
		return fmt.Errorf("listing workflows: %w", err)
	}
	if len(workflows) == 0 {
		return nil
	}

	projects, err := s.projects.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	seeded := 0
	for _, p := range projects {
		existing, err := s.typeWorkflows.ListByProject(ctx, p.ID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("project_id", p.ID.String()).Msg("failed to check type workflows")
			continue
		}
		if len(existing) > 0 {
			continue
		}
		s.seedTypeWorkflows(ctx, p.ID, workflows)
		seeded++
	}

	if seeded > 0 {
		log.Ctx(ctx).Info().Int("count", seeded).Msg("backfilled type-workflow mappings for existing projects")
	}
	return nil
}

// seedTypeWorkflows populates default type-workflow mappings for a new project.
func (s *ProjectService) seedTypeWorkflows(ctx context.Context, projectID uuid.UUID, workflows []model.Workflow) {
	if len(workflows) == 0 {
		return
	}

	// Find Task Workflow and Ticket Workflow by name
	var taskWfID, ticketWfID uuid.UUID
	for _, wf := range workflows {
		switch wf.Name {
		case "Task Workflow":
			taskWfID = wf.ID
		case "Ticket Workflow":
			ticketWfID = wf.ID
		}
	}

	// Fallback: use the first default workflow for all types
	if taskWfID == uuid.Nil {
		for _, wf := range workflows {
			if wf.IsDefault {
				taskWfID = wf.ID
				break
			}
		}
	}
	if ticketWfID == uuid.Nil {
		ticketWfID = taskWfID
	}
	if taskWfID == uuid.Nil {
		return
	}

	// Map types to workflows
	typeMapping := map[string]uuid.UUID{
		model.WorkItemTypeTask:     taskWfID,
		model.WorkItemTypeBug:      taskWfID,
		model.WorkItemTypeEpic:     taskWfID,
		model.WorkItemTypeTicket:   ticketWfID,
		model.WorkItemTypeFeedback: ticketWfID,
	}

	for itemType, wfID := range typeMapping {
		mapping := &model.ProjectTypeWorkflow{
			ID:           uuid.New(),
			ProjectID:    projectID,
			WorkItemType: itemType,
			WorkflowID:   wfID,
		}
		if err := s.typeWorkflows.Upsert(ctx, mapping); err != nil {
			log.Ctx(ctx).Warn().Err(err).
				Str("work_item_type", itemType).
				Msg("failed to seed type workflow mapping")
		}
	}
}

// requireMembership checks that the user is a member of the project or a global admin.
// Returns ErrNotFound (not ErrForbidden) to avoid leaking project existence.
func (s *ProjectService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

// requireRole checks that the user has one of the allowed roles or is a global admin.
func (s *ProjectService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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

// RequireProjectRole gets a project by key and checks that the user has one of the allowed roles.
// This is a public method for use by other handlers that need project-level authorization.
func (s *ProjectService) RequireProjectRole(ctx context.Context, info *model.AuthInfo, projectKey string, allowedRoles ...string) (*model.Project, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, allowedRoles...); err != nil {
		return nil, err
	}

	return project, nil
}

func isValidProjectRole(role string) bool {
	switch role {
	case model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember, model.ProjectRoleViewer:
		return true
	}
	return false
}
