package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// SLARepository defines persistence operations for SLA targets and elapsed tracking.
type SLARepository interface {
	ListTargetsByProject(ctx context.Context, projectID uuid.UUID) ([]model.SLAStatusTarget, error)
	ListTargetsByProjectAndType(ctx context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) ([]model.SLAStatusTarget, error)
	GetTarget(ctx context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID, statusName string) (*model.SLAStatusTarget, error)
	BulkUpsertTargets(ctx context.Context, targets []model.SLAStatusTarget) ([]model.SLAStatusTarget, error)
	DeleteTarget(ctx context.Context, id uuid.UUID) error
	DeleteTargetsByTypeAndWorkflow(ctx context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) error
	InitElapsedOnCreate(ctx context.Context, workItemID uuid.UUID, statusName string, enteredAt time.Time) error
	UpsertElapsedOnEnter(ctx context.Context, workItemID uuid.UUID, statusName string, now time.Time) error
	UpdateElapsedOnLeave(ctx context.Context, workItemID uuid.UUID, statusName string, now time.Time) error
	GetElapsed(ctx context.Context, workItemID uuid.UUID, statusName string) (*model.SLAElapsed, error)
	ListElapsedByWorkItemIDs(ctx context.Context, ids []uuid.UUID) ([]model.SLAElapsed, error)
}

// BulkUpsertSLAInput holds the input for bulk-upserting SLA targets.
type BulkUpsertSLAInput struct {
	WorkItemType string
	WorkflowID   uuid.UUID
	Targets      []SLATargetInput
}

// SLATargetInput holds the input for a single SLA target.
type SLATargetInput struct {
	StatusName    string
	TargetSeconds int
	CalendarMode  string
}

// SLAService handles SLA business logic and authorization.
type SLAService struct {
	sla       SLARepository
	projects  ProjectRepository
	members   ProjectMemberRepository
	workflows WorkflowRepository
}

// NewSLAService creates a new SLAService.
func NewSLAService(
	sla SLARepository,
	projects ProjectRepository,
	members ProjectMemberRepository,
	workflows WorkflowRepository,
) *SLAService {
	return &SLAService{
		sla:       sla,
		projects:  projects,
		members:   members,
		workflows: workflows,
	}
}

// ListTargets returns all SLA targets for a project.
func (s *SLAService) ListTargets(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.SLAStatusTarget, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.sla.ListTargetsByProject(ctx, project.ID)
}

// BulkUpsertTargets creates or updates SLA targets for a type+workflow combination.
func (s *SLAService) BulkUpsertTargets(ctx context.Context, info *model.AuthInfo, projectKey string, input BulkUpsertSLAInput) ([]model.SLAStatusTarget, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return nil, err
	}

	// Validate work item type
	if !isValidWorkItemType(input.WorkItemType) {
		return nil, fmt.Errorf("invalid work item type %q: %w", input.WorkItemType, model.ErrValidation)
	}

	// Validate workflow exists and get its statuses
	wf, err := s.workflows.GetByID(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", model.ErrValidation)
	}

	// Build status name → category map
	statusCategories := make(map[string]string, len(wf.Statuses))
	for _, st := range wf.Statuses {
		statusCategories[st.Name] = st.Category
	}

	// Validate each target
	for _, t := range input.Targets {
		if t.TargetSeconds <= 0 {
			return nil, fmt.Errorf("target_seconds must be positive: %w", model.ErrValidation)
		}
		if t.CalendarMode != model.CalendarMode24x7 && t.CalendarMode != model.CalendarModeBusinessHours {
			return nil, fmt.Errorf("invalid calendar_mode %q: %w", t.CalendarMode, model.ErrValidation)
		}
		category, exists := statusCategories[t.StatusName]
		if !exists {
			return nil, fmt.Errorf("status %q does not exist in workflow %q: %w", t.StatusName, wf.Name, model.ErrValidation)
		}
		if category == model.CategoryDone || category == model.CategoryCancelled {
			return nil, fmt.Errorf("cannot set SLA on terminal status %q (%s): %w", t.StatusName, category, model.ErrValidation)
		}
	}

	// Build target models
	targets := make([]model.SLAStatusTarget, len(input.Targets))
	for i, t := range input.Targets {
		targets[i] = model.SLAStatusTarget{
			ID:            uuid.New(),
			ProjectID:     project.ID,
			WorkItemType:  input.WorkItemType,
			WorkflowID:    input.WorkflowID,
			StatusName:    t.StatusName,
			TargetSeconds: t.TargetSeconds,
			CalendarMode:  t.CalendarMode,
		}
	}

	result, err := s.sla.BulkUpsertTargets(ctx, targets)
	if err != nil {
		return nil, fmt.Errorf("upserting SLA targets: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("work_item_type", input.WorkItemType).
		Int("targets_count", len(result)).
		Msg("SLA targets upserted")

	return result, nil
}

// DeleteTarget deletes a single SLA target.
func (s *SLAService) DeleteTarget(ctx context.Context, info *model.AuthInfo, projectKey string, targetID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID, model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
		return err
	}

	return s.sla.DeleteTarget(ctx, targetID)
}

