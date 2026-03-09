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

func (r *mockNamespaceMemberRepo) RemoveAllByNamespace(_ context.Context, nsID uuid.UUID) error {
	for key, m := range r.members {
		if m.NamespaceID == nsID {
			delete(r.members, key)
		}
	}
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

func (r *mockNamespaceProjectRepo) GetByKeyAndNamespace(_ context.Context, namespaceID uuid.UUID, key string) (*model.Project, error) {
	for i := range r.projects {
		if r.projects[i].Key == key && r.projects[i].NamespaceID != nil && *r.projects[i].NamespaceID == namespaceID {
			return &r.projects[i], nil
		}
	}
	return nil, model.ErrNotFound
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

func (r *mockNamespaceUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, model.ErrNotFound
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

type mockNamespaceSystemSettingsRepo struct {
	settings map[string]*model.SystemSetting
}

func newMockNamespaceSystemSettingsRepo() *mockNamespaceSystemSettingsRepo {
	return &mockNamespaceSystemSettingsRepo{
		settings: make(map[string]*model.SystemSetting),
	}
}

func (r *mockNamespaceSystemSettingsRepo) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	s, ok := r.settings[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

// --- Test helper ---

func newTestNamespaceService() (*NamespaceService, *mockNamespaceRepo, *mockNamespaceMemberRepo, *mockNamespaceProjectRepo, *mockNamespaceUserRepo) {
	nsRepo := newMockNamespaceRepo()
	memberRepo := newMockNamespaceMemberRepo()
	projectRepo := newMockNamespaceProjectRepo()
	userRepo := newMockNamespaceUserRepo()
	settingsRepo := newMockNamespaceSystemSettingsRepo()
	svc := NewNamespaceService(nsRepo, memberRepo, projectRepo, userRepo, settingsRepo)
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
	if ns.DisplayName != model.DefaultBrandName {
		t.Fatalf("expected display name %q, got %q", model.DefaultBrandName, ns.DisplayName)
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

// --- Phase 3 CRUD tests ---

func newTestNamespaceServiceWithSettings() (*NamespaceService, *mockNamespaceRepo, *mockNamespaceMemberRepo, *mockNamespaceProjectRepo, *mockNamespaceUserRepo, *mockNamespaceSystemSettingsRepo) {
	nsRepo := newMockNamespaceRepo()
	memberRepo := newMockNamespaceMemberRepo()
	projectRepo := newMockNamespaceProjectRepo()
	userRepo := newMockNamespaceUserRepo()
	settingsRepo := newMockNamespaceSystemSettingsRepo()
	svc := NewNamespaceService(nsRepo, memberRepo, projectRepo, userRepo, settingsRepo)
	return svc, nsRepo, memberRepo, projectRepo, userRepo, settingsRepo
}

func enableNamespaces(settingsRepo *mockNamespaceSystemSettingsRepo) {
	settingsRepo.settings[model.SettingNamespacesEnabled] = &model.SystemSetting{
		Key:   model.SettingNamespacesEnabled,
		Value: []byte(`true`),
	}
}

func nsAdminAuthInfo() *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "admin@test.com",
		GlobalRole: model.RoleAdmin,
	}
}

func nsUserAuthInfo(userID uuid.UUID) *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     userID,
		Email:      "user@test.com",
		GlobalRole: model.RoleUser,
	}
}

func TestCreateNamespace_Success(t *testing.T) {
	svc, _, memberRepo, _, userRepo, settingsRepo := newTestNamespaceServiceWithSettings()
	enableNamespaces(settingsRepo)
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	ns, err := svc.CreateNamespace(context.Background(), info, "acme", "Acme Corp")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ns.Slug != "acme" {
		t.Fatalf("expected slug 'acme', got %q", ns.Slug)
	}
	if ns.DisplayName != "Acme Corp" {
		t.Fatalf("expected display name 'Acme Corp', got %q", ns.DisplayName)
	}
	if ns.IsDefault {
		t.Fatal("should not be default")
	}

	// Creator should be owner
	member, err := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, userID)
	if err != nil {
		t.Fatalf("expected creator to be member, got %v", err)
	}
	if member.Role != model.NamespaceRoleOwner {
		t.Fatalf("expected owner role, got %q", member.Role)
	}
}

func TestCreateNamespace_DisabledFeature(t *testing.T) {
	svc, _, _, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	// Don't enable namespaces
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	_, err := svc.CreateNamespace(context.Background(), info, "acme", "Acme Corp")
	if !errors.Is(err, model.ErrNamespacesDisabled) {
		t.Fatalf("expected ErrNamespacesDisabled, got %v", err)
	}
}

func TestCreateNamespace_InvalidSlug(t *testing.T) {
	svc, _, _, _, userRepo, settingsRepo := newTestNamespaceServiceWithSettings()
	enableNamespaces(settingsRepo)
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	cases := []string{"A", "ABC", "1abc", "-abc", "a"} // too short, uppercase, starts with number, starts with hyphen, too short
	for _, slug := range cases {
		_, err := svc.CreateNamespace(context.Background(), info, slug, "Test")
		if !errors.Is(err, model.ErrValidation) {
			t.Fatalf("expected ErrValidation for slug %q, got %v", slug, err)
		}
	}
}

func TestCreateNamespace_ReservedSlug(t *testing.T) {
	svc, _, _, _, userRepo, settingsRepo := newTestNamespaceServiceWithSettings()
	enableNamespaces(settingsRepo)
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	for slug := range reservedSlugs {
		_, err := svc.CreateNamespace(context.Background(), info, slug, "Test")
		if !errors.Is(err, model.ErrValidation) {
			t.Fatalf("expected ErrValidation for reserved slug %q, got %v", slug, err)
		}
	}
}

func TestCreateNamespace_DuplicateSlug(t *testing.T) {
	svc, _, _, _, userRepo, settingsRepo := newTestNamespaceServiceWithSettings()
	enableNamespaces(settingsRepo)
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	_, err := svc.CreateNamespace(context.Background(), info, "acme", "Acme")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.CreateNamespace(context.Background(), info, "acme", "Acme 2")
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestGetNamespace_AsGlobalAdmin(t *testing.T) {
	svc, nsRepo, _, _, _, _ := newTestNamespaceServiceWithSettings()
	info := nsAdminAuthInfo()

	// Create a namespace directly in the repo
	ns := &model.Namespace{ID: uuid.New(), Slug: "test-ns", DisplayName: "Test", CreatedBy: info.UserID}
	nsRepo.Create(context.Background(), ns)

	result, err := svc.GetNamespace(context.Background(), info, "test-ns")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Slug != "test-ns" {
		t.Fatalf("expected slug 'test-ns', got %q", result.Slug)
	}
}

func TestGetNamespace_NonMemberDenied(t *testing.T) {
	svc, nsRepo, _, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "secret", DisplayName: "Secret"}
	nsRepo.Create(context.Background(), ns)

	_, err := svc.GetNamespace(context.Background(), info, "secret")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateNamespace_Success(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "test-ns", DisplayName: "Old Name"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: userID, Role: model.NamespaceRoleOwner})

	newName := "New Name"
	updated, err := svc.UpdateNamespace(context.Background(), info, "test-ns", nil, &newName, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.DisplayName != "New Name" {
		t.Fatalf("expected 'New Name', got %q", updated.DisplayName)
	}
}

func TestUpdateNamespace_DefaultDenied(t *testing.T) {
	svc, nsRepo, _, _, _, _ := newTestNamespaceServiceWithSettings()
	info := nsAdminAuthInfo()

	ns := &model.Namespace{ID: uuid.New(), Slug: model.DefaultNamespaceSlug, DisplayName: "Default", IsDefault: true}
	nsRepo.Create(context.Background(), ns)

	newName := "Not Default"
	_, err := svc.UpdateNamespace(context.Background(), info, model.DefaultNamespaceSlug, nil, &newName, nil, nil)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteNamespace_Success(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "doomed", DisplayName: "Doomed"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: userID, Role: model.NamespaceRoleOwner})

	err := svc.DeleteNamespace(context.Background(), info, "doomed")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = nsRepo.GetBySlug(context.Background(), "doomed")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatal("expected namespace to be deleted")
	}
}

