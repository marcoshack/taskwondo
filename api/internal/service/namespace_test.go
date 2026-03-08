package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock repositories ---

type mockNamespaceRepo struct {
	namespaces map[uuid.UUID]*model.Namespace
	bySlug     map[string]*model.Namespace
}

func newMockNamespaceRepo() *mockNamespaceRepo {
	return &mockNamespaceRepo{
		namespaces: make(map[uuid.UUID]*model.Namespace),
		bySlug:     make(map[string]*model.Namespace),
	}
}

func (r *mockNamespaceRepo) Create(_ context.Context, ns *model.Namespace) error {
	if _, ok := r.bySlug[ns.Slug]; ok {
		return model.ErrAlreadyExists
	}
	r.namespaces[ns.ID] = ns
	r.bySlug[ns.Slug] = ns
	return nil
}

func (r *mockNamespaceRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Namespace, error) {
	ns, ok := r.namespaces[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return ns, nil
}

func (r *mockNamespaceRepo) GetBySlug(_ context.Context, slug string) (*model.Namespace, error) {
	ns, ok := r.bySlug[slug]
	if !ok {
		return nil, model.ErrNotFound
	}
	return ns, nil
}

func (r *mockNamespaceRepo) GetDefault(_ context.Context) (*model.Namespace, error) {
	for _, ns := range r.namespaces {
		if ns.IsDefault {
			return ns, nil
		}
	}
	return nil, model.ErrNotFound
}

func (r *mockNamespaceRepo) List(_ context.Context) ([]model.Namespace, error) {
	var result []model.Namespace
	for _, ns := range r.namespaces {
		result = append(result, *ns)
	}
	return result, nil
}

func (r *mockNamespaceRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]model.Namespace, error) {
	return r.List(context.Background())
}

func (r *mockNamespaceRepo) Update(_ context.Context, ns *model.Namespace) error {
	existing, ok := r.namespaces[ns.ID]
	if !ok {
		return model.ErrNotFound
	}
	delete(r.bySlug, existing.Slug)
	r.namespaces[ns.ID] = ns
	r.bySlug[ns.Slug] = ns
	return nil
}

func (r *mockNamespaceRepo) Delete(_ context.Context, id uuid.UUID) error {
	ns, ok := r.namespaces[id]
	if !ok {
		return model.ErrNotFound
	}
	if ns.IsDefault {
		return model.ErrNotFound // Cannot delete default
	}
	delete(r.namespaces, id)
	delete(r.bySlug, ns.Slug)
	return nil
}

