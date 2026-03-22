package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

var namespaceSlugRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]{1,29}$`)

// Reserved namespace slugs that cannot be used.
var reservedSlugs = map[string]bool{
	"default": true,
	"api":     true,
	"admin":   true,
	"system":  true,
}

// NamespaceRepository defines persistence operations for namespaces.
type NamespaceRepository interface {
	Create(ctx context.Context, ns *model.Namespace) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Namespace, error)
	GetBySlug(ctx context.Context, slug string) (*model.Namespace, error)
	GetDefault(ctx context.Context) (*model.Namespace, error)
	List(ctx context.Context) ([]model.Namespace, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Namespace, error)
	Update(ctx context.Context, ns *model.Namespace) error
	Delete(ctx context.Context, id uuid.UUID) error
	HasProjects(ctx context.Context, id uuid.UUID) (bool, error)
	CountNonDefault(ctx context.Context) (int, error)
}

// NamespaceMemberRepository defines persistence operations for namespace members.
type NamespaceMemberRepository interface {
	Add(ctx context.Context, member *model.NamespaceMember) error
	GetByNamespaceAndUser(ctx context.Context, namespaceID, userID uuid.UUID) (*model.NamespaceMember, error)
	ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]model.NamespaceMemberWithUser, error)
	UpdateRole(ctx context.Context, namespaceID, userID uuid.UUID, role string) error
	Remove(ctx context.Context, namespaceID, userID uuid.UUID) error
	RemoveAllByNamespace(ctx context.Context, namespaceID uuid.UUID) error
	CountByRole(ctx context.Context, namespaceID uuid.UUID, role string) (int, error)
	CountOwnedByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// NamespaceProjectRepository defines the project operations needed by the namespace service.
type NamespaceProjectRepository interface {
	ListAll(ctx context.Context) ([]model.Project, error)
	SetNamespaceID(ctx context.Context, projectID, namespaceID uuid.UUID) error
	GetByKeyAndNamespace(ctx context.Context, namespaceID uuid.UUID, key string) (*model.Project, error)
}

// NamespaceUserRepository defines the user operations needed by the namespace service.
type NamespaceUserRepository interface {
	ListAll(ctx context.Context) ([]model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// NamespaceSystemSettingsRepository defines the system settings operations needed by the namespace service.
type NamespaceSystemSettingsRepository interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
}

// NamespaceUserSettingsRepository defines user setting operations needed by the namespace service.
type NamespaceUserSettingsRepository interface {
	Get(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) (*model.UserSetting, error)
}

// NamespaceService handles namespace business logic.
type NamespaceService struct {
	namespaces     NamespaceRepository
	members        NamespaceMemberRepository
	projects       NamespaceProjectRepository
	users          NamespaceUserRepository
	systemSettings NamespaceSystemSettingsRepository
	userSettings   NamespaceUserSettingsRepository
}

// NewNamespaceService creates a new NamespaceService.
func NewNamespaceService(namespaces NamespaceRepository, members NamespaceMemberRepository, projects NamespaceProjectRepository, users NamespaceUserRepository, systemSettings NamespaceSystemSettingsRepository, userSettings NamespaceUserSettingsRepository) *NamespaceService {
	return &NamespaceService{
		namespaces:     namespaces,
		members:        members,
		projects:       projects,
		users:          users,
		systemSettings: systemSettings,
		userSettings:   userSettings,
	}
}

// --- Namespace CRUD ---

// CreateNamespace creates a new namespace. Requires namespaces to be enabled.
func (s *NamespaceService) CreateNamespace(ctx context.Context, info *model.AuthInfo, slug, displayName string) (*model.Namespace, error) {
	if err := s.requireNamespacesEnabled(ctx); err != nil {
		return nil, err
	}

	// Enforce per-user namespace limit (admins are exempt)
	if info.GlobalRole != model.RoleAdmin {
		limit, err := s.resolveNamespaceLimit(ctx, info.UserID)
		if err != nil {
			return nil, fmt.Errorf("resolving namespace limit: %w", err)
		}
		if limit > 0 {
			count, err := s.members.CountOwnedByUser(ctx, info.UserID)
			if err != nil {
				return nil, fmt.Errorf("counting user namespaces: %w", err)
			}
			if count >= limit {
				return nil, fmt.Errorf("namespace ownership limit reached; contact an administrator to increase your limit: %w", model.ErrForbidden)
			}
		}
	}

	if !namespaceSlugRegexp.MatchString(slug) {
		return nil, fmt.Errorf("namespace slug must be 2-30 lowercase alphanumeric characters or hyphens, starting with a letter: %w", model.ErrValidation)
	}

	if reservedSlugs[slug] {
		return nil, fmt.Errorf("namespace slug %q is reserved: %w", slug, model.ErrValidation)
	}

	if err := s.checkReservedSlug(ctx, slug); err != nil {
		return nil, err
	}

	if displayName == "" {
		return nil, fmt.Errorf("display_name is required: %w", model.ErrValidation)
	}

	// Check for duplicate slug
	existing, err := s.namespaces.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("namespace slug %q already in use: %w", slug, model.ErrAlreadyExists)
	}
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("checking namespace slug: %w", err)
	}

	ns := &model.Namespace{
		ID:          uuid.New(),
		Slug:        slug,
		DisplayName: displayName,
		Icon:        "building2",
		Color:       "slate",
		IsDefault:   false,
		CreatedBy:   info.UserID,
	}

	if err := s.namespaces.Create(ctx, ns); err != nil {
		return nil, fmt.Errorf("creating namespace: %w", err)
	}

	// Add creator as namespace owner
	member := &model.NamespaceMember{
		NamespaceID: ns.ID,
		UserID:      info.UserID,
		Role:        model.NamespaceRoleOwner,
	}
	if err := s.members.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("adding namespace owner: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_id", ns.ID.String()).
		Str("slug", ns.Slug).
		Str("user_id", info.UserID.String()).
		Msg("namespace created")

	return s.namespaces.GetByID(ctx, ns.ID)
}

// GetNamespace returns a namespace by slug. User must be a member or global admin.
func (s *NamespaceService) GetNamespace(ctx context.Context, info *model.AuthInfo, slug string) (*model.Namespace, error) {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if err := s.requireNamespaceMembership(ctx, info, ns.ID); err != nil {
		return nil, err
	}

	return ns, nil
}

// ListUserNamespaces returns all namespaces the user belongs to.
func (s *NamespaceService) ListUserNamespaces(ctx context.Context, info *model.AuthInfo) ([]model.Namespace, error) {
	if info.GlobalRole == model.RoleAdmin && !s.adminHidesNonMember(ctx, info.UserID) {
		return s.namespaces.List(ctx)
	}
	return s.namespaces.ListByUser(ctx, info.UserID)
}

// UpdateNamespace modifies a namespace. Requires owner or admin role in the namespace.
func (s *NamespaceService) UpdateNamespace(ctx context.Context, info *model.AuthInfo, slug string, newSlug, displayName, icon, color *string) (*model.Namespace, error) {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if ns.IsDefault {
		return nil, fmt.Errorf("cannot modify the default namespace: %w", model.ErrForbidden)
	}

	if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner, model.NamespaceRoleAdmin); err != nil {
		return nil, err
	}

	if newSlug != nil {
		if !namespaceSlugRegexp.MatchString(*newSlug) {
			return nil, fmt.Errorf("namespace slug must be 2-30 lowercase alphanumeric characters or hyphens, starting with a letter: %w", model.ErrValidation)
		}
		if reservedSlugs[*newSlug] {
			return nil, fmt.Errorf("namespace slug %q is reserved: %w", *newSlug, model.ErrValidation)
		}
		if err := s.checkReservedSlug(ctx, *newSlug); err != nil {
			return nil, err
		}
		// Check for slug collision
		if *newSlug != ns.Slug {
			existing, err := s.namespaces.GetBySlug(ctx, *newSlug)
			if err == nil && existing != nil {
				return nil, fmt.Errorf("namespace slug %q already in use: %w", *newSlug, model.ErrAlreadyExists)
			}
			if err != nil && err != model.ErrNotFound {
				return nil, fmt.Errorf("checking namespace slug: %w", err)
			}
		}
		ns.Slug = *newSlug
	}

	if displayName != nil {
		if *displayName == "" {
			return nil, fmt.Errorf("display_name cannot be empty: %w", model.ErrValidation)
		}
		ns.DisplayName = *displayName
	}

	if icon != nil {
		if !model.ValidNamespaceIcons[*icon] {
			return nil, fmt.Errorf("invalid namespace icon %q: %w", *icon, model.ErrValidation)
		}
		ns.Icon = *icon
	}

	if color != nil {
		if !model.ValidNamespaceColors[*color] {
			return nil, fmt.Errorf("invalid namespace color %q: %w", *color, model.ErrValidation)
		}
		ns.Color = *color
	}

	if err := s.namespaces.Update(ctx, ns); err != nil {
		return nil, fmt.Errorf("updating namespace: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_id", ns.ID.String()).
		Str("slug", ns.Slug).
		Msg("namespace updated")

	return s.namespaces.GetBySlug(ctx, ns.Slug)
}

// DeleteNamespace removes a namespace. Requires owner role; namespace must be empty.
func (s *NamespaceService) DeleteNamespace(ctx context.Context, info *model.AuthInfo, slug string) error {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if ns.IsDefault {
		return fmt.Errorf("cannot delete the default namespace: %w", model.ErrForbidden)
	}

	if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner); err != nil {
		return err
	}

	hasProjects, err := s.namespaces.HasProjects(ctx, ns.ID)
	if err != nil {
		return fmt.Errorf("checking namespace projects: %w", err)
	}
	if hasProjects {
		return fmt.Errorf("namespace still contains projects; migrate or delete them first: %w", model.ErrNamespaceNotEmpty)
	}

	// Remove all members before deleting (FK constraint)
	if err := s.members.RemoveAllByNamespace(ctx, ns.ID); err != nil {
		return fmt.Errorf("removing namespace members: %w", err)
	}

	if err := s.namespaces.Delete(ctx, ns.ID); err != nil {
		return fmt.Errorf("deleting namespace: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_id", ns.ID.String()).
		Str("slug", ns.Slug).
		Str("user_id", info.UserID.String()).
		Msg("namespace deleted")

	return nil
}

// --- Namespace Member Management ---

// AddNamespaceMember adds a user to a namespace. Requires owner or admin role.
func (s *NamespaceService) AddNamespaceMember(ctx context.Context, info *model.AuthInfo, slug string, targetUserID uuid.UUID, role string) (*model.NamespaceMember, error) {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner, model.NamespaceRoleAdmin); err != nil {
		return nil, err
	}

	if !isValidNamespaceRole(role) {
		return nil, fmt.Errorf("invalid namespace role %q: %w", role, model.ErrValidation)
	}

	// Only owners can add other owners
	if role == model.NamespaceRoleOwner {
		if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner); err != nil {
			return nil, fmt.Errorf("only owners can add other owners: %w", model.ErrForbidden)
		}
	}

	// Enforce namespace limit when adding as owner
	if role == model.NamespaceRoleOwner {
		if err := s.checkNamespaceLimit(ctx, targetUserID); err != nil {
			return nil, err
		}
	}

	// Verify user exists
	if _, err := s.users.GetByID(ctx, targetUserID); err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	// Check if already a member
	existing, err := s.members.GetByNamespaceAndUser(ctx, ns.ID, targetUserID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("user is already a member of this namespace: %w", model.ErrAlreadyExists)
	}
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("checking membership: %w", err)
	}

	member := &model.NamespaceMember{
		NamespaceID: ns.ID,
		UserID:      targetUserID,
		Role:        role,
	}

	if err := s.members.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("adding namespace member: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_slug", slug).
		Str("user_id", targetUserID.String()).
		Str("role", role).
		Msg("member added to namespace")

	return member, nil
}

// ListNamespaceMembers returns members of a namespace. Requires membership or admin.
func (s *NamespaceService) ListNamespaceMembers(ctx context.Context, info *model.AuthInfo, slug string) ([]model.NamespaceMemberWithUser, error) {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if err := s.requireNamespaceMembership(ctx, info, ns.ID); err != nil {
		return nil, err
	}

	return s.members.ListByNamespace(ctx, ns.ID)
}

// UpdateNamespaceMemberRole changes a member's role. Owner only.
func (s *NamespaceService) UpdateNamespaceMemberRole(ctx context.Context, info *model.AuthInfo, slug string, targetUserID uuid.UUID, role string) error {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner); err != nil {
		return err
	}

	if !isValidNamespaceRole(role) {
		return fmt.Errorf("invalid namespace role %q: %w", role, model.ErrValidation)
	}

	// Get target member to check current role
	target, err := s.members.GetByNamespaceAndUser(ctx, ns.ID, targetUserID)
	if err != nil {
		return err
	}

	// Enforce namespace limit when promoting to owner
	if role == model.NamespaceRoleOwner && target.Role != model.NamespaceRoleOwner {
		if err := s.checkNamespaceLimit(ctx, targetUserID); err != nil {
			return err
		}
	}

	// Protect last owner
	if target.Role == model.NamespaceRoleOwner && role != model.NamespaceRoleOwner {
		count, err := s.members.CountByRole(ctx, ns.ID, model.NamespaceRoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return fmt.Errorf("cannot demote the last owner: %w", model.ErrConflict)
		}
	}

	if err := s.members.UpdateRole(ctx, ns.ID, targetUserID, role); err != nil {
		return fmt.Errorf("updating namespace member role: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_slug", slug).
		Str("user_id", targetUserID.String()).
		Str("new_role", role).
		Msg("namespace member role updated")

	return nil
}

// RemoveNamespaceMember removes a user from a namespace. Owner/admin only. Cannot remove last owner.
func (s *NamespaceService) RemoveNamespaceMember(ctx context.Context, info *model.AuthInfo, slug string, targetUserID uuid.UUID) error {
	ns, err := s.namespaces.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner, model.NamespaceRoleAdmin); err != nil {
		return err
	}

	// Get target member to check role
	target, err := s.members.GetByNamespaceAndUser(ctx, ns.ID, targetUserID)
	if err != nil {
		return err
	}

	// Only owners can remove owners
	if target.Role == model.NamespaceRoleOwner {
		if err := s.requireNamespaceRole(ctx, info, ns.ID, model.NamespaceRoleOwner); err != nil {
			return fmt.Errorf("only owners can remove other owners: %w", model.ErrForbidden)
		}

		// Protect last owner
		count, err := s.members.CountByRole(ctx, ns.ID, model.NamespaceRoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return fmt.Errorf("cannot remove the last owner: %w", model.ErrConflict)
		}
	}

	if err := s.members.Remove(ctx, ns.ID, targetUserID); err != nil {
		return fmt.Errorf("removing namespace member: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_slug", slug).
		Str("user_id", targetUserID.String()).
		Msg("member removed from namespace")

	return nil
}

// --- Project Migration ---

// MigrateProject moves a project from one namespace to another.
func (s *NamespaceService) MigrateProject(ctx context.Context, info *model.AuthInfo, projectKey, fromSlug, toSlug string) error {
	fromNs, err := s.namespaces.GetBySlug(ctx, fromSlug)
	if err != nil {
		return fmt.Errorf("source namespace: %w", err)
	}

	toNs, err := s.namespaces.GetBySlug(ctx, toSlug)
	if err != nil {
		return fmt.Errorf("target namespace: %w", err)
	}

	// Actor must be admin/owner in source namespace (or global admin)
	if err := s.requireNamespaceRole(ctx, info, fromNs.ID, model.NamespaceRoleOwner, model.NamespaceRoleAdmin); err != nil {
		return fmt.Errorf("insufficient permissions in source namespace: %w", model.ErrForbidden)
	}

	// Actor must be admin/owner in target namespace (or global admin)
	if err := s.requireNamespaceRole(ctx, info, toNs.ID, model.NamespaceRoleOwner, model.NamespaceRoleAdmin); err != nil {
		return fmt.Errorf("insufficient permissions in target namespace: %w", model.ErrForbidden)
	}

	// Find the project in the source namespace
	project, err := s.projects.GetByKeyAndNamespace(ctx, fromNs.ID, projectKey)
	if err != nil {
		return fmt.Errorf("project not found in source namespace: %w", err)
	}

	// Check key doesn't collide in target namespace
	existing, err := s.projects.GetByKeyAndNamespace(ctx, toNs.ID, projectKey)
	if err == nil && existing != nil {
		return fmt.Errorf("project key %q already exists in target namespace %q: %w", projectKey, toSlug, model.ErrAlreadyExists)
	}
	if err != nil && err != model.ErrNotFound {
		return fmt.Errorf("checking target namespace: %w", err)
	}

	// Move the project
	if err := s.projects.SetNamespaceID(ctx, project.ID, toNs.ID); err != nil {
		return fmt.Errorf("migrating project: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("from_namespace", fromSlug).
		Str("to_namespace", toSlug).
		Str("user_id", info.UserID.String()).
		Msg("project migrated between namespaces")

	return nil
}

// --- Namespace Feature Flag ---

// IsNamespacesEnabled checks whether the namespaces feature is enabled.
func (s *NamespaceService) IsNamespacesEnabled(ctx context.Context) (bool, error) {
	setting, err := s.systemSettings.Get(ctx, model.SettingNamespacesEnabled)
	if err != nil {
		if err == model.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("checking namespaces setting: %w", err)
	}
	var enabled bool
	if err := json.Unmarshal(setting.Value, &enabled); err != nil {
		return false, nil
	}
	return enabled, nil
}

// GetDefault returns the default namespace.
func (s *NamespaceService) GetDefault(ctx context.Context) (*model.Namespace, error) {
	return s.namespaces.GetDefault(ctx)
}

// GetBySlug returns a namespace by slug (no auth check — used by middleware).
func (s *NamespaceService) GetBySlug(ctx context.Context, slug string) (*model.Namespace, error) {
	return s.namespaces.GetBySlug(ctx, slug)
}

// --- Startup / Seeding ---

// getBrandName reads the brand_name system setting, falling back to model.DefaultBrandName.
func (s *NamespaceService) getBrandName(ctx context.Context) string {
	setting, err := s.systemSettings.Get(ctx, "brand_name")
	if err != nil {
		return model.DefaultBrandName
	}
	var name string
	if err := json.Unmarshal(setting.Value, &name); err != nil || name == "" {
		return model.DefaultBrandName
	}
	return name
}

// UpdateDefaultNamespaceDisplayName updates the default namespace's display name
// to match the given brand name. Called when the brand_name system setting changes.
func (s *NamespaceService) UpdateDefaultNamespaceDisplayName(ctx context.Context, name string) error {
	if name == "" {
		name = model.DefaultBrandName
	}
	ns, err := s.namespaces.GetDefault(ctx)
	if err != nil {
		return fmt.Errorf("getting default namespace: %w", err)
	}
	if ns.DisplayName == name {
		return nil
	}
	ns.DisplayName = name
	if err := s.namespaces.Update(ctx, ns); err != nil {
		return fmt.Errorf("updating default namespace display name: %w", err)
	}
	log.Ctx(ctx).Info().
		Str("brand_name", name).
		Msg("default namespace display name synced with brand")
	return nil
}

// SeedDefaultNamespace creates the default namespace if it doesn't exist and backfills
// existing projects that have no namespace_id.
func (s *NamespaceService) SeedDefaultNamespace(ctx context.Context) error {
	brandName := s.getBrandName(ctx)

	// Check if default namespace already exists
	existing, err := s.namespaces.GetDefault(ctx)
	if err == nil {
		// Sync display name and icon for the default namespace
		needsUpdate := false
		if existing.DisplayName != brandName {
			existing.DisplayName = brandName
			needsUpdate = true
		}
		if existing.Icon != "globe" {
			existing.Icon = "globe"
			needsUpdate = true
		}
		if needsUpdate {
			if err := s.namespaces.Update(ctx, existing); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to sync default namespace")
			}
		}

		log.Ctx(ctx).Debug().
			Str("namespace_id", existing.ID.String()).
			Msg("default namespace already exists, checking backfill")

		// Backfill any projects without a namespace
		return s.backfillProjects(ctx, existing.ID)
	}
	if err != model.ErrNotFound {
		return fmt.Errorf("checking default namespace: %w", err)
	}

	// Find an admin user to use as creator
	adminID, err := s.findAdminUser(ctx)
	if err != nil {
		return fmt.Errorf("finding admin user for default namespace: %w", err)
	}

	ns := &model.Namespace{
		ID:          uuid.New(),
		Slug:        model.DefaultNamespaceSlug,
		DisplayName: brandName,
		Icon:        "globe",
		Color:       "slate",
		IsDefault:   true,
		CreatedBy:   adminID,
	}

	if err := s.namespaces.Create(ctx, ns); err != nil {
		return fmt.Errorf("creating default namespace: %w", err)
	}

	// Add the admin as namespace owner
	member := &model.NamespaceMember{
		NamespaceID: ns.ID,
		UserID:      adminID,
		Role:        model.NamespaceRoleOwner,
	}
	if err := s.members.Add(ctx, member); err != nil {
		return fmt.Errorf("adding default namespace owner: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("namespace_id", ns.ID.String()).
		Msg("default namespace created")

	// Backfill existing projects
	return s.backfillProjects(ctx, ns.ID)
}

// --- Helpers ---

// adminHidesNonMember checks whether the admin user has enabled the
// "hide_non_member_projects" global preference. Returns false on any error.
func (s *NamespaceService) adminHidesNonMember(ctx context.Context, userID uuid.UUID) bool {
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

// requireNamespacesEnabled checks the feature flag.
func (s *NamespaceService) requireNamespacesEnabled(ctx context.Context) error {
	enabled, err := s.IsNamespacesEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return model.ErrNamespacesDisabled
	}
	return nil
}

// requireNamespaceMembership checks that the user is a member of the namespace or a global admin.
func (s *NamespaceService) requireNamespaceMembership(ctx context.Context, info *model.AuthInfo, namespaceID uuid.UUID) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}

	_, err := s.members.GetByNamespaceAndUser(ctx, namespaceID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking namespace membership: %w", err)
	}
	return nil
}

// requireNamespaceRole checks that the user has one of the allowed roles or is a global admin.
func (s *NamespaceService) requireNamespaceRole(ctx context.Context, info *model.AuthInfo, namespaceID uuid.UUID, allowedRoles ...string) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}

	member, err := s.members.GetByNamespaceAndUser(ctx, namespaceID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking namespace membership: %w", err)
	}

	for _, role := range allowedRoles {
		if member.Role == role {
			return nil
		}
	}

	return model.ErrForbidden
}

// backfillProjects assigns the given namespace to all projects that don't have one.
func (s *NamespaceService) backfillProjects(ctx context.Context, namespaceID uuid.UUID) error {
	projects, err := s.projects.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("listing projects for backfill: %w", err)
	}

	count := 0
	for _, p := range projects {
		if p.NamespaceID != nil {
			continue
		}
		if err := s.projects.SetNamespaceID(ctx, p.ID, namespaceID); err != nil {
			return fmt.Errorf("backfilling project %s: %w", p.ID, err)
		}
		count++
	}

	if count > 0 {
		log.Ctx(ctx).Info().
			Int("count", count).
			Str("namespace_id", namespaceID.String()).
			Msg("backfilled projects with default namespace")
	}

	return nil
}

// findAdminUser returns the ID of any admin user.
func (s *NamespaceService) findAdminUser(ctx context.Context) (uuid.UUID, error) {
	users, err := s.users.ListAll(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("listing users: %w", err)
	}
	for _, u := range users {
		if u.GlobalRole == model.RoleAdmin {
			return u.ID, nil
		}
	}
	// Fallback: use the first user if no admin exists
	if len(users) > 0 {
		return users[0].ID, nil
	}
	return uuid.Nil, fmt.Errorf("no users found to create default namespace")
}

// checkNamespaceLimit checks whether the target user has reached their namespace ownership limit.
// Admins are exempt.
func (s *NamespaceService) checkNamespaceLimit(ctx context.Context, userID uuid.UUID) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("looking up user for namespace limit: %w", err)
	}
	if user.GlobalRole == model.RoleAdmin {
		return nil
	}

	limit, err := s.resolveNamespaceLimit(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolving namespace limit: %w", err)
	}
	if limit > 0 {
		count, err := s.members.CountOwnedByUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("counting user namespaces: %w", err)
		}
		if count >= limit {
			return fmt.Errorf("namespace ownership limit reached; contact an administrator to increase the limit: %w", model.ErrForbidden)
		}
	}
	return nil
}

// resolveNamespaceLimit returns the effective namespace limit for a user.
// Priority: per-user MaxNamespaces > global system setting > hardcoded default.
func (s *NamespaceService) resolveNamespaceLimit(ctx context.Context, userID uuid.UUID) (int, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("could not fetch user for namespace limit; using global default")
		return s.globalNamespaceLimit(ctx), nil
	}
	if user.MaxNamespaces != nil {
		return *user.MaxNamespaces, nil
	}
	return s.globalNamespaceLimit(ctx), nil
}

// globalNamespaceLimit returns the global max_namespaces_per_user setting, or the default.
func (s *NamespaceService) globalNamespaceLimit(ctx context.Context) int {
	if setting, err := s.systemSettings.Get(ctx, model.SettingMaxNamespacesPerUser); err == nil {
		var v int
		if json.Unmarshal(setting.Value, &v) == nil && v >= 0 {
			return v
		}
	}
	return model.DefaultMaxNamespacesPerUser
}

// ResolveEffectiveLimit returns the effective namespace limit for the given auth info.
// Admins get 0 (unlimited).
func (s *NamespaceService) ResolveEffectiveLimit(ctx context.Context, info *model.AuthInfo) (int, error) {
	if info.GlobalRole == model.RoleAdmin {
		return 0, nil
	}
	return s.resolveNamespaceLimit(ctx, info.UserID)
}

// CountOwnedByUser returns the number of namespaces owned by the given user.
func (s *NamespaceService) CountOwnedByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.members.CountOwnedByUser(ctx, userID)
}

func isValidNamespaceRole(role string) bool {
	switch role {
	case model.NamespaceRoleOwner, model.NamespaceRoleAdmin, model.NamespaceRoleMember:
		return true
	}
	return false
}

// checkReservedSlug checks the slug against the admin-managed deny list.
func (s *NamespaceService) checkReservedSlug(ctx context.Context, slug string) error {
	setting, err := s.systemSettings.Get(ctx, model.SettingReservedNamespaceSlugs)
	if err != nil {
		return nil // no deny list configured — allow
	}
	var denyList []string
	if err := json.Unmarshal(setting.Value, &denyList); err != nil {
		return nil // malformed — allow
	}
	for _, reserved := range denyList {
		if reserved == slug {
			return fmt.Errorf("namespace slug %q is reserved: %w", slug, model.ErrValidation)
		}
	}
	return nil
}
