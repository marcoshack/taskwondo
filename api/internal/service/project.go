package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

var projectKeyRegexp = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,4}$`)

// ProjectRepository defines the persistence operations for projects.
type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) error
	GetByKey(ctx context.Context, key string) (*model.Project, error)
	GetByKeyAndNamespace(ctx context.Context, namespaceID uuid.UUID, key string) (*model.Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Project, error)
	ListAll(ctx context.Context) ([]model.Project, error)
	GetSummaries(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectSummary, error)
	CountByOwner(ctx context.Context, userID uuid.UUID) (int, error)
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
	ResolveNamespaces(ctx context.Context, projectKeys []string) (map[string]model.ProjectNamespaceInfo, error)
	ResolveNamespacesByIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectNamespaceInfo, error)
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

// ProjectInviteRepository defines persistence operations for project invites.
type ProjectInviteRepository interface {
	Create(ctx context.Context, invite *model.ProjectInvite) error
	GetByCode(ctx context.Context, code string) (*model.ProjectInvite, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.ProjectInvite, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectInvite, error)
	IncrementUseCount(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProject(ctx context.Context, projectID uuid.UUID) error
}

// ProjectService handles project business logic and authorization.
type ProjectService struct {
	projects       ProjectRepository
	members        ProjectMemberRepository
	users          UserRepository
	workflows      WorkflowRepository
	typeWorkflows  ProjectTypeWorkflowRepository
	systemSettings SystemSettingRepositoryInterface
	invites        ProjectInviteRepository
	inboxItems     InboxRepository
	watchers       WatcherRepository
	userSettings   UserSettingRepository
	publisher      EventPublisher
	embedCache     *FeatureFlagCache
}

// SetPublisher configures the event publisher for embed events.
func (s *ProjectService) SetPublisher(p EventPublisher) {
	s.publisher = p
}

// SetEmbedCache configures the feature flag cache for semantic search indexing.
func (s *ProjectService) SetEmbedCache(cache *FeatureFlagCache) {
	s.embedCache = cache
}

// NewProjectService creates a new ProjectService.
func NewProjectService(projects ProjectRepository, members ProjectMemberRepository, users UserRepository, workflows WorkflowRepository, typeWorkflows ProjectTypeWorkflowRepository, systemSettings SystemSettingRepositoryInterface, invites ProjectInviteRepository, inboxItems InboxRepository, watchers WatcherRepository, userSettings UserSettingRepository) *ProjectService {
	return &ProjectService{
		projects:       projects,
		members:        members,
		users:          users,
		workflows:      workflows,
		typeWorkflows:  typeWorkflows,
		systemSettings: systemSettings,
		invites:        invites,
		inboxItems:     inboxItems,
		watchers:       watchers,
		userSettings:   userSettings,
	}
}

// Create creates a new project and adds the creator as owner.
func (s *ProjectService) Create(ctx context.Context, info *model.AuthInfo, name, key string, description *string, defaultWorkflowID *uuid.UUID) (*model.Project, error) {
	// Enforce per-user project limit (admins are exempt)
	if info.GlobalRole != model.RoleAdmin {
		limit, err := s.resolveProjectLimit(ctx, info.UserID)
		if err != nil {
			return nil, fmt.Errorf("resolving project limit: %w", err)
		}

		// limit == 0 means unlimited
		if limit > 0 {
			count, err := s.projects.CountByOwner(ctx, info.UserID)
			if err != nil {
				return nil, fmt.Errorf("counting user projects: %w", err)
			}
			if count >= limit {
				return nil, model.NewKeyedError(model.ErrForbidden, "project_limit_reached",
					"project ownership limit reached; contact an administrator to increase your limit", nil)
			}
		}
	}

	if !projectKeyRegexp.MatchString(key) {
		return nil, model.NewKeyedError(model.ErrConflict, "project_key_invalid",
			"project key must be 2-5 uppercase alphanumeric characters starting with a letter", nil)
	}

	if err := s.checkReservedKey(ctx, key); err != nil {
		return nil, err
	}

	// Check for duplicate key (namespace-scoped if namespace context present)
	namespaceID := model.NamespaceIDFromContext(ctx)
	if namespaceID != uuid.Nil {
		existing, err := s.projects.GetByKeyAndNamespace(ctx, namespaceID, key)
		if err == nil && existing != nil {
			return nil, model.NewKeyedError(model.ErrAlreadyExists, "project_key_in_use",
				fmt.Sprintf("project key %q already in use", key), map[string]string{"key": key})
		}
		if err != nil && err != model.ErrNotFound {
			return nil, fmt.Errorf("checking project key: %w", err)
		}
	} else {
		existing, err := s.projects.GetByKey(ctx, key)
		if err == nil && existing != nil {
			return nil, model.NewKeyedError(model.ErrAlreadyExists, "project_key_in_use",
				fmt.Sprintf("project key %q already in use", key), map[string]string{"key": key})
		}
		if err != nil && err != model.ErrNotFound {
			return nil, fmt.Errorf("checking project key: %w", err)
		}
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

	// Set namespace from context
	var nsPtr *uuid.UUID
	if namespaceID != uuid.Nil {
		nsPtr = &namespaceID
	}

	project := &model.Project{
		ID:                uuid.New(),
		Name:              name,
		Key:               key,
		Description:       description,
		DefaultWorkflowID: defaultWorkflowID,
		NamespaceID:       nsPtr,
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

	// Publish embed.index event for semantic search
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeProject, created.ID, &created.ID)

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

// List returns projects visible to the authenticated user, filtered by namespace context.
func (s *ProjectService) List(ctx context.Context, info *model.AuthInfo) ([]model.Project, error) {
	var projects []model.Project
	var err error
	if info.GlobalRole == model.RoleAdmin && !s.adminHidesNonMemberProjects(ctx, info.UserID) {
		projects, err = s.projects.ListAll(ctx)
	} else {
		projects, err = s.projects.ListByUser(ctx, info.UserID)
	}
	if err != nil {
		return nil, err
	}

	// Filter by namespace context if present
	namespaceID := model.NamespaceIDFromContext(ctx)
	if namespaceID == uuid.Nil {
		return projects, nil
	}
	filtered := make([]model.Project, 0, len(projects))
	for _, p := range projects {
		if p.NamespaceID != nil && *p.NamespaceID == namespaceID {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
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

// CountOwnedByUser returns the number of projects owned by the given user.
func (s *ProjectService) CountOwnedByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.projects.CountByOwner(ctx, userID)
}

// ResolveProjectNamespaces returns namespace info for the given project keys.
func (s *ProjectService) ResolveProjectNamespaces(ctx context.Context, projectKeys []string) (map[string]model.ProjectNamespaceInfo, error) {
	return s.projects.ResolveNamespaces(ctx, projectKeys)
}

// ResolveProjectNamespacesByIDs returns namespace info keyed by project ID.
func (s *ProjectService) ResolveProjectNamespacesByIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectNamespaceInfo, error) {
	return s.projects.ResolveNamespacesByIDs(ctx, projectIDs)
}

// resolveProjectLimit returns the effective project limit for a user.
// Priority: per-user MaxProjects > global system setting > hardcoded default.
func (s *ProjectService) resolveProjectLimit(ctx context.Context, userID uuid.UUID) (int, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		// If user not found, fall back to global limit
		log.Ctx(ctx).Warn().Err(err).Msg("could not fetch user for project limit; using global default")
		return s.globalProjectLimit(ctx), nil
	}
	if user.MaxProjects != nil {
		return *user.MaxProjects, nil
	}
	return s.globalProjectLimit(ctx), nil
}

// globalProjectLimit returns the global max_projects_per_user setting, or the default.
func (s *ProjectService) globalProjectLimit(ctx context.Context) int {
	if setting, err := s.systemSettings.Get(ctx, model.SettingMaxProjectsPerUser); err == nil {
		var v int
		if json.Unmarshal(setting.Value, &v) == nil && v >= 0 {
			return v
		}
	}
	return model.DefaultMaxProjectsPerUser
}

// ResolveEffectiveLimit returns the effective project limit for the given auth info.
// Admins get 0 (unlimited).
func (s *ProjectService) ResolveEffectiveLimit(ctx context.Context, info *model.AuthInfo) (int, error) {
	if info.GlobalRole == model.RoleAdmin {
		return 0, nil
	}
	return s.resolveProjectLimit(ctx, info.UserID)
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
		if err := validateBusinessHours(businessHours); err != nil {
			return nil, err
		}
		project.BusinessHours = businessHours
	}

	if err := s.projects.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	// Reindex project embedding
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeProject, project.ID, &project.ID)

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

	// Clean up user-facing references to the project's work items
	inboxRemoved, err := s.inboxItems.RemoveByProjectID(ctx, project.ID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to remove inbox items for deleted project")
	}
	watchersRemoved, err := s.watchers.RemoveByProjectID(ctx, project.ID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to remove watchers for deleted project")
	}
	if err := s.invites.DeleteByProject(ctx, project.ID); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to remove invites for deleted project")
	}

	// Delete project embedding
	publishEmbedDelete(ctx, s.publisher, s.embedCache, model.EntityTypeProject, project.ID)

	log.Ctx(ctx).Info().
		Str("project_id", project.ID.String()).
		Str("project_key", project.Key).
		Str("user_id", info.UserID.String()).
		Int("inbox_items_removed", inboxRemoved).
		Int("watchers_removed", watchersRemoved).
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

	// Publish member-added notification (best-effort)
	s.publishMemberAdded(ctx, project, userID, info.UserID, role)

	return member, nil
}

// ListMembers returns members of a project and the total member count. Requires membership.
// Viewers only see owners, admins, and themselves to avoid leaking membership info.
// The total count always reflects the full membership regardless of filtering.
func (s *ProjectService) ListMembers(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.ProjectMemberWithUser, int, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, 0, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, 0, err
	}

	members, err := s.members.ListByProject(ctx, project.ID)
	if err != nil {
		return nil, 0, err
	}

	totalCount := len(members)

	// Viewers only see owners, admins, and themselves
	if info.GlobalRole != model.RoleAdmin {
		callerRole := ""
		for _, m := range members {
			if m.UserID == info.UserID {
				callerRole = m.Role
				break
			}
		}
		if callerRole == model.ProjectRoleViewer {
			filtered := make([]model.ProjectMemberWithUser, 0, len(members))
			for _, m := range members {
				if m.Role == model.ProjectRoleOwner || m.Role == model.ProjectRoleAdmin || m.UserID == info.UserID {
					filtered = append(filtered, m)
				}
			}
			return filtered, totalCount, nil
		}
	}

	return members, totalCount, nil
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

// SeedDefaultTypeWorkflows ensures the default_type_workflows system setting exists.
// If the setting is absent, it builds the default mapping from the seeded Task and Ticket workflows.
// This is idempotent and safe to call on every startup.
func (s *ProjectService) SeedDefaultTypeWorkflows(ctx context.Context) error {
	// Check if the setting already exists
	_, err := s.systemSettings.Get(ctx, model.SettingDefaultTypeWorkflows)
	if err == nil {
		return nil // already set
	}
	if err != model.ErrNotFound {
		return fmt.Errorf("checking default type workflows setting: %w", err)
	}

	workflows, err := s.workflows.List(ctx)
	if err != nil {
		return fmt.Errorf("listing workflows: %w", err)
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
		return nil // no workflows available
	}

	typeMapping := map[string]string{
		model.WorkItemTypeTask:     taskWfID.String(),
		model.WorkItemTypeBug:      taskWfID.String(),
		model.WorkItemTypeEpic:     taskWfID.String(),
		model.WorkItemTypeTicket:   ticketWfID.String(),
		model.WorkItemTypeFeedback: ticketWfID.String(),
	}

	value, err := json.Marshal(typeMapping)
	if err != nil {
		return fmt.Errorf("marshalling default type workflows: %w", err)
	}

	setting := &model.SystemSetting{
		Key:   model.SettingDefaultTypeWorkflows,
		Value: value,
	}
	if err := s.systemSettings.Upsert(ctx, setting); err != nil {
		return fmt.Errorf("saving default type workflows setting: %w", err)
	}

	log.Ctx(ctx).Info().Msg("seeded default type-workflow mappings")
	return nil
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
// It reads the default_type_workflows system setting first, falling back to
// hardcoded name-based lookup if the setting is absent or invalid.
func (s *ProjectService) seedTypeWorkflows(ctx context.Context, projectID uuid.UUID, workflows []model.Workflow) {
	if len(workflows) == 0 {
		return
	}

	// Build a lookup of available workflow IDs for validation
	wfByID := make(map[string]bool, len(workflows))
	for _, wf := range workflows {
		wfByID[wf.ID.String()] = true
	}

	// Try to use the system setting
	typeMapping := s.resolveDefaultTypeWorkflows(ctx, wfByID)

	// Fallback: hardcoded name-based lookup
	if len(typeMapping) == 0 {
		var taskWfID, ticketWfID uuid.UUID
		for _, wf := range workflows {
			switch wf.Name {
			case "Task Workflow":
				taskWfID = wf.ID
			case "Ticket Workflow":
				ticketWfID = wf.ID
			}
		}
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
		typeMapping = map[string]uuid.UUID{
			model.WorkItemTypeTask:     taskWfID,
			model.WorkItemTypeBug:      taskWfID,
			model.WorkItemTypeEpic:     taskWfID,
			model.WorkItemTypeTicket:   ticketWfID,
			model.WorkItemTypeFeedback: ticketWfID,
		}
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

// resolveDefaultTypeWorkflows reads the default_type_workflows system setting and
// returns a validated map of type → workflow UUID. Returns nil if the setting is absent or invalid.
func (s *ProjectService) resolveDefaultTypeWorkflows(ctx context.Context, validWfIDs map[string]bool) map[string]uuid.UUID {
	setting, err := s.systemSettings.Get(ctx, model.SettingDefaultTypeWorkflows)
	if err != nil {
		return nil
	}

	var raw map[string]string
	if err := json.Unmarshal(setting.Value, &raw); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("invalid default_type_workflows setting")
		return nil
	}

	result := make(map[string]uuid.UUID, len(raw))
	for itemType, wfIDStr := range raw {
		if !validWfIDs[wfIDStr] {
			log.Ctx(ctx).Warn().Str("work_item_type", itemType).Str("workflow_id", wfIDStr).
				Msg("default_type_workflows references unknown workflow, skipping")
			continue
		}
		wfID, err := uuid.Parse(wfIDStr)
		if err != nil {
			continue
		}
		result[itemType] = wfID
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// CreateInvite creates a new invite link for a project. Requires owner or admin role.
func (s *ProjectService) CreateInvite(ctx context.Context, info *model.AuthInfo, projectKey, role string, expiresAt *time.Time, maxUses int) (*model.ProjectInvite, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if !isValidInviteRole(role) {
		return nil, fmt.Errorf("invalid invite role %q (owner not allowed): %w", role, model.ErrValidation)
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("generating invite code: %w", err)
	}

	invite := &model.ProjectInvite{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Code:      code,
		Role:      role,
		CreatedBy: info.UserID,
		ExpiresAt: expiresAt,
		MaxUses:   maxUses,
	}

	if err := s.invites.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("creating invite: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("invite_code", code).
		Str("role", role).
		Msg("project invite created")

	return invite, nil
}

// CreateEmailInviteResult contains the result of an email-based invite attempt.
type CreateEmailInviteResult struct {
	Invite      *model.ProjectInvite  // Non-nil when an invite was created (user doesn't exist)
	Member      *model.ProjectMember  // Non-nil when user was added directly (user exists)
	DirectAdd   bool                  // True if user already existed and was added directly
}

// CreateEmailInvite handles an email-based invite. If the user exists, adds them directly;
// otherwise creates a personal invite and sends an email notification.
func (s *ProjectService) CreateEmailInvite(ctx context.Context, info *model.AuthInfo, projectKey, email, role string, expiresAt *time.Time) (*CreateEmailInviteResult, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	if !isValidInviteRole(role) {
		return nil, fmt.Errorf("invalid invite role %q (owner not allowed): %w", role, model.ErrValidation)
	}

	// Only owners can invite as owner (shouldn't reach here due to isValidInviteRole, but be safe)
	if role == model.ProjectRoleOwner {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner); err != nil {
			return nil, model.ErrForbidden
		}
	}

	// Check if the user already exists
	existingUser, err := s.users.GetByEmail(ctx, email)
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("looking up user by email: %w", err)
	}

	if existingUser != nil {
		// User exists — add them directly
		existing, err := s.members.GetByProjectAndUser(ctx, project.ID, existingUser.ID)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("user is already a member of this project: %w", model.ErrAlreadyExists)
		}
		if err != nil && err != model.ErrNotFound {
			return nil, fmt.Errorf("checking membership: %w", err)
		}

		member := &model.ProjectMember{
			ID:        uuid.New(),
			ProjectID: project.ID,
			UserID:    existingUser.ID,
			Role:      role,
		}
		if err := s.members.Add(ctx, member); err != nil {
			return nil, fmt.Errorf("adding member: %w", err)
		}

		log.Ctx(ctx).Info().
			Str("project_key", projectKey).
			Str("email", email).
			Str("role", role).
			Msg("user added directly via email invite (existing user)")

		// Publish member-added notification (reuse existing flow)
		s.publishMemberAdded(ctx, project, existingUser.ID, info.UserID, role)

		return &CreateEmailInviteResult{Member: member, DirectAdd: true}, nil
	}

	// User doesn't exist — create a personal invite and send email
	code, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("generating invite code: %w", err)
	}

	invite := &model.ProjectInvite{
		ID:           uuid.New(),
		ProjectID:    project.ID,
		Code:         code,
		Role:         role,
		CreatedBy:    info.UserID,
		InviteeEmail: &email,
		ExpiresAt:    expiresAt,
		MaxUses:      1,
	}

	if err := s.invites.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("creating invite: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("invite_code", code).
		Str("invitee_email", email).
		Str("role", role).
		Msg("email invite created for non-existing user")

	// Publish invite email notification
	s.publishInviteEmail(ctx, project, info, email, code, role)

	return &CreateEmailInviteResult{Invite: invite}, nil
}

func (s *ProjectService) publishInviteEmail(ctx context.Context, project *model.Project, info *model.AuthInfo, email, code, role string) {
	if s.publisher == nil {
		return
	}
	inviter, err := s.users.GetByID(ctx, info.UserID)
	inviterName := "A team member"
	if err == nil {
		inviterName = inviter.DisplayName
	}
	evt := model.InviteEmailEvent{
		ProjectKey:   project.Key,
		ProjectName:  project.Name,
		InviteeEmail: email,
		InviterName:  inviterName,
		InviteCode:   code,
		Role:         role,
	}
	if err := s.publisher.Publish("notification.invite_email", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish invite email notification")
	}
}

// ListInvites returns all invites for a project. Requires owner or admin role.
func (s *ProjectService) ListInvites(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.ProjectInvite, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	return s.invites.ListByProject(ctx, project.ID)
}

// DeleteInvite revokes an invite link. Requires owner or admin role.
func (s *ProjectService) DeleteInvite(ctx context.Context, info *model.AuthInfo, projectKey string, inviteID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	// Verify the invite belongs to this project
	invite, err := s.invites.GetByID(ctx, inviteID)
	if err != nil {
		return err
	}
	if invite.ProjectID != project.ID {
		return model.ErrNotFound
	}

	if err := s.invites.Delete(ctx, inviteID); err != nil {
		return fmt.Errorf("deleting invite: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("invite_id", inviteID.String()).
		Msg("project invite revoked")

	return nil
}

// GetInviteInfo returns public information about an invite for the join page.
// No authentication required.
func (s *ProjectService) GetInviteInfo(ctx context.Context, code string) (*model.ProjectInviteInfo, error) {
	invite, err := s.invites.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.GetByID(ctx, invite.ProjectID)
	if err != nil {
		return nil, err
	}

	// Don't expose invites for deleted projects
	if project.DeletedAt != nil {
		return nil, model.ErrNotFound
	}

	info := &model.ProjectInviteInfo{
		ProjectName: project.Name,
		ProjectKey:  project.Key,
		Role:        invite.Role,
		Expired:     invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()),
		Full:        invite.MaxUses > 0 && invite.UseCount >= invite.MaxUses,
	}

	return info, nil
}

// AcceptInviteResult contains the result of accepting an invite.
type AcceptInviteResult struct {
	Project        *model.Project
	RoleNotApplied bool   // true if user already had a higher role
	ExistingRole   string // populated when RoleNotApplied is true
	InviteRole     string // populated when RoleNotApplied is true
}

// roleRank returns a numeric rank for project roles (higher = more access).
func roleRank(role string) int {
	switch role {
	case model.ProjectRoleOwner:
		return 4
	case model.ProjectRoleAdmin:
		return 3
	case model.ProjectRoleMember:
		return 2
	case model.ProjectRoleViewer:
		return 1
	default:
		return 0
	}
}

// AcceptInvite uses an invite code to join a project. Requires authentication.
func (s *ProjectService) AcceptInvite(ctx context.Context, info *model.AuthInfo, code string) (*AcceptInviteResult, error) {
	invite, err := s.invites.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.GetByID(ctx, invite.ProjectID)
	if err != nil {
		return nil, err
	}

	if project.DeletedAt != nil {
		return nil, model.ErrNotFound
	}

	// Check expiry
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("invite has expired: %w", model.ErrValidation)
	}

	// Check max uses
	if invite.MaxUses > 0 && invite.UseCount >= invite.MaxUses {
		return nil, fmt.Errorf("invite has reached maximum uses: %w", model.ErrValidation)
	}

	// Check if already a member — only upgrade, never downgrade
	result := &AcceptInviteResult{Project: project}
	existing, err := s.members.GetByProjectAndUser(ctx, project.ID, info.UserID)
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("checking membership: %w", err)
	}
	if existing != nil {
		if roleRank(invite.Role) > roleRank(existing.Role) {
			// Upgrade to the higher role from the invite
			if err := s.members.UpdateRole(ctx, project.ID, info.UserID, invite.Role); err != nil {
				return nil, fmt.Errorf("updating member role: %w", err)
			}
			log.Ctx(ctx).Info().
				Str("project_key", project.Key).
				Str("user_id", info.UserID.String()).
				Str("old_role", existing.Role).
				Str("new_role", invite.Role).
				Msg("upgraded member role via invite")
		} else if roleRank(invite.Role) < roleRank(existing.Role) {
			// Invite would downgrade — ignore but inform the caller
			result.RoleNotApplied = true
			result.ExistingRole = existing.Role
			result.InviteRole = invite.Role
			log.Ctx(ctx).Info().
				Str("project_key", project.Key).
				Str("user_id", info.UserID.String()).
				Str("existing_role", existing.Role).
				Str("invite_role", invite.Role).
				Msg("invite role not applied: user already has higher access")
		}
		// Same role: no-op
	} else {
		// Add as new member
		member := &model.ProjectMember{
			ID:        uuid.New(),
			ProjectID: project.ID,
			UserID:    info.UserID,
			Role:      invite.Role,
		}
		if err := s.members.Add(ctx, member); err != nil {
			return nil, fmt.Errorf("adding member: %w", err)
		}
	}

	// Increment use count (atomic, respects max_uses)
	if err := s.invites.IncrementUseCount(ctx, invite.ID); err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("invite_id", invite.ID.String()).Msg("failed to increment invite use count")
	}

	log.Ctx(ctx).Info().
		Str("project_key", project.Key).
		Str("user_id", info.UserID.String()).
		Str("role", invite.Role).
		Str("invite_code", code).
		Msg("user joined project via invite")

	return result, nil
}

const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateInviteCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := make([]byte, 8)
	for i := range b {
		code[i] = base62Chars[int(b[i])%len(base62Chars)]
	}
	return string(code), nil
}

func isValidInviteRole(role string) bool {
	switch role {
	case model.ProjectRoleAdmin, model.ProjectRoleMember, model.ProjectRoleViewer:
		return true
	}
	return false
}

// requireMembership checks that the user is a member of the project or a global admin.
// Returns ErrNotFound (not ErrForbidden) to avoid leaking project existence.
// adminHidesNonMemberProjects checks whether the admin user has enabled the
// "hide_non_member_projects" global preference. Returns false on any error.
func (s *ProjectService) adminHidesNonMemberProjects(ctx context.Context, userID uuid.UUID) bool {
	if s.userSettings == nil {
		return false
	}
	setting, err := s.userSettings.Get(ctx, userID, nil, "hide_non_member_projects")
	if err != nil {
		return false
	}
	var hide bool
	if err := json.Unmarshal(setting.Value, &hide); err != nil {
		return false
	}
	return hide
}

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

func validateBusinessHours(bh *model.BusinessHoursConfig) error {
	if bh.Timezone == "" {
		return fmt.Errorf("timezone is required: %w", model.ErrValidation)
	}
	if _, err := time.LoadLocation(bh.Timezone); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", bh.Timezone, model.ErrValidation)
	}
	if bh.StartHour < 0 || bh.StartHour > 23 {
		return fmt.Errorf("start_hour must be 0-23: %w", model.ErrValidation)
	}
	if bh.EndHour < 0 || bh.EndHour > 23 {
		return fmt.Errorf("end_hour must be 0-23: %w", model.ErrValidation)
	}
	if bh.EndHour <= bh.StartHour {
		return fmt.Errorf("end_hour must be greater than start_hour: %w", model.ErrValidation)
	}
	if len(bh.Days) == 0 {
		return fmt.Errorf("at least one business day is required: %w", model.ErrValidation)
	}
	for _, d := range bh.Days {
		if d < 0 || d > 6 {
			return fmt.Errorf("invalid day %d, must be 0-6: %w", d, model.ErrValidation)
		}
	}
	return nil
}

// publishMemberAdded publishes a member-added notification event (best-effort).
func (s *ProjectService) publishMemberAdded(_ context.Context, project *model.Project, userID, addedByID uuid.UUID, role string) {
	if s.publisher == nil {
		return
	}
	// Don't notify when users add themselves (shouldn't happen, but be safe)
	if userID == addedByID {
		return
	}
	evt := model.MemberAddedEvent{
		ProjectID:   project.ID,
		ProjectKey:  project.Key,
		ProjectName: project.Name,
		UserID:      userID,
		AddedByID:   addedByID,
		Role:        role,
	}
	if err := s.publisher.Publish("notification.member_added", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish member added notification")
	}
}

// checkReservedKey checks the project key against the admin-managed deny list.
func (s *ProjectService) checkReservedKey(ctx context.Context, key string) error {
	setting, err := s.systemSettings.Get(ctx, model.SettingReservedProjectKeys)
	if err != nil {
		return nil // no deny list configured — allow
	}
	var denyList []string
	if err := json.Unmarshal(setting.Value, &denyList); err != nil {
		return nil // malformed — allow
	}
	for _, reserved := range denyList {
		if reserved == key {
			return model.NewKeyedError(model.ErrValidation, "project_key_reserved",
				fmt.Sprintf("project key %q is reserved", key), map[string]string{"key": key})
		}
	}
	return nil
}
