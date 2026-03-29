package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock repositories ---

type mockProjectRepo struct {
	projects         map[uuid.UUID]*model.Project
	byKey            map[string]*model.Project
	listAllCalled    bool
	listByUserCalled bool
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[uuid.UUID]*model.Project),
		byKey:    make(map[string]*model.Project),
	}
}

func (m *mockProjectRepo) Create(_ context.Context, project *model.Project) error {
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	m.projects[project.ID] = project
	m.byKey[project.Key] = project
	return nil
}

func (m *mockProjectRepo) GetByKey(_ context.Context, key string) (*model.Project, error) {
	p, ok := m.byKey[key]
	if !ok || p.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) GetByKeyAndNamespace(_ context.Context, namespaceID uuid.UUID, key string) (*model.Project, error) {
	p, ok := m.byKey[key]
	if !ok || p.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	if p.NamespaceID == nil || *p.NamespaceID != namespaceID {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok || p.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]model.Project, error) {
	m.listByUserCalled = true
	// This mock doesn't filter by user — the service layer handles that via membership.
	// For testing, return all non-deleted projects.
	var result []model.Project
	for _, p := range m.projects {
		if p.DeletedAt == nil {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockProjectRepo) ListAll(_ context.Context) ([]model.Project, error) {
	m.listAllCalled = true
	var result []model.Project
	for _, p := range m.projects {
		if p.DeletedAt == nil {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockProjectRepo) Update(_ context.Context, project *model.Project) error {
	existing, ok := m.projects[project.ID]
	if !ok || existing.DeletedAt != nil {
		return model.ErrNotFound
	}
	// Remove old key mapping if key changed
	if existing.Key != project.Key {
		delete(m.byKey, existing.Key)
	}
	now := time.Now()
	project.UpdatedAt = now
	m.projects[project.ID] = project
	m.byKey[project.Key] = project
	return nil
}

func (m *mockProjectRepo) Delete(_ context.Context, id uuid.UUID) error {
	p, ok := m.projects[id]
	if !ok || p.DeletedAt != nil {
		return model.ErrNotFound
	}
	now := time.Now()
	p.DeletedAt = &now
	return nil
}

func (m *mockProjectRepo) GetSummaries(_ context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectSummary, error) {
	result := make(map[uuid.UUID]model.ProjectSummary, len(projectIDs))
	for _, id := range projectIDs {
		result[id] = model.ProjectSummary{}
	}
	return result, nil
}

func (m *mockProjectRepo) CountByOwner(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, p := range m.projects {
		if p.DeletedAt == nil {
			count++
		}
	}
	// This mock counts all non-deleted projects as owned by the user.
	// For more precise testing, the caller can pre-populate the repo.
	return count, nil
}

func (m *mockProjectRepo) ResolveNamespaces(_ context.Context, _ []string) (map[string]model.ProjectNamespaceInfo, error) {
	return nil, nil
}

func (m *mockProjectRepo) ResolveNamespacesByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.ProjectNamespaceInfo, error) {
	return nil, nil
}

type mockProjectMemberRepo struct {
	members map[string]*model.ProjectMember // key: "projectID:userID"
}

func newMockProjectMemberRepo() *mockProjectMemberRepo {
	return &mockProjectMemberRepo{
		members: make(map[string]*model.ProjectMember),
	}
}

func memberKey(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockProjectMemberRepo) Add(_ context.Context, member *model.ProjectMember) error {
	key := memberKey(member.ProjectID, member.UserID)
	if _, exists := m.members[key]; exists {
		return model.ErrAlreadyExists
	}
	member.CreatedAt = time.Now()
	m.members[key] = member
	return nil
}

func (m *mockProjectMemberRepo) GetByProjectAndUser(_ context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error) {
	key := memberKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return member, nil
}

func (m *mockProjectMemberRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectMemberWithUser, error) {
	var result []model.ProjectMemberWithUser
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, model.ProjectMemberWithUser{
				ProjectMember: *member,
				Email:         "user@example.com",
				DisplayName:   "Test User",
			})
		}
	}
	return result, nil
}

func (m *mockProjectMemberRepo) UpdateRole(_ context.Context, projectID, userID uuid.UUID, role string) error {
	key := memberKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return model.ErrNotFound
	}
	member.Role = role
	return nil
}

func (m *mockProjectMemberRepo) Remove(_ context.Context, projectID, userID uuid.UUID) error {
	key := memberKey(projectID, userID)
	if _, ok := m.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.members, key)
	return nil
}

func (m *mockProjectMemberRepo) CountByRole(_ context.Context, projectID uuid.UUID, role string) (int, error) {
	count := 0
	for _, member := range m.members {
		if member.ProjectID == projectID && member.Role == role {
			count++
		}
	}
	return count, nil
}

type mockProjectInviteRepo struct {
	invites map[uuid.UUID]*model.ProjectInvite
	byCode  map[string]*model.ProjectInvite
}

func newMockProjectInviteRepo() *mockProjectInviteRepo {
	return &mockProjectInviteRepo{
		invites: make(map[uuid.UUID]*model.ProjectInvite),
		byCode:  make(map[string]*model.ProjectInvite),
	}
}

func (m *mockProjectInviteRepo) Create(_ context.Context, invite *model.ProjectInvite) error {
	invite.CreatedAt = time.Now()
	m.invites[invite.ID] = invite
	m.byCode[invite.Code] = invite
	return nil
}

func (m *mockProjectInviteRepo) GetByCode(_ context.Context, code string) (*model.ProjectInvite, error) {
	inv, ok := m.byCode[code]
	if !ok {
		return nil, model.ErrNotFound
	}
	return inv, nil
}

