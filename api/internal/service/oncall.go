package service

import (
	"context"
	"fmt"
	"slices"
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
	Members        []model.OncallRotationMemberWithUser `json:"members"`
	ActiveOverride *model.OncallOverride                `json:"active_override,omitempty"`
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

	firstUserID := input.MemberIDs[0]
	nextRotation := computeNextRotation(input.StartDate, input.RotationTime, input.Timezone, input.PeriodDays)

	rot := &model.OncallRotation{
		ID:              uuid.New(),
		TeamID:          teamID,
		PeriodDays:      input.PeriodDays,
		RotationTime:    input.RotationTime,
		Timezone:        input.Timezone,
		StartDate:       input.StartDate,
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
func (s *OncallService) GetRotation(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID) (*OncallRotationResult, error) {
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

	return s.getRotationResult(ctx, rot)
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
		rot.RotationTime = *input.RotationTime
	}
	if input.Timezone != nil {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", *input.Timezone, model.ErrValidation)
		}
		rot.Timezone = *input.Timezone
	}
	startDateChanged := false
	if input.StartDate != nil && *input.StartDate != rot.StartDate {
		rot.StartDate = *input.StartDate
		startDateChanged = true
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

// AdvanceRotation advances the rotation to the next member.
// Returns the old and new user IDs for notification purposes.
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

	// Advance position (wrapping)
	newPosition := (rot.CurrentPosition + 1) % len(members)
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

	// Update rotation state
	rot.CurrentPosition = newPosition
	rot.CurrentUserID = &newUserID
	nextRotation := computeNextRotationFromNow(rot.Timezone, rot.PeriodDays)
	rot.NextRotationAt = nextRotation

	if _, err := s.oncall.Update(ctx, rot); err != nil {
		return nil, fmt.Errorf("updating rotation: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("rotation_id", rotationID.String()).
		Str("old_user_id", oldUserID.String()).
		Str("new_user_id", newUserID.String()).
		Int("new_position", newPosition).
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

	if err := s.overrides.Delete(ctx, overrideID); err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Str("override_id", overrideID.String()).
		Msg("oncall override cancelled")

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

// GetActiveOverride returns the currently active override for a team's rotation (if any).
func (s *OncallService) GetActiveOverride(ctx context.Context, teamID uuid.UUID) (*model.OncallOverride, error) {
	if s.overrides == nil {
		return nil, nil
	}

	rot, err := s.oncall.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return s.overrides.GetActiveOverride(ctx, rot.ID)
}

// ListOverridesInRange returns overrides for a rotation within a time range (for calendar view).
func (s *OncallService) ListOverridesInRange(ctx context.Context, info *model.AuthInfo, projectKey string, teamID uuid.UUID, from, to time.Time) ([]model.OncallOverrideWithUser, error) {
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

	return s.overrides.ListOverridesInRange(ctx, rot.ID, from, to)
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

	// Include active override if available
	if s.overrides != nil {
		active, err := s.overrides.GetActiveOverride(ctx, rot.ID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to check active override")
		} else {
			result.ActiveOverride = active
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

func computeNextRotation(startDate, rotationTime, timezone string, periodDays int) *time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	// Parse start date and rotation time
	t, err := time.ParseInLocation("2006-01-02 15:04:05", startDate+" "+rotationTime, loc)
	if err != nil {
		// Fallback: just use start date at noon
		t, _ = time.ParseInLocation("2006-01-02", startDate, loc)
		t = t.Add(12 * time.Hour)
	}

	now := time.Now()

	// If start date+time is in the future, the first rotation happens then
	if t.After(now) {
		utc := t.UTC()
		return &utc
	}

	// Otherwise, add periods until we're in the future
	next := t.Add(time.Duration(periodDays) * 24 * time.Hour)
	for next.Before(now) {
		next = next.Add(time.Duration(periodDays) * 24 * time.Hour)
	}

	utc := next.UTC()
	return &utc
}

func computeNextRotationFromNow(_ string, periodDays int) *time.Time {
	next := time.Now().Add(time.Duration(periodDays) * 24 * time.Hour)
	return &next
}
