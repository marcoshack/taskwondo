package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// StatsRepository defines persistence operations for stats snapshots.
type StatsRepository interface {
	Timeline(ctx context.Context, projectID uuid.UUID, since time.Time, daily bool) ([]model.ProjectStatsSnapshot, error)
}

// StatsService handles stats business logic and authorization.
type StatsService struct {
	stats    StatsRepository
	projects ProjectRepository
	members  ProjectMemberRepository
}

// NewStatsService creates a new StatsService.
func NewStatsService(stats StatsRepository, projects ProjectRepository, members ProjectMemberRepository) *StatsService {
	return &StatsService{
		stats:    stats,
		projects: projects,
		members:  members,
	}
}

// Timeline returns project-level stats snapshots for the given time range.
func (s *StatsService) Timeline(ctx context.Context, info *model.AuthInfo, projectKey string, rangeStr string) ([]model.ProjectStatsSnapshot, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	since, err := parseSince(rangeStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), model.ErrValidation)
	}

	daily := needsDailyGranularity(rangeStr)
	return s.stats.Timeline(ctx, project.ID, since, daily)
}

var dayRe = regexp.MustCompile(`^(\d+)d$`)

const maxRange = 365 * 24 * time.Hour

// parseSince converts a range string like "24h", "3d", "30d", or "2h30m" into
// a time.Time representing now minus that duration. Supports all time.ParseDuration
// units plus day notation (<N>d → <N*24>h). Max range: 365d.
func parseSince(rangeStr string) (time.Time, error) {
	s := rangeStr
	if m := dayRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		s = fmt.Sprintf("%dh", n*24)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid range %q", rangeStr)
	}
	if d <= 0 || d > maxRange {
		return time.Time{}, fmt.Errorf("range must be between 1s and 365d")
	}
	return time.Now().Add(-d), nil
}

// needsDailyGranularity returns true if the range exceeds 7 days, meaning
// the chart should display daily aggregated data instead of hourly.
func needsDailyGranularity(rangeStr string) bool {
	s := rangeStr
	if m := dayRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		s = fmt.Sprintf("%dh", n*24)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return false
	}
	return d > 7*24*time.Hour
}

func (s *StatsService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	_, err := s.members.GetByProjectAndUser(ctx, projectID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking membership: %w", err)
	}
	return nil
}