func TestDeleteNamespace_DefaultDenied(t *testing.T) {
	svc, nsRepo, _, _, _, _ := newTestNamespaceServiceWithSettings()
	info := nsAdminAuthInfo()

	ns := &model.Namespace{ID: uuid.New(), Slug: model.DefaultNamespaceSlug, DisplayName: "Default", IsDefault: true}
	nsRepo.Create(context.Background(), ns)

	err := svc.DeleteNamespace(context.Background(), info, model.DefaultNamespaceSlug)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteNamespace_NotEmptyDenied(t *testing.T) {
	// Use a custom namespace repo that returns true for HasProjects
	nsRepo := &mockNamespaceRepoWithProjects{mockNamespaceRepo: newMockNamespaceRepo()}
	memberRepo := newMockNamespaceMemberRepo()
	projectRepo := newMockNamespaceProjectRepo()
	userRepo := newMockNamespaceUserRepo()
	settingsRepo := newMockNamespaceSystemSettingsRepo()
	svc := NewNamespaceService(nsRepo, memberRepo, projectRepo, userRepo, settingsRepo)

	userID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(userID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "has-projects", DisplayName: "Has Projects"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: userID, Role: model.NamespaceRoleOwner})

	err := svc.DeleteNamespace(context.Background(), info, "has-projects")
	if !errors.Is(err, model.ErrNamespaceNotEmpty) {
		t.Fatalf("expected ErrNamespaceNotEmpty, got %v", err)
	}
}