func (m *mockProjectInviteRepo) GetByID(_ context.Context, id uuid.UUID) (*model.ProjectInvite, error) {
	inv, ok := m.invites[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return inv, nil
}

func (m *mockProjectInviteRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectInvite, error) {
	var result []model.ProjectInvite
	for _, inv := range m.invites {
		if inv.ProjectID == projectID {
			result = append(result, *inv)
		}
	}
	return result, nil
}

func (m *mockProjectInviteRepo) IncrementUseCount(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invites[id]
	if !ok {
		return model.ErrNotFound
	}
	if inv.MaxUses > 0 && inv.UseCount >= inv.MaxUses {
		return model.ErrNotFound
	}
	inv.UseCount++
	return nil
}

func (m *mockProjectInviteRepo) Delete(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invites[id]
	if !ok {
		return model.ErrNotFound
	}
	delete(m.byCode, inv.Code)
	delete(m.invites, id)
	return nil
}

func (m *mockProjectInviteRepo) DeleteByProject(_ context.Context, projectID uuid.UUID) error {
	for id, inv := range m.invites {
		if inv.ProjectID == projectID {
			delete(m.byCode, inv.Code)
			delete(m.invites, id)
		}
	}
	return nil
}

// noopInboxRepo is a minimal InboxRepository for project tests (only RemoveByProjectID needed).
type noopInboxRepo struct{}

func (noopInboxRepo) Add(context.Context, *model.InboxItem) error                  { return nil }
func (noopInboxRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error           { return nil }
func (noopInboxRepo) RemoveByID(context.Context, uuid.UUID, uuid.UUID) error       { return nil }
func (noopInboxRepo) List(context.Context, uuid.UUID, bool, string, []string, *uuid.UUID, *uuid.UUID, int) (*model.InboxItemList, error) {
	return &model.InboxItemList{}, nil
}
func (noopInboxRepo) CountByUser(context.Context, uuid.UUID) (int, error)    { return 0, nil }
func (noopInboxRepo) CountAllByUser(context.Context, uuid.UUID) (int, error) { return 0, nil }
func (noopInboxRepo) UpdatePosition(context.Context, uuid.UUID, uuid.UUID, int) error {
	return nil
}
func (noopInboxRepo) MaxPosition(context.Context, uuid.UUID) (int, error)       { return 0, nil }
func (noopInboxRepo) RemoveCompleted(context.Context, uuid.UUID) (int, error)   { return 0, nil }
func (noopInboxRepo) Exists(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil }
func (noopInboxRepo) GetWorkItemProjectID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (noopInboxRepo) RemoveByProjectID(context.Context, uuid.UUID) (int, error) { return 0, nil }

// noopWatcherRepo is a minimal WatcherRepository for project tests (only RemoveByProjectID needed).
type noopWatcherRepo struct{}

func (noopWatcherRepo) Create(context.Context, *model.WorkItemWatcher) error { return nil }
func (noopWatcherRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error   { return nil }
func (noopWatcherRepo) ListByWorkItem(context.Context, uuid.UUID) ([]model.WorkItemWatcherWithUser, error) {
	return nil, nil
}
func (noopWatcherRepo) CountByWorkItem(context.Context, uuid.UUID) (int, error)      { return 0, nil }
func (noopWatcherRepo) IsWatching(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil }
func (noopWatcherRepo) ListWatchedItemIDs(context.Context, uuid.UUID, *uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (noopWatcherRepo) RemoveByProjectID(context.Context, uuid.UUID) (int, error) { return 0, nil }

// --- Test helpers ---

type testProjectSetup struct {
	svc              *ProjectService
	projectRepo      *mockProjectRepo
	memberRepo       *mockProjectMemberRepo
	userRepo         *mockUserRepo
	workflowRepo     *mockWorkflowRepo
	typeWorkflowRepo *mockTypeWorkflowRepo
	settingsRepo     *mockSystemSettingRepo
	inviteRepo       *mockProjectInviteRepo
	userSettingRepo  *mockUserSettingRepo
}

func newTestProjectSetup() *testProjectSetup {
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	userRepo := newMockUserRepo()
	workflowRepo := newMockWorkflowRepo()
	typeWorkflowRepo := newMockTypeWorkflowRepo()
	settingsRepo := newMockSystemSettingRepo()
	inviteRepo := newMockProjectInviteRepo()
	userSettingRepo := newMockUserSettingRepo()
	svc := NewProjectService(projectRepo, memberRepo, userRepo, workflowRepo, typeWorkflowRepo, settingsRepo, inviteRepo, noopInboxRepo{}, noopWatcherRepo{}, userSettingRepo)
	return &testProjectSetup{
		svc:              svc,
		projectRepo:      projectRepo,
		memberRepo:       memberRepo,
		userRepo:         userRepo,
		workflowRepo:     workflowRepo,
		typeWorkflowRepo: typeWorkflowRepo,
		settingsRepo:     settingsRepo,
		inviteRepo:       inviteRepo,
		userSettingRepo:  userSettingRepo,
	}
}

func newTestProjectService() (*ProjectService, *mockProjectRepo, *mockProjectMemberRepo, *mockUserRepo) {
	s := newTestProjectSetup()
	return s.svc, s.projectRepo, s.memberRepo, s.userRepo
}

func adminAuthInfo() *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "admin@example.com",
		GlobalRole: model.RoleAdmin,
	}
}

func userAuthInfo() *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "user@example.com",
		GlobalRole: model.RoleUser,
	}
}

// --- Tests ---

func TestCreateProject_Success(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	desc := "Test project"
	project, err := svc.Create(context.Background(), info, "Test Project", "TEST", &desc, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if project.Name != "Test Project" {
		t.Fatalf("expected name 'Test Project', got %s", project.Name)
	}
	if project.Key != "TEST" {
		t.Fatalf("expected key 'TEST', got %s", project.Key)
	}
}

func TestCreateProject_InvalidKey(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	tests := []struct {
		name string
		key  string
	}{
		{"lowercase", "test"},
		{"too short", "T"},
		{"starts with digit", "1TEST"},
		{"has spaces", "TE ST"},
		{"has special chars", "TE-ST"},
		{"six chars", "ABCDEF"},
		{"too long", "TOOLONGKEYNAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), info, "Test", tt.key, nil, nil)
			if err == nil {
				t.Fatalf("expected error for key %q, got nil", tt.key)
			}
		})
	}
}

