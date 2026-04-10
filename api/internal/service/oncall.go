package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// OncallRotationRepository defines persistence operations for on-call rotations.
type OncallRotationRepository interface {
	Create(ctx context.Context, rot *model.OncallRotation) (*model.OncallRotation, error)
	GetByTeamID(ctx context.Context, teamID uuid.UUID) (*model.OncallRotation, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.OncallRotation, error)
	Update(ctx context.Context, rot *model.OncallRotation) (*model.OncallRotation, error)
	Delete(ctx context.Context, teamID uuid.UUID) error
	SetMembers(ctx context.Context, rotationID uuid.UUID, members []model.OncallRotationMember) error
	ListMembers(ctx context.Context, rotationID uuid.UUID) ([]model.OncallRotationMemberWithUser, error)
	CreateHistory(ctx context.Context, h *model.OncallRotationHistory) error
	EndCurrentHistory(ctx context.Context, rotationID uuid.UUID) error
	ListHistory(ctx context.Context, rotationID uuid.UUID, limit, offset int) ([]model.OncallRotationHistoryWithUser, int, error)
	ListDueRotations(ctx context.Context) ([]model.OncallRotation, error)
}

// OncallOverrideRepository defines persistence operations for on-call overrides.
type OncallOverrideRepository interface {
	Create(ctx context.Context, o *model.OncallOverride) (*model.OncallOverride, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.OncallOverride, error)
	Update(ctx context.Context, o *model.OncallOverride) (*model.OncallOverride, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByRotation(ctx context.Context, rotationID uuid.UUID) ([]model.OncallOverrideWithUser, error)
	GetActiveOverride(ctx context.Context, rotationID uuid.UUID) (*model.OncallOverride, error)
	ListOverridesInRange(ctx context.Context, rotationID uuid.UUID, from, to time.Time) ([]model.OncallOverrideWithUser, error)
	ListStaleOverrideRotations(ctx context.Context) ([]model.OverrideTransition, error)
}

// CreateOncallRotationInput holds the input for creating an on-call rotation.
type CreateOncallRotationInput struct {
	PeriodDays   int
	RotationTime string // "HH:MM:SS"
	Timezone     string
	StartDate    string // "YYYY-MM-DD"
	MemberIDs    []uuid.UUID
}

// UpdateOncallRotationInput holds the input for updating an on-call rotation.
type UpdateOncallRotationInput struct {
	PeriodDays   *int
	RotationTime *string
	Timezone     *string
	StartDate    *string
	MemberIDs    []uuid.UUID
}

// OncallRotationResult is returned by GetRotation with members included.
type OncallRotationResult struct {
	model.OncallRotation
	Members   []model.OncallRotationMemberWithUser `json:"members"`
	Overrides []model.OncallOverrideWithUser       `json:"overrides,omitempty"`
	Shifts    []ScheduleShift                      `json:"shifts,omitempty"`
}

// CreateOncallOverrideInput holds the input for creating an on-call override.
type CreateOncallOverrideInput struct {
	OverrideUserID uuid.UUID
	StartAt        time.Time
	EndAt          time.Time
	Reason         *string
}

// UpdateOncallOverrideInput holds the input for updating an on-call override.
type UpdateOncallOverrideInput struct {
	OverrideUserID *uuid.UUID
	StartAt        *time.Time
	EndAt          *time.Time
	Reason         *string
	ClearReason    bool
}

// AdvanceResult holds the old and new user IDs after a rotation advance.
type AdvanceResult struct {
	OldUserID      uuid.UUID
	NewUserID      uuid.UUID
	RotationID     uuid.UUID
	TeamID         uuid.UUID
	NextRotationAt *time.Time
}

// ScheduleShift represents a single on-call shift in the projected schedule.
type ScheduleShift struct {
	UserID     uuid.UUID  `json:"user_id"`
	StartAt    time.Time  `json:"start_at"`
	EndAt      time.Time  `json:"end_at"`
	IsOverride bool       `json:"is_override"`
	OverrideID *uuid.UUID `json:"override_id,omitempty"`
}

// OncallService handles on-call rotation business logic and authorization.
type OncallService struct {
	oncall    OncallRotationRepository
	overrides OncallOverrideRepository
	teams     TeamRepository
	projects  ProjectRepository
	members   ProjectMemberRepository
	publisher EventPublisher
}

// NewOncallService creates a new OncallService.
func NewOncallService(
	oncall OncallRotationRepository,
	teams TeamRepository,
	projects ProjectRepository,
	members ProjectMemberRepository,
) *OncallService {
	return &OncallService{
		oncall:   oncall,
		teams:    teams,
		projects: projects,
		members:  members,
	}
}

// SetOverrideRepository sets the override repository (optional dependency).
func (s *OncallService) SetOverrideRepository(repo OncallOverrideRepository) {
	s.overrides = repo
}

// SetPublisher sets the event publisher for override notifications.
func (s *OncallService) SetPublisher(p EventPublisher) {
	s.publisher = p
}

// CreateRotation creates a new on-call rotation for a team.
func (s *OncallService) CreateRotation(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, input CreateOncallRotationInput) (*OncallRotationResult, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	if len(input.MemberIDs) == 0 {
		return nil, fmt.Errorf("at least one member is required: %w", model.ErrValidation)
	}

	if err := s.validateMembers(ctx, teamID, input.MemberIDs); err != nil {
		return nil, err
	}

	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", input.Timezone, model.ErrValidation)
	}

	if input.PeriodDays <= 0 {
		return nil, fmt.Errorf("period_days must be greater than 0: %w", model.ErrValidation)
	}

	startDate, err := parseDate(input.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date %q: %w", input.StartDate, model.ErrValidation)
	}
	rotationTime, err := parseTimeOfDay(input.RotationTime)
	if err != nil {
		return nil, fmt.Errorf("invalid rotation_time %q: %w", input.RotationTime, model.ErrValidation)
	}

	firstUserID := input.MemberIDs[0]
	nextRotation := computeNextRotation(startDate, rotationTime, input.Timezone, input.PeriodDays)

	rot := &model.OncallRotation{
		ID:              uuid.New(),
		TeamID:          teamID,
		PeriodDays:      input.PeriodDays,
		RotationTime:    rotationTime,
		Timezone:        input.Timezone,
		StartDate:       startDate,
		CurrentUserID:   &firstUserID,
		CurrentPosition: 0,
		NextRotationAt:  nextRotation,
	}

	rot, err = s.oncall.Create(ctx, rot)
	if err != nil {
		return nil, fmt.Errorf("creating oncall rotation: %w", err)
	}

	members := buildRotationMembers(rot.ID, input.MemberIDs)
	if err := s.oncall.SetMembers(ctx, rot.ID, members); err != nil {
		return nil, fmt.Errorf("setting rotation members: %w", err)
	}

	// Create first history entry
	history := &model.OncallRotationHistory{
		ID:         uuid.Must(uuid.NewV7()),
		RotationID: rot.ID,
		UserID:     firstUserID,
		StartedAt:  time.Now(),
	}
	if err := s.oncall.CreateHistory(ctx, history); err != nil {
		return nil, fmt.Errorf("creating initial history: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("rotation_id", rot.ID.String()).
		Str("team_id", teamID.String()).
		Msg("oncall rotation created")

	return s.getRotationResult(ctx, rot)
}

// GetRotation returns the on-call rotation for a team with members.
// When rangeStart and rangeEnd are provided, it also computes the projected schedule shifts.
func (s *OncallService) GetRotation(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, rangeStart, rangeEnd *time.Time) (*OncallRotationResult, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	result, err := s.getRotationResult(ctx, rot)
	if err != nil {
		return nil, err
	}

	// Compute schedule shifts when a date range is requested
	if rangeStart != nil && rangeEnd != nil && len(result.Members) > 0 {
		shifts := computeRotationShifts(rot, result.Members, *rangeStart, *rangeEnd)

		if s.overrides != nil {
			overridesInRange, err := s.overrides.ListOverridesInRange(ctx, rot.ID, *rangeStart, *rangeEnd)
			if err != nil {
				return nil, fmt.Errorf("listing overrides in range: %w", err)
			}
			if len(overridesInRange) > 0 {
				shifts = applyOverrides(shifts, overridesInRange)
			}
		}

		result.Shifts = shifts
	}

	return result, nil
}

// UpdateRotation modifies an on-call rotation.
func (s *OncallService) UpdateRotation(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, input UpdateOncallRotationInput) (*OncallRotationResult, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if input.PeriodDays != nil {
		if *input.PeriodDays <= 0 {
			return nil, fmt.Errorf("period_days must be greater than 0: %w", model.ErrValidation)
		}
		rot.PeriodDays = *input.PeriodDays
	}
	if input.RotationTime != nil {
		rt, err := parseTimeOfDay(*input.RotationTime)
		if err != nil {
			return nil, fmt.Errorf("invalid rotation_time %q: %w", *input.RotationTime, model.ErrValidation)
		}
		rot.RotationTime = rt
	}
	if input.Timezone != nil {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", *input.Timezone, model.ErrValidation)
		}
		rot.Timezone = *input.Timezone
	}
	startDateChanged := false
	if input.StartDate != nil {
		sd, err := parseDate(*input.StartDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date %q: %w", *input.StartDate, model.ErrValidation)
		}
		if !sd.Equal(rot.StartDate) {
			rot.StartDate = sd
			startDateChanged = true
		}
	}

	if input.MemberIDs != nil {
		if len(input.MemberIDs) == 0 {
			return nil, fmt.Errorf("at least one member is required: %w", model.ErrValidation)
		}

		if err := s.validateMembers(ctx, teamID, input.MemberIDs); err != nil {
			return nil, err
		}

		members := buildRotationMembers(rot.ID, input.MemberIDs)
		if err := s.oncall.SetMembers(ctx, rot.ID, members); err != nil {
			return nil, fmt.Errorf("setting rotation members: %w", err)
		}

		// If current on-call user was removed, advance to next valid member
		if rot.CurrentUserID != nil && !containsUserID(input.MemberIDs, *rot.CurrentUserID) {
			rot.CurrentPosition = 0
			rot.CurrentUserID = &input.MemberIDs[0]
		}
	}

	// When start_date changes, reset rotation: first member is oncall from now
	// until the new start date, then rotation continues normally.
	if startDateChanged {
		memberList, err := s.oncall.ListMembers(ctx, rot.ID)
		if err != nil {
			return nil, fmt.Errorf("listing members for start date reset: %w", err)
		}
		if len(memberList) > 0 {
			rot.CurrentPosition = 0
			firstUserID := memberList[0].UserID
			rot.CurrentUserID = &firstUserID

			if err := s.oncall.EndCurrentHistory(ctx, rot.ID); err != nil {
				return nil, fmt.Errorf("ending current history: %w", err)
			}
			history := &model.OncallRotationHistory{
				ID:         uuid.Must(uuid.NewV7()),
				RotationID: rot.ID,
				UserID:     firstUserID,
				StartedAt:  time.Now(),
			}
			if err := s.oncall.CreateHistory(ctx, history); err != nil {
				return nil, fmt.Errorf("creating history for start date reset: %w", err)
			}
		}
	}

	rot.NextRotationAt = computeNextRotation(rot.StartDate, rot.RotationTime, rot.Timezone, rot.PeriodDays)

	rot, err = s.oncall.Update(ctx, rot)
	if err != nil {
		return nil, fmt.Errorf("updating oncall rotation: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("rotation_id", rot.ID.String()).
		Str("team_id", teamID.String()).
		Msg("oncall rotation updated")

	return s.getRotationResult(ctx, rot)
}

// DeleteRotation removes the on-call rotation for a team.
func (s *OncallService) DeleteRotation(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team.ProjectID != project.ID {
		return model.ErrNotFound
	}

	if err := s.oncall.Delete(ctx, teamID); err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Str("team_id", teamID.String()).
		Msg("oncall rotation deleted")

	return nil
}

// ListHistory returns paginated history for a team's on-call rotation.
func (s *OncallService) ListHistory(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, limit, offset int) ([]model.OncallRotationHistoryWithUser, int, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, 0, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, 0, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, 0, err
	}
	if team.ProjectID != project.ID {
		return nil, 0, model.ErrNotFound
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.oncall.ListHistory(ctx, rot.ID, limit, offset)
}

// AdvanceRotation advances the rotation to the correct member based on the
// current time. If multiple periods have elapsed since the last rotation
// (e.g. the worker was down), it skips ahead to the right position rather
// than advancing one step at a time.
func (s *OncallService) AdvanceRotation(ctx context.Context, rotationID uuid.UUID) (*AdvanceResult, error) {
	rot, err := s.oncall.GetByID(ctx, rotationID)
	if err != nil {
		return nil, fmt.Errorf("loading rotation: %w", err)
	}

	members, err := s.oncall.ListMembers(ctx, rotationID)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("rotation has no members: %w", model.ErrValidation)
	}

	oldUserID := uuid.Nil
	if rot.CurrentUserID != nil {
		oldUserID = *rot.CurrentUserID
	}

	// Compute how many periods have elapsed since next_rotation_at.
	// If the worker was late or down, we may need to skip multiple positions.
	periodsToAdvance := 1
	if rot.NextRotationAt != nil {
		period := time.Duration(rot.PeriodDays) * 24 * time.Hour
		elapsed := time.Since(*rot.NextRotationAt)
		if elapsed > 0 {
			periodsToAdvance = 1 + int(elapsed/period)
		}
	}

	newPosition := (rot.CurrentPosition + periodsToAdvance) % len(members)
	newUserID := members[newPosition].UserID

	// End current history
	if err := s.oncall.EndCurrentHistory(ctx, rotationID); err != nil {
		return nil, fmt.Errorf("ending current history: %w", err)
	}

	// Create new history entry
	history := &model.OncallRotationHistory{
		ID:         uuid.Must(uuid.NewV7()),
		RotationID: rotationID,
		UserID:     newUserID,
		StartedAt:  time.Now(),
	}
	if err := s.oncall.CreateHistory(ctx, history); err != nil {
		return nil, fmt.Errorf("creating history: %w", err)
	}

	// Update rotation state — compute next_rotation_at aligned to the
	// rotation schedule (start_date + rotation_time in timezone), not
	// relative to now, to prevent time drift.
	rot.CurrentPosition = newPosition
	rot.CurrentUserID = &newUserID
	rot.IsOverride = false
	rot.NextRotationAt = computeNextRotation(rot.StartDate, rot.RotationTime, rot.Timezone, rot.PeriodDays)

	// Check for active override — if one exists, current_user_id should be the
	// override user, but position still advances.
	if s.overrides != nil {
		active, err := s.overrides.GetActiveOverride(ctx, rotationID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to check active override during rotation advance")
		} else if active != nil {
			rot.CurrentUserID = &active.OverrideUserID
			rot.IsOverride = true
		}
	}

	if _, err := s.oncall.Update(ctx, rot); err != nil {
		return nil, fmt.Errorf("updating rotation: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("rotation_id", rotationID.String()).
		Str("old_user_id", oldUserID.String()).
		Str("new_user_id", newUserID.String()).
		Int("new_position", newPosition).
		Int("periods_advanced", periodsToAdvance).
		Msg("oncall rotation advanced")

	return &AdvanceResult{
		OldUserID:      oldUserID,
		NewUserID:      newUserID,
		RotationID:     rotationID,
		TeamID:         rot.TeamID,
		NextRotationAt: rot.NextRotationAt,
	}, nil
}

// --- Override methods ---

// CreateOverride creates a new on-call override for a team's rotation.
func (s *OncallService) CreateOverride(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, input CreateOncallOverrideInput) (*model.OncallOverride, error) {
	if s.overrides == nil {
		return nil, fmt.Errorf("override repository not configured")
	}

	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Validate time range
	if !input.StartAt.Before(input.EndAt) {
		return nil, fmt.Errorf("start_at must be before end_at: %w", model.ErrValidation)
	}
	if input.EndAt.Before(time.Now()) {
		return nil, fmt.Errorf("override end_at must be in the future: %w", model.ErrValidation)
	}

	// Authorization: admins/owners can create for anyone, members only for themselves
	isAdmin := info.GlobalRole == model.RoleAdmin
	if !isAdmin {
		member, err := s.members.GetByProjectAndUser(ctx, project.ID, info.UserID)
		if err != nil {
			if err == model.ErrNotFound {
				return nil, model.ErrNotFound
			}
			return nil, fmt.Errorf("checking membership: %w", err)
		}
		isAdmin = member.Role == model.ProjectRoleOwner || member.Role == model.ProjectRoleAdmin
	}

	if !isAdmin && input.OverrideUserID != info.UserID {
		return nil, model.ErrForbidden
	}

	// Customers cannot be override users
	overrideMember, err := s.members.GetByProjectAndUser(ctx, project.ID, input.OverrideUserID)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, fmt.Errorf("override user is not a project member: %w", model.ErrValidation)
		}
		return nil, fmt.Errorf("checking override user membership: %w", err)
	}
	if overrideMember.Role == model.ProjectRoleCustomer {
		return nil, fmt.Errorf("customers cannot be assigned as on-call override: %w", model.ErrValidation)
	}

	override := &model.OncallOverride{
		ID:             uuid.Must(uuid.NewV7()),
		RotationID:     rot.ID,
		OverrideUserID: input.OverrideUserID,
		StartAt:        input.StartAt,
		EndAt:          input.EndAt,
		Reason:         input.Reason,
		CreatedBy:      info.UserID,
	}

	result, err := s.overrides.Create(ctx, override)
	if err != nil {
		return nil, fmt.Errorf("creating oncall override: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("override_id", result.ID.String()).
		Str("rotation_id", rot.ID.String()).
		Str("override_user_id", input.OverrideUserID.String()).
		Msg("oncall override created")

	// If override is immediately active, update rotation state
	if !input.StartAt.After(time.Now()) {
		rot.CurrentUserID = &input.OverrideUserID
		rot.IsOverride = true
		if _, err := s.oncall.Update(ctx, rot); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update rotation after creating active override")
		}
	}

	// Publish event for notifications
	if s.publisher != nil {
		scheduledUserID := uuid.Nil
		if rot.CurrentUserID != nil {
			scheduledUserID = *rot.CurrentUserID
		}
		evt := model.OncallOverrideCreatedEvent{
			OverrideID:     result.ID,
			RotationID:     rot.ID,
			TeamID:         teamID,
			TeamName:       team.Name,
			ProjectID:      project.ID,
			ProjectKey:     project.Key,
			ProjectName:    project.Name,
			OverrideUserID: input.OverrideUserID,
			ScheduledUser:  scheduledUserID,
			StartAt:        input.StartAt,
			EndAt:          input.EndAt,
			Reason:         input.Reason,
		}
		if err := s.publisher.Publish("oncall.override.created", evt); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to publish oncall override created event")
		}
	}

	return result, nil
}

// ListOverrides returns active and upcoming overrides for a team's rotation.
func (s *OncallService) ListOverrides(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID) ([]model.OncallOverrideWithUser, error) {
	if s.overrides == nil {
		return nil, nil
	}

	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return s.overrides.ListByRotation(ctx, rot.ID)
}

// UpdateOverride modifies an existing on-call override.
func (s *OncallService) UpdateOverride(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, overrideID uuid.UUID, input UpdateOncallOverrideInput) (*model.OncallOverride, error) {
	if s.overrides == nil {
		return nil, fmt.Errorf("override repository not configured")
	}

	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.ProjectID != project.ID {
		return nil, model.ErrNotFound
	}

	override, err := s.overrides.GetByID(ctx, overrideID)
	if err != nil {
		return nil, err
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if override.RotationID != rot.ID {
		return nil, model.ErrNotFound
	}

	// Authorization: creator or admins/owners can update
	if override.CreatedBy != info.UserID {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return nil, err
		}
	}

	if input.OverrideUserID != nil {
		// Validate new override user is not a customer
		overrideMember, err := s.members.GetByProjectAndUser(ctx, project.ID, *input.OverrideUserID)
		if err != nil {
			if err == model.ErrNotFound {
				return nil, fmt.Errorf("override user is not a project member: %w", model.ErrValidation)
			}
			return nil, fmt.Errorf("checking override user membership: %w", err)
		}
		if overrideMember.Role == model.ProjectRoleCustomer {
			return nil, fmt.Errorf("customers cannot be assigned as on-call override: %w", model.ErrValidation)
		}
		override.OverrideUserID = *input.OverrideUserID
	}
	if input.StartAt != nil {
		override.StartAt = *input.StartAt
	}
	if input.EndAt != nil {
		override.EndAt = *input.EndAt
	}
	if input.ClearReason {
		override.Reason = nil
	} else if input.Reason != nil {
		override.Reason = input.Reason
	}

	// Validate time range
	if !override.StartAt.Before(override.EndAt) {
		return nil, fmt.Errorf("start_at must be before end_at: %w", model.ErrValidation)
	}

	result, err := s.overrides.Update(ctx, override)
	if err != nil {
		return nil, fmt.Errorf("updating oncall override: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("override_id", overrideID.String()).
		Msg("oncall override updated")

	// Reconcile rotation state after update
	now := time.Now()
	overrideIsActive := !override.StartAt.After(now) && override.EndAt.After(now)

	if overrideIsActive {
		// This override is now active — set override user
		rot.CurrentUserID = &override.OverrideUserID
		rot.IsOverride = true
		if _, err := s.oncall.Update(ctx, rot); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update rotation after override update")
		}
	} else if rot.IsOverride {
		// Override was moved out of active window — check for other active overrides
		active, err := s.overrides.GetActiveOverride(ctx, rot.ID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to check active override after update")
		} else if active != nil {
			rot.CurrentUserID = &active.OverrideUserID
			rot.IsOverride = true
			if _, err := s.oncall.Update(ctx, rot); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to update rotation with fallback override")
			}
		} else {
			// No active override — restore scheduled user
			members, err := s.oncall.ListMembers(ctx, rot.ID)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to list members for scheduled user restore")
			} else if len(members) > 0 {
				scheduledUser := computeScheduledUserNow(rot, members)
				rot.CurrentUserID = &scheduledUser
				rot.IsOverride = false
				if _, err := s.oncall.Update(ctx, rot); err != nil {
					log.Ctx(ctx).Error().Err(err).Msg("failed to restore scheduled user after override update")
				}
			}
		}
	}

	return result, nil
}

// DeleteOverride cancels an on-call override.
func (s *OncallService) DeleteOverride(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, overrideID uuid.UUID) error {
	if s.overrides == nil {
		return fmt.Errorf("override repository not configured")
	}

	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team.ProjectID != project.ID {
		return model.ErrNotFound
	}

	override, err := s.overrides.GetByID(ctx, overrideID)
	if err != nil {
		return err
	}

	// Verify the override belongs to this team's rotation
	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return err
	}
	if override.RotationID != rot.ID {
		return model.ErrNotFound
	}

	// Authorization: creator or admins/owners can cancel
	if override.CreatedBy != info.UserID {
		if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return err
		}
	}

	// Check if the override was active before deleting it
	now := time.Now()
	wasActive := !override.StartAt.After(now) && override.EndAt.After(now)

	if err := s.overrides.Delete(ctx, overrideID); err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Str("override_id", overrideID.String()).
		Msg("oncall override cancelled")

	// If the deleted override was active, reconcile rotation state
	if wasActive {
		active, err := s.overrides.GetActiveOverride(ctx, rot.ID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to check active override after delete")
		} else if active != nil {
			// Another override is still active
			rot.CurrentUserID = &active.OverrideUserID
			rot.IsOverride = true
			if _, err := s.oncall.Update(ctx, rot); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to update rotation with fallback override")
			}
		} else {
			// No active override — restore scheduled user
			members, err := s.oncall.ListMembers(ctx, rot.ID)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to list members for scheduled user restore")
			} else if len(members) > 0 {
				scheduledUser := computeScheduledUserNow(rot, members)
				rot.CurrentUserID = &scheduledUser
				rot.IsOverride = false
				if _, err := s.oncall.Update(ctx, rot); err != nil {
					log.Ctx(ctx).Error().Err(err).Msg("failed to restore scheduled user after override delete")
				}
			}
		}
	}

	// Publish event for notifications
	if s.publisher != nil {
		scheduledUserID := uuid.Nil
		if rot.CurrentUserID != nil {
			scheduledUserID = *rot.CurrentUserID
		}
		evt := model.OncallOverrideCancelledEvent{
			OverrideID:     overrideID,
			RotationID:     rot.ID,
			TeamID:         teamID,
			TeamName:       team.Name,
			ProjectID:      project.ID,
			ProjectKey:     project.Key,
			ProjectName:    project.Name,
			OverrideUserID: override.OverrideUserID,
			ScheduledUser:  scheduledUserID,
			StartAt:        override.StartAt,
			EndAt:          override.EndAt,
		}
		if err := s.publisher.Publish("oncall.override.cancelled", evt); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to publish oncall override cancelled event")
		}
	}

	return nil
}