// mockNamespaceRepoWithProjects always reports that namespaces have projects.
type mockNamespaceRepoWithProjects struct {
	*mockNamespaceRepo
}

func (r *mockNamespaceRepoWithProjects) HasProjects(_ context.Context, _ uuid.UUID) (bool, error) {
	return true, nil
}

// --- Namespace Member Management Tests ---

func TestAddNamespaceMember_Success(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	targetID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "team", DisplayName: "Team"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})

	member, err := svc.AddNamespaceMember(context.Background(), info, "team", targetID, model.NamespaceRoleAdmin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if member.Role != model.NamespaceRoleAdmin {
		t.Fatalf("expected admin role, got %q", member.Role)
	}
}

func TestAddNamespaceMember_DuplicateDenied(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	targetID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "team", DisplayName: "Team"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: targetID, Role: model.NamespaceRoleMember})

	_, err := svc.AddNamespaceMember(context.Background(), info, "team", targetID, model.NamespaceRoleMember)
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUpdateNamespaceMemberRole_Success(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	targetID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "team", DisplayName: "Team"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: targetID, Role: model.NamespaceRoleMember})

	err := svc.UpdateNamespaceMemberRole(context.Background(), info, "team", targetID, model.NamespaceRoleAdmin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, targetID)
	if updated.Role != model.NamespaceRoleAdmin {
		t.Fatalf("expected admin, got %q", updated.Role)
	}
}

func TestUpdateNamespaceMemberRole_ProtectLastOwner(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "solo", DisplayName: "Solo"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})

	err := svc.UpdateNamespaceMemberRole(context.Background(), info, "solo", ownerID, model.NamespaceRoleAdmin)
	if !errors.Is(err, model.ErrConflict) {
		t.Fatalf("expected ErrConflict for last owner demotion, got %v", err)
	}
}

func TestRemoveNamespaceMember_Success(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	targetID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "team", DisplayName: "Team"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: targetID, Role: model.NamespaceRoleMember})

	err := svc.RemoveNamespaceMember(context.Background(), info, "team", targetID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = memberRepo.GetByNamespaceAndUser(context.Background(), ns.ID, targetID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatal("expected member to be removed")
	}
}

func TestRemoveNamespaceMember_ProtectLastOwner(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "solo", DisplayName: "Solo"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})

	err := svc.RemoveNamespaceMember(context.Background(), info, "solo", ownerID)
	if !errors.Is(err, model.ErrConflict) {
		t.Fatalf("expected ErrConflict for last owner removal, got %v", err)
	}
}

// --- Project Migration Tests ---

