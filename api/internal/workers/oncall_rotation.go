package workers

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// oncallTeamRepository is the minimal interface for looking up teams.
type oncallTeamRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error)
}

// oncallProjectRepository is the minimal interface for loading projects by ID.
type oncallProjectRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

// OncallRotationTask is a periodic task that scans for due rotations and advances them.
type OncallRotationTask struct {
	oncall    service.OncallRotationRepository
	service   *service.OncallService
	teams     oncallTeamRepository
	projects  oncallProjectRepository
	publisher slaEventPublisher
	logger    zerolog.Logger
}

// NewOncallRotationTask creates a new OncallRotationTask.
func NewOncallRotationTask(
	oncall service.OncallRotationRepository,
	svc *service.OncallService,
	teams oncallTeamRepository,
	projects oncallProjectRepository,
	publisher slaEventPublisher,
	logger zerolog.Logger,
) *OncallRotationTask {
	return &OncallRotationTask{
		oncall:    oncall,
		service:   svc,
		teams:     teams,
		projects:  projects,
		publisher: publisher,
		logger:    logger,
	}
}

// Run executes the on-call rotation scan.
func (t *OncallRotationTask) Run(ctx context.Context) error {
	rotations, err := t.oncall.ListDueRotations(ctx)
	if err != nil {
		return err
	}

	if len(rotations) == 0 {
		return nil
	}

	t.logger.Info().Int("due_rotations", len(rotations)).Msg("processing due oncall rotations")

	for _, rot := range rotations {
		result, err := t.service.AdvanceRotation(ctx, rot.ID)
		if err != nil {
			t.logger.Error().Err(err).
				Str("rotation_id", rot.ID.String()).
				Msg("failed to advance oncall rotation")
			continue
		}

		// Look up team and project for the event payload
		team, err := t.teams.GetByID(ctx, result.TeamID)
		if err != nil {
			t.logger.Error().Err(err).
				Str("team_id", result.TeamID.String()).
				Msg("failed to load team for oncall event")
			continue
		}

		project, err := t.projects.GetByID(ctx, team.ProjectID)
		if err != nil {
			t.logger.Error().Err(err).
				Str("project_id", team.ProjectID.String()).
				Msg("failed to load project for oncall event")
			continue
		}

		evt := model.OncallRotationAdvancedEvent{
			RotationID:     result.RotationID,
			TeamID:         result.TeamID,
			ProjectID:      project.ID,
			ProjectKey:     project.Key,
			ProjectName:    project.Name,
			TeamName:       team.Name,
			OldUserID:      result.OldUserID,
			NewUserID:      result.NewUserID,
			NextRotationAt: result.NextRotationAt,
		}

		if err := t.publisher.Publish("oncall.rotation.advanced", evt); err != nil {
			t.logger.Error().Err(err).
				Str("rotation_id", result.RotationID.String()).
				Msg("failed to publish oncall rotation event")
		}
	}

	return nil
}