// ReconcileOverrides is called by the worker to fix rotations where is_override
// is stale (override started or ended without synchronous update).
func (s *OncallService) ReconcileOverrides(ctx context.Context) error {
	if s.overrides == nil {
		return nil
	}

	transitions, err := s.overrides.ListStaleOverrideRotations(ctx)
	if err != nil {
		return fmt.Errorf("listing stale override rotations: %w", err)
	}

	if len(transitions) == 0 {
		return nil
	}

	log.Ctx(ctx).Info().Int("transitions", len(transitions)).Msg("reconciling stale override rotations")

	for _, t := range transitions {
		rot, err := s.oncall.GetByID(ctx, t.RotationID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).
				Str("rotation_id", t.RotationID.String()).
				Msg("failed to load rotation for override reconciliation")
			continue
		}

		switch t.Type {
		case "ended":
			// Override ended — restore scheduled user
			members, err := s.oncall.ListMembers(ctx, rot.ID)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).
					Str("rotation_id", rot.ID.String()).
					Msg("failed to list members for override reconciliation")
				continue
			}
			if len(members) > 0 {
				scheduledUser := computeScheduledUserNow(rot, members)
				rot.CurrentUserID = &scheduledUser
				rot.IsOverride = false
				if _, err := s.oncall.Update(ctx, rot); err != nil {
					log.Ctx(ctx).Error().Err(err).
						Str("rotation_id", rot.ID.String()).
						Msg("failed to restore scheduled user during reconciliation")
				}
			}

		case "started":
			// Override started — set override user
			if t.OverrideUserID != nil {
				rot.CurrentUserID = t.OverrideUserID
				rot.IsOverride = true
				if _, err := s.oncall.Update(ctx, rot); err != nil {
					log.Ctx(ctx).Error().Err(err).
						Str("rotation_id", rot.ID.String()).
						Msg("failed to set override user during reconciliation")
				}
			}
		}
	}

	return nil
}

