package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// slaProjectRepository is the minimal interface for loading projects.
type slaProjectRepository interface {
	ListAll(ctx context.Context) ([]model.Project, error)
}

// slaEscalationRepository is the minimal interface for loading escalation lists and mappings.
type slaEscalationRepository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]model.EscalationList, error)
	ListMappings(ctx context.Context, projectID uuid.UUID) ([]model.TypeEscalationMapping, error)
}

// slaNotificationRepository tracks which notifications have been sent.
type slaNotificationRepository interface {
	HasBeenSent(ctx context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) (bool, error)
	RecordSent(ctx context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) error
}

// slaWorkItemRepository loads active work items for SLA checking.
type slaWorkItemRepository interface {
	List(ctx context.Context, projectID uuid.UUID, filter *model.WorkItemFilter) (*model.WorkItemList, error)
}

// slaWorkflowRepository is the minimal interface for loading workflow status categories.
type slaWorkflowRepository interface {
	ListStatuses(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error)
}

// slaTypeWorkflowRepository loads type-to-workflow mappings for a project.
type slaTypeWorkflowRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectTypeWorkflow, error)
}

// slaInfoComputer computes SLA info for batches of work items.
type slaInfoComputer interface {
	ComputeSLAInfoBatch(ctx context.Context, items []model.WorkItem, projectID uuid.UUID, businessHours *model.BusinessHoursConfig) map[uuid.UUID]*model.SLAInfo
}

// slaEventPublisher publishes SLA breach events.
type slaEventPublisher interface {
	Publish(subject string, data any) error
}

// SLAMonitorTask is a periodic task that scans for SLA threshold crossings
// and publishes notification events for newly breached thresholds.
type SLAMonitorTask struct {
	projects      slaProjectRepository
	slaComputer   slaInfoComputer
	escalations   slaEscalationRepository
	notifications slaNotificationRepository
	items         slaWorkItemRepository
	workflows     slaWorkflowRepository
	typeWorkflows slaTypeWorkflowRepository
	publisher     slaEventPublisher
	logger        zerolog.Logger
}

// NewSLAMonitorTask creates a new SLAMonitorTask.
func NewSLAMonitorTask(
	projects slaProjectRepository,
	slaComputer slaInfoComputer,
	escalations slaEscalationRepository,
	notifications slaNotificationRepository,
	items slaWorkItemRepository,
	workflows slaWorkflowRepository,
	typeWorkflows slaTypeWorkflowRepository,
	publisher slaEventPublisher,
	logger zerolog.Logger,
) *SLAMonitorTask {
	return &SLAMonitorTask{
		projects:      projects,
		slaComputer:   slaComputer,
		escalations:   escalations,
		notifications: notifications,
		items:         items,
		workflows:     workflows,
		typeWorkflows: typeWorkflows,
		publisher:     publisher,
		logger:        logger,
	}
}

// Run executes the SLA monitoring scan.
func (t *SLAMonitorTask) Run(ctx context.Context) error {
	projects, err := t.projects.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	for _, project := range projects {
		if err := t.scanProject(ctx, &project); err != nil {
			t.logger.Error().Err(err).
				Str("project_id", project.ID.String()).
				Str("project_key", project.Key).
				Msg("failed to scan project for SLA breaches")
		}
	}

	return nil
}

