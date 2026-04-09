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

	result, err := svc.GetRotation(context.Background(), info, "TEST", team.ID, nil, nil)
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

	_, err := svc.GetRotation(context.Background(), info, "TEST", team.ID, nil, nil)
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

	_, err = svc.GetRotation(context.Background(), info, "TEST", team.ID, nil, nil)
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

func TestOncallAdvance_MissedRotationsCatchUp(t *testing.T) {
	svc, oncallRepo, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	// Create rotation with 2-day period and 3 members
	input := CreateOncallRotationInput{
		PeriodDays:   2,
		RotationTime: "12:00:00",
		Timezone:     "UTC",
		StartDate:    "2026-04-01",
		MemberIDs:    memberIDs,
	}
	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, input)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Simulate the worker being down: set next_rotation_at to 5 days ago
	// (missed 2 full periods with a 2-day period = should advance 3 positions)
	rot := oncallRepo.rotations[created.ID]
	pastDue := time.Now().Add(-5 * 24 * time.Hour)
	rot.NextRotationAt = &pastDue
	oncallRepo.rotations[created.ID] = rot

	result, err := svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("advance failed: %v", err)
	}

	// 3 positions from 0 with 3 members: (0 + 3) % 3 = 0, wraps back to first member
	if result.NewUserID != memberIDs[0] {
		t.Fatalf("expected wrap to member[0] %s after 3-position skip, got %s", memberIDs[0], result.NewUserID)
	}

	// Verify next_rotation_at is aligned to the rotation schedule, not now + period
	updated := oncallRepo.rotations[created.ID]
	if updated.NextRotationAt == nil {
		t.Fatal("expected next_rotation_at to be set")
	}
	nextRot := *updated.NextRotationAt
	if !nextRot.After(time.Now()) {
		t.Fatalf("expected next_rotation_at in the future, got %v", nextRot)
	}
	// It should be at 12:00 UTC (the configured rotation_time)
	if nextRot.UTC().Hour() != 12 || nextRot.UTC().Minute() != 0 {
		t.Fatalf("expected next_rotation_at at 12:00 UTC, got %s", nextRot.UTC().Format(time.RFC3339))
	}
}

