package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock team repository ---

type mockTeamRepo struct {
	teams   map[uuid.UUID]*model.Team
	members map[string]*model.TeamMember // key: teamID:userID
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{
		teams:   make(map[uuid.UUID]*model.Team),
		members: make(map[string]*model.TeamMember),
	}
}

func tmKey(teamID, userID uuid.UUID) string {
	return teamID.String() + ":" + userID.String()
}

func (m *mockTeamRepo) Create(_ context.Context, t *model.Team) error {
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	m.teams[t.ID] = t
	return nil
}

func (m *mockTeamRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t, nil
}

func (m *mockTeamRepo) List(_ context.Context, projectID uuid.UUID) ([]model.Team, error) {
	var result []model.Team
	for _, t := range m.teams {
		if t.ProjectID == projectID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTeamRepo) Update(_ context.Context, t *model.Team) error {
	if _, ok := m.teams[t.ID]; !ok {
		return model.ErrNotFound
	}
	t.UpdatedAt = time.Now()
	m.teams[t.ID] = t
	return nil
}

func (m *mockTeamRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.teams[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.teams, id)
	return nil
}

func (m *mockTeamRepo) AddMember(_ context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	key := tmKey(teamID, userID)
	if _, exists := m.members[key]; exists {
		return nil, model.ErrAlreadyExists
	}
	member := &model.TeamMember{
		ID:        uuid.New(),
		TeamID:    teamID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	m.members[key] = member
	return member, nil
}

func (m *mockTeamRepo) RemoveMember(_ context.Context, teamID, userID uuid.UUID) error {
	key := tmKey(teamID, userID)
	if _, ok := m.members[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.members, key)
	return nil
}

func (m *mockTeamRepo) ListMembers(_ context.Context, teamID uuid.UUID) ([]model.TeamMemberWithUser, error) {
	var result []model.TeamMemberWithUser
	for _, member := range m.members {
		if member.TeamID == teamID {
			result = append(result, model.TeamMemberWithUser{
				TeamMember:  *member,
				Email:       "user@example.com",
				DisplayName: "Test User",
			})
		}
	}
	return result, nil
}

// --- Helpers ---

func newTestTeamService() (*TeamService, *mockTeamRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	teamRepo := newMockTeamRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewTeamService(teamRepo, projectRepo, memberRepo)
	return svc, teamRepo, projectRepo, memberRepo
}

func setupTeamProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
	t.Helper()
	project := &model.Project{
		ID:   uuid.New(),
		Name: "Test Project",
		Key:  "TEST",
	}
	projectRepo.Create(context.Background(), project)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    info.UserID,
		Role:      role,
	})
	return project
}

func validCreateTeamInput() CreateTeamInput {
	return CreateTeamInput{
		Name: "Engineering",
	}
}

// --- Tests ---

func TestTeamCreate_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	team, err := svc.Create(context.Background(), info, "TEST", validCreateTeamInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if team.Name != "Engineering" {
		t.Fatalf("expected name 'Engineering', got %s", team.Name)
	}
}

func TestTeamCreate_EmptyName(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	input := validCreateTeamInput()
	input.Name = ""
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestTeamCreate_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateTeamInput())
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTeamGet_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Design",
	}
	teamRepo.Create(context.Background(), team)

	result, err := svc.Get(context.Background(), info, "TEST", team.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Design" {
		t.Fatalf("expected name 'Design', got %s", result.Name)
	}
}

func TestTeamGet_WrongProject(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: uuid.New(), // different project
		Name:      "Other Team",
	}
	teamRepo.Create(context.Background(), team)

	_, err := svc.Get(context.Background(), info, "TEST", team.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong project, got %v", err)
	}
}

func TestTeamList_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	teamRepo.Create(context.Background(), &model.Team{
		ID: uuid.New(), ProjectID: project.ID, Name: "Team A",
	})
	teamRepo.Create(context.Background(), &model.Team{
		ID: uuid.New(), ProjectID: project.ID, Name: "Team B",
	})

	teams, err := svc.List(context.Background(), info, "TEST")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}
}

func TestTeamUpdate_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Old Name",
	}
	teamRepo.Create(context.Background(), team)

	newName := "New Name"
	updated, err := svc.Update(context.Background(), info, "TEST", team.ID, UpdateTeamInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("expected name 'New Name', got %s", updated.Name)
	}
}

func TestTeamUpdate_ClearDescription(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	desc := "some description"
	team := &model.Team{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		Name:        "Team",
		Description: &desc,
	}
	teamRepo.Create(context.Background(), team)

	updated, err := svc.Update(context.Background(), info, "TEST", team.ID, UpdateTeamInput{
		ClearDescription: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Description != nil {
		t.Fatal("expected description to be cleared")
	}
}

func TestTeamDelete_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Delete Me",
	}
	teamRepo.Create(context.Background(), team)

	err := svc.Delete(context.Background(), info, "TEST", team.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = teamRepo.GetByID(context.Background(), team.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatal("expected team to be deleted")
	}
}

func TestTeamDelete_MemberForbidden(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Team",
	}
	teamRepo.Create(context.Background(), team)

	err := svc.Delete(context.Background(), info, "TEST", team.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTeamAddMember_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Team",
	}
	teamRepo.Create(context.Background(), team)

	targetUserID := uuid.New()
	// Target user must be a project member (non-customer) to be added to a team
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    targetUserID,
		Role:      model.ProjectRoleMember,
	})

	member, err := svc.AddMember(context.Background(), info, "TEST", team.ID, targetUserID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if member.TeamID != team.ID {
		t.Fatalf("expected team ID %s, got %s", team.ID, member.TeamID)
	}
	if member.UserID != targetUserID {
		t.Fatalf("expected user ID %s, got %s", targetUserID, member.UserID)
	}
}

func TestTeamRemoveMember_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Team",
	}
	teamRepo.Create(context.Background(), team)

	targetUserID := uuid.New()
	teamRepo.AddMember(context.Background(), team.ID, targetUserID)

	err := svc.RemoveMember(context.Background(), info, "TEST", team.ID, targetUserID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTeamListMembers_Success(t *testing.T) {
	svc, teamRepo, projectRepo, memberRepo := newTestTeamService()
	info := userAuthInfo()
	project := setupTeamProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Team",
	}
	teamRepo.Create(context.Background(), team)

	teamRepo.AddMember(context.Background(), team.ID, uuid.New())
	teamRepo.AddMember(context.Background(), team.ID, uuid.New())

	members, err := svc.ListMembers(context.Background(), info, "TEST", team.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}
