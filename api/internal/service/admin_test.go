package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock repositories for admin service ---

type mockAdminUserRepo struct {
	byID    map[uuid.UUID]*model.User
	byEmail map[string]*model.User
}

func newMockAdminUserRepo() *mockAdminUserRepo {
	return &mockAdminUserRepo{
		byID:    make(map[uuid.UUID]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (m *mockAdminUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockAdminUserRepo) ListAll(_ context.Context) ([]model.User, error) {
	var result []model.User
	for _, u := range m.byID {
		result = append(result, *u)
	}
	return result, nil
}

func (m *mockAdminUserRepo) UpdateGlobalRole(_ context.Context, id uuid.UUID, role string) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.GlobalRole = role
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockAdminUserRepo) UpdateIsActive(_ context.Context, id uuid.UUID, isActive bool) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.IsActive = isActive
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockAdminUserRepo) CountByRole(_ context.Context, role string) (int, error) {
	count := 0
	for _, u := range m.byID {
		if u.GlobalRole == role && u.IsActive {
			count++
		}
	}
	return count, nil
}

func (m *mockAdminUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockAdminUserRepo) Create(_ context.Context, user *model.User) error {
	m.byID[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockAdminUserRepo) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string, forceChange bool) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.PasswordHash = hash
	u.ForcePasswordChange = forceChange
	return nil
}

func (m *mockAdminUserRepo) UpdateMaxProjects(_ context.Context, id uuid.UUID, maxProjects *int) error {
	u, ok := m.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	u.MaxProjects = maxProjects
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockAdminUserRepo) UpdateMaxNamespaces(_ context.Context, _ uuid.UUID, _ *int) error {
	return nil
}

func (m *mockAdminUserRepo) addUser(role string, active bool) *model.User {
	u := &model.User{
		ID:          uuid.New(),
		Email:       uuid.New().String()[:8] + "@test.com",
		DisplayName: "Test User",
		GlobalRole:  role,
		IsActive:    active,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return u
}

type mockAdminProjectRepo struct {
	byID map[uuid.UUID]*model.Project
}

func newMockAdminProjectRepo() *mockAdminProjectRepo {
	return &mockAdminProjectRepo{byID: make(map[uuid.UUID]*model.Project)}
}

func (m *mockAdminProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockAdminProjectRepo) addProject(key, name string) *model.Project {
	p := &model.Project{
		ID:        uuid.New(),
		Key:       key,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.byID[p.ID] = p
	return p
}

type mockAdminMemberRepo struct {
	members map[string]*model.ProjectMember // key: "projectID:userID"
}

func newMockAdminMemberRepo() *mockAdminMemberRepo {
	return &mockAdminMemberRepo{members: make(map[string]*model.ProjectMember)}
}

func adminMemberKey(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockAdminMemberRepo) Add(_ context.Context, member *model.ProjectMember) error {
	key := adminMemberKey(member.ProjectID, member.UserID)
	if _, exists := m.members[key]; exists {
		return model.ErrAlreadyExists
	}
	member.CreatedAt = time.Now()
	m.members[key] = member
	return nil
}

func (m *mockAdminMemberRepo) GetByProjectAndUser(_ context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error) {
	key := adminMemberKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return member, nil
}

func (m *mockAdminMemberRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]model.ProjectMemberWithProject, error) {
	var result []model.ProjectMemberWithProject
	for _, member := range m.members {
		if member.UserID == userID {
			ownerCount := 0
			for _, m2 := range m.members {
				if m2.ProjectID == member.ProjectID && m2.Role == "owner" {
					ownerCount++
				}
			}
			result = append(result, model.ProjectMemberWithProject{
				ProjectMember: *member,
				ProjectName:   "Test Project",
				ProjectKey:    "TEST",
				OwnerCount:    ownerCount,
			})
		}
	}
	return result, nil
}

func (m *mockAdminMemberRepo) CountByRole(_ context.Context, projectID uuid.UUID, role string) (int, error) {
	count := 0
	for _, member := range m.members {
		if member.ProjectID == projectID && member.Role == role {
			count++
		}
	}
	return count, nil
}

func (m *mockAdminMemberRepo) UpdateRole(_ context.Context, projectID, userID uuid.UUID, role string) error {
	key := adminMemberKey(projectID, userID)
	member, ok := m.members[key]
	if !ok {
		return model.ErrNotFound
	}
	member.Role = role
	return nil
}

func (m *mockAdminMemberRepo) Remove(_ context.Context, projectID, userID uuid.UUID) error {
	key := adminMemberKey(projectID, userID)
	if _, ok := m.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.members, key)
	return nil
}

// --- Test helpers ---