// --- helpers ---

func (s *OncallService) validateMembers(ctx context.Context, teamID uuid.UUID, memberIDs []uuid.UUID) error {
	teamMembers, err := s.teams.ListMembers(ctx, teamID)
	if err != nil {
		return fmt.Errorf("listing team members: %w", err)
	}

	memberSet := make(map[uuid.UUID]bool, len(teamMembers))
	for _, m := range teamMembers {
		memberSet[m.UserID] = true
	}

	for _, id := range memberIDs {
		if !memberSet[id] {
			return fmt.Errorf("user %s is not a team member: %w", id, model.ErrValidation)
		}
	}

	return nil
}

func (s *OncallService) getRotationResult(ctx context.Context, rot *model.OncallRotation) (*OncallRotationResult, error) {
	members, err := s.oncall.ListMembers(ctx, rot.ID)
	if err != nil {
		return nil, fmt.Errorf("listing rotation members: %w", err)
	}

	result := &OncallRotationResult{
		OncallRotation: *rot,
		Members:        members,
	}

	// Include overrides (active + upcoming) if available
	if s.overrides != nil {
		overrides, err := s.overrides.ListByRotation(ctx, rot.ID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to list overrides")
		} else {
			result.Overrides = overrides

			// Add override users to members list if not already present
			memberSet := make(map[uuid.UUID]bool, len(members))
			for _, m := range members {
				memberSet[m.UserID] = true
			}
			for _, ov := range overrides {
				if !memberSet[ov.OverrideUserID] {
					memberSet[ov.OverrideUserID] = true
					result.Members = append(result.Members, model.OncallRotationMemberWithUser{
						OncallRotationMember: model.OncallRotationMember{
							ID:         ov.ID, // use override ID as a synthetic member ID
							RotationID: rot.ID,
							UserID:     ov.OverrideUserID,
							Position:   -1,
						},
						DisplayName: ov.OverrideUserName,
						AvatarURL:   ov.OverrideAvatar,
					})
				}
			}
		}
	}

	return result, nil
}