func TestCreateProject_DuplicateKey(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "First", "DUPE", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Create(context.Background(), info, "Second", "DUPE", nil, nil)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestCreateProject_DenyListKey(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()

	// Add a key to the deny list
	s.settingsRepo.settings[model.SettingReservedProjectKeys] = &model.SystemSetting{
		Key:   model.SettingReservedProjectKeys,
		Value: []byte(`["ADMIN","SYS"]`),
	}

	_, err := s.svc.Create(context.Background(), info, "Admin Project", "ADMIN", nil, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for deny-listed key, got %v", err)
	}

	// A non-denied key should succeed
	proj, err := s.svc.Create(context.Background(), info, "Good Project", "GOOD", nil, nil)
	if err != nil {
		t.Fatalf("expected no error for allowed key, got %v", err)
	}
	if proj.Key != "GOOD" {
		t.Fatalf("expected key 'GOOD', got %q", proj.Key)
	}
}

func TestCreateProject_CreatorBecomesOwner(t *testing.T) {
	svc, _, memberRepo, _ := newTestProjectService()
	info := userAuthInfo()

	project, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	member, err := memberRepo.GetByProjectAndUser(context.Background(), project.ID, info.UserID)
	if err != nil {
		t.Fatalf("expected creator to be a member, got %v", err)
	}
	if member.Role != model.ProjectRoleOwner {
		t.Fatalf("expected creator role 'owner', got %s", member.Role)
	}
}

func TestGetProject_MemberCanAccess(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	project, err := svc.Get(context.Background(), info, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if project.Key != "TT" {
		t.Fatalf("expected key 'TT', got %s", project.Key)
	}
}

func TestGetProject_NonMemberGetNotFound(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()
	other := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Get(context.Background(), other, "TT")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

func TestGetProject_AdminCanAccess(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()
	admin := adminAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	project, err := svc.Get(context.Background(), admin, "TT")
	if err != nil {
		t.Fatalf("expected admin to access project, got %v", err)
	}
	if project.Key != "TT" {
		t.Fatalf("expected key 'TT', got %s", project.Key)
	}
}

func TestUpdateProject_OwnerCanUpdate(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	newName := "Updated Name"
	project, err := svc.Update(context.Background(), info, "TT", &newName, nil, nil, false, nil, nil, false, nil, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if project.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %s", project.Name)
	}
}

func TestUpdateProject_MemberCannotUpdate(t *testing.T) {
	svc, _, memberRepo, _ := newTestProjectService()
	owner := userAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add a regular member
	memberInfo := userAuthInfo()
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    memberInfo.UserID,
		Role:      model.ProjectRoleMember,
	})

	newName := "Hacked"
	_, err = svc.Update(context.Background(), memberInfo, "TT", &newName, nil, nil, false, nil, nil, false, nil, false)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteProject_OwnerCanDelete(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(context.Background(), info, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not be found after deletion
	_, err = svc.Get(context.Background(), info, "TT")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteProject_AdminRoleCannotDelete(t *testing.T) {
	svc, _, memberRepo, _ := newTestProjectService()
	owner := userAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add project admin (not owner)
	adminMember := userAuthInfo()
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    adminMember.UserID,
		Role:      model.ProjectRoleAdmin,
	})

	err = svc.Delete(context.Background(), adminMember, "TT")
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden for project admin delete, got %v", err)
	}
}

func TestAddMember_Success(t *testing.T) {
	svc, _, _, userRepo := newTestProjectService()
	owner := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create the target user in the user repo
	newUserID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:       newUserID,
		Email:    "new@example.com",
		IsActive: true,
	})

	member, err := svc.AddMember(context.Background(), owner, "TT", newUserID, model.ProjectRoleMember)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if member.Role != model.ProjectRoleMember {
		t.Fatalf("expected role 'member', got %s", member.Role)
	}
}

func TestAddMember_DuplicateFails(t *testing.T) {
	svc, _, _, userRepo := newTestProjectService()
	owner := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	newUserID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:       newUserID,
		Email:    "new@example.com",
		IsActive: true,
	})

	_, err = svc.AddMember(context.Background(), owner, "TT", newUserID, model.ProjectRoleMember)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.AddMember(context.Background(), owner, "TT", newUserID, model.ProjectRoleMember)
	if err == nil {
		t.Fatal("expected error for duplicate member")
	}
}

func TestAddMember_MemberCannotAdd(t *testing.T) {
	svc, _, memberRepo, userRepo := newTestProjectService()
	owner := userAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add a regular member
	memberInfo := userAuthInfo()
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    memberInfo.UserID,
		Role:      model.ProjectRoleMember,
	})

	targetID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:       targetID,
		Email:    "target@example.com",
		IsActive: true,
	})

	_, err = svc.AddMember(context.Background(), memberInfo, "TT", targetID, model.ProjectRoleMember)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateMemberRole_Success(t *testing.T) {
	svc, _, memberRepo, userRepo := newTestProjectService()
	owner := userAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	memberID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:       memberID,
		Email:    "member@example.com",
		IsActive: true,
	})
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    memberID,
		Role:      model.ProjectRoleMember,
	})

	err = svc.UpdateMemberRole(context.Background(), owner, "TT", memberID, model.ProjectRoleAdmin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify role changed
	member, _ := memberRepo.GetByProjectAndUser(context.Background(), project.ID, memberID)
	if member.Role != model.ProjectRoleAdmin {
		t.Fatalf("expected role 'admin', got %s", member.Role)
	}
}

func TestUpdateMemberRole_CannotDemoteLastOwner(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Try to demote the only owner
	err = svc.UpdateMemberRole(context.Background(), owner, "TT", owner.UserID, model.ProjectRoleMember)
	if err == nil {
		t.Fatal("expected error when demoting last owner")
	}
}

func TestRemoveMember_Success(t *testing.T) {
	svc, _, memberRepo, userRepo := newTestProjectService()
	owner := userAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	memberID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:       memberID,
		Email:    "member@example.com",
		IsActive: true,
	})
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    memberID,
		Role:      model.ProjectRoleMember,
	})

	err = svc.RemoveMember(context.Background(), owner, "TT", memberID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify member removed
	_, err = memberRepo.GetByProjectAndUser(context.Background(), project.ID, memberID)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound after removal, got %v", err)
	}
}

func TestRemoveMember_CannotRemoveLastOwner(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.RemoveMember(context.Background(), owner, "TT", owner.UserID)
	if err == nil {
		t.Fatal("expected error when removing last owner")
	}
}

