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

func (m *mockStatsRepo) Timeline(_ context.Context, projectID uuid.UUID, since time.Time, daily bool) ([]model.ProjectStatsSnapshot, error) {
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

	_, err := svc.Timeline(context.Background(), info, "STAT", "abc")
	if err == nil {
		t.Fatal("expected validation error for invalid range")
	}
}

func TestStatsTimeline_CustomRange(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestStatsService()
	info := userAuthInfo()
	setupStatsProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	// 30d should now be valid
	_, err := svc.Timeline(context.Background(), info, "STAT", "30d")
	if err != nil {
		t.Fatalf("expected no error for 30d, got %v", err)
	}
}

func TestParseSince(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"1 hour", "1h", false},
		{"24 hours", "24h", false},
		{"3 days", "3d", false},
		{"7 days", "7d", false},
		{"14 days", "14d", false},
		{"30 days", "30d", false},
		{"365 days", "365d", false},
		{"compound", "2h30m", false},
		{"30 minutes", "30m", false},
		{"empty", "", true},
		{"garbage", "abc", true},
		{"zero days", "0d", true},
		{"negative hours", "-5h", true},
		{"over max days", "400d", true},
		{"over max hours", "9000h", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSince(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSince(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNeedsDailyGranularity(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"7d", false},
		{"8d", true},
		{"168h", false},
		{"169h", true},
		{"24h", false},
		{"30d", true},
		{"3d", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := needsDailyGranularity(tt.input)
			if got != tt.want {
				t.Errorf("needsDailyGranularity(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
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