func TestMigrateProject_Success(t *testing.T) {
	svc, nsRepo, memberRepo, projectRepo, userRepo, _ := newTestNamespaceServiceWithSettings()
	info := nsAdminAuthInfo()
	userRepo.addUser(model.RoleAdmin)

	fromNs := &model.Namespace{ID: uuid.New(), Slug: "source", DisplayName: "Source"}
	toNs := &model.Namespace{ID: uuid.New(), Slug: "target", DisplayName: "Target"}
	nsRepo.Create(context.Background(), fromNs)
	nsRepo.Create(context.Background(), toNs)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: fromNs.ID, UserID: info.UserID, Role: model.NamespaceRoleOwner})
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: toNs.ID, UserID: info.UserID, Role: model.NamespaceRoleOwner})

	projectRepo.projects = []model.Project{
		{ID: uuid.New(), Key: "PROJ", Name: "Test Project", NamespaceID: &fromNs.ID},
	}

	err := svc.MigrateProject(context.Background(), info, "PROJ", "source", "target")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Project should now be in target namespace
	if *projectRepo.projects[0].NamespaceID != toNs.ID {
		t.Fatal("expected project to be in target namespace")
	}
}

func TestMigrateProject_KeyCollision(t *testing.T) {
	svc, nsRepo, _, projectRepo, _, _ := newTestNamespaceServiceWithSettings()
	info := nsAdminAuthInfo()

	fromNs := &model.Namespace{ID: uuid.New(), Slug: "source", DisplayName: "Source"}
	toNs := &model.Namespace{ID: uuid.New(), Slug: "target", DisplayName: "Target"}
	nsRepo.Create(context.Background(), fromNs)
	nsRepo.Create(context.Background(), toNs)

	projectRepo.projects = []model.Project{
		{ID: uuid.New(), Key: "DUP", Name: "Source Project", NamespaceID: &fromNs.ID},
		{ID: uuid.New(), Key: "DUP", Name: "Target Project", NamespaceID: &toNs.ID},
	}

	err := svc.MigrateProject(context.Background(), info, "DUP", "source", "target")
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestListUserNamespaces_AdminSeesAll(t *testing.T) {
	svc, nsRepo, _, _, _, _ := newTestNamespaceServiceWithSettings()
	info := nsAdminAuthInfo()

	nsRepo.Create(context.Background(), &model.Namespace{ID: uuid.New(), Slug: "ns1", DisplayName: "NS1"})
	nsRepo.Create(context.Background(), &model.Namespace{ID: uuid.New(), Slug: "ns2", DisplayName: "NS2"})

	result, err := svc.ListUserNamespaces(context.Background(), info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(result))
	}
}

func TestIsNamespacesEnabled_DefaultFalse(t *testing.T) {
	svc, _, _, _, _, _ := newTestNamespaceServiceWithSettings()

	enabled, err := svc.IsNamespacesEnabled(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if enabled {
		t.Fatal("expected namespaces to be disabled by default")
	}
}

func TestIsNamespacesEnabled_Enabled(t *testing.T) {
	svc, _, _, _, _, settingsRepo := newTestNamespaceServiceWithSettings()
	enableNamespaces(settingsRepo)

	enabled, err := svc.IsNamespacesEnabled(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatal("expected namespaces to be enabled")
	}
}

func TestAddNamespaceMember_InvalidRole(t *testing.T) {
	svc, nsRepo, memberRepo, _, userRepo, _ := newTestNamespaceServiceWithSettings()
	ownerID := userRepo.addUser(model.RoleUser)
	targetID := userRepo.addUser(model.RoleUser)
	info := nsUserAuthInfo(ownerID)

	ns := &model.Namespace{ID: uuid.New(), Slug: "team", DisplayName: "Team"}
	nsRepo.Create(context.Background(), ns)
	memberRepo.Add(context.Background(), &model.NamespaceMember{NamespaceID: ns.ID, UserID: ownerID, Role: model.NamespaceRoleOwner})

	_, err := svc.AddNamespaceMember(context.Background(), info, "team", targetID, "invalid-role")
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid role, got %v", err)
	}
}