func TestListProjects_UserSeesOwnProjects(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), info, "Project 2", "P2", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	projects, err := svc.List(context.Background(), info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestListMembers(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	members, _, err := svc.ListMembers(context.Background(), owner, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member (owner), got %d", len(members))
	}
}

func TestListMembers_ViewerSeesOnlyOwnersAdminsAndSelf(t *testing.T) {
	svc, _, _, userRepo := newTestProjectService()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add an admin member
	adminUser := &model.User{ID: uuid.New(), Email: "padmin@example.com", IsActive: true}
	userRepo.Create(ctx, adminUser)
	adminInfo := &model.AuthInfo{UserID: adminUser.ID, Email: adminUser.Email, GlobalRole: model.RoleUser}
	svc.AddMember(ctx, owner, "TT", adminUser.ID, model.ProjectRoleAdmin)

	// Add a regular member
	regularUser := &model.User{ID: uuid.New(), Email: "regular@example.com", IsActive: true}
	userRepo.Create(ctx, regularUser)
	svc.AddMember(ctx, owner, "TT", regularUser.ID, model.ProjectRoleMember)

	// Add a viewer
	viewerUser := &model.User{ID: uuid.New(), Email: "viewer@example.com", IsActive: true}
	userRepo.Create(ctx, viewerUser)
	svc.AddMember(ctx, owner, "TT", viewerUser.ID, model.ProjectRoleViewer)

	viewerInfo := &model.AuthInfo{UserID: viewerUser.ID, Email: viewerUser.Email, GlobalRole: model.RoleUser}

	// Viewer should only see owner, admin, and themselves
	members, totalCount, err := svc.ListMembers(ctx, viewerInfo, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("expected 3 members (owner + admin + self), got %d", len(members))
	}
	if totalCount != 4 {
		t.Fatalf("expected total_count 4, got %d", totalCount)
	}
	roles := map[string]bool{}
	for _, m := range members {
		roles[m.Role] = true
	}
	if !roles["owner"] || !roles["admin"] || !roles["viewer"] {
		t.Fatalf("expected owner, admin, and viewer roles, got %v", roles)
	}

	// Non-viewer (admin) should see all 4 members
	allMembers, adminTotal, err := svc.ListMembers(ctx, adminInfo, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(allMembers) != 4 {
		t.Fatalf("expected 4 members for admin, got %d", len(allMembers))
	}
	if adminTotal != 4 {
		t.Fatalf("expected total_count 4 for admin, got %d", adminTotal)
	}
}

func TestAddMember_InvalidRole(t *testing.T) {
	svc, _, _, userRepo := newTestProjectService()
	owner := userAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	targetID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:       targetID,
		Email:    "target@example.com",
		IsActive: true,
	})

	_, err = svc.AddMember(context.Background(), owner, "TT", targetID, "superadmin")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestGlobalAdminCanDeleteAnyProject(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()
	admin := adminAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(context.Background(), admin, "TT")
	if err != nil {
		t.Fatalf("expected global admin to delete project, got %v", err)
	}
}

func TestUpdateProject_ChangeKey(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	newKey := "NKEY"
	project, err := svc.Update(context.Background(), info, "TT", nil, &newKey, nil, false, nil, nil, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Key != "NKEY" {
		t.Fatalf("expected key 'NKEY', got %s", project.Key)
	}
}

func TestUpdateProject_InvalidKey(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	badKey := "bad-key"
	_, err = svc.Update(context.Background(), info, "TT", nil, &badKey, nil, false, nil, nil, false, nil, false)
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestUpdateProject_DuplicateKey(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Project A", "AA", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), info, "Project B", "BB", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dupKey := "AA"
	_, err = svc.Update(context.Background(), info, "BB", nil, &dupKey, nil, false, nil, nil, false, nil, false)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestUpdateProject_Description(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	desc := "New description"
	project, err := svc.Update(context.Background(), info, "TT", nil, nil, &desc, false, nil, nil, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Description == nil || *project.Description != "New description" {
		t.Fatal("expected description to be set")
	}

	// Clear description
	project, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, true, nil, nil, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Description != nil {
		t.Fatal("expected description to be cleared")
	}
}

func TestUpdateProject_DefaultWorkflow(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	wfID := uuid.New()
	project, err := svc.Update(context.Background(), info, "TT", nil, nil, nil, false, &wfID, nil, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.DefaultWorkflowID == nil || *project.DefaultWorkflowID != wfID {
		t.Fatal("expected default workflow to be set")
	}
}

func TestListProject_AdminSeesAll(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()
	admin := adminAuthInfo()

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	projects, err := svc.List(context.Background(), admin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) < 1 {
		t.Fatal("expected admin to see all projects")
	}
}

func TestListProject_AdminHidesNonMemberProjects(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()

	_, err := s.svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Without preference, admin should call ListAll
	s.projectRepo.listAllCalled = false
	s.projectRepo.listByUserCalled = false
	_, err = s.svc.List(context.Background(), admin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.projectRepo.listAllCalled {
		t.Fatal("expected ListAll to be called for admin without preference")
	}
	if s.projectRepo.listByUserCalled {
		t.Fatal("expected ListByUser NOT to be called for admin without preference")
	}

	// Set the hide preference for admin
	s.userSettingRepo.settings[settingKey(admin.UserID, nil, "hide_non_member_projects")] = &model.UserSetting{
		UserID: admin.UserID,
		Key:    "hide_non_member_projects",
		Value:  json.RawMessage(`true`),
	}

	// With preference, admin should call ListByUser
	s.projectRepo.listAllCalled = false
	s.projectRepo.listByUserCalled = false
	_, err = s.svc.List(context.Background(), admin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.projectRepo.listAllCalled {
		t.Fatal("expected ListAll NOT to be called for admin with hide preference")
	}
	if !s.projectRepo.listByUserCalled {
		t.Fatal("expected ListByUser to be called for admin with hide preference")
	}
}

func TestListMembers_AdminBypass(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()
	admin := adminAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = project

	members, _, err := svc.ListMembers(context.Background(), admin, "TT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Admin should be able to list even without being a member
	_ = members
}

func TestRemoveMember_OwnerRemovesMember(t *testing.T) {
	svc, _, memberRepo, userRepo := newTestProjectService()
	owner := userAuthInfo()
	target := userAuthInfo()

	// Create user record for target
	userRepo.Create(context.Background(), &model.User{
		ID:       target.UserID,
		Email:    "target@example.com",
		IsActive: true,
	})

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    target.UserID,
		Role:      model.ProjectRoleMember,
	})

	err = svc.RemoveMember(context.Background(), owner, "TT", target.UserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Type Workflow Tests ---

func TestCreateProject_SeedsTypeWorkflows(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()

	// Seed two default workflows matching the expected names
	taskWf := &model.Workflow{ID: uuid.New(), Name: "Task Workflow", IsDefault: true}
	ticketWf := &model.Workflow{ID: uuid.New(), Name: "Ticket Workflow", IsDefault: true}
	s.workflowRepo.Create(context.Background(), taskWf)
	s.workflowRepo.Create(context.Background(), ticketWf)

	project, err := s.svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify 5 mappings were created
	mappings, err := s.typeWorkflowRepo.ListByProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("expected no error listing mappings, got %v", err)
	}
	if len(mappings) != 5 {
		t.Fatalf("expected 5 type-workflow mappings, got %d", len(mappings))
	}

	// Check specific mappings
	byType := make(map[string]uuid.UUID)
	for _, m := range mappings {
		byType[m.WorkItemType] = m.WorkflowID
	}
	if byType[model.WorkItemTypeTask] != taskWf.ID {
		t.Errorf("expected task→Task Workflow, got %v", byType[model.WorkItemTypeTask])
	}
	if byType[model.WorkItemTypeBug] != taskWf.ID {
		t.Errorf("expected bug→Task Workflow, got %v", byType[model.WorkItemTypeBug])
	}
	if byType[model.WorkItemTypeEpic] != taskWf.ID {
		t.Errorf("expected epic→Task Workflow, got %v", byType[model.WorkItemTypeEpic])
	}
	if byType[model.WorkItemTypeTicket] != ticketWf.ID {
		t.Errorf("expected ticket→Ticket Workflow, got %v", byType[model.WorkItemTypeTicket])
	}
	if byType[model.WorkItemTypeFeedback] != ticketWf.ID {
		t.Errorf("expected feedback→Ticket Workflow, got %v", byType[model.WorkItemTypeFeedback])
	}
}

func TestCreateProject_SeedsWithSingleWorkflow(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()

	// Only one workflow — all types should map to it
	wf := &model.Workflow{ID: uuid.New(), Name: "Custom Workflow", IsDefault: true}
	s.workflowRepo.Create(context.Background(), wf)

	project, err := s.svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mappings, err := s.typeWorkflowRepo.ListByProject(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 5 {
		t.Fatalf("expected 5 mappings, got %d", len(mappings))
	}
	for _, m := range mappings {
		if m.WorkflowID != wf.ID {
			t.Errorf("expected all types to map to single workflow, type %s maps to %v", m.WorkItemType, m.WorkflowID)
		}
	}
}

func TestGetTypeWorkflows_Success(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()

	project, err := s.svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Manually add a mapping
	wfID := uuid.New()
	s.typeWorkflowRepo.Upsert(context.Background(), &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: wfID,
	})

	mappings, err := s.svc.GetTypeWorkflows(context.Background(), info, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mappings) < 1 {
		t.Fatal("expected at least 1 mapping")
	}
}

func TestGetTypeWorkflows_NonMemberDenied(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	other := userAuthInfo()

	_, err := s.svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.GetTypeWorkflows(context.Background(), other, "TT")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

func TestUpdateTypeWorkflow_Success(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()

	_, err := s.svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a workflow to assign
	wf := &model.Workflow{ID: uuid.New(), Name: "New Workflow"}
	s.workflowRepo.Create(context.Background(), wf)

	mapping, err := s.svc.UpdateTypeWorkflow(context.Background(), info, "TT", model.WorkItemTypeTask, wf.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mapping.WorkflowID != wf.ID {
		t.Errorf("expected workflow ID %v, got %v", wf.ID, mapping.WorkflowID)
	}
	if mapping.WorkItemType != model.WorkItemTypeTask {
		t.Errorf("expected type 'task', got %s", mapping.WorkItemType)
	}
}

func TestUpdateTypeWorkflow_MemberDenied(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	member := userAuthInfo()

	project, err := s.svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add as regular member
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID,
		UserID: member.UserID, Role: model.ProjectRoleMember,
	})

	wf := &model.Workflow{ID: uuid.New(), Name: "Wf"}
	s.workflowRepo.Create(context.Background(), wf)

	_, err = s.svc.UpdateTypeWorkflow(context.Background(), member, "TT", model.WorkItemTypeTask, wf.ID)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSeedExistingProjectTypeWorkflows_BackfillsMissing(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Create project without workflows available (so no seeding happens during Create)
	project, err := s.svc.Create(ctx, info, "Legacy", "LEG", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify no mappings exist
	mappings, _ := s.typeWorkflowRepo.ListByProject(ctx, project.ID)
	if len(mappings) != 0 {
		t.Fatalf("expected 0 mappings before backfill, got %d", len(mappings))
	}

	// Now add workflows and run backfill
	taskWf := &model.Workflow{ID: uuid.New(), Name: "Task Workflow", IsDefault: true}
	ticketWf := &model.Workflow{ID: uuid.New(), Name: "Ticket Workflow", IsDefault: true}
	s.workflowRepo.Create(ctx, taskWf)
	s.workflowRepo.Create(ctx, ticketWf)

	err = s.svc.SeedExistingProjectTypeWorkflows(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify mappings were backfilled
	mappings, _ = s.typeWorkflowRepo.ListByProject(ctx, project.ID)
	if len(mappings) != 5 {
		t.Fatalf("expected 5 mappings after backfill, got %d", len(mappings))
	}
}

func TestSeedExistingProjectTypeWorkflows_SkipsExisting(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Seed workflows first so Create() auto-seeds mappings
	taskWf := &model.Workflow{ID: uuid.New(), Name: "Task Workflow", IsDefault: true}
	s.workflowRepo.Create(ctx, taskWf)

	project, err := s.svc.Create(ctx, info, "Modern", "MOD", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should already have mappings
	before, _ := s.typeWorkflowRepo.ListByProject(ctx, project.ID)
	if len(before) != 5 {
		t.Fatalf("expected 5 pre-existing mappings, got %d", len(before))
	}

	// Now change one mapping to a custom workflow
	customWf := &model.Workflow{ID: uuid.New(), Name: "Custom", IsDefault: false}
	s.workflowRepo.Create(ctx, customWf)
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: customWf.ID,
	})

	// Run backfill — should NOT overwrite existing mappings
	err = s.svc.SeedExistingProjectTypeWorkflows(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	after, _ := s.typeWorkflowRepo.ListByProject(ctx, project.ID)
	byType := make(map[string]uuid.UUID)
	for _, m := range after {
		byType[m.WorkItemType] = m.WorkflowID
	}

	// Task should still have the custom workflow, not overwritten
	if byType[model.WorkItemTypeTask] != customWf.ID {
		t.Errorf("expected task mapping preserved as custom workflow, got %v", byType[model.WorkItemTypeTask])
	}
}

func TestUpdateTypeWorkflow_WorkflowNotFound(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()

	_, err := s.svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UpdateTypeWorkflow(context.Background(), info, "TT", model.WorkItemTypeTask, uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent workflow")
	}
}

func TestUpdateProject_AllowedComplexityValues(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set allowed complexity values
	vals := []int{1, 2, 3, 5, 8, 13}
	project, err := svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, vals, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(project.AllowedComplexityValues) != 6 {
		t.Fatalf("expected 6 allowed values, got %d", len(project.AllowedComplexityValues))
	}

	// Clear allowed complexity values
	project, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, nil, true, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(project.AllowedComplexityValues) != 0 {
		t.Fatalf("expected empty allowed values, got %d", len(project.AllowedComplexityValues))
	}
}

func TestUpdateProject_AllowedComplexityValues_InvalidValues(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Zero value should fail
	_, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, []int{0, 1, 2}, false, nil, false)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for zero value, got %v", err)
	}

	// Negative value should fail
	_, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, []int{-1, 3}, false, nil, false)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for negative value, got %v", err)
	}

	// Duplicates should fail
	_, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, []int{1, 3, 3}, false, nil, false)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for duplicate values, got %v", err)
	}
}

// --- Project Limit Tests ---

func TestCreateProject_LimitEnforced(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Set limit to 2
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`2`),
	})

	// First two should succeed
	_, err := s.svc.Create(ctx, info, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatalf("expected no error for project 1, got %v", err)
	}
	_, err = s.svc.Create(ctx, info, "Project 2", "P2", nil, nil)
	if err != nil {
		t.Fatalf("expected no error for project 2, got %v", err)
	}

	// Third should fail
	_, err = s.svc.Create(ctx, info, "Project 3", "P3", nil, nil)
	if err == nil {
		t.Fatal("expected error for project 3 exceeding limit")
	}
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateProject_LimitAtExactBoundary(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Set limit to 1
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`1`),
	})

	// First project at the boundary — should succeed
	_, err := s.svc.Create(ctx, info, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatalf("expected no error at boundary, got %v", err)
	}

	// Second project exceeds — should fail
	_, err = s.svc.Create(ctx, info, "Project 2", "P2", nil, nil)
	if err == nil {
		t.Fatal("expected error exceeding limit of 1")
	}
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateProject_LimitErrorMessageContainsCount(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`1`),
	})

	_, _ = s.svc.Create(ctx, info, "Project 1", "P1", nil, nil)
	_, err := s.svc.Create(ctx, info, "Project 2", "P2", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should mention limit reached and tell user to contact admin
	errMsg := err.Error()
	if !strings.Contains(errMsg, "ownership limit reached") || !strings.Contains(errMsg, "administrator") {
		t.Fatalf("expected error message with limit and admin contact info, got: %s", errMsg)
	}
}