func (s *OncallService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *OncallService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	member, err := s.members.GetByProjectAndUser(ctx, projectID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking membership: %w", err)
	}
	if slices.Contains(allowedRoles, member.Role) {
		return nil
	}
	return model.ErrForbidden
}

func buildRotationMembers(rotationID uuid.UUID, userIDs []uuid.UUID) []model.OncallRotationMember {
	members := make([]model.OncallRotationMember, len(userIDs))
	for i, uid := range userIDs {
		members[i] = model.OncallRotationMember{
			ID:         uuid.New(),
			RotationID: rotationID,
			UserID:     uid,
			Position:   i,
		}
	}
	return members
}

func containsUserID(ids []uuid.UUID, target uuid.UUID) bool {
	return slices.Contains(ids, target)
}

// rotationEpoch combines a rotation's start date and time-of-day into a single
// time.Time in the rotation's configured timezone.
func rotationEpoch(startDate, rotationTime time.Time, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	y, m, d := startDate.Date()
	return time.Date(y, m, d, rotationTime.Hour(), rotationTime.Minute(), rotationTime.Second(), 0, loc)
}

// computeScheduledUserNow returns the user who should be on-call per the base
// rotation at the current time (ignoring overrides). The logic mirrors
// computeRotationShifts: compute rotationEpoch, then periodsElapsed, then
// memberIdx = periodsElapsed % len(members).
func computeScheduledUserNow(rot *model.OncallRotation, members []model.OncallRotationMemberWithUser) uuid.UUID {
	t0 := rotationEpoch(rot.StartDate, rot.RotationTime, rot.Timezone)
	period := time.Duration(rot.PeriodDays) * 24 * time.Hour
	now := time.Now()

	elapsed := now.Sub(t0)
	periodsElapsed := int(elapsed / period)
	if elapsed < 0 && elapsed%period != 0 {
		periodsElapsed--
	}

	numMembers := len(members)
	memberIdx := periodsElapsed % numMembers
	if memberIdx < 0 {
		memberIdx += numMembers
	}

	return members[memberIdx].UserID
}