func (r *mockNamespaceRepo) HasProjects(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (r *mockNamespaceRepo) CountNonDefault(_ context.Context) (int, error) {
	count := 0
	for _, ns := range r.namespaces {
		if !ns.IsDefault {
			count++
		}
	}
	return count, nil
}

type mockNamespaceMemberRepo struct {
	members map[string]*model.NamespaceMember // key: "namespaceID:userID"
}

func newMockNamespaceMemberRepo() *mockNamespaceMemberRepo {
	return &mockNamespaceMemberRepo{
		members: make(map[string]*model.NamespaceMember),
	}
}

func nsMemberKey(nsID, userID uuid.UUID) string {
	return nsID.String() + ":" + userID.String()
}

func (r *mockNamespaceMemberRepo) Add(_ context.Context, m *model.NamespaceMember) error {
	key := nsMemberKey(m.NamespaceID, m.UserID)
	if _, ok := r.members[key]; ok {
		return model.ErrAlreadyExists
	}
	r.members[key] = m
	return nil
}

func (r *mockNamespaceMemberRepo) GetByNamespaceAndUser(_ context.Context, nsID, userID uuid.UUID) (*model.NamespaceMember, error) {
	m, ok := r.members[nsMemberKey(nsID, userID)]
	if !ok {
		return nil, model.ErrNotFound
	}
	return m, nil
}

func (r *mockNamespaceMemberRepo) ListByNamespace(_ context.Context, nsID uuid.UUID) ([]model.NamespaceMemberWithUser, error) {
	var result []model.NamespaceMemberWithUser
	for _, m := range r.members {
		if m.NamespaceID == nsID {
			result = append(result, model.NamespaceMemberWithUser{
				NamespaceMember: *m,
				DisplayName:     "Test User",
				Email:           "test@example.com",
			})
		}
	}
	return result, nil
}

func (r *mockNamespaceMemberRepo) UpdateRole(_ context.Context, nsID, userID uuid.UUID, role string) error {
	m, ok := r.members[nsMemberKey(nsID, userID)]
	if !ok {
		return model.ErrNotFound
	}
	m.Role = role
	return nil
}

func (r *mockNamespaceMemberRepo) Remove(_ context.Context, nsID, userID uuid.UUID) error {
	key := nsMemberKey(nsID, userID)
	if _, ok := r.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(r.members, key)
	return nil
}

func (r *mockNamespaceMemberRepo) CountByRole(_ context.Context, nsID uuid.UUID, role string) (int, error) {
	count := 0
	for _, m := range r.members {
		if m.NamespaceID == nsID && m.Role == role {
			count++
		}
	}
	return count, nil
}

type mockNamespaceProjectRepo struct {
	projects []model.Project
}

func newMockNamespaceProjectRepo() *mockNamespaceProjectRepo {
	return &mockNamespaceProjectRepo{}
}

func (r *mockNamespaceProjectRepo) ListAll(_ context.Context) ([]model.Project, error) {
	return r.projects, nil
}

func (r *mockNamespaceProjectRepo) SetNamespaceID(_ context.Context, projectID, namespaceID uuid.UUID) error {
	for i := range r.projects {
		if r.projects[i].ID == projectID {
			r.projects[i].NamespaceID = &namespaceID
			return nil
		}
	}
	return model.ErrNotFound
}

type mockNamespaceUserRepo struct {
	users []model.User
}

func newMockNamespaceUserRepo() *mockNamespaceUserRepo {
	return &mockNamespaceUserRepo{}
}

func (r *mockNamespaceUserRepo) ListAll(_ context.Context) ([]model.User, error) {
	return r.users, nil
}

func (r *mockNamespaceUserRepo) addUser(role string) uuid.UUID {
	id := uuid.New()
	r.users = append(r.users, model.User{
		ID:          id,
		Email:       id.String() + "@test.com",
		DisplayName: "User " + id.String()[:8],
		GlobalRole:  role,
		IsActive:    true,
	})
	return id
}

// --- Test helper ---

func newTestNamespaceService() (*NamespaceService, *mockNamespaceRepo, *mockNamespaceMemberRepo, *mockNamespaceProjectRepo, *mockNamespaceUserRepo) {
	nsRepo := newMockNamespaceRepo()
	memberRepo := newMockNamespaceMemberRepo()
	projectRepo := newMockNamespaceProjectRepo()
	userRepo := newMockNamespaceUserRepo()
	svc := NewNamespaceService(nsRepo, memberRepo, projectRepo, userRepo)
	return svc, nsRepo, memberRepo, projectRepo, userRepo
}

// --- Tests ---

func TestSeedDefaultNamespace_CreatesNew(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo := newTestNamespaceService()
	adminID := userRepo.addUser(model.RoleAdmin)

	err := svc.SeedDefaultNamespace(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify default namespace was created
	ns, err := nsRepo.GetDefault(context.Background())
	if err != nil {
		t.Fatalf("expected to find default namespace, got %v", err)
	}
	if ns.Slug != model.DefaultNamespaceSlug {
		t.Fatalf("expected slug %q, got %q", model.DefaultNamespaceSlug, ns.Slug)
	}
	if !ns.IsDefault {
		t.Fatal("expected is_default to be true")
	}
	if ns.DisplayName != "Default" {
		t.Fatalf("expected display name 'Default', got %q", ns.DisplayName)
	}

	// Verify admin was added as owner
	member, err := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, adminID)
	if err != nil {
		t.Fatalf("expected admin to be a member, got %v", err)
	}
	if member.Role != model.NamespaceRoleOwner {
		t.Fatalf("expected admin role 'owner', got %q", member.Role)
	}
}

func TestSeedDefaultNamespace_Idempotent(t *testing.T) {
	svc, nsRepo, _, _, userRepo := newTestNamespaceService()
	userRepo.addUser(model.RoleAdmin)

	// Seed twice
	if err := svc.SeedDefaultNamespace(context.Background()); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	if err := svc.SeedDefaultNamespace(context.Background()); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	// Should still have exactly one namespace
	all, _ := nsRepo.List(context.Background())
	if len(all) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(all))
	}
}

