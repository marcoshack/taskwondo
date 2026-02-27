package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock ---

type mockStatsRepo struct {
	snapshots []model.ProjectStatsSnapshot
}

func (m *mockStatsRepo) Timeline(_ context.Context, projectID uuid.UUID, since time.Time) ([]model.ProjectStatsSnapshot, error) {
	var result []model.ProjectStatsSnapshot
	for _, s := range m.snapshots {
		if s.ProjectID == projectID && !s.CapturedAt.Before(since) {
			result = append(result, s)
		}
	}
	return result, nil
}

// --- Helpers ---

func newTestStatsService() (*StatsService, *mockStatsRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	statsRepo := &mockStatsRepo{}
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewStatsService(statsRepo, projectRepo, memberRepo)
	return svc, statsRepo, projectRepo, memberRepo
}

func setupStatsProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
	t.Helper()
	project := &model.Project{
		ID:   uuid.New(),
		Name: "Test Project",
		Key:  "STAT",
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

// --- Tests ---

func TestStatsTimeline_Success(t *testing.T) {
	svc, statsRepo, projectRepo, memberRepo := newTestStatsService()
	info := userAuthInfo()
	project := setupStatsProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	now := time.Now().Truncate(time.Hour)
	statsRepo.snapshots = []model.ProjectStatsSnapshot{
		{ID: uuid.New(), ProjectID: project.ID, TodoCount: 5, InProgressCount: 3, DoneCount: 2, CancelledCount: 1, CapturedAt: now.Add(-2 * time.Hour)},
		{ID: uuid.New(), ProjectID: project.ID, TodoCount: 6, InProgressCount: 4, DoneCount: 3, CancelledCount: 1, CapturedAt: now.Add(-1 * time.Hour)},
	}

	result, err := svc.Timeline(context.Background(), info, "STAT", "24h")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(result))
	}
	if result[0].TodoCount != 5 {
		t.Fatalf("expected todo_count 5, got %d", result[0].TodoCount)
	}
}

func TestStatsTimeline_InvalidRange(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestStatsService()
	info := userAuthInfo()
	setupStatsProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Timeline(context.Background(), info, "STAT", "30d")
	if err == nil {
		t.Fatal("expected validation error for invalid range")
	}
}

func TestStatsTimeline_NonMember(t *testing.T) {
	svc, _, projectRepo, _ := newTestStatsService()
	info := userAuthInfo()

	// Create project without adding user as member
	project := &model.Project{ID: uuid.New(), Name: "Other", Key: "OTH"}
	projectRepo.Create(context.Background(), project)

	_, err := svc.Timeline(context.Background(), info, "OTH", "7d")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStatsTimeline_AdminAccess(t *testing.T) {
	svc, _, projectRepo, _ := newTestStatsService()
	info := adminAuthInfo()

	// Admin can access without membership
	project := &model.Project{ID: uuid.New(), Name: "Other", Key: "ADM"}
	projectRepo.Create(context.Background(), project)

	_, err := svc.Timeline(context.Background(), info, "ADM", "7d")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStatsTimeline_ProjectNotFound(t *testing.T) {
	svc, _, _, _ := newTestStatsService()
	info := userAuthInfo()

	_, err := svc.Timeline(context.Background(), info, "NOPE", "7d")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