func newTestAdminService() (*AdminService, *mockAdminUserRepo, *mockAdminProjectRepo, *mockAdminMemberRepo) {
	userRepo := newMockAdminUserRepo()
	projectRepo := newMockAdminProjectRepo()
	memberRepo := newMockAdminMemberRepo()
	svc := NewAdminService(userRepo, projectRepo, memberRepo)
	return svc, userRepo, projectRepo, memberRepo
}

// --- Tests ---

func TestAdminListUsers(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	userRepo.addUser(model.RoleAdmin, true)
	userRepo.addUser(model.RoleUser, true)
	userRepo.addUser(model.RoleUser, false) // inactive user

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
}

func TestAdminUpdateUser_ChangeRole(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	role := model.RoleAdmin
	updated, err := svc.UpdateUser(ctx, caller.ID, target.ID, &role, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.GlobalRole != model.RoleAdmin {
		t.Fatalf("expected role admin, got %s", updated.GlobalRole)
	}
}

func TestAdminUpdateUser_InvalidRole(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	role := "superadmin"
	_, err := svc.UpdateUser(ctx, caller.ID, target.ID, &role, nil, nil, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestAdminUpdateUser_CannotChangeOwnRole(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)

	role := model.RoleUser
	_, err := svc.UpdateUser(ctx, caller.ID, caller.ID, &role, nil, nil, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestAdminUpdateUser_CannotDisableSelf(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)

	active := false
	_, err := svc.UpdateUser(ctx, caller.ID, caller.ID, nil, &active, nil, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestAdminUpdateUser_CannotRemoveLastAdmin(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleAdmin, true)

	// Demote caller (but we're calling as target, so demoting caller)
	role := model.RoleUser
	// First demote one admin — should succeed since there are 2
	_, err := svc.UpdateUser(ctx, target.ID, caller.ID, &role, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Now try to demote the last admin — should fail
	_, err = svc.UpdateUser(ctx, target.ID, target.ID, &role, nil, nil, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for last admin, got %v", err)
	}
}

func TestAdminUpdateUser_DisableUser(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	active := false
	updated, err := svc.UpdateUser(ctx, caller.ID, target.ID, nil, &active, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.IsActive {
		t.Fatal("expected user to be inactive")
	}
}

func TestAdminUpdateUser_NotFound(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)

	role := model.RoleAdmin
	_, err := svc.UpdateUser(ctx, caller.ID, uuid.New(), &role, nil, nil, nil)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAdminListUserProjects(t *testing.T) {
	svc, userRepo, projectRepo, memberRepo := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)
	proj := projectRepo.addProject("TEST", "Test Project")

	memberRepo.members[adminMemberKey(proj.ID, user.ID)] = &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: proj.ID,
		UserID:    user.ID,
		Role:      model.ProjectRoleMember,
		CreatedAt: time.Now(),
	}

	projects, err := svc.ListUserProjects(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].ProjectKey != "TEST" {
		t.Fatalf("expected project key TEST, got %s", projects[0].ProjectKey)
	}
}

func TestAdminAddUserToProject_Success(t *testing.T) {
	svc, userRepo, projectRepo, memberRepo := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)
	proj := projectRepo.addProject("TEST", "Test Project")

	err := svc.AddUserToProject(ctx, user.ID, proj.ID, model.ProjectRoleMember)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify membership was created
	key := adminMemberKey(proj.ID, user.ID)
	if _, ok := memberRepo.members[key]; !ok {
		t.Fatal("expected membership to exist")
	}
}

func TestAdminAddUserToProject_Duplicate(t *testing.T) {
	svc, userRepo, projectRepo, _ := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)
	proj := projectRepo.addProject("TEST", "Test Project")

	if err := svc.AddUserToProject(ctx, user.ID, proj.ID, model.ProjectRoleMember); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err := svc.AddUserToProject(ctx, user.ID, proj.ID, model.ProjectRoleMember)
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAdminAddUserToProject_InvalidRole(t *testing.T) {
	svc, userRepo, projectRepo, _ := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)
	proj := projectRepo.addProject("TEST", "Test Project")

	err := svc.AddUserToProject(ctx, user.ID, proj.ID, "superadmin")
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestAdminAddUserToProject_UserNotFound(t *testing.T) {
	svc, _, projectRepo, _ := newTestAdminService()
	ctx := context.Background()

	proj := projectRepo.addProject("TEST", "Test Project")

	err := svc.AddUserToProject(ctx, uuid.New(), proj.ID, model.ProjectRoleMember)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAdminAddUserToProject_ProjectNotFound(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)

	err := svc.AddUserToProject(ctx, user.ID, uuid.New(), model.ProjectRoleMember)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAdminRemoveUserFromProject_Success(t *testing.T) {
	svc, userRepo, projectRepo, memberRepo := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)
	proj := projectRepo.addProject("TEST", "Test Project")

	// Add membership first
	if err := svc.AddUserToProject(ctx, user.ID, proj.ID, model.ProjectRoleMember); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err := svc.RemoveUserFromProject(ctx, user.ID, proj.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	key := adminMemberKey(proj.ID, user.ID)
	if _, ok := memberRepo.members[key]; ok {
		t.Fatal("expected membership to be removed")
	}
}

func TestAdminRemoveUserFromProject_NotFound(t *testing.T) {
	svc, userRepo, projectRepo, _ := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)
	proj := projectRepo.addProject("TEST", "Test Project")

	err := svc.RemoveUserFromProject(ctx, user.ID, proj.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- CreateUser tests ---

func TestCreateUser_Success(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	user, password, err := svc.CreateUser(ctx, "new@test.com", "New User")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "new@test.com" {
		t.Fatalf("expected email new@test.com, got %s", user.Email)
	}
	if user.DisplayName != "New User" {
		t.Fatalf("expected display name 'New User', got %s", user.DisplayName)
	}
	if user.GlobalRole != model.RoleUser {
		t.Fatalf("expected role user, got %s", user.GlobalRole)
	}
	if !user.IsActive {
		t.Fatal("expected user to be active")
	}
	if !user.ForcePasswordChange {
		t.Fatal("expected ForcePasswordChange to be true")
	}
	if len(password) != 12 {
		t.Fatalf("expected 12-char password, got %d", len(password))
	}
	// Verify user is stored
	if _, err := userRepo.GetByEmail(ctx, "new@test.com"); err != nil {
		t.Fatalf("expected user to be persisted, got %v", err)
	}
}

func TestCreateUser_EmailAlreadyExists(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	existing := userRepo.addUser(model.RoleUser, true)

	_, _, err := svc.CreateUser(ctx, existing.Email, "Another User")
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestCreateUser_EmptyEmail(t *testing.T) {
	svc, _, _, _ := newTestAdminService()
	ctx := context.Background()

	_, _, err := svc.CreateUser(ctx, "", "Some Name")
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateUser_EmptyDisplayName(t *testing.T) {
	svc, _, _, _ := newTestAdminService()
	ctx := context.Background()

	_, _, err := svc.CreateUser(ctx, "test@test.com", "")
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

// --- ResetUserPassword tests ---

func TestResetUserPassword_Success(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	user := userRepo.addUser(model.RoleUser, true)

	password, err := svc.ResetUserPassword(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(password) != 12 {
		t.Fatalf("expected 12-char password, got %d", len(password))
	}
	// Verify force_password_change is set
	updated, _ := userRepo.GetByID(ctx, user.ID)
	if !updated.ForcePasswordChange {
		t.Fatal("expected ForcePasswordChange to be true after reset")
	}
}

// --- MaxProjects tests ---

func TestAdminUpdateUser_SetMaxProjects(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	limit := 10
	updated, err := svc.UpdateUser(ctx, caller.ID, target.ID, nil, nil, &limit, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.MaxProjects == nil || *updated.MaxProjects != 10 {
		t.Fatalf("expected max_projects=10, got %v", updated.MaxProjects)
	}
}

func TestAdminUpdateUser_ClearMaxProjects(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	// First set it
	limit := 10
	_, err := svc.UpdateUser(ctx, caller.ID, target.ID, nil, nil, &limit, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Clear it with -1
	clear := -1
	updated, err := svc.UpdateUser(ctx, caller.ID, target.ID, nil, nil, &clear, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.MaxProjects != nil {
		t.Fatalf("expected max_projects=nil after clear, got %v", updated.MaxProjects)
	}
}

func TestAdminUpdateUser_SetMaxProjectsZero(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	limit := 0
	updated, err := svc.UpdateUser(ctx, caller.ID, target.ID, nil, nil, &limit, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.MaxProjects == nil || *updated.MaxProjects != 0 {
		t.Fatalf("expected max_projects=0 (unlimited), got %v", updated.MaxProjects)
	}
}

func TestAdminUpdateUser_InvalidMaxProjects(t *testing.T) {
	svc, userRepo, _, _ := newTestAdminService()
	ctx := context.Background()

	caller := userRepo.addUser(model.RoleAdmin, true)
	target := userRepo.addUser(model.RoleUser, true)

	invalid := -2
	_, err := svc.UpdateUser(ctx, caller.ID, target.ID, nil, nil, &invalid, nil)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for -2, got %v", err)
	}
}

func TestResetUserPassword_UserNotFound(t *testing.T) {
	svc, _, _, _ := newTestAdminService()
	ctx := context.Background()

	_, err := svc.ResetUserPassword(ctx, uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