func TestCreateProject_AdminBypassesLimit(t *testing.T) {
	s := newTestProjectSetup()
	admin := adminAuthInfo()
	ctx := context.Background()

	// Set limit to 1
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`1`),
	})

	// Admin should bypass the limit entirely
	_, err := s.svc.Create(ctx, admin, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = s.svc.Create(ctx, admin, "Project 2", "P2", nil, nil)
	if err != nil {
		t.Fatalf("expected admin to bypass limit, got %v", err)
	}
	_, err = s.svc.Create(ctx, admin, "Project 3", "P3", nil, nil)
	if err != nil {
		t.Fatalf("expected admin to bypass limit, got %v", err)
	}
}

func TestCreateProject_DefaultLimitWhenSettingNotConfigured(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// No setting configured — default is 5
	for i := range 5 {
		key := string(rune('A'+i)) + "A"
		_, err := s.svc.Create(ctx, info, "Project", key, nil, nil)
		if err != nil {
			t.Fatalf("expected no error for project %d, got %v", i+1, err)
		}
	}

	// 6th should fail with default limit
	_, err := s.svc.Create(ctx, info, "Project 6", "FA", nil, nil)
	if err == nil {
		t.Fatal("expected error for project exceeding default limit of 5")
	}
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateProject_ZeroLimitMeansUnlimited(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Set limit to 0 — should mean unlimited
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`0`),
	})

	// Should be able to create many projects
	for i := range 10 {
		key := string(rune('A'+i)) + "A"
		_, err := s.svc.Create(ctx, info, "Project", key, nil, nil)
		if err != nil {
			t.Fatalf("expected no error with unlimited (0) limit for project %d, got %v", i+1, err)
		}
	}
}