func TestOncallAdvance_NextRotationAlignedToSchedule(t *testing.T) {
	svc, oncallRepo, teamRepo, projectRepo, memberRepo := newTestOncallService()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("advance failed: %v", err)
	}

	// After advance, next_rotation_at should be at 12:00 UTC (not now + 7 days)
	updated := oncallRepo.rotations[created.ID]
	if updated.NextRotationAt == nil {
		t.Fatal("expected next_rotation_at to be set")
	}
	nextRot := updated.NextRotationAt.UTC()
	if nextRot.Hour() != 12 || nextRot.Minute() != 0 {
		t.Fatalf("expected next_rotation_at at 12:00 UTC, got %s", nextRot.Format(time.RFC3339))
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
	expectedDate, _ := parseDate(newStartDate)
	if !result.StartDate.Equal(expectedDate) {
		t.Fatalf("expected start_date %s, got %s", newStartDate, result.StartDate.Format("2006-01-02"))
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

// --- Mock oncall override repository ---

type mockOncallOverrideRepo struct {
	overrides        map[uuid.UUID]*model.OncallOverride
	staleTransitions []model.OverrideTransition
}

func newMockOncallOverrideRepo() *mockOncallOverrideRepo {
	return &mockOncallOverrideRepo{
		overrides: make(map[uuid.UUID]*model.OncallOverride),
	}
}

func (m *mockOncallOverrideRepo) Create(_ context.Context, o *model.OncallOverride) (*model.OncallOverride, error) {
	o.CreatedAt = time.Now()
	m.overrides[o.ID] = o
	return o, nil
}

func (m *mockOncallOverrideRepo) GetByID(_ context.Context, id uuid.UUID) (*model.OncallOverride, error) {
	o, ok := m.overrides[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return o, nil
}

func (m *mockOncallOverrideRepo) Update(_ context.Context, o *model.OncallOverride) (*model.OncallOverride, error) {
	if _, ok := m.overrides[o.ID]; !ok {
		return nil, model.ErrNotFound
	}
	m.overrides[o.ID] = o
	return o, nil
}

func (m *mockOncallOverrideRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.overrides[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.overrides, id)
	return nil
}

func (m *mockOncallOverrideRepo) ListByRotation(_ context.Context, rotationID uuid.UUID) ([]model.OncallOverrideWithUser, error) {
	now := time.Now()
	var result []model.OncallOverrideWithUser
	for _, o := range m.overrides {
		if o.RotationID == rotationID && o.EndAt.After(now) {
			result = append(result, model.OncallOverrideWithUser{
				OncallOverride:   *o,
				OverrideUserName: "Override User",
				CreatedByName:    "Creator",
			})
		}
	}
	return result, nil
}

func (m *mockOncallOverrideRepo) GetActiveOverride(_ context.Context, rotationID uuid.UUID) (*model.OncallOverride, error) {
	now := time.Now()
	var latest *model.OncallOverride
	for _, o := range m.overrides {
		if o.RotationID == rotationID && !o.StartAt.After(now) && o.EndAt.After(now) {
			if latest == nil || o.CreatedAt.After(latest.CreatedAt) {
				oCopy := *o
				latest = &oCopy
			}
		}
	}
	return latest, nil
}

func (m *mockOncallOverrideRepo) ListStaleOverrideRotations(_ context.Context) ([]model.OverrideTransition, error) {
	return m.staleTransitions, nil
}

func (m *mockOncallOverrideRepo) ListOverridesInRange(_ context.Context, rotationID uuid.UUID, from, to time.Time) ([]model.OncallOverrideWithUser, error) {
	var result []model.OncallOverrideWithUser
	for _, o := range m.overrides {
		if o.RotationID == rotationID && o.StartAt.Before(to) && o.EndAt.After(from) {
			result = append(result, model.OncallOverrideWithUser{
				OncallOverride:   *o,
				OverrideUserName: "Override User",
				CreatedByName:    "Creator",
			})
		}
	}
	return result, nil
}

func newTestOncallServiceWithOverrides() (*OncallService, *mockOncallRotationRepo, *mockOncallOverrideRepo, *mockTeamRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	oncallRepo := newMockOncallRotationRepo()
	overrideRepo := newMockOncallOverrideRepo()
	teamRepo := newMockTeamRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewOncallService(oncallRepo, teamRepo, projectRepo, memberRepo)
	svc.SetOverrideRepository(overrideRepo)
	return svc, oncallRepo, overrideRepo, teamRepo, projectRepo, memberRepo
}

func TestOncallOverrideCreate_Success(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	overrideInput := CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(1 * time.Hour),
		EndAt:          time.Now().Add(25 * time.Hour),
	}

	result, err := svc.CreateOverride(context.Background(), info, "TEST", team.ID, overrideInput)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.OverrideUserID != memberIDs[1] {
		t.Fatalf("expected override_user_id %s, got %s", memberIDs[1], result.OverrideUserID)
	}
}

func TestOncallOverrideCreate_MemberCanCreateForSelf(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleMember)

	// Create rotation as admin first
	adminInfo := &model.AuthInfo{UserID: uuid.New(), GlobalRole: model.RoleAdmin}
	_, err := svc.CreateRotation(context.Background(), adminInfo, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Member can create override for themselves
	overrideInput := CreateOncallOverrideInput{
		OverrideUserID: info.UserID,
		StartAt:        time.Now().Add(1 * time.Hour),
		EndAt:          time.Now().Add(25 * time.Hour),
	}

	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, overrideInput)
	if err != nil {
		t.Fatalf("expected no error for self-override, got %v", err)
	}
}

func TestOncallOverrideCreate_MemberCannotCreateForOther(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleMember)

	adminInfo := &model.AuthInfo{UserID: uuid.New(), GlobalRole: model.RoleAdmin}
	_, err := svc.CreateRotation(context.Background(), adminInfo, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Member cannot create override for someone else
	overrideInput := CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(1 * time.Hour),
		EndAt:          time.Now().Add(25 * time.Hour),
	}

	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, overrideInput)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestOncallOverrideCreate_InvalidTimeRange(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// start_at after end_at
	overrideInput := CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(25 * time.Hour),
		EndAt:          time.Now().Add(1 * time.Hour),
	}

	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, overrideInput)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestOncallOverrideCreate_PastEndTime(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	overrideInput := CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(-48 * time.Hour),
		EndAt:          time.Now().Add(-24 * time.Hour),
	}

	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, overrideInput)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for past end_at, got %v", err)
	}
}

