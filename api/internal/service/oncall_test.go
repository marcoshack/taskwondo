package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock oncall rotation repository ---

type mockOncallRotationRepo struct {
	rotations map[uuid.UUID]*model.OncallRotation
	byTeam    map[uuid.UUID]*model.OncallRotation
	members   map[uuid.UUID][]model.OncallRotationMember
	history   map[uuid.UUID][]model.OncallRotationHistory
}

func newMockOncallRotationRepo() *mockOncallRotationRepo {
	return &mockOncallRotationRepo{
		rotations: make(map[uuid.UUID]*model.OncallRotation),
		byTeam:    make(map[uuid.UUID]*model.OncallRotation),
		members:   make(map[uuid.UUID][]model.OncallRotationMember),
		history:   make(map[uuid.UUID][]model.OncallRotationHistory),
	}
}

func (m *mockOncallRotationRepo) Create(_ context.Context, rot *model.OncallRotation) (*model.OncallRotation, error) {
	now := time.Now()
	rot.CreatedAt = now
	rot.UpdatedAt = now
	m.rotations[rot.ID] = rot
	m.byTeam[rot.TeamID] = rot
	return rot, nil
}

func (m *mockOncallRotationRepo) GetByTeamID(_ context.Context, teamID uuid.UUID) (*model.OncallRotation, error) {
	rot, ok := m.byTeam[teamID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return rot, nil
}

func (m *mockOncallRotationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.OncallRotation, error) {
	rot, ok := m.rotations[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return rot, nil
}

func (m *mockOncallRotationRepo) Update(_ context.Context, rot *model.OncallRotation) (*model.OncallRotation, error) {
	if _, ok := m.rotations[rot.ID]; !ok {
		return nil, model.ErrNotFound
	}
	rot.UpdatedAt = time.Now()
	m.rotations[rot.ID] = rot
	m.byTeam[rot.TeamID] = rot
	return rot, nil
}

func (m *mockOncallRotationRepo) Delete(_ context.Context, teamID uuid.UUID) error {
	rot, ok := m.byTeam[teamID]
	if !ok {
		return model.ErrNotFound
	}
	delete(m.rotations, rot.ID)
	delete(m.byTeam, teamID)
	return nil
}

func (m *mockOncallRotationRepo) SetMembers(_ context.Context, rotationID uuid.UUID, members []model.OncallRotationMember) error {
	m.members[rotationID] = members
	return nil
}

func (m *mockOncallRotationRepo) ListMembers(_ context.Context, rotationID uuid.UUID) ([]model.OncallRotationMemberWithUser, error) {
	members := m.members[rotationID]
	result := make([]model.OncallRotationMemberWithUser, len(members))
	for i, member := range members {
		result[i] = model.OncallRotationMemberWithUser{
			OncallRotationMember: member,
			Email:                "user@example.com",
			DisplayName:          "Test User",
		}
	}
	return result, nil
}

func (m *mockOncallRotationRepo) CreateHistory(_ context.Context, h *model.OncallRotationHistory) error {
	m.history[h.RotationID] = append(m.history[h.RotationID], *h)
	return nil
}

func (m *mockOncallRotationRepo) EndCurrentHistory(_ context.Context, rotationID uuid.UUID) error {
	entries := m.history[rotationID]
	for i := range entries {
		if entries[i].EndedAt == nil {
			now := time.Now()
			entries[i].EndedAt = &now
		}
	}
	m.history[rotationID] = entries
	return nil
}

func (m *mockOncallRotationRepo) ListHistory(_ context.Context, rotationID uuid.UUID, limit, offset int) ([]model.OncallRotationHistoryWithUser, int, error) {
	entries := m.history[rotationID]
	total := len(entries)

	if offset >= total {
		return nil, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	result := make([]model.OncallRotationHistoryWithUser, 0, end-offset)
	for _, h := range entries[offset:end] {
		result = append(result, model.OncallRotationHistoryWithUser{
			OncallRotationHistory: h,
			DisplayName:           "Test User",
		})
	}
	return result, total, nil
}

func (m *mockOncallRotationRepo) ListDueRotations(_ context.Context) ([]model.OncallRotation, error) {
	var result []model.OncallRotation
	now := time.Now()
	for _, rot := range m.rotations {
		if rot.NextRotationAt != nil && !rot.NextRotationAt.After(now) {
			result = append(result, *rot)
		}
	}
	return result, nil
}

// --- Helpers ---

func newTestOncallService() (*OncallService, *mockOncallRotationRepo, *mockTeamRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	oncallRepo := newMockOncallRotationRepo()
	teamRepo := newMockTeamRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewOncallService(oncallRepo, teamRepo, projectRepo, memberRepo)
	return svc, oncallRepo, teamRepo, projectRepo, memberRepo
}

func setupOncallProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, teamRepo *mockTeamRepo, info *model.AuthInfo, role string) (*model.Project, *model.Team, []uuid.UUID) {
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

	team := &model.Team{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Engineering",
	}
	teamRepo.Create(context.Background(), team)

	// Add team members
	user1 := info.UserID
	user2 := uuid.New()
	user3 := uuid.New()
	teamRepo.AddMember(context.Background(), team.ID, user1)
	teamRepo.AddMember(context.Background(), team.ID, user2)
	teamRepo.AddMember(context.Background(), team.ID, user3)

	// Add user2 and user3 as project members too
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    user2,
		Role:      model.ProjectRoleMember,
	})
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    user3,
		Role:      model.ProjectRoleMember,
	})

	return project, team, []uuid.UUID{user1, user2, user3}
}