func TestCreateProject_LimitUpdatedDynamically(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Start with limit of 2
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`2`),
	})

	_, err := s.svc.Create(ctx, info, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.Create(ctx, info, "Project 2", "P2", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// At limit — should fail
	_, err = s.svc.Create(ctx, info, "Project 3", "P3", nil, nil)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden at limit, got %v", err)
	}

	// Admin raises the limit to 5
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`5`),
	})

	// Now creation should succeed again
	_, err = s.svc.Create(ctx, info, "Project 3", "P3", nil, nil)
	if err != nil {
		t.Fatalf("expected creation to succeed after limit increase, got %v", err)
	}
}

func TestCreateProject_InvalidSettingValueUsesDefault(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Set an invalid (non-numeric) JSON value
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`"not a number"`),
	})

	// Should fall back to default (5)
	for i := range 5 {
		key := string(rune('A'+i)) + "A"
		_, err := s.svc.Create(ctx, info, "Project", key, nil, nil)
		if err != nil {
			t.Fatalf("expected no error for project %d with invalid setting, got %v", i+1, err)
		}
	}

	_, err := s.svc.Create(ctx, info, "Project 6", "FA", nil, nil)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden with default limit after invalid setting, got %v", err)
	}
}

func TestCreateProject_NegativeSettingValueUsesDefault(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Set a negative value — should be ignored, use default (5)
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`-3`),
	})

	for i := range 5 {
		key := string(rune('A'+i)) + "A"
		_, err := s.svc.Create(ctx, info, "Project", key, nil, nil)
		if err != nil {
			t.Fatalf("expected no error for project %d, got %v", i+1, err)
		}
	}

	_, err := s.svc.Create(ctx, info, "Project 6", "FA", nil, nil)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden with default limit, got %v", err)
	}
}

func TestCreateProject_DeletedProjectsDoNotCountTowardLimit(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`2`),
	})

	// Create 2 projects (at limit)
	_, err := s.svc.Create(ctx, info, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.Create(ctx, info, "Project 2", "P2", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Delete one
	err = s.svc.Delete(ctx, info, "P1")
	if err != nil {
		t.Fatal(err)
	}

	// Now should be able to create again (1 active, limit 2)
	_, err = s.svc.Create(ctx, info, "Project 3", "P3", nil, nil)
	if err != nil {
		t.Fatalf("expected creation to succeed after deleting a project, got %v", err)
	}
}


func TestCreateProject_PerUserLimitOverridesGlobal(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Register user in user repo with per-user limit of 2
	perUserLimit := 2
	s.userRepo.Create(ctx, &model.User{
		ID:          info.UserID,
		Email:       info.Email,
		IsActive:    true,
		MaxProjects: &perUserLimit,
	})

	// Set global limit to 10 — should be ignored because per-user limit takes precedence
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`10`),
	})

	// First two should succeed (per-user limit is 2)
	_, err := s.svc.Create(ctx, info, "Project 1", "P1", nil, nil)
	if err != nil {
		t.Fatalf("expected no error for project 1, got %v", err)
	}
	_, err = s.svc.Create(ctx, info, "Project 2", "P2", nil, nil)
	if err != nil {
		t.Fatalf("expected no error for project 2, got %v", err)
	}

	// Third should fail (per-user limit of 2 overrides global 10)
	_, err = s.svc.Create(ctx, info, "Project 3", "P3", nil, nil)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden with per-user limit override, got %v", err)
	}
}