func TestOncallOverrideList_Success(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create two overrides
	for i := 0; i < 2; i++ {
		_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
			OverrideUserID: memberIDs[1],
			StartAt:        time.Now().Add(time.Duration(i+1) * 24 * time.Hour),
			EndAt:          time.Now().Add(time.Duration(i+2) * 24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("create override %d failed: %v", i, err)
		}
	}

	list, err := svc.ListOverrides(context.Background(), info, "TEST", team.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(list))
	}
}

func TestOncallOverrideDelete_CreatorCanDelete(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	override, err := svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(1 * time.Hour),
		EndAt:          time.Now().Add(25 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	err = svc.DeleteOverride(context.Background(), info, "TEST", team.ID, override.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestOncallOverrideDelete_NonCreatorMemberForbidden(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	adminInfo := &model.AuthInfo{UserID: uuid.New(), GlobalRole: model.RoleAdmin}
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleMember)

	_, err := svc.CreateRotation(context.Background(), adminInfo, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Admin creates override
	override, err := svc.CreateOverride(context.Background(), adminInfo, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(1 * time.Hour),
		EndAt:          time.Now().Add(25 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	// Regular member (non-creator) cannot delete
	err = svc.DeleteOverride(context.Background(), info, "TEST", team.ID, override.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestOncallOverrideActiveOverride(t *testing.T) {
	svc, _, overrideRepo, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create an active override (starts in the past, ends in the future)
	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[2],
		StartAt:        time.Now().Add(-1 * time.Hour),
		EndAt:          time.Now().Add(23 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	// Verify active override via the repository directly
	rot, _ := svc.oncall.GetByTeamID(context.Background(), team.ID)
	active, err := overrideRepo.GetActiveOverride(context.Background(), rot.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if active == nil {
		t.Fatal("expected active override, got nil")
	}
	if active.OverrideUserID != memberIDs[2] {
		t.Fatalf("expected override_user_id %s, got %s", memberIDs[2], active.OverrideUserID)
	}
}

func TestOncallOverrideGetRotation_IncludesActiveOverride(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create an active override
	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[2],
		StartAt:        time.Now().Add(-1 * time.Hour),
		EndAt:          time.Now().Add(23 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	result, err := svc.GetRotation(context.Background(), info, "TEST", team.ID, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.IsOverride {
		t.Fatal("expected IsOverride to be true when active override exists")
	}
	if len(result.Overrides) == 0 {
		t.Fatal("expected overrides in rotation result, got none")
	}
	foundOverrideUser := false
	for _, o := range result.Overrides {
		if o.OverrideUserID == memberIDs[2] {
			foundOverrideUser = true
			break
		}
	}
	if !foundOverrideUser {
		t.Fatalf("expected override_user_id %s in overrides list", memberIDs[2])
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

// --- rotationEpoch tests ---

func TestRotationEpoch_Basic(t *testing.T) {
	sd := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)
	rt := time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)
	result := rotationEpoch(sd, rt, "UTC")
	expected := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestRotationEpoch_Timezone(t *testing.T) {
	sd := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	rt := time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)
	result := rotationEpoch(sd, rt, "America/New_York")
	// April 6 is during EDT (UTC-4)
	expectedUTC := time.Date(2026, 4, 6, 16, 0, 0, 0, time.UTC)
	if !result.UTC().Equal(expectedUTC) {
		t.Errorf("expected UTC %v, got UTC %v", expectedUTC, result.UTC())
	}
}

// --- Schedule projection tests ---

func TestComputeRotationShifts_Basic(t *testing.T) {
	rot := &model.OncallRotation{
		StartDate:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		RotationTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC),
		Timezone:     "UTC",
		PeriodDays:   7,
	}
	userA := uuid.New()
	userB := uuid.New()
	members := []model.OncallRotationMemberWithUser{
		{OncallRotationMember: model.OncallRotationMember{UserID: userA, Position: 0}, DisplayName: "Alice"},
		{OncallRotationMember: model.OncallRotationMember{UserID: userB, Position: 1}, DisplayName: "Bob"},
	}

	// Request schedule for April 1-30
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	shifts := computeRotationShifts(rot, members, rangeStart, rangeEnd)

	// T0 = 2026-04-01 12:00 UTC
	// Shift boundaries: Apr 1 12:00, Apr 8 12:00, Apr 15 12:00, Apr 22 12:00, Apr 29 12:00
	// Clipped to range: [Apr 1 00:00, May 1 00:00]
	// Expected shifts:
	//   Alice: Apr 1 00:00 → Apr 1 12:00 (clipped from previous period)
	//   Wait — actually Alice is member[0], period 0 starts at T0. Before T0 is period -1 = member[1] (Bob).
	//   Let me re-check.

	// The first shift boundary at or before rangeStart (Apr 1 00:00):
	// T0 = Apr 1 12:00. rangeStart = Apr 1 00:00 is before T0.
	// periodsElapsed = floor((00:00 - 12:00) / 7d) = floor(-12h / 168h) = -1
	// shiftStart = T0 + (-1 * 7d) = Mar 25 12:00
	// memberIdx = -1 % 2 = -1 + 2 = 1 → Bob
	// Clipped: Apr 1 00:00 → Apr 1 12:00 (Bob)
	// Then shift at T0: memberIdx = 0 → Alice: Apr 1 12:00 → Apr 8 12:00
	// etc.

	if len(shifts) < 4 {
		t.Fatalf("expected at least 4 shifts, got %d", len(shifts))
	}

	// First shift should be Bob (from before T0, clipped to range start)
	if shifts[0].UserID != userB {
		t.Errorf("shift 0: expected Bob (%s), got %s", userB, shifts[0].UserID)
	}
	if !shifts[0].StartAt.Equal(rangeStart) {
		t.Errorf("shift 0 start: expected %v, got %v", rangeStart, shifts[0].StartAt)
	}
	expectedT0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if !shifts[0].EndAt.Equal(expectedT0) {
		t.Errorf("shift 0 end: expected %v, got %v", expectedT0, shifts[0].EndAt)
	}

	// Second shift should be Alice starting at T0
	if shifts[1].UserID != userA {
		t.Errorf("shift 1: expected Alice (%s), got %s", userA, shifts[1].UserID)
	}
	if !shifts[1].StartAt.Equal(expectedT0) {
		t.Errorf("shift 1 start: expected %v, got %v", expectedT0, shifts[1].StartAt)
	}
}

func TestComputeRotationShifts_FutureStartDate(t *testing.T) {
	rot := &model.OncallRotation{
		StartDate:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		RotationTime: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC),
		Timezone:     "UTC",
		PeriodDays:   3,
	}
	userA := uuid.New()
	userB := uuid.New()
	members := []model.OncallRotationMemberWithUser{
		{OncallRotationMember: model.OncallRotationMember{UserID: userA, Position: 0}, DisplayName: "Alice"},
		{OncallRotationMember: model.OncallRotationMember{UserID: userB, Position: 1}, DisplayName: "Bob"},
	}

	// Range fully after start date
	rangeStart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)

	shifts := computeRotationShifts(rot, members, rangeStart, rangeEnd)

	// T0 = May 1 00:00 UTC. Period = 3 days, 2 members.
	// May 1 → May 4: Alice (period 0, member 0)
	// May 4 → May 7: Bob (period 1, member 1)
	if len(shifts) != 2 {
		t.Fatalf("expected 2 shifts, got %d", len(shifts))
	}
	if shifts[0].UserID != userA {
		t.Errorf("shift 0: expected Alice, got %s", shifts[0].UserID)
	}
	if shifts[1].UserID != userB {
		t.Errorf("shift 1: expected Bob, got %s", shifts[1].UserID)
	}
}

func TestComputeRotationShifts_Timezone(t *testing.T) {
	rot := &model.OncallRotation{
		StartDate:    time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
		RotationTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC),
		Timezone:     "America/New_York",
		PeriodDays:   7,
	}
	userA := uuid.New()
	userB := uuid.New()
	members := []model.OncallRotationMemberWithUser{
		{OncallRotationMember: model.OncallRotationMember{UserID: userA, Position: 0}, DisplayName: "Hack"},
		{OncallRotationMember: model.OncallRotationMember{UserID: userB, Position: 1}, DisplayName: "Hackzm"},
	}

	// This is the prod scenario: start_date=Apr 6, period=7d, rotation_time=12:00 EDT
	// T0 = Apr 6 12:00 EDT = Apr 6 16:00 UTC
	// Hack (member 0) is on-call from T0 to T0+7d = Apr 13 16:00 UTC
	rangeStart := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)

	shifts := computeRotationShifts(rot, members, rangeStart, rangeEnd)

	// Before T0 (Apr 6 00:00 → Apr 6 16:00): should be Hackzm (prev period)
	if shifts[0].UserID != userB {
		t.Errorf("shift 0: expected Hackzm (prev period), got %s", shifts[0].UserID)
	}
	t0UTC := time.Date(2026, 4, 6, 16, 0, 0, 0, time.UTC)
	if !shifts[0].EndAt.Equal(t0UTC) {
		t.Errorf("shift 0 end: expected %v, got %v", t0UTC, shifts[0].EndAt)
	}

	// From T0: Hack is on-call
	if shifts[1].UserID != userA {
		t.Errorf("shift 1: expected Hack, got %s", shifts[1].UserID)
	}
	nextRotation := time.Date(2026, 4, 13, 16, 0, 0, 0, time.UTC)
	if !shifts[1].EndAt.Equal(nextRotation) {
		t.Errorf("shift 1 end: expected %v, got %v", nextRotation, shifts[1].EndAt)
	}

	// From Apr 13 16:00: Hackzm is on-call (clipped to range end)
	if shifts[2].UserID != userB {
		t.Errorf("shift 2: expected Hackzm, got %s", shifts[2].UserID)
	}
}

func TestApplyOverrides_SplitsShift(t *testing.T) {
	userA := uuid.New()
	userC := uuid.New()

	shifts := []ScheduleShift{
		{
			UserID:  userA,
			StartAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			EndAt:   time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		},
	}

	overrides := []model.OncallOverrideWithUser{
		{
			OncallOverride: model.OncallOverride{
				ID:             uuid.New(),
				OverrideUserID: userC,
				StartAt:        time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
				EndAt:          time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
				CreatedAt:      time.Now(),
			},
			OverrideUserName: "Charlie",
		},
	}

	result := applyOverrides(shifts, overrides)

	// Should produce: Alice(Apr 1-3), Charlie(Apr 3-5), Alice(Apr 5-8)
	if len(result) != 3 {
		t.Fatalf("expected 3 shifts, got %d", len(result))
	}
	if result[0].UserID != userA || result[0].IsOverride {
		t.Errorf("shift 0: expected Alice (not override)")
	}
	if !result[0].EndAt.Equal(time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("shift 0 end: expected Apr 3, got %v", result[0].EndAt)
	}
	if result[1].UserID != userC || !result[1].IsOverride {
		t.Errorf("shift 1: expected Charlie (override)")
	}
	if result[2].UserID != userA || result[2].IsOverride {
		t.Errorf("shift 2: expected Alice (not override)")
	}
}

func TestApplyOverrides_MergesConsecutiveSameUser(t *testing.T) {
	userA := uuid.New()

	shifts := []ScheduleShift{
		{
			UserID:  userA,
			StartAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			EndAt:   time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:  userA,
			StartAt: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
			EndAt:   time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		},
	}

	// Override replaces same user — should still merge back
	overrides := []model.OncallOverrideWithUser{
		{
			OncallOverride: model.OncallOverride{
				ID:             uuid.New(),
				OverrideUserID: userA,
				StartAt:        time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
				EndAt:          time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
				CreatedAt:      time.Now(),
			},
			OverrideUserName: "Alice",
		},
	}

	result := applyOverrides(shifts, overrides)

	// After override: Alice(Apr 1-2 not-override), Alice(Apr 2-3 override), Alice(Apr 3-4 not-override), Alice(Apr 4-8 not-override)
	// Consecutive non-override Alice shifts merge, but override Alice doesn't merge with non-override Alice
	// So: Alice(Apr 1-2), Alice-override(Apr 2-3), Alice(Apr 3-8)
	if len(result) != 3 {
		t.Fatalf("expected 3 shifts (non-override, override, non-override), got %d", len(result))
	}
	if result[1].IsOverride != true {
		t.Error("shift 1 should be an override")
	}
}

func TestGetSchedule_Integration(t *testing.T) {
	svc, oncallRepo, overrideRepo, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	// Create rotation
	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Get schedule for April
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	result, err := svc.GetRotation(context.Background(), info, "TEST", team.ID, &rangeStart, &rangeEnd)
	if err != nil {
		t.Fatalf("get schedule failed: %v", err)
	}

	if len(result.Shifts) == 0 {
		t.Fatal("expected shifts, got none")
	}
	if len(result.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(result.Members))
	}
	if result.PeriodDays != 7 {
		t.Fatalf("expected period_days 7, got %d", result.PeriodDays)
	}

	// Add an override and verify it appears in schedule
	rot := oncallRepo.byTeam[team.ID]
	override := &model.OncallOverride{
		ID:             uuid.Must(uuid.NewV7()),
		RotationID:     rot.ID,
		OverrideUserID: memberIDs[2],
		StartAt:        time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		EndAt:          time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
		CreatedBy:      info.UserID,
		CreatedAt:      time.Now(),
	}
	overrideRepo.overrides[override.ID] = override

	result, err = svc.GetRotation(context.Background(), info, "TEST", team.ID, &rangeStart, &rangeEnd)
	if err != nil {
		t.Fatalf("get schedule with override failed: %v", err)
	}

	// Check that at least one shift is an override
	hasOverride := false
	for _, s := range result.Shifts {
		if s.IsOverride {
			hasOverride = true
			break
		}
	}
	if !hasOverride {
		t.Error("expected at least one override shift in schedule")
	}

	// Check overrides list is populated
	if len(result.Overrides) != 1 {
		t.Fatalf("expected 1 override in result, got %d", len(result.Overrides))
	}
}

// --- Override transition tests ---

func TestCreateOverride_ImmediateActive(t *testing.T) {
	svc, oncallRepo, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create an override that is immediately active (start_at in the past)
	overrideInput := CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(-1 * time.Minute),
		EndAt:          time.Now().Add(24 * time.Hour),
	}

	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, overrideInput)
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	// Verify the rotation was updated to reflect the active override
	rot := oncallRepo.byTeam[team.ID]
	if !rot.IsOverride {
		t.Fatal("expected IsOverride to be true after creating immediately active override")
	}
	if rot.CurrentUserID == nil || *rot.CurrentUserID != memberIDs[1] {
		t.Fatalf("expected CurrentUserID to be override user %s, got %v", memberIDs[1], rot.CurrentUserID)
	}
}

func TestDeleteOverride_RestoresScheduledUser(t *testing.T) {
	svc, oncallRepo, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create an active override
	override, err := svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[2],
		StartAt:        time.Now().Add(-1 * time.Minute),
		EndAt:          time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	// Confirm override is active
	rot := oncallRepo.byTeam[team.ID]
	if !rot.IsOverride {
		t.Fatal("expected IsOverride to be true after creating active override")
	}

	// Delete the override
	err = svc.DeleteOverride(context.Background(), info, "TEST", team.ID, override.ID)
	if err != nil {
		t.Fatalf("delete override failed: %v", err)
	}

	// Verify the rotation was restored to scheduled user
	rot = oncallRepo.byTeam[team.ID]
	if rot.IsOverride {
		t.Fatal("expected IsOverride to be false after deleting the only active override")
	}

	// The scheduled user should be based on the rotation position.
	// computeScheduledUserNow determines the correct user based on time elapsed.
	members, _ := oncallRepo.ListMembers(context.Background(), rot.ID)
	expectedUser := computeScheduledUserNow(rot, members)
	if rot.CurrentUserID == nil || *rot.CurrentUserID != expectedUser {
		t.Fatalf("expected CurrentUserID to be scheduled user %s, got %v", expectedUser, rot.CurrentUserID)
	}
}

func TestDeleteOverride_FallsBackToAnotherOverride(t *testing.T) {
	svc, oncallRepo, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create two overlapping active overrides for different users
	override1, err := svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[1],
		StartAt:        time.Now().Add(-2 * time.Minute),
		EndAt:          time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override 1 failed: %v", err)
	}

	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[2],
		StartAt:        time.Now().Add(-1 * time.Minute),
		EndAt:          time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override 2 failed: %v", err)
	}

	// Delete the first override
	err = svc.DeleteOverride(context.Background(), info, "TEST", team.ID, override1.ID)
	if err != nil {
		t.Fatalf("delete override 1 failed: %v", err)
	}

	// The second override should still be active
	rot := oncallRepo.byTeam[team.ID]
	if !rot.IsOverride {
		t.Fatal("expected IsOverride to remain true when another active override exists")
	}
	if rot.CurrentUserID == nil || *rot.CurrentUserID != memberIDs[2] {
		t.Fatalf("expected CurrentUserID to be second override user %s, got %v", memberIDs[2], rot.CurrentUserID)
	}
}

func TestAdvanceRotation_DuringActiveOverride(t *testing.T) {
	svc, oncallRepo, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Create an active override
	_, err = svc.CreateOverride(context.Background(), info, "TEST", team.ID, CreateOncallOverrideInput{
		OverrideUserID: memberIDs[2],
		StartAt:        time.Now().Add(-1 * time.Minute),
		EndAt:          time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create override failed: %v", err)
	}

	// Advance the rotation
	result, err := svc.AdvanceRotation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("advance rotation failed: %v", err)
	}

	// Position should have advanced
	rot := oncallRepo.rotations[created.ID]
	if rot.CurrentPosition != 1 {
		t.Fatalf("expected CurrentPosition to advance to 1, got %d", rot.CurrentPosition)
	}

	// But CurrentUserID should still be the override user, not the scheduled user
	if rot.CurrentUserID == nil || *rot.CurrentUserID != memberIDs[2] {
		t.Fatalf("expected CurrentUserID to be override user %s, got %v", memberIDs[2], rot.CurrentUserID)
	}
	if !rot.IsOverride {
		t.Fatal("expected IsOverride to be true during active override")
	}

	// The advance result returns the scheduled new user (position-based), not the override user
	if result.NewUserID != memberIDs[1] {
		t.Fatalf("expected advance result new user to be %s (position 1), got %s", memberIDs[1], result.NewUserID)
	}
}

func TestReconcileOverrides_OverrideStarted(t *testing.T) {
	svc, oncallRepo, overrideRepo, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Directly insert an active override in the mock (bypassing the service to simulate
	// a future override that has since become active without synchronous update)
	overrideUserID := memberIDs[1]
	override := &model.OncallOverride{
		ID:             uuid.Must(uuid.NewV7()),
		RotationID:     created.ID,
		OverrideUserID: overrideUserID,
		StartAt:        time.Now().Add(-10 * time.Minute),
		EndAt:          time.Now().Add(24 * time.Hour),
		CreatedBy:      info.UserID,
		CreatedAt:      time.Now(),
	}
	overrideRepo.overrides[override.ID] = override

	// Rotation still has IsOverride=false (stale state)
	rot := oncallRepo.rotations[created.ID]
	if rot.IsOverride {
		t.Fatal("expected IsOverride to be false before reconciliation")
	}

	// Set up stale transitions to report a "started" transition
	overrideRepo.staleTransitions = []model.OverrideTransition{
		{
			RotationID:     created.ID,
			OverrideUserID: &overrideUserID,
			Type:           "started",
		},
	}

	// Reconcile
	err = svc.ReconcileOverrides(context.Background())
	if err != nil {
		t.Fatalf("reconcile overrides failed: %v", err)
	}

	// Verify rotation was updated
	rot = oncallRepo.rotations[created.ID]
	if !rot.IsOverride {
		t.Fatal("expected IsOverride to be true after reconciling started override")
	}
	if rot.CurrentUserID == nil || *rot.CurrentUserID != overrideUserID {
		t.Fatalf("expected CurrentUserID to be override user %s, got %v", overrideUserID, rot.CurrentUserID)
	}
}

func TestReconcileOverrides_OverrideEnded(t *testing.T) {
	svc, oncallRepo, overrideRepo, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Manually set rotation to override state (simulating a previously active override)
	rot := oncallRepo.rotations[created.ID]
	rot.IsOverride = true
	overrideUser := memberIDs[2]
	rot.CurrentUserID = &overrideUser
	oncallRepo.rotations[created.ID] = rot
	oncallRepo.byTeam[team.ID] = rot

	// Set up stale transitions to report an "ended" transition
	overrideRepo.staleTransitions = []model.OverrideTransition{
		{
			RotationID:     created.ID,
			OverrideUserID: nil,
			Type:           "ended",
		},
	}

	// Reconcile
	err = svc.ReconcileOverrides(context.Background())
	if err != nil {
		t.Fatalf("reconcile overrides failed: %v", err)
	}

	// Verify rotation was restored
	rot = oncallRepo.rotations[created.ID]
	if rot.IsOverride {
		t.Fatal("expected IsOverride to be false after reconciling ended override")
	}

	// CurrentUserID should be the scheduled user
	members, _ := oncallRepo.ListMembers(context.Background(), created.ID)
	expectedUser := computeScheduledUserNow(rot, members)
	if rot.CurrentUserID == nil || *rot.CurrentUserID != expectedUser {
		t.Fatalf("expected CurrentUserID to be scheduled user %s, got %v", expectedUser, rot.CurrentUserID)
	}
}

func TestReconcileOverrides_NoMismatch(t *testing.T) {
	svc, oncallRepo, overrideRepo, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	created, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// No stale transitions
	overrideRepo.staleTransitions = nil

	// Record current state
	rotBefore := *oncallRepo.rotations[created.ID]

	err = svc.ReconcileOverrides(context.Background())
	if err != nil {
		t.Fatalf("reconcile overrides failed: %v", err)
	}

	// Rotation should be unchanged
	rotAfter := oncallRepo.rotations[created.ID]
	if rotAfter.IsOverride != rotBefore.IsOverride {
		t.Fatalf("expected IsOverride to remain %v, got %v", rotBefore.IsOverride, rotAfter.IsOverride)
	}
	if rotAfter.UpdatedAt != rotBefore.UpdatedAt {
		t.Fatal("expected rotation to not be updated when there are no stale transitions")
	}
}

func TestComputeScheduledUserNow(t *testing.T) {
	// Create a rotation with start_date ~21 days in the past, 7-day period, 3 members
	// After 21 days with 7-day period: 3 periods elapsed, 3 % 3 = 0 -> member[0]
	startDate := time.Now().Add(-21 * 24 * time.Hour)
	rot := &model.OncallRotation{
		ID:           uuid.New(),
		StartDate:    startDate,
		RotationTime: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), // midnight
		Timezone:     "UTC",
		PeriodDays:   7,
	}

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	members := []model.OncallRotationMemberWithUser{
		{OncallRotationMember: model.OncallRotationMember{UserID: user1, Position: 0}},
		{OncallRotationMember: model.OncallRotationMember{UserID: user2, Position: 1}},
		{OncallRotationMember: model.OncallRotationMember{UserID: user3, Position: 2}},
	}

	result := computeScheduledUserNow(rot, members)

	// Compute expected: t0 = startDate at midnight UTC
	t0 := rotationEpoch(rot.StartDate, rot.RotationTime, rot.Timezone)
	period := time.Duration(rot.PeriodDays) * 24 * time.Hour
	elapsed := time.Since(t0)
	periodsElapsed := int(elapsed / period)
	expectedIdx := periodsElapsed % len(members)
	if expectedIdx < 0 {
		expectedIdx += len(members)
	}
	expectedUser := members[expectedIdx].UserID

	if result != expectedUser {
		t.Fatalf("expected scheduled user %s (member[%d]), got %s", expectedUser, expectedIdx, result)
	}
}

func TestGetRotation_WithDateRange(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Request rotation with a date range
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	result, err := svc.GetRotation(context.Background(), info, "TEST", team.ID, &rangeStart, &rangeEnd)
	if err != nil {
		t.Fatalf("get rotation failed: %v", err)
	}

	if len(result.Shifts) == 0 {
		t.Fatal("expected Shifts to be non-empty when date range is provided")
	}

	// Verify shifts cover the range
	for _, shift := range result.Shifts {
		if shift.StartAt.After(rangeEnd) || shift.EndAt.Before(rangeStart) {
			t.Fatalf("shift %v-%v is outside requested range %v-%v", shift.StartAt, shift.EndAt, rangeStart, rangeEnd)
		}
	}
}

func TestGetRotation_WithoutDateRange(t *testing.T) {
	svc, _, _, teamRepo, projectRepo, memberRepo := newTestOncallServiceWithOverrides()
	info := userAuthInfo()
	_, team, memberIDs := setupOncallProject(t, projectRepo, memberRepo, teamRepo, info, model.ProjectRoleAdmin)

	_, err := svc.CreateRotation(context.Background(), info, "TEST", team.ID, validCreateOncallInput(memberIDs))
	if err != nil {
		t.Fatalf("create rotation failed: %v", err)
	}

	// Request rotation without a date range
	result, err := svc.GetRotation(context.Background(), info, "TEST", team.ID, nil, nil)
	if err != nil {
		t.Fatalf("get rotation failed: %v", err)
	}

	if len(result.Shifts) != 0 {
		t.Fatalf("expected Shifts to be empty when no date range is provided, got %d shifts", len(result.Shifts))
	}
}