// ComputeSLAInfo computes the SLA status for a work item's current status.
// Returns nil if no SLA target is defined for the item's current status.
func (s *SLAService) ComputeSLAInfo(ctx context.Context, item *model.WorkItem, workflowID uuid.UUID, businessHours *model.BusinessHoursConfig) *model.SLAInfo {
	target, err := s.sla.GetTarget(ctx, item.ProjectID, item.Type, workflowID, item.Status)
	if err != nil {
		return nil // No SLA target for this status
	}

	elapsed, err := s.sla.GetElapsed(ctx, item.ID, item.Status)
	if err != nil {
		return nil // No elapsed record yet
	}

	totalElapsed := elapsed.ElapsedSeconds
	if elapsed.LastEnteredAt != nil {
		// Item is currently in this status — add live elapsed time
		now := time.Now()
		if target.CalendarMode == model.CalendarModeBusinessHours && businessHours != nil {
			totalElapsed += CalculateBusinessSeconds(*elapsed.LastEnteredAt, now, *businessHours)
		} else {
			totalElapsed += int(now.Sub(*elapsed.LastEnteredAt).Seconds())
		}
	}

	remaining := target.TargetSeconds - totalElapsed
	percentage := 0
	if target.TargetSeconds > 0 {
		percentage = (totalElapsed * 100) / target.TargetSeconds
	}

	status := model.SLAStatusOnTrack
	if percentage >= 100 {
		status = model.SLAStatusBreached
	} else if percentage >= 75 {
		status = model.SLAStatusWarning
	}

	return &model.SLAInfo{
		TargetSeconds:    target.TargetSeconds,
		ElapsedSeconds:   totalElapsed,
		RemainingSeconds: remaining,
		Percentage:       percentage,
		Status:           status,
	}
}

