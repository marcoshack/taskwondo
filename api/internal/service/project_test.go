package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock repositories ---

type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
	byKey    map[string]*model.Project
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

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok || p.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]model.Project, error) {
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

// --- Test helpers ---

type testProjectSetup struct {
	svc              *ProjectService
	projectRepo      *mockProjectRepo
	memberRepo       *mockProjectMemberRepo
	userRepo         *mockUserRepo
	workflowRepo     *mockWorkflowRepo
	typeWorkflowRepo *mockTypeWorkflowRepo
}

func newTestProjectSetup() *testProjectSetup {
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	userRepo := newMockUserRepo()
	workflowRepo := newMockWorkflowRepo()
	typeWorkflowRepo := newMockTypeWorkflowRepo()
	svc := NewProjectService(projectRepo, memberRepo, userRepo, workflowRepo, typeWorkflowRepo)
	return &testProjectSetup{
		svc:              svc,
		projectRepo:      projectRepo,
		memberRepo:       memberRepo,
		userRepo:         userRepo,
		workflowRepo:     workflowRepo,
		typeWorkflowRepo: typeWorkflowRepo,
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
	project, err := svc.Update(context.Background(), info, "TT", &newName, nil, nil, false, nil, nil, false)
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
	_, err = svc.Update(context.Background(), memberInfo, "TT", &newName, nil, nil, false, nil, nil, false)
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

	members, err := svc.ListMembers(context.Background(), owner, "TT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member (owner), got %d", len(members))
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
	project, err := svc.Update(context.Background(), info, "TT", nil, &newKey, nil, false, nil, nil, false)
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
	_, err = svc.Update(context.Background(), info, "TT", nil, &badKey, nil, false, nil, nil, false)
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
	_, err = svc.Update(context.Background(), info, "BB", nil, &dupKey, nil, false, nil, nil, false)
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
	project, err := svc.Update(context.Background(), info, "TT", nil, nil, &desc, false, nil, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Description == nil || *project.Description != "New description" {
		t.Fatal("expected description to be set")
	}

	// Clear description
	project, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, true, nil, nil, false)
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
	project, err := svc.Update(context.Background(), info, "TT", nil, nil, nil, false, &wfID, nil, false)
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

func TestListMembers_AdminBypass(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	owner := userAuthInfo()
	admin := adminAuthInfo()

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = project

	members, err := svc.ListMembers(context.Background(), admin, "TT")
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
	project, err := svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, vals, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(project.AllowedComplexityValues) != 6 {
		t.Fatalf("expected 6 allowed values, got %d", len(project.AllowedComplexityValues))
	}

	// Clear allowed complexity values
	project, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, nil, true)
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
	_, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, []int{0, 1, 2}, false)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for zero value, got %v", err)
	}

	// Negative value should fail
	_, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, []int{-1, 3}, false)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for negative value, got %v", err)
	}

	// Duplicates should fail
	_, err = svc.Update(context.Background(), info, "TT", nil, nil, nil, false, nil, []int{1, 3, 3}, false)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for duplicate values, got %v", err)
	}
}