func computeNextRotation(startDate, rotationTime time.Time, timezone string, periodDays int) *time.Time {
	t := rotationEpoch(startDate, rotationTime, timezone)

	now := time.Now()

	// If start date+time is in the future, the first rotation happens then
	if t.After(now) {
		utc := t.UTC()
		return &utc
	}

	// Otherwise, add periods until we're in the future
	period := time.Duration(periodDays) * 24 * time.Hour
	next := t.Add(period)
	for next.Before(now) {
		next = next.Add(period)
	}

	utc := next.UTC()
	return &utc
}

// parseDate parses a "YYYY-MM-DD" string from API input into a time.Time (UTC midnight).
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// parseTimeOfDay parses an "HH:MM:SS" or "HH:MM" string from API input into a time.Time.
// Only the hour/minute/second components are meaningful; the date is zero-valued.
func parseTimeOfDay(s string) (time.Time, error) {
	t, err := time.Parse("15:04:05", s)
	if err != nil {
		t, err = time.Parse("15:04", s)
	}
	return t, err
}

// computeRotationShifts projects the base rotation schedule (without overrides) for a time range.
func computeRotationShifts(rot *model.OncallRotation, members []model.OncallRotationMemberWithUser, rangeStart, rangeEnd time.Time) []ScheduleShift {
	t0 := rotationEpoch(rot.StartDate, rot.RotationTime, rot.Timezone)

	period := time.Duration(rot.PeriodDays) * 24 * time.Hour
	numMembers := len(members)

	// Find the first shift boundary at or before rangeStart.
	// elapsed periods = floor((rangeStart - t0) / period)
	elapsed := rangeStart.Sub(t0)
	periodsElapsed := int(elapsed / period)
	if elapsed < 0 && elapsed%period != 0 {
		periodsElapsed--
	}

	shiftStart := t0.Add(time.Duration(periodsElapsed) * period)

	// Ensure we start at or before rangeStart
	for shiftStart.After(rangeStart) {
		shiftStart = shiftStart.Add(-period)
		periodsElapsed--
	}

	var shifts []ScheduleShift
	for shiftStart.Before(rangeEnd) {
		shiftEnd := shiftStart.Add(period)

		// Member index: periodsElapsed mod numMembers (handle negative)
		memberIdx := periodsElapsed % numMembers
		if memberIdx < 0 {
			memberIdx += numMembers
		}

		// Clip to requested range
		clippedStart := shiftStart
		clippedEnd := shiftEnd
		if clippedStart.Before(rangeStart) {
			clippedStart = rangeStart
		}
		if clippedEnd.After(rangeEnd) {
			clippedEnd = rangeEnd
		}

		member := members[memberIdx]
		shifts = append(shifts, ScheduleShift{
			UserID:     member.UserID,
			StartAt:    clippedStart,
			EndAt:      clippedEnd,
			IsOverride: false,
		})

		shiftStart = shiftEnd
		periodsElapsed++
	}

	return shifts
}