func validCreateOncallInput(memberIDs []uuid.UUID) CreateOncallRotationInput {
	return CreateOncallRotationInput{
		PeriodDays:   7,
		RotationTime: "12:00:00",
		Timezone:     "UTC",
		StartDate:    "2026-04-01",
		MemberIDs:    memberIDs,
	}
}

// --- Tests ---

func TestOncallCreate_Success(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	result, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.TeamID != team.ID {
		t.Fatalf("expected team_id %s, got %s", team.ID, result.TeamID)
	}
	if result.PeriodDays != 7 {
		t.Fatalf("expected period_days 7, got %d", result.PeriodDays)
	}
	if result.CurrentUserID == nil || *result.CurrentUserID != memberIDs[0] {
		t.Fatal("expected current_user_id to be first member")
	}
	if result.NextRotationAt == nil {
		t.Fatal("expected next_rotation_at to be set")
	}
	if len(result.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(result.Members))
	}
}

func TestOncallCreate_NoMembers(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, _ := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, CreateOncallRotationInput{
		PeriodDays:   7,
		RotationTime: "12:00:00",
		Timezone:     "UTC",
		StartDate:    "2026-04-01",
		MemberIDs:    nil,
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestOncallCreate_InvalidTimezone(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	input := validCreateOncallInput(memberIDs)
	input.Timezone = "Invalid/Timezone"
	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestOncallCreate_InvalidPeriodDays(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	input := validCreateOncallInput(memberIDs)
	input.PeriodDays = 0
	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestOncallCreate_MemberForbidden(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleMember)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestOncallCreate_NonTeamMember(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	// Include a user ID that's not a team member
	input := validCreateOncallInput(append(memberIDs, uuid.New()))
	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestOncallGet_Success(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	result, err := svc.GetRotation(context.Background(), info, "TEST", team.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.TeamID != team.ID {
		t.Fatalf("expected team_id %s, got %s", team.ID, result.TeamID)
	}
}

func TestOncallGet_NotFound(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, _ := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleMember)

	_, err := svc.GetRotation(context.Background(), info, "TEST", team.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestOncallUpdate_Success(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	newPeriod := 14
	result, err := svc.UpdateRotation(context.Background(), info, "TEST", team.ID, UpdateOncallRotationInput{
		PeriodDays: &newPeriod,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.PeriodDays != 14 {
		t.Fatalf("expected period_days 14, got %d", result.PeriodDays)
	}
}

func TestOncallUpdate_RemoveCurrentMember(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Remove the current on-call user (first member) from the rotation
	newMembers := memberIDs[1:]
	result, err := svc.UpdateRotation(context.Background(), info, "TEST", team.ID, UpdateOncallRotationInput{
		MemberIDs: newMembers,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Current user should have changed since the previous current was removed
	if result.CurrentUserID != nil && *result.CurrentUserID == *created.CurrentUserID {
		t.Fatal("expected current user to change after removal")
	}
	if result.CurrentUserID == nil || *result.CurrentUserID != newMembers[0] {
		t.Fatal("expected current user to be first of new members")
	}
}

func TestOncallDelete_Success(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	err = svc.DeleteRotation(context.Background(), info, "TEST", team.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetRotation(context.Background(), info, "TEST", team.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestOncallDelete_MemberForbidden(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, _ := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleMember)

	err := svc.DeleteRotation(context.Background(), info, "TEST", team.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestOncallListHistory_Success(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	history, total, err := svc.ListHistory(context.Background(), info, "TEST", team.ID, 20, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 history entry, got %d", total)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}
}

func TestOncallAdvance_Success(t *testing.T) {
	svc, oncallRepo, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	result, err := svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.OldUserID != memberIDs[0] {
		t.Fatalf("expected old user to be %s, got %s", memberIDs[0], result.OldUserID)
	}
	if result.NewUserID != memberIDs[1] {
		t.Fatalf("expected new user to be %s, got %s", memberIDs[1], result.NewUserID)
	}

	// Verify rotation state updated
	rot, _ := oncallRepo.GetByID(context.Background(), created.ID)
	if rot.CurrentPosition != 1 {
		t.Fatalf("expected position 1, got %d", rot.CurrentPosition)
	}
}

func TestOncallAdvance_WrapAround(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Advance through all members
	var lastResult *AdvanceResult
	for i := 0; i < len(memberIDs); i++ {
		lastResult, err = svc.AdvanceRotation(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("advance %d failed: %v", i, err)
		}
	}

	// After advancing len(members) times from position 0, we should wrap back to member[0]
	if lastResult.NewUserID != memberIDs[0] {
		t.Fatalf("expected wrap to first member %s, got %s", memberIDs[0], lastResult.NewUserID)
	}
}

func TestOncallAdvance_SingleMember(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	singleMember := memberIDs[:1]
	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(singleMember))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	result, err := svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// With a single member, advance should wrap back to the same user
	if result.NewUserID != singleMember[0] {
		t.Fatalf("expected same user %s after advance, got %s", singleMember[0], result.NewUserID)
	}
}

func TestOncallCreate_WrongProject(t *testing.T) {
	svc, _, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, _, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	// Create a team in a different project
	otherTeam := &model.Team{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Name:      "Other Team",
	}
	teamRepo.Create(context.Background(), otherTeam)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", otherTeam.ID, validCreateOncallInput(memberIDs))
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for team in wrong project, got %v", err)
	}
}

func TestOncallUpdate_StartDateResetsRotation(t *testing.T) {
	svc, oncallRepo, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Advance to second member
	_, err = svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("advance failed: %v", err)
	}

	rot, _ := oncallRepo.GetByID(context.Background(), created.ID)
	if rot.CurrentPosition != 1 {
		t.Fatalf("expected position 1 after advance, got %d", rot.CurrentPosition)
	}

	// Update with a new start_date — should reset to first member
	newStartDate := "2026-04-10"
	result, err := svc.UpdateRotation(context.Background(), info, "TEST", team.ID, UpdateOncallRotationInput{
		StartDate: &newStartDate,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.CurrentPosition != 0 {
		t.Fatalf("expected position reset to 0, got %d", result.CurrentPosition)
	}
	if result.CurrentUserID == nil || *result.CurrentUserID != memberIDs[0] {
		t.Fatal("expected current user to be reset to first member")
	}
	if result.StartDate != newStartDate {
		t.Fatalf("expected start_date %s, got %s", newStartDate, result.StartDate)
	}

	// History should have 3 entries: initial, advance, and reset
	history := oncallRepo.history[created.ID]
	if len(history) != 3 {
		t.Fatalf("expected 3 history entries, got %d", len(history))
	}
	// The second entry (from advance) should have been ended
	if history[1].EndedAt == nil {
		t.Fatal("expected second history entry to have ended_at set")
	}
}

func TestOncallUpdate_SameStartDateNoReset(t *testing.T) {
	svc, oncallRepo, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Advance to second member
	_, err = svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("advance failed: %v", err)
	}

	// Update with the same start_date — should NOT reset
	sameDate := "2026-04-01"
	result, err := svc.UpdateRotation(context.Background(), info, "TEST", team.ID, UpdateOncallRotationInput{
		StartDate: &sameDate,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.CurrentPosition != 1 {
		t.Fatalf("expected position to stay at 1, got %d", result.CurrentPosition)
	}

	// History should have only 2 entries (initial + advance), no extra reset
	history := oncallRepo.history[created.ID]
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
}