// ComputeSLAInfoBatch computes SLA info for multiple work items efficiently.
func (s *SLAService) ComputeSLAInfoBatch(ctx context.Context, items []model.WorkItem, projectID uuid.UUID, businessHours *model.BusinessHoursConfig) map[uuid.UUID]*model.SLAInfo {
	result := make(map[uuid.UUID]*model.SLAInfo, len(items))
	if len(items) == 0 {
		return result
	}

	// Load all SLA targets for this project
	targets, err := s.sla.ListTargetsByProject(ctx, projectID)
	if err != nil {
		return result
	}

	// Build lookup: (type, workflowID, status) → target
	type targetKey struct {
		workItemType string
		workflowID   uuid.UUID
		statusName   string
	}
	targetMap := make(map[targetKey]*model.SLAStatusTarget, len(targets))
	for i := range targets {
		t := &targets[i]
		targetMap[targetKey{t.WorkItemType, t.WorkflowID, t.StatusName}] = t
	}

	// Batch load elapsed records
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	elapsedRecords, err := s.sla.ListElapsedByWorkItemIDs(ctx, ids)
	if err != nil {
		return result
	}

	// Build lookup: (workItemID, status) → elapsed
	type elapsedKey struct {
		workItemID uuid.UUID
		statusName string
	}
	elapsedMap := make(map[elapsedKey]*model.SLAElapsed, len(elapsedRecords))
	for i := range elapsedRecords {
		e := &elapsedRecords[i]
		elapsedMap[elapsedKey{e.WorkItemID, e.StatusName}] = e
	}

	now := time.Now()
	for _, item := range items {
		// Find matching target for this item's type + status
		// (targets contain workflow_id, but we match on type+status since a project
		// typically maps one workflow per type)
		var matchedTarget *model.SLAStatusTarget
		for k, t := range targetMap {
			if k.workItemType == item.Type && k.statusName == item.Status {
				matchedTarget = t
				break
			}
		}
		if matchedTarget == nil {
			continue
		}

		elapsed := elapsedMap[elapsedKey{item.ID, item.Status}]
		if elapsed == nil {
			continue
		}

		totalElapsed := elapsed.ElapsedSeconds
		if elapsed.LastEnteredAt != nil {
			if matchedTarget.CalendarMode == model.CalendarModeBusinessHours && businessHours != nil {
				totalElapsed += CalculateBusinessSeconds(*elapsed.LastEnteredAt, now, *businessHours)
			} else {
				totalElapsed += int(now.Sub(*elapsed.LastEnteredAt).Seconds())
			}
		}

		remaining := matchedTarget.TargetSeconds - totalElapsed
		percentage := 0
		if matchedTarget.TargetSeconds > 0 {
			percentage = (totalElapsed * 100) / matchedTarget.TargetSeconds
		}

		status := model.SLAStatusOnTrack
		if percentage >= 100 {
			status = model.SLAStatusBreached
		} else if percentage >= 75 {
			status = model.SLAStatusWarning
		}

		result[item.ID] = &model.SLAInfo{
			TargetSeconds:    matchedTarget.TargetSeconds,
			ElapsedSeconds:   totalElapsed,
			RemainingSeconds: remaining,
			Percentage:       percentage,
			Status:           status,
		}
	}

	return result
}

// ComputeSLAForItems computes SLA info for work items given a project key.
// Returns a map of work item ID → SLA info. Items without SLA targets are omitted.
func (s *SLAService) ComputeSLAForItems(ctx context.Context, projectKey string, items []model.WorkItem) map[uuid.UUID]*model.SLAInfo {
	if len(items) == 0 {
		return nil
	}
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil
	}
	return s.ComputeSLAInfoBatch(ctx, items, project.ID, project.BusinessHours)
}

// CalculateBusinessSeconds returns the number of business seconds between two times.
func CalculateBusinessSeconds(from, to time.Time, config model.BusinessHoursConfig) int {
	if to.Before(from) {
		return 0
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
	}

	from = from.In(loc)
	to = to.In(loc)

	// Build set of business days for O(1) lookup
	businessDays := make(map[time.Weekday]bool, len(config.Days))
	for _, d := range config.Days {
		businessDays[time.Weekday(d)] = true
	}

	businessSecondsPerDay := (config.EndHour - config.StartHour) * 3600
	if businessSecondsPerDay <= 0 {
		return 0
	}

	totalSeconds := 0
	current := from

	for current.Before(to) {
		if !businessDays[current.Weekday()] {
			// Skip non-business days
			current = time.Date(current.Year(), current.Month(), current.Day()+1, config.StartHour, 0, 0, 0, loc)
			continue
		}

		dayStart := time.Date(current.Year(), current.Month(), current.Day(), config.StartHour, 0, 0, 0, loc)
		dayEnd := time.Date(current.Year(), current.Month(), current.Day(), config.EndHour, 0, 0, 0, loc)

		// Clamp to business hours
		periodStart := current
		if periodStart.Before(dayStart) {
			periodStart = dayStart
		}
		periodEnd := to
		if periodEnd.After(dayEnd) {
			periodEnd = dayEnd
		}

		if periodStart.Before(periodEnd) {
			totalSeconds += int(periodEnd.Sub(periodStart).Seconds())
		}

		// Move to next day's start
		current = time.Date(current.Year(), current.Month(), current.Day()+1, config.StartHour, 0, 0, 0, loc)
	}

	return totalSeconds
}

