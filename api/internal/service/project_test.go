package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
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

func newTestProjectService() (*ProjectService, *mockProjectRepo, *mockProjectMemberRepo, *mockUserRepo) {
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	userRepo := newMockUserRepo()
	svc := NewProjectService(projectRepo, memberRepo, userRepo)
	return svc, projectRepo, memberRepo, userRepo
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
	project, err := svc.Create(context.Background(), info, "Test Project", "TEST", &desc)
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
		{"too long", "TOOLONGKEYNAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), info, "Test", tt.key, nil)
			if err == nil {
				t.Fatalf("expected error for key %q, got nil", tt.key)
			}
		})
	}
}

func TestCreateProject_DuplicateKey(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "First", "DUPE", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Create(context.Background(), info, "Second", "DUPE", nil)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestCreateProject_CreatorBecomesOwner(t *testing.T) {
	svc, _, memberRepo, _ := newTestProjectService()
	info := userAuthInfo()

	project, err := svc.Create(context.Background(), info, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil)
	if err != nil {
		t.Fatal(err)
	}

	newName := "Updated Name"
	project, err := svc.Update(context.Background(), info, "TT", &newName, nil, nil, false)
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

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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
	_, err = svc.Update(context.Background(), memberInfo, "TT", &newName, nil, nil, false)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteProject_OwnerCanDelete(t *testing.T) {
	svc, _, _, _ := newTestProjectService()
	info := userAuthInfo()

	_, err := svc.Create(context.Background(), info, "Test", "TT", nil)
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

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	project, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), info, "Project 1", "P1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), info, "Project 2", "P2", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
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

	_, err := svc.Create(context.Background(), owner, "Test", "TT", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(context.Background(), admin, "TT")
	if err != nil {
		t.Fatalf("expected global admin to delete project, got %v", err)
	}
}