// applyOverrides splits base rotation shifts where overrides are active.
// Overrides sorted by created_at ASC so later-created overrides take priority.
func applyOverrides(shifts []ScheduleShift, overrides []model.OncallOverrideWithUser) []ScheduleShift {
	sort.Slice(overrides, func(i, j int) bool {
		return overrides[i].CreatedAt.Before(overrides[j].CreatedAt)
	})

	for _, ov := range overrides {
		var newShifts []ScheduleShift
		for _, shift := range shifts {
			// No overlap: shift ends before override starts or shift starts at/after override ends
			if !shift.StartAt.Before(ov.EndAt) || !ov.StartAt.Before(shift.EndAt) {
				newShifts = append(newShifts, shift)
				continue
			}

			// Before portion (part of shift before override starts)
			if shift.StartAt.Before(ov.StartAt) {
				before := shift
				before.EndAt = ov.StartAt
				newShifts = append(newShifts, before)
			}

			// Override portion
			ovStart := shift.StartAt
			if ov.StartAt.After(ovStart) {
				ovStart = ov.StartAt
			}
			ovEnd := shift.EndAt
			if ov.EndAt.Before(ovEnd) {
				ovEnd = ov.EndAt
			}
			ovID := ov.ID
			newShifts = append(newShifts, ScheduleShift{
				UserID:     ov.OverrideUserID,
				StartAt:    ovStart,
				EndAt:      ovEnd,
				IsOverride: true,
				OverrideID: &ovID,
			})

			// After portion (part of shift after override ends)
			if shift.EndAt.After(ov.EndAt) {
				after := shift
				after.StartAt = ov.EndAt
				newShifts = append(newShifts, after)
			}
		}
		shifts = newShifts
	}

	// Merge consecutive shifts with the same user
	return mergeConsecutiveShifts(shifts)
}

// mergeConsecutiveShifts merges adjacent shifts that have the same user and override status.
func mergeConsecutiveShifts(shifts []ScheduleShift) []ScheduleShift {
	if len(shifts) <= 1 {
		return shifts
	}
	merged := []ScheduleShift{shifts[0]}
	for _, s := range shifts[1:] {
		last := &merged[len(merged)-1]
		if last.UserID == s.UserID && last.IsOverride == s.IsOverride && last.EndAt.Equal(s.StartAt) {
			last.EndAt = s.EndAt
		} else {
			merged = append(merged, s)
		}
	}
	return merged
}