func TestSeedDefaultNamespace_BackfillsProjects(t *testing.T) {
	svc, _, _, projectRepo, userRepo := newTestNamespaceService()
	userRepo.addUser(model.RoleAdmin)

	// Add projects without namespace
	p1ID := uuid.New()
	p2ID := uuid.New()
	projectRepo.projects = []model.Project{
		{ID: p1ID, Name: "Project 1", Key: "P1"},
		{ID: p2ID, Name: "Project 2", Key: "P2"},
	}

	err := svc.SeedDefaultNamespace(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Both projects should now have namespace_id set
	for _, p := range projectRepo.projects {
		if p.NamespaceID == nil {
			t.Fatalf("project %s still has nil namespace_id", p.Key)
		}
	}
}

func TestSeedDefaultNamespace_SkipsAlreadyBackfilled(t *testing.T) {
	svc, _, _, projectRepo, userRepo := newTestNamespaceService()
	userRepo.addUser(model.RoleAdmin)

	existingNS := uuid.New()
	p1ID := uuid.New()
	p2ID := uuid.New()
	projectRepo.projects = []model.Project{
		{ID: p1ID, Name: "Project 1", Key: "P1", NamespaceID: &existingNS},
		{ID: p2ID, Name: "Project 2", Key: "P2"},
	}

	err := svc.SeedDefaultNamespace(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// P1 should keep its original namespace
	if *projectRepo.projects[0].NamespaceID != existingNS {
		t.Fatal("expected P1 to keep its original namespace_id")
	}
	// P2 should be backfilled
	if projectRepo.projects[1].NamespaceID == nil {
		t.Fatal("expected P2 to be backfilled")
	}
}

func TestSeedDefaultNamespace_NoUsersError(t *testing.T) {
	svc, _, _, _, _ := newTestNamespaceService()

	err := svc.SeedDefaultNamespace(context.Background())
	if err == nil {
		t.Fatal("expected error when no users exist")
	}
}

func TestSeedDefaultNamespace_FallsBackToNonAdmin(t *testing.T) {
	svc, nsRepo, _, _, userRepo := newTestNamespaceService()
	userRepo.addUser(model.RoleUser) // non-admin user

	err := svc.SeedDefaultNamespace(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ns, err := nsRepo.GetDefault(context.Background())
	if err != nil {
		t.Fatalf("expected default namespace, got %v", err)
	}
	if ns.CreatedBy != userRepo.users[0].ID {
		t.Fatal("expected namespace to be created by the fallback user")
	}
}

func TestSeedDefaultNamespace_BackfillOnReseed(t *testing.T) {
	svc, _, _, projectRepo, userRepo := newTestNamespaceService()
	userRepo.addUser(model.RoleAdmin)

	// First seed with no projects
	if err := svc.SeedDefaultNamespace(context.Background()); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}

	// Add a project without namespace (simulating a project created before namespace was required)
	p1ID := uuid.New()
	projectRepo.projects = append(projectRepo.projects, model.Project{
		ID: p1ID, Name: "Late Project", Key: "LP",
	})

	// Second seed should backfill the new project
	if err := svc.SeedDefaultNamespace(context.Background()); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	for _, p := range projectRepo.projects {
		if p.NamespaceID == nil {
			t.Fatalf("project %s should have been backfilled", p.Key)
		}
	}
}

// --- Model-level tests ---

func TestNamespaceConstants(t *testing.T) {
	if model.NamespaceRoleOwner != "owner" {
		t.Fatalf("expected owner, got %s", model.NamespaceRoleOwner)
	}
	if model.NamespaceRoleAdmin != "admin" {
		t.Fatalf("expected admin, got %s", model.NamespaceRoleAdmin)
	}
	if model.NamespaceRoleMember != "member" {
		t.Fatalf("expected member, got %s", model.NamespaceRoleMember)
	}
	if model.DefaultNamespaceSlug != "default" {
		t.Fatalf("expected 'default', got %s", model.DefaultNamespaceSlug)
	}
}

func TestErrorTypes(t *testing.T) {
	if !errors.Is(model.ErrNamespacesDisabled, model.ErrNamespacesDisabled) {
		t.Fatal("ErrNamespacesDisabled should match itself")
	}
	if !errors.Is(model.ErrNamespaceNotEmpty, model.ErrNamespaceNotEmpty) {
		t.Fatal("ErrNamespaceNotEmpty should match itself")
	}
}