// AddBusinessSeconds returns the absolute time when the given number of business
// seconds will have elapsed starting from `from`. This is the inverse of
// CalculateBusinessSeconds.
func AddBusinessSeconds(from time.Time, seconds int, config model.BusinessHoursConfig) time.Time {
	if seconds <= 0 {
		return from
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		loc = time.UTC
	}

	current := from.In(loc)

	businessDays := make(map[time.Weekday]bool, len(config.Days))
	for _, d := range config.Days {
		businessDays[time.Weekday(d)] = true
	}

	remaining := seconds

	for remaining > 0 {
		if !businessDays[current.Weekday()] {
			// Skip to next day's start
			current = time.Date(current.Year(), current.Month(), current.Day()+1, config.StartHour, 0, 0, 0, loc)
			continue
		}

		dayStart := time.Date(current.Year(), current.Month(), current.Day(), config.StartHour, 0, 0, 0, loc)
		dayEnd := time.Date(current.Year(), current.Month(), current.Day(), config.EndHour, 0, 0, 0, loc)

		// Snap to business hours
		if current.Before(dayStart) {
			current = dayStart
		}
		if !current.Before(dayEnd) {
			// Past end of business hours — advance to next day
			current = time.Date(current.Year(), current.Month(), current.Day()+1, config.StartHour, 0, 0, 0, loc)
			continue
		}

		available := int(dayEnd.Sub(current).Seconds())
		if remaining <= available {
			return current.Add(time.Duration(remaining) * time.Second)
		}

		remaining -= available
		current = time.Date(current.Year(), current.Month(), current.Day()+1, config.StartHour, 0, 0, 0, loc)
	}

	return current
}

// ComputeSLATargetAt computes the absolute deadline for a work item's current
// SLA target. Returns nil if no SLA target exists for the item's current status.
func (s *SLAService) ComputeSLATargetAt(ctx context.Context, item *model.WorkItem, workflowID uuid.UUID, businessHours *model.BusinessHoursConfig) *time.Time {
	target, err := s.sla.GetTarget(ctx, item.ProjectID, item.Type, workflowID, item.Status)
	if err != nil {
		return nil
	}

	elapsed, err := s.sla.GetElapsed(ctx, item.ID, item.Status)
	elapsedSeconds := 0
	if err == nil {
		elapsedSeconds = elapsed.ElapsedSeconds
	}

	remaining := target.TargetSeconds - elapsedSeconds
	if remaining <= 0 {
		// Already breached — return a time in the past
		t := time.Now().Add(-time.Duration(-remaining) * time.Second)
		return &t
	}

	now := time.Now()
	if target.CalendarMode == model.CalendarModeBusinessHours && businessHours != nil {
		t := AddBusinessSeconds(now, remaining, *businessHours)
		return &t
	}

	t := now.Add(time.Duration(remaining) * time.Second)
	return &t
}

// ComputeSLATargetAtSimple computes the SLA deadline for a newly created item
// where elapsed time is known to be zero. Avoids DB lookups.
func ComputeSLATargetAtSimple(targetSeconds int, calendarMode string, businessHours *model.BusinessHoursConfig) *time.Time {
	now := time.Now()
	if calendarMode == model.CalendarModeBusinessHours && businessHours != nil {
		t := AddBusinessSeconds(now, targetSeconds, *businessHours)
		return &t
	}
	t := now.Add(time.Duration(targetSeconds) * time.Second)
	return &t
}

func (s *SLAService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *SLAService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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
	for _, role := range allowedRoles {
		if member.Role == role {
			return nil
		}
	}
	return model.ErrForbidden
}