func TestCreateProject_PerUserLimitZeroMeansUnlimited(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// Register user with per-user limit of 0 (unlimited)
	perUserLimit := 0
	s.userRepo.Create(ctx, &model.User{
		ID:          info.UserID,
		Email:       info.Email,
		IsActive:    true,
		MaxProjects: &perUserLimit,
	})

	// Set global limit to 1
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`1`),
	})

	// Should be able to create many projects (per-user 0 = unlimited overrides global 1)
	for i := range 5 {
		key := string(rune('A'+i)) + "A"
		_, err := s.svc.Create(ctx, info, "Project", key, nil, nil)
		if err != nil {
			t.Fatalf("expected no error with per-user unlimited limit for project %d, got %v", i+1, err)
		}
	}
}

func TestResolveEffectiveLimit_AdminGetsUnlimited(t *testing.T) {
	s := newTestProjectSetup()
	admin := adminAuthInfo()
	ctx := context.Background()

	// Set a restrictive global limit
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`1`),
	})

	limit, err := s.svc.ResolveEffectiveLimit(ctx, admin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if limit != 0 {
		t.Fatalf("expected 0 (unlimited) for admin, got %d", limit)
	}
}

func TestResolveEffectiveLimit_UserGetsPerUserLimit(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	perUserLimit := 8
	s.userRepo.Create(ctx, &model.User{
		ID:          info.UserID,
		Email:       info.Email,
		IsActive:    true,
		MaxProjects: &perUserLimit,
	})

	// Global limit of 3 should be overridden
	s.settingsRepo.Upsert(ctx, &model.SystemSetting{
		Key:   model.SettingMaxProjectsPerUser,
		Value: json.RawMessage(`3`),
	})

	limit, err := s.svc.ResolveEffectiveLimit(ctx, info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if limit != 8 {
		t.Fatalf("expected 8 (per-user limit), got %d", limit)
	}
}

func TestResolveEffectiveLimit_UserGetsGlobalDefault(t *testing.T) {
	s := newTestProjectSetup()
	info := userAuthInfo()
	ctx := context.Background()

	// User without per-user limit
	s.userRepo.Create(ctx, &model.User{
		ID:       info.UserID,
		Email:    info.Email,
		IsActive: true,
	})

	limit, err := s.svc.ResolveEffectiveLimit(ctx, info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if limit != model.DefaultMaxProjectsPerUser {
		t.Fatalf("expected default %d, got %d", model.DefaultMaxProjectsPerUser, limit)
	}
}

func TestValidateBusinessHours(t *testing.T) {
	tests := []struct {
		name    string
		config  *model.BusinessHoursConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 17, Timezone: "UTC",
			},
			wantErr: false,
		},
		{
			name: "valid config with named timezone",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 8, EndHour: 18, Timezone: "America/New_York",
			},
			wantErr: false,
		},
		{
			name: "empty timezone",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 17, Timezone: "",
			},
			wantErr: true,
		},
		{
			name: "invalid timezone",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 17, Timezone: "Invalid/Timezone",
			},
			wantErr: true,
		},
		{
			name: "start_hour negative",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: -1, EndHour: 17, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "start_hour too high",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 24, EndHour: 17, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "end_hour too high",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 24, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "end_hour equals start_hour",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 9, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "end_hour less than start_hour",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 2, 3, 4, 5}, StartHour: 17, EndHour: 9, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "no days",
			config: &model.BusinessHoursConfig{
				Days: []int{}, StartHour: 9, EndHour: 17, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "invalid day value",
			config: &model.BusinessHoursConfig{
				Days: []int{1, 7}, StartHour: 9, EndHour: 17, Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "negative day value",
			config: &model.BusinessHoursConfig{
				Days: []int{-1, 1}, StartHour: 9, EndHour: 17, Timezone: "UTC",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBusinessHours(tt.config)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

// --- Invite Tests ---

func TestCreateInvite_Success(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if invite.Role != model.ProjectRoleMember {
		t.Fatalf("expected role 'member', got %s", invite.Role)
	}
	if len(invite.Code) != 8 {
		t.Fatalf("expected 8-char code, got %d chars: %s", len(invite.Code), invite.Code)
	}
	if invite.MaxUses != 0 {
		t.Fatalf("expected max_uses 0, got %d", invite.MaxUses)
	}
}

func TestCreateInvite_WithExpiration(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleViewer, &expiresAt, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if invite.ExpiresAt == nil {
		t.Fatal("expected expires_at to be set")
	}
	if invite.MaxUses != 10 {
		t.Fatalf("expected max_uses 10, got %d", invite.MaxUses)
	}
}

func TestCreateInvite_OwnerRoleDenied(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleOwner, nil, 0)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for owner role, got %v", err)
	}
}

func TestCreateInvite_MemberCannotCreate(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	member := userAuthInfo()
	ctx := context.Background()

	project, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	s.memberRepo.Add(ctx, &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID,
		UserID: member.UserID, Role: model.ProjectRoleMember,
	})

	_, err = s.svc.CreateInvite(ctx, member, "TT", model.ProjectRoleMember, nil, 0)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestListInvites_Success(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleViewer, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	invites, err := s.svc.ListInvites(ctx, owner, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(invites))
	}
}

func TestDeleteInvite_Success(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = s.svc.DeleteInvite(ctx, owner, "TT", invite.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	invites, _ := s.svc.ListInvites(ctx, owner, "TT")
	if len(invites) != 0 {
		t.Fatalf("expected 0 invites after delete, got %d", len(invites))
	}
}

func TestAcceptInvite_Success(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	joiner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.svc.AcceptInvite(ctx, joiner, invite.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Project.Key != "TT" {
		t.Fatalf("expected project key 'TT', got %s", result.Project.Key)
	}

	// Verify membership was created
	members, _, _ := s.svc.ListMembers(ctx, owner, "TT")
	found := false
	for _, m := range members {
		if m.UserID == joiner.UserID && m.Role == model.ProjectRoleMember {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected joiner to be a member")
	}

	// Verify use count incremented
	updated, _ := s.inviteRepo.GetByCode(ctx, invite.Code)
	if updated.UseCount != 1 {
		t.Fatalf("expected use_count 1, got %d", updated.UseCount)
	}
}

func TestAcceptInvite_Expired(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	joiner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	expired := time.Now().Add(-time.Hour)
	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, &expired, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.AcceptInvite(ctx, joiner, invite.Code)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for expired invite, got %v", err)
	}
}

func TestAcceptInvite_MaxUsesReached(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 1)
	if err != nil {
		t.Fatal(err)
	}

	// First accept should work
	joiner1 := userAuthInfo()
	_, err = s.svc.AcceptInvite(ctx, joiner1, invite.Code)
	if err != nil {
		t.Fatalf("expected first accept to succeed, got %v", err)
	}

	// Second accept should fail
	joiner2 := userAuthInfo()
	_, err = s.svc.AcceptInvite(ctx, joiner2, invite.Code)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for max uses, got %v", err)
	}
}

