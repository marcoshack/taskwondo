package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

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
	CountByRole(ctx context.Context, namespaceID uuid.UUID, role string) (int, error)
}

// NamespaceProjectRepository defines the project operations needed by the namespace service.
type NamespaceProjectRepository interface {
	ListAll(ctx context.Context) ([]model.Project, error)
	SetNamespaceID(ctx context.Context, projectID, namespaceID uuid.UUID) error
}

// NamespaceUserRepository defines the user operations needed by the namespace service.
type NamespaceUserRepository interface {
	ListAll(ctx context.Context) ([]model.User, error)
}

// NamespaceService handles namespace business logic.
type NamespaceService struct {
	namespaces NamespaceRepository
	members    NamespaceMemberRepository
	projects   NamespaceProjectRepository
	users      NamespaceUserRepository
}

// NewNamespaceService creates a new NamespaceService.
func NewNamespaceService(namespaces NamespaceRepository, members NamespaceMemberRepository, projects NamespaceProjectRepository, users NamespaceUserRepository) *NamespaceService {
	return &NamespaceService{
		namespaces: namespaces,
		members:    members,
		projects:   projects,
		users:      users,
	}
}

// SeedDefaultNamespace creates the default namespace if it doesn't exist and backfills
// existing projects that have no namespace_id.
func (s *NamespaceService) SeedDefaultNamespace(ctx context.Context) error {
	// Check if default namespace already exists
	existing, err := s.namespaces.GetDefault(ctx)
	if err == nil {
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
		DisplayName: "Default",
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
