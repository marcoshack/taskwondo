package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// StatsRepository defines persistence operations for stats snapshots.
type StatsRepository interface {
	Timeline(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.ProjectStatsSnapshot, error)
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

	return s.stats.Timeline(ctx, project.ID, since)
}

func parseSince(rangeStr string) (time.Time, error) {
	now := time.Now()
	switch rangeStr {
	case "24h":
		return now.Add(-24 * time.Hour), nil
	case "3d":
		return now.Add(-3 * 24 * time.Hour), nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("invalid range %q, must be 24h, 3d, or 7d", rangeStr)
	}
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