func TestAcceptInvite_AlreadyMember_SameRole(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	joiner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Joiner accepts invite and becomes member
	_, err = s.svc.AcceptInvite(ctx, joiner, invite.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Create another member invite, joiner accepts again — should succeed silently
	invite2, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.AcceptInvite(ctx, joiner, invite2.Code)
	if err != nil {
		t.Fatalf("expected no error for same-role re-accept, got %v", err)
	}
}

func TestAcceptInvite_AlreadyMember_RoleUpgrade(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	joiner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Join as viewer first
	viewerInvite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleViewer, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.AcceptInvite(ctx, joiner, viewerInvite.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Now accept a member invite — role should be upgraded
	memberInvite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.AcceptInvite(ctx, joiner, memberInvite.Code)
	if err != nil {
		t.Fatalf("expected no error for role upgrade, got %v", err)
	}

	// Verify the role was updated
	members, _, err := s.svc.ListMembers(ctx, owner, "TT")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range members {
		if m.UserID == joiner.UserID {
			if m.Role != model.ProjectRoleMember {
				t.Fatalf("expected role %q after upgrade, got %q", model.ProjectRoleMember, m.Role)
			}
			return
		}
	}
	t.Fatal("joiner not found in members list")
}

func TestAcceptInvite_AlreadyMember_NoDowngrade(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	joiner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Join as admin first
	adminInvite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleAdmin, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.svc.AcceptInvite(ctx, joiner, adminInvite.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Now accept a viewer invite — role should NOT be downgraded
	viewerInvite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleViewer, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.svc.AcceptInvite(ctx, joiner, viewerInvite.Code)
	if err != nil {
		t.Fatalf("expected no error for no-downgrade, got %v", err)
	}

	// Verify the result indicates role was not applied
	if !result.RoleNotApplied {
		t.Fatal("expected RoleNotApplied to be true")
	}
	if result.ExistingRole != model.ProjectRoleAdmin {
		t.Fatalf("expected existing role %q, got %q", model.ProjectRoleAdmin, result.ExistingRole)
	}
	if result.InviteRole != model.ProjectRoleViewer {
		t.Fatalf("expected invite role %q, got %q", model.ProjectRoleViewer, result.InviteRole)
	}

	// Verify the role was NOT changed
	members, _, err := s.svc.ListMembers(ctx, owner, "TT")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range members {
		if m.UserID == joiner.UserID {
			if m.Role != model.ProjectRoleAdmin {
				t.Fatalf("expected role %q preserved, got %q", model.ProjectRoleAdmin, m.Role)
			}
			return
		}
	}
	t.Fatal("joiner not found in members list")
}

func TestAcceptInvite_NotFound(t *testing.T) {
	s := newTestProjectSetup()
	joiner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.AcceptInvite(ctx, joiner, "nonexistent")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetInviteInfo_Success(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test Project", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	info, err := s.svc.GetInviteInfo(ctx, invite.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.ProjectName != "Test Project" {
		t.Fatalf("expected project name 'Test Project', got %s", info.ProjectName)
	}
	if info.ProjectKey != "TT" {
		t.Fatalf("expected project key 'TT', got %s", info.ProjectKey)
	}
	if info.Role != model.ProjectRoleMember {
		t.Fatalf("expected role 'member', got %s", info.Role)
	}
	if info.Expired || info.Full {
		t.Fatal("expected not expired and not full")
	}
}

func TestGetInviteInfo_ExpiredAndFull(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	expired := time.Now().Add(-time.Hour)
	invite, err := s.svc.CreateInvite(ctx, owner, "TT", model.ProjectRoleMember, &expired, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Manually set use_count to max
	inv, _ := s.inviteRepo.GetByCode(ctx, invite.Code)
	inv.UseCount = 1

	info, err := s.svc.GetInviteInfo(ctx, invite.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !info.Expired {
		t.Fatal("expected expired to be true")
	}
	if !info.Full {
		t.Fatal("expected full to be true")
	}
}

// --- Email Invite Tests ---

func TestCreateEmailInvite_ExistingUser_DirectAdd(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a user that "exists"
	existingUser := &model.User{
		ID:          uuid.New(),
		Email:       "existing@test.com",
		DisplayName: "Existing User",
	}
	s.userRepo.Create(ctx, existingUser)

	result, err := s.svc.CreateEmailInvite(ctx, owner, "TT", "existing@test.com", model.ProjectRoleMember, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.DirectAdd {
		t.Fatal("expected DirectAdd to be true")
	}
	if result.Member == nil {
		t.Fatal("expected Member to be set")
	}
	if result.Member.UserID != existingUser.ID {
		t.Fatalf("expected member user ID %s, got %s", existingUser.ID, result.Member.UserID)
	}
	if result.Invite != nil {
		t.Fatal("expected Invite to be nil")
	}
}

func TestCreateEmailInvite_NonExistingUser_CreatesInvite(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.svc.CreateEmailInvite(ctx, owner, "TT", "newuser@test.com", model.ProjectRoleMember, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.DirectAdd {
		t.Fatal("expected DirectAdd to be false")
	}
	if result.Invite == nil {
		t.Fatal("expected Invite to be set")
	}
	if result.Invite.InviteeEmail == nil || *result.Invite.InviteeEmail != "newuser@test.com" {
		t.Fatalf("expected invitee_email 'newuser@test.com', got %v", result.Invite.InviteeEmail)
	}
	if result.Invite.MaxUses != 1 {
		t.Fatalf("expected max_uses 1, got %d", result.Invite.MaxUses)
	}
}

func TestCreateEmailInvite_AlreadyMember_Error(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	project, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	existingUser := &model.User{
		ID:          uuid.New(),
		Email:       "member@test.com",
		DisplayName: "Already Member",
	}
	s.userRepo.Create(ctx, existingUser)

	// Add them as a member
	s.memberRepo.Add(ctx, &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID,
		UserID: existingUser.ID, Role: model.ProjectRoleMember,
	})

	_, err = s.svc.CreateEmailInvite(ctx, owner, "TT", "member@test.com", model.ProjectRoleMember, nil)
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestCreateEmailInvite_OwnerRoleDenied(t *testing.T) {
	s := newTestProjectSetup()
	owner := userAuthInfo()
	ctx := context.Background()

	_, err := s.svc.Create(ctx, owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateEmailInvite(ctx, owner, "TT", "anyone@test.com", model.ProjectRoleOwner, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for owner role, got %v", err)
	}
}