func (t *SLAMonitorTask) scanProject(ctx context.Context, project *model.Project) error {
	// Load escalation lists for this project; skip if none defined
	lists, err := t.escalations.List(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("loading escalation lists: %w", err)
	}
	if len(lists) == 0 {
		return nil
	}

	// Load type-to-escalation-list mappings
	mappings, err := t.escalations.ListMappings(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("loading escalation mappings: %w", err)
	}
	if len(mappings) == 0 {
		return nil
	}

	// Build type → escalation list lookup
	listByID := make(map[uuid.UUID]*model.EscalationList, len(lists))
	for i := range lists {
		listByID[lists[i].ID] = &lists[i]
	}
	typeToList := make(map[string]*model.EscalationList, len(mappings))
	for _, m := range mappings {
		if l, ok := listByID[m.EscalationListID]; ok {
			typeToList[m.WorkItemType] = l
		}
	}

	// Determine active statuses from the workflows referenced by this project
	activeStatuses := t.resolveActiveStatuses(ctx, project)
	if len(activeStatuses) == 0 {
		return nil
	}

	// Load active work items in non-terminal statuses
	items, err := t.loadActiveItems(ctx, project.ID, activeStatuses)
	if err != nil {
		return fmt.Errorf("loading active items: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	// Batch compute SLA info using the SLA service
	slaInfoMap := t.slaComputer.ComputeSLAInfoBatch(ctx, items, project.ID, project.BusinessHours)

	// Check each item against escalation thresholds
	published := 0
	for _, item := range items {
		escList, ok := typeToList[item.Type]
		if !ok {
			continue
		}

		slaInfo, ok := slaInfoMap[item.ID]
		if !ok || slaInfo == nil {
			continue
		}

		// Skip paused items (outside business hours)
		if slaInfo.Paused {
			continue
		}

		// Check each escalation level
		for levelIdx, level := range escList.Levels {
			if slaInfo.Percentage < level.ThresholdPct {
				continue
			}

			escalationLevel := levelIdx + 1

			// Skip retroactive notifications: if the threshold was already
			// crossed before the escalation list was created, this is a
			// pre-existing breach that should not trigger a notification.
			if !escList.CreatedAt.IsZero() && slaInfo.TargetSeconds > 0 {
				secondsSinceListCreated := int(time.Since(escList.CreatedAt).Seconds())
				elapsedAtListCreation := slaInfo.ElapsedSeconds - secondsSinceListCreated
				if elapsedAtListCreation > 0 {
					pctAtListCreation := (elapsedAtListCreation * 100) / slaInfo.TargetSeconds
					if pctAtListCreation >= level.ThresholdPct {
						t.logger.Debug().
							Str("work_item_id", item.ID.String()).
							Int("threshold_pct", level.ThresholdPct).
							Int("pct_at_list_creation", pctAtListCreation).
							Msg("skipping retroactive SLA notification")
						continue
					}
				}
			}

			// Check if already sent
			sent, err := t.notifications.HasBeenSent(ctx, item.ID, item.Status, escalationLevel, level.ThresholdPct)
			if err != nil {
				t.logger.Warn().Err(err).
					Str("work_item_id", item.ID.String()).
					Msg("failed to check SLA notification sent status")
				continue
			}
			if sent {
				continue
			}

			// Publish breach event
			evt := model.SLABreachEvent{
				WorkItemID:       item.ID,
				ProjectID:        project.ID,
				ProjectKey:       project.Key,
				ItemNumber:       item.ItemNumber,
				Title:            item.Title,
				StatusName:       item.Status,
				SLAPercentage:    slaInfo.Percentage,
				TargetSeconds:    slaInfo.TargetSeconds,
				ElapsedSeconds:   slaInfo.ElapsedSeconds,
				EscalationLevel:  escalationLevel,
				EscalationListID: escList.ID,
				ThresholdPct:     level.ThresholdPct,
			}

			if err := t.publisher.Publish("notification.sla_breach", evt); err != nil {
				t.logger.Error().Err(err).
					Str("work_item_id", item.ID.String()).
					Int("threshold_pct", level.ThresholdPct).
					Msg("failed to publish SLA breach event")
				continue
			}

			// Record sent immediately to prevent duplicates from subsequent
			// scans that may run before the notification sender processes the event.
			if err := t.notifications.RecordSent(ctx, item.ID, item.Status, escalationLevel, level.ThresholdPct); err != nil {
				t.logger.Warn().Err(err).
					Str("work_item_id", item.ID.String()).
					Int("threshold_pct", level.ThresholdPct).
					Msg("failed to record SLA notification sent in monitor")
			}

			published++
			t.logger.Info().
				Str("project_key", project.Key).
				Int("item_number", item.ItemNumber).
				Int("percentage", slaInfo.Percentage).
				Int("threshold_pct", level.ThresholdPct).
				Int("escalation_level", escalationLevel).
				Msg("SLA breach event published")
		}
	}

	if published > 0 {
		t.logger.Info().
			Str("project_key", project.Key).
			Int("events_published", published).
			Msg("SLA monitor scan completed for project")
	}

	return nil
}

// resolveActiveStatuses collects non-terminal status names across all workflows
// used by this project.
func (t *SLAMonitorTask) resolveActiveStatuses(ctx context.Context, project *model.Project) map[string]bool {
	active := make(map[string]bool)
	seenWorkflows := make(map[uuid.UUID]bool)

	addStatuses := func(wfID uuid.UUID) {
		if seenWorkflows[wfID] {
			return
		}
		seenWorkflows[wfID] = true
		statuses, err := t.workflows.ListStatuses(ctx, wfID)
		if err != nil {
			return
		}
		for _, s := range statuses {
			if s.Category != model.CategoryDone && s.Category != model.CategoryCancelled {
				active[s.Name] = true
			}
		}
	}

	if project.DefaultWorkflowID != nil {
		addStatuses(*project.DefaultWorkflowID)
	}

	// Also include statuses from type-specific workflow mappings
	typeWorkflows, err := t.typeWorkflows.ListByProject(ctx, project.ID)
	if err == nil {
		for _, tw := range typeWorkflows {
			addStatuses(tw.WorkflowID)
		}
	}

	return active
}

// loadActiveItems loads work items in non-terminal statuses for a project.
func (t *SLAMonitorTask) loadActiveItems(ctx context.Context, projectID uuid.UUID, activeStatuses map[string]bool) ([]model.WorkItem, error) {
	statuses := make([]string, 0, len(activeStatuses))
	for s := range activeStatuses {
		statuses = append(statuses, s)
	}
	if len(statuses) == 0 {
		return nil, nil
	}

	var allItems []model.WorkItem
	var cursor *uuid.UUID
	for {
		filter := &model.WorkItemFilter{
			Statuses: statuses,
			Limit:    100,
			Cursor:   cursor,
			Sort:     "item_number",
			Order:    "asc",
		}

		result, err := t.items.List(ctx, projectID, filter)
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, result.Items...)

		if !result.HasMore || len(result.Items) == 0 {
			break
		}
		lastID := result.Items[len(result.Items)-1].ID
		cursor = &lastID
	}

	return allItems, nil
}

// formatDuration formats a number of seconds into a human-readable string (e.g. "2h 30m").
func formatDuration(seconds int) string {
	if seconds < 0 {
		return "overdue by " + formatDuration(-seconds)
	}
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60

	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", mins)
}
