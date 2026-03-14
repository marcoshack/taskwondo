package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/storage"
)

// WorkItemRepository defines persistence operations for work items.
type WorkItemRepository interface {
	Create(ctx context.Context, item *model.WorkItem) error
	GetByProjectAndNumber(ctx context.Context, projectID uuid.UUID, itemNumber int) (*model.WorkItem, error)
	List(ctx context.Context, projectID uuid.UUID, filter *model.WorkItemFilter) (*model.WorkItemList, error)
	Update(ctx context.Context, item *model.WorkItem) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// WorkItemEventRepository defines persistence operations for work item events.
type WorkItemEventRepository interface {
	Create(ctx context.Context, event *model.WorkItemEvent) error
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemEvent, error)
	ListByWorkItemFiltered(ctx context.Context, workItemID uuid.UUID, visibility string) ([]model.WorkItemEventWithActor, error)
}

// CommentRepository defines persistence operations for comments.
type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Comment, error)
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID, visibility string) ([]model.Comment, error)
	Update(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// WorkItemRelationRepository defines persistence operations for work item relations.
type WorkItemRelationRepository interface {
	Create(ctx context.Context, relation *model.WorkItemRelation) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.WorkItemRelation, error)
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemRelation, error)
	ListByWorkItemWithDetails(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemRelationWithDetails, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AttachmentRepository defines persistence operations for attachments.
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *model.Attachment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Attachment, error)
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.Attachment, error)
	UpdateComment(ctx context.Context, id uuid.UUID, comment string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TimeEntryRepository defines persistence operations for time entries.
type TimeEntryRepository interface {
	Create(ctx context.Context, entry *model.TimeEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.TimeEntry, error)
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.TimeEntry, error)
	Update(ctx context.Context, entry *model.TimeEntry) error
	Delete(ctx context.Context, id uuid.UUID) error
	SumByWorkItem(ctx context.Context, workItemID uuid.UUID) (int, error)
}

// WatcherRepository defines persistence operations for work item watchers.
type WatcherRepository interface {
	Create(ctx context.Context, watcher *model.WorkItemWatcher) error
	Delete(ctx context.Context, workItemID, userID uuid.UUID) error
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemWatcherWithUser, error)
	CountByWorkItem(ctx context.Context, workItemID uuid.UUID) (int, error)
	IsWatching(ctx context.Context, workItemID, userID uuid.UUID) (bool, error)
	ListWatchedItemIDs(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID) ([]uuid.UUID, error)
	RemoveByProjectID(ctx context.Context, projectID uuid.UUID) (int, error)
}

// EventPublisher publishes async events (e.g. notifications) to a message broker.
type EventPublisher interface {
	Publish(subject string, data any) error
}

// CreateWorkItemInput holds the input for creating a work item.
type CreateWorkItemInput struct {
	Type         string
	Title        string
	Description  *string
	Priority     string
	AssigneeID   *uuid.UUID
	Labels       []string
	Complexity   *int
	ParentID     *uuid.UUID
	QueueID      *uuid.UUID
	MilestoneID  *uuid.UUID
	Visibility   string
	DueDate      *time.Time
	CustomFields map[string]interface{}
	WatcherIDs   []uuid.UUID
}

// UpdateWorkItemInput holds the input for updating a work item.
type UpdateWorkItemInput struct {
	Title            *string
	Description      *string
	ClearDescription bool
	Status           *string
	Priority         *string
	Type             *string
	AssigneeID       *uuid.UUID
	ClearAssignee    bool
	Labels           *[]string
	Complexity       *int
	ClearComplexity  bool
	Visibility       *string
	DueDate          *time.Time
	ClearDueDate     bool
	ParentID         *uuid.UUID
	ClearParent      bool
	QueueID          *uuid.UUID
	ClearQueue       bool
	MilestoneID      *uuid.UUID
	ClearMilestone   bool
	EstimatedSeconds *int
	ClearEstimate    bool
	CustomFields     map[string]interface{}
}

// CreateTimeEntryInput holds the input for logging a time entry.
type CreateTimeEntryInput struct {
	StartedAt       time.Time
	DurationSeconds int
	Description     string
}

// UpdateTimeEntryInput holds the input for updating a time entry.
type UpdateTimeEntryInput struct {
	StartedAt       *time.Time
	DurationSeconds *int
	Description     *string
	ClearDescription bool
}

// CreateCommentInput holds the input for creating a comment.
type CreateCommentInput struct {
	Body       string
	Visibility string
}

// CreateRelationInput holds the input for creating a relation.
type CreateRelationInput struct {
	TargetDisplayID string
	RelationType    string
}

// CreateAttachmentInput holds the input for uploading an attachment.
type CreateAttachmentInput struct {
	Filename    string
	ContentType string
	Size        int64
	Comment     string
	Reader      io.Reader
}

// RelationWithDisplay is a relation enriched with display IDs and titles.
type RelationWithDisplay struct {
	model.WorkItemRelation
	SourceDisplayID string
	SourceTitle     string
	TargetDisplayID string
	TargetTitle     string
}

// WorkItemService handles work item business logic and authorization.
type WorkItemService struct {
	items         WorkItemRepository
	events        WorkItemEventRepository
	comments      CommentRepository
	relations     WorkItemRelationRepository
	attachments   AttachmentRepository
	timeEntries   TimeEntryRepository
	watchers      WatcherRepository
	projects      ProjectRepository
	members       ProjectMemberRepository
	workflows     WorkflowRepository
	typeWorkflows ProjectTypeWorkflowRepository
	queues        QueueRepository
	milestones    MilestoneRepository
	sla           SLARepository
	slaService    *SLAService
	fileStorage   storage.Storage
	maxUploadSize int64
	publisher     EventPublisher
	embedCache    *FeatureFlagCache
}

// NewWorkItemService creates a new WorkItemService.
func NewWorkItemService(
	items WorkItemRepository,
	events WorkItemEventRepository,
	comments CommentRepository,
	relations WorkItemRelationRepository,
	attachments AttachmentRepository,
	timeEntries TimeEntryRepository,
	watchers WatcherRepository,
	projects ProjectRepository,
	members ProjectMemberRepository,
	workflows WorkflowRepository,
	typeWorkflows ProjectTypeWorkflowRepository,
	queues QueueRepository,
	milestones MilestoneRepository,
	sla SLARepository,
	slaService *SLAService,
	fileStorage storage.Storage,
	maxUploadSize int64,
) *WorkItemService {
	return &WorkItemService{
		items:         items,
		events:        events,
		comments:      comments,
		relations:     relations,
		attachments:   attachments,
		timeEntries:   timeEntries,
		watchers:      watchers,
		projects:      projects,
		members:       members,
		workflows:     workflows,
		typeWorkflows: typeWorkflows,
		queues:        queues,
		milestones:    milestones,
		sla:           sla,
		slaService:    slaService,
		fileStorage:   fileStorage,
		maxUploadSize: maxUploadSize,
	}
}

// SetPublisher configures the event publisher for async notifications.
func (s *WorkItemService) SetPublisher(p EventPublisher) {
	s.publisher = p
}

// SetEmbedCache configures the feature flag cache for semantic search indexing.
func (s *WorkItemService) SetEmbedCache(cache *FeatureFlagCache) {
	s.embedCache = cache
}

// Create creates a new work item in the given project.
func (s *WorkItemService) Create(ctx context.Context, info *model.AuthInfo, projectKey string, input CreateWorkItemInput) (*model.WorkItem, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	// Validate type
	if !isValidWorkItemType(input.Type) {
		return nil, fmt.Errorf("invalid work item type %q: %w", input.Type, model.ErrValidation)
	}

	// Validate title
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("title is required: %w", model.ErrValidation)
	}

	// Default and validate priority
	if input.Priority == "" {
		input.Priority = model.PriorityMedium
	}
	if !isValidPriority(input.Priority) {
		return nil, fmt.Errorf("invalid priority %q: %w", input.Priority, model.ErrValidation)
	}

	// Default and validate visibility
	if input.Visibility == "" {
		input.Visibility = model.VisibilityInternal
	}
	if !isValidVisibility(input.Visibility) {
		return nil, fmt.Errorf("invalid visibility %q: %w", input.Visibility, model.ErrValidation)
	}

	// Validate assignee is a project member and not a viewer
	if input.AssigneeID != nil {
		member, err := s.members.GetByProjectAndUser(ctx, project.ID, *input.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("assignee must be a project member: %w", model.ErrValidation)
		}
		if member.Role == model.ProjectRoleViewer {
			return nil, fmt.Errorf("viewers cannot be assigned to work items: %w", model.ErrValidation)
		}
	}

	// Validate queue belongs to this project
	if input.QueueID != nil {
		q, err := s.queues.GetByID(ctx, *input.QueueID)
		if err != nil {
			return nil, fmt.Errorf("queue not found: %w", model.ErrValidation)
		}
		if q.ProjectID != project.ID {
			return nil, fmt.Errorf("queue does not belong to this project: %w", model.ErrValidation)
		}
	}

	// Validate milestone belongs to this project
	if input.MilestoneID != nil {
		m, err := s.milestones.GetByID(ctx, *input.MilestoneID)
		if err != nil {
			return nil, fmt.Errorf("milestone not found: %w", model.ErrValidation)
		}
		if m.ProjectID != project.ID {
			return nil, fmt.Errorf("milestone does not belong to this project: %w", model.ErrValidation)
		}
	}

	// Validate complexity against project settings
	if input.Complexity != nil {
		if err := validateComplexity(*input.Complexity, project.AllowedComplexityValues); err != nil {
			return nil, err
		}
	}

	// Validate watcher IDs are project members
	for _, watcherID := range input.WatcherIDs {
		if _, err := s.members.GetByProjectAndUser(ctx, project.ID, watcherID); err != nil {
			return nil, fmt.Errorf("watcher must be a project member: %w", model.ErrValidation)
		}
	}

	labels := input.Labels
	if labels == nil {
		labels = []string{}
	}
	customFields := input.CustomFields
	if customFields == nil {
		customFields = map[string]interface{}{}
	}

	// Determine initial status from the type-specific workflow
	initialStatus := "open"
	var slaTargetAt *time.Time
	if wfID, err := s.resolveWorkflowID(ctx, project.ID, input.Type, project.DefaultWorkflowID); err == nil {
		if status, err := s.workflows.GetInitialStatus(ctx, wfID); err == nil {
			initialStatus = status.Name
		}
		// Compute SLA deadline for the initial status (elapsed is 0 for new items)
		if target, err := s.sla.GetTarget(ctx, project.ID, input.Type, wfID, initialStatus); err == nil {
			slaTargetAt = ComputeSLATargetAtSimple(target.TargetSeconds, target.CalendarMode, project.BusinessHours)
		}
	}

	item := &model.WorkItem{
		ID:           uuid.Must(uuid.NewV7()),
		ProjectID:    project.ID,
		QueueID:      input.QueueID,
		MilestoneID:  input.MilestoneID,
		ParentID:     input.ParentID,
		Type:         input.Type,
		Title:        input.Title,
		Description:  input.Description,
		Status:       initialStatus,
		Priority:     input.Priority,
		AssigneeID:   input.AssigneeID,
		ReporterID:   info.UserID,
		Visibility:   input.Visibility,
		Labels:       labels,
		Complexity:   input.Complexity,
		CustomFields: customFields,
		DueDate:      input.DueDate,
		SLATargetAt:  slaTargetAt,
	}

	if err := s.items.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("creating work item: %w", err)
	}

	// Record "created" event
	s.recordEvent(ctx, item.ID, info, "created", nil, nil, nil)

	// Initialize SLA elapsed tracking for the initial status
	if err := s.sla.InitElapsedOnCreate(ctx, item.ID, item.Status, time.Now()); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to initialize SLA elapsed tracking")
	}

	// Bulk-insert watchers specified at creation time
	for _, watcherUserID := range input.WatcherIDs {
		w := &model.WorkItemWatcher{
			ID:         uuid.New(),
			WorkItemID: item.ID,
			UserID:     watcherUserID,
			AddedBy:    info.UserID,
		}
		if err := s.watchers.Create(ctx, w); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("user_id", watcherUserID.String()).Msg("failed to add watcher on create")
		}
	}

	// Publish assignment notification if item was created with an assignee
	s.publishAssignment(ctx, projectKey, item, info.UserID)

	// Publish new item notification for project members
	s.publishNewItem(ctx, projectKey, project.ID, item, info.UserID)

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Int("item_number", item.ItemNumber).
		Str("type", item.Type).
		Msg("work item created")

	// Re-fetch to get DB-assigned timestamps
	created, err := s.items.GetByProjectAndNumber(ctx, project.ID, item.ItemNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching created work item: %w", err)
	}

	// Publish embed.index event for semantic search
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeWorkItem, created.ID, &created.ProjectID)

	return created, nil
}

// Get returns a work item by project key and item number.
func (s *WorkItemService) Get(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) (*model.WorkItem, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
}

// List returns work items matching the given filter.
func (s *WorkItemService) List(ctx context.Context, info *model.AuthInfo, projectKey string, filter *model.WorkItemFilter) (*model.WorkItemList, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	// Resolve "assignee=me" into AssigneeIDs
	if filter.AssigneeMe {
		filter.AssigneeIDs = append(filter.AssigneeIDs, info.UserID)
	}

	// Sanitize defaults
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Sort == "" {
		filter.Sort = "created_at"
	}
	if !isValidSortField(filter.Sort) {
		filter.Sort = "created_at"
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}
	if filter.Order != "asc" && filter.Order != "desc" {
		filter.Order = "desc"
	}

	return s.items.List(ctx, project.ID, filter)
}

// Update modifies a work item, recording events for each field change.
func (s *WorkItemService) Update(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, input UpdateWorkItemInput) (*model.WorkItem, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	// Track field changes for watcher notifications
	type fieldChange struct {
		field, oldVal, newVal string
	}
	var watcherChanges []fieldChange
	var statusOld, statusNew, statusCategory string // for status change notification

	// Apply changes and record events
	if input.Title != nil && *input.Title != item.Title {
		if strings.TrimSpace(*input.Title) == "" {
			return nil, fmt.Errorf("title cannot be empty: %w", model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, info, "title", item.Title, *input.Title)
		watcherChanges = append(watcherChanges, fieldChange{"title", item.Title, *input.Title})
		item.Title = *input.Title
	}

	if input.Description != nil {
		oldDesc := ""
		if item.Description != nil {
			oldDesc = *item.Description
		}
		if *input.Description != oldDesc {
			s.recordFieldChange(ctx, item.ID, info, "description", oldDesc, *input.Description)
			watcherChanges = append(watcherChanges, fieldChange{"description", oldDesc, *input.Description})
			item.Description = input.Description
		}
	}

	// Process type change BEFORE status change so workflow resolution uses the new type
	typeChanged := false
	if input.Type != nil && *input.Type != item.Type {
		if !isValidWorkItemType(*input.Type) {
			return nil, fmt.Errorf("invalid work item type %q: %w", *input.Type, model.ErrValidation)
		}

		// Check if current status is valid in the new type's workflow.
		// If incompatible, auto-reset to the target workflow's initial status.
		if newWfID, err := s.resolveWorkflowID(ctx, project.ID, *input.Type, project.DefaultWorkflowID); err == nil {
			statuses, _ := s.workflows.ListStatuses(ctx, newWfID)
			statusExists := false
			effectiveStatus := item.Status
			if input.Status != nil {
				effectiveStatus = *input.Status
			}
			for _, st := range statuses {
				if st.Name == effectiveStatus {
					statusExists = true
					break
				}
			}
			if !statusExists {
				initialStatus, err := s.workflows.GetInitialStatus(ctx, newWfID)
				if err != nil {
					return nil, fmt.Errorf("getting initial status for workflow: %w", err)
				}
				log.Ctx(ctx).Info().
					Str("old_status", effectiveStatus).
					Str("new_status", initialStatus.Name).
					Str("new_type", *input.Type).
					Msg("auto-resetting status to initial for new workflow")
				s.recordFieldChange(ctx, item.ID, info, "status", item.Status, initialStatus.Name)
				// Manage resolved_at based on status category change
				newCategory, _ := s.workflows.GetStatusCategory(ctx, newWfID, initialStatus.Name)
				oldCategory, _ := s.workflows.GetStatusCategory(ctx, newWfID, item.Status)
				if oldCategory != newCategory {
					now := time.Now()
					if newCategory == "done" || newCategory == "cancelled" {
						item.ResolvedAt = &now
					} else {
						item.ResolvedAt = nil
					}
				}
				item.Status = initialStatus.Name
				input.Status = nil // prevent duplicate status processing below
			}
		}

		s.recordFieldChange(ctx, item.ID, info, "type", item.Type, *input.Type)
		watcherChanges = append(watcherChanges, fieldChange{"type", item.Type, *input.Type})
		item.Type = *input.Type
		typeChanged = true
	}

	if input.Status != nil && *input.Status != item.Status {
		// Resolve workflow for the item's (potentially updated) type
		if wfID, err := s.resolveWorkflowID(ctx, project.ID, item.Type, project.DefaultWorkflowID); err == nil {
			// Skip transition validation when type changed — the type-change block
			// already verified the new status exists in the target workflow.
			// Normal transition rules don't apply across workflow boundaries.
			if !typeChanged {
				valid, err := s.workflows.ValidateTransition(ctx, wfID, item.Status, *input.Status)
				if err != nil {
					return nil, fmt.Errorf("validating transition: %w", err)
				}
				if !valid {
					return nil, fmt.Errorf("transition from %q to %q is not allowed in this workflow: %w",
						item.Status, *input.Status, model.ErrInvalidTransition)
				}
			}

			// Manage resolved_at based on status category
			newCategory, _ := s.workflows.GetStatusCategory(ctx, wfID, *input.Status)
			oldCategory, _ := s.workflows.GetStatusCategory(ctx, wfID, item.Status)

			if (newCategory == model.CategoryDone || newCategory == model.CategoryCancelled) &&
				oldCategory != model.CategoryDone && oldCategory != model.CategoryCancelled {
				now := time.Now()
				item.ResolvedAt = &now
			} else if (newCategory == model.CategoryTodo || newCategory == model.CategoryInProgress) &&
				(oldCategory == model.CategoryDone || oldCategory == model.CategoryCancelled) {
				item.ResolvedAt = nil
			}
		}

		// SLA elapsed tracking: leave old status, enter new status
		now := time.Now()
		if err := s.accumulateElapsedOnLeave(ctx, project, item, now); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to update SLA elapsed on leave")
		}
		if err := s.sla.UpsertElapsedOnEnter(ctx, item.ID, *input.Status, now); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to upsert SLA elapsed on enter")
		}

		s.recordFieldChange(ctx, item.ID, info, "status", item.Status, *input.Status)
		watcherChanges = append(watcherChanges, fieldChange{"status", item.Status, *input.Status})
		statusOld = item.Status
		statusNew = *input.Status
		if wfID, err := s.resolveWorkflowID(ctx, project.ID, item.Type, project.DefaultWorkflowID); err == nil {
			statusCategory, _ = s.workflows.GetStatusCategory(ctx, wfID, statusNew)
		}
		item.Status = *input.Status

		// Recompute SLA deadline for the new status
		if s.slaService != nil {
			if wfID, err := s.resolveWorkflowID(ctx, project.ID, item.Type, project.DefaultWorkflowID); err == nil {
				item.SLATargetAt = s.slaService.ComputeSLATargetAt(ctx, item, wfID, project.BusinessHours)
			}
		}
	}

	// If type changed but status didn't, recompute SLA (different type may have different SLA target)
	if typeChanged && (input.Status == nil || *input.Status == item.Status) {
		if s.slaService != nil {
			if wfID, err := s.resolveWorkflowID(ctx, project.ID, item.Type, project.DefaultWorkflowID); err == nil {
				item.SLATargetAt = s.slaService.ComputeSLATargetAt(ctx, item, wfID, project.BusinessHours)
			}
		}
	}

	if input.Priority != nil && *input.Priority != item.Priority {
		if !isValidPriority(*input.Priority) {
			return nil, fmt.Errorf("invalid priority %q: %w", *input.Priority, model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, info, "priority", item.Priority, *input.Priority)
		watcherChanges = append(watcherChanges, fieldChange{"priority", item.Priority, *input.Priority})
		item.Priority = *input.Priority
	}

	if input.ClearAssignee {
		if item.AssigneeID != nil {
			s.recordEvent(ctx, item.ID, info, "unassigned", strPtr("assignee_id"), strPtr(item.AssigneeID.String()), nil)
			watcherChanges = append(watcherChanges, fieldChange{"assignee", item.AssigneeID.String(), ""})
			item.AssigneeID = nil
		}
	} else if input.AssigneeID != nil {
		oldAssignee := ""
		if item.AssigneeID != nil {
			oldAssignee = item.AssigneeID.String()
		}
		newAssignee := input.AssigneeID.String()
		if oldAssignee != newAssignee {
			// Validate assignee is a project member and not a viewer
			member, err := s.members.GetByProjectAndUser(ctx, project.ID, *input.AssigneeID)
			if err != nil {
				return nil, fmt.Errorf("assignee must be a project member: %w", model.ErrValidation)
			}
			if member.Role == model.ProjectRoleViewer {
				return nil, fmt.Errorf("viewers cannot be assigned to work items: %w", model.ErrValidation)
			}
			s.recordEvent(ctx, item.ID, info, "assigned", strPtr("assignee_id"), strPtr(oldAssignee), strPtr(newAssignee))
			watcherChanges = append(watcherChanges, fieldChange{"assignee", oldAssignee, newAssignee})
			item.AssigneeID = input.AssigneeID
			s.publishAssignment(ctx, project.Key, item, info.UserID)
		}
	}

	if input.Labels != nil {
		oldLabels := strings.Join(item.Labels, ",")
		newLabels := strings.Join(*input.Labels, ",")
		if oldLabels != newLabels {
			s.recordFieldChange(ctx, item.ID, info, "labels", oldLabels, newLabels)
			watcherChanges = append(watcherChanges, fieldChange{"labels", oldLabels, newLabels})
			item.Labels = *input.Labels
		}
	}

	if input.ClearComplexity {
		if item.Complexity != nil {
			s.recordEvent(ctx, item.ID, info, "complexity_cleared", strPtr("complexity"), strPtr(strconv.Itoa(*item.Complexity)), nil)
			item.Complexity = nil
		}
	} else if input.Complexity != nil {
		oldVal := ""
		if item.Complexity != nil {
			oldVal = strconv.Itoa(*item.Complexity)
		}
		newVal := strconv.Itoa(*input.Complexity)
		if oldVal != newVal {
			if err := validateComplexity(*input.Complexity, project.AllowedComplexityValues); err != nil {
				return nil, err
			}
			s.recordFieldChange(ctx, item.ID, info, "complexity", oldVal, newVal)
			item.Complexity = input.Complexity
		}
	}

	if input.Visibility != nil && *input.Visibility != item.Visibility {
		if !isValidVisibility(*input.Visibility) {
			return nil, fmt.Errorf("invalid visibility %q: %w", *input.Visibility, model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, info, "visibility", item.Visibility, *input.Visibility)
		item.Visibility = *input.Visibility
	}

	if input.ClearDueDate {
		if item.DueDate != nil {
			s.recordEvent(ctx, item.ID, info, "due_date_cleared", strPtr("due_date"), strPtr(item.DueDate.Format(time.DateOnly)), nil)
			watcherChanges = append(watcherChanges, fieldChange{"due_date", item.DueDate.Format(time.DateOnly), ""})
			item.DueDate = nil
		}
	} else if input.DueDate != nil {
		oldDueDate := ""
		if item.DueDate != nil {
			oldDueDate = item.DueDate.Format(time.DateOnly)
		}
		newDueDate := input.DueDate.Format(time.DateOnly)
		if oldDueDate != newDueDate {
			s.recordEvent(ctx, item.ID, info, "due_date_set", strPtr("due_date"), strPtr(oldDueDate), strPtr(newDueDate))
			watcherChanges = append(watcherChanges, fieldChange{"due_date", oldDueDate, newDueDate})
			item.DueDate = input.DueDate
		}
	}

	if input.ClearParent {
		item.ParentID = nil
	} else if input.ParentID != nil {
		item.ParentID = input.ParentID
	}

	if input.ClearQueue {
		item.QueueID = nil
	} else if input.QueueID != nil {
		// Validate queue belongs to this project
		q, err := s.queues.GetByID(ctx, *input.QueueID)
		if err != nil {
			return nil, fmt.Errorf("queue not found: %w", model.ErrValidation)
		}
		if q.ProjectID != project.ID {
			return nil, fmt.Errorf("queue does not belong to this project: %w", model.ErrValidation)
		}
		item.QueueID = input.QueueID
	}

	if input.ClearMilestone {
		if item.MilestoneID != nil {
			watcherChanges = append(watcherChanges, fieldChange{"milestone", item.MilestoneID.String(), ""})
		}
		item.MilestoneID = nil
	} else if input.MilestoneID != nil {
		// Validate milestone belongs to this project
		m, err := s.milestones.GetByID(ctx, *input.MilestoneID)
		if err != nil {
			return nil, fmt.Errorf("milestone not found: %w", model.ErrValidation)
		}
		if m.ProjectID != project.ID {
			return nil, fmt.Errorf("milestone does not belong to this project: %w", model.ErrValidation)
		}
		oldMilestone := ""
		if item.MilestoneID != nil {
			oldMilestone = item.MilestoneID.String()
		}
		if oldMilestone != input.MilestoneID.String() {
			watcherChanges = append(watcherChanges, fieldChange{"milestone", oldMilestone, input.MilestoneID.String()})
		}
		item.MilestoneID = input.MilestoneID
	}

	if input.CustomFields != nil {
		item.CustomFields = input.CustomFields
	}

	if input.ClearEstimate {
		if item.EstimatedSeconds != nil {
			s.recordEvent(ctx, item.ID, info, "estimate_cleared", strPtr("estimated_seconds"), strPtr(strconv.Itoa(*item.EstimatedSeconds)), nil)
			item.EstimatedSeconds = nil
		}
	} else if input.EstimatedSeconds != nil {
		if *input.EstimatedSeconds <= 0 {
			return nil, fmt.Errorf("estimated_seconds must be positive: %w", model.ErrValidation)
		}
		oldVal := ""
		if item.EstimatedSeconds != nil {
			oldVal = strconv.Itoa(*item.EstimatedSeconds)
		}
		newVal := strconv.Itoa(*input.EstimatedSeconds)
		if oldVal != newVal {
			s.recordEvent(ctx, item.ID, info, "estimate_set", strPtr("estimated_seconds"), strPtr(oldVal), strPtr(newVal))
			item.EstimatedSeconds = input.EstimatedSeconds
		}
	}

	if err := s.items.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("updating work item: %w", err)
	}

	// Publish watcher notifications for all field changes (best-effort, after DB update)
	for _, ch := range watcherChanges {
		s.publishWatcherNotification(ctx, project.Key, project.ID, item, info.UserID, "field_change", ch.field, ch.oldVal, ch.newVal, "")
	}

	// Publish status change notification for assignee (best-effort)
	if statusNew != "" && statusCategory != "" {
		s.publishStatusChange(ctx, project.Key, project.ID, item, info.UserID, statusOld, statusNew, statusCategory)
	}

	// Publish embed.index event for semantic search (reindex on update)
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeWorkItem, item.ID, &project.ID)

	// Re-fetch to get updated timestamp
	return s.items.GetByProjectAndNumber(ctx, project.ID, item.ItemNumber)
}

// Delete soft-deletes a work item.
func (s *WorkItemService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return err
	}

	if err := s.items.Delete(ctx, item.ID); err != nil {
		return fmt.Errorf("deleting work item: %w", err)
	}

	// Publish embed.delete event for semantic search
	publishEmbedDelete(ctx, s.publisher, s.embedCache, model.EntityTypeWorkItem, item.ID)

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Int("item_number", itemNumber).
		Str("user_id", info.UserID.String()).
		Msg("work item deleted")

	return nil
}

// --- Comment methods ---

// CreateComment creates a new comment on a work item.
func (s *WorkItemService) CreateComment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, input CreateCommentInput) (*model.Comment, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.Body) == "" {
		return nil, fmt.Errorf("body is required: %w", model.ErrValidation)
	}

	if input.Visibility == "" {
		input.Visibility = model.VisibilityInternal
	}
	if input.Visibility != model.VisibilityInternal && input.Visibility != model.VisibilityPublic {
		return nil, fmt.Errorf("invalid comment visibility %q: %w", input.Visibility, model.ErrValidation)
	}

	comment := &model.Comment{
		ID:         uuid.Must(uuid.NewV7()),
		WorkItemID: item.ID,
		AuthorID:   &info.UserID,
		Body:       input.Body,
		Visibility: input.Visibility,
	}

	if err := s.comments.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("creating comment: %w", err)
	}

	// Record "comment_added" event
	s.recordEventWithMetadata(ctx, item.ID, info, "comment_added", input.Visibility, map[string]interface{}{
		"comment_id": comment.ID.String(),
		"preview":    input.Body,
	})

	// Publish watcher notification for the new comment
	preview := input.Body
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	s.publishWatcherNotification(ctx, projectKey, project.ID, item, info.UserID, "comment_added", "", "", "", preview)

	// Publish comment-on-assigned notification for assignee
	s.publishCommentOnAssigned(ctx, projectKey, project.ID, item, info.UserID, preview)

	// Publish embed.index event for the new comment
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeComment, comment.ID, &project.ID)

	// Re-fetch to get DB-assigned timestamps
	return s.comments.GetByID(ctx, comment.ID)
}

// ListComments returns comments for a work item.
func (s *WorkItemService) ListComments(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, visibility string) ([]model.Comment, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	return s.comments.ListByWorkItem(ctx, item.ID, visibility)
}

// UpdateComment updates a comment's body. Only the author can edit their own comments.
func (s *WorkItemService) UpdateComment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, commentID uuid.UUID, body string) (*model.Comment, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	comment, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// Only the author can edit their own comments
	if comment.AuthorID == nil || *comment.AuthorID != info.UserID {
		return nil, model.ErrForbidden
	}

	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("body is required: %w", model.ErrValidation)
	}

	oldBody := comment.Body
	comment.Body = body
	if err := s.comments.Update(ctx, comment); err != nil {
		return nil, fmt.Errorf("updating comment: %w", err)
	}

	// Record "comment_updated" event
	s.recordEventWithMetadata(ctx, item.ID, info, "comment_updated", comment.Visibility, map[string]interface{}{
		"comment_id":  commentID.String(),
		"old_preview": oldBody,
		"preview":     body,
	})

	// Reindex comment embedding
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeComment, commentID, &project.ID)

	return s.comments.GetByID(ctx, commentID)
}

// DeleteComment soft-deletes a comment. Only the author or project owner/admin can delete.
func (s *WorkItemService) DeleteComment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, commentID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return err
	}

	comment, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	isAuthor := comment.AuthorID != nil && *comment.AuthorID == info.UserID
	if !isAuthor {
		if err := s.requireRole(ctx, info, project.ID,
			model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return model.ErrForbidden
		}
	}

	if err := s.comments.Delete(ctx, commentID); err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "comment_deleted", model.VisibilityInternal, map[string]interface{}{
		"comment_id": commentID.String(),
	})

	// Delete comment embedding
	publishEmbedDelete(ctx, s.publisher, s.embedCache, model.EntityTypeComment, commentID)

	return nil
}

// --- Relation methods ---

// CreateRelation creates a relationship between two work items.
func (s *WorkItemService) CreateRelation(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, input CreateRelationInput) (*RelationWithDisplay, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	sourceItem, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	// Validate relation type
	if !isValidRelationType(input.RelationType) {
		return nil, fmt.Errorf("invalid relation type %q: %w", input.RelationType, model.ErrValidation)
	}

	// Parse target display ID (e.g. "INFRA-38")
	targetKey, targetNumber, err := parseDisplayID(input.TargetDisplayID)
	if err != nil {
		return nil, fmt.Errorf("invalid target_display_id %q: %w", input.TargetDisplayID, model.ErrValidation)
	}

	targetProject, err := s.projects.GetByKey(ctx, targetKey)
	if err != nil {
		return nil, fmt.Errorf("target project not found: %w", model.ErrValidation)
	}

	targetItem, err := s.items.GetByProjectAndNumber(ctx, targetProject.ID, targetNumber)
	if err != nil {
		return nil, fmt.Errorf("target work item not found: %w", model.ErrValidation)
	}

	// Prevent self-relation
	if sourceItem.ID == targetItem.ID {
		return nil, fmt.Errorf("cannot create relation to self: %w", model.ErrValidation)
	}

	relation := &model.WorkItemRelation{
		ID:           uuid.Must(uuid.NewV7()),
		SourceID:     sourceItem.ID,
		TargetID:     targetItem.ID,
		RelationType: input.RelationType,
		CreatedBy:    info.UserID,
	}

	if err := s.relations.Create(ctx, relation); err != nil {
		return nil, fmt.Errorf("creating relation: %w", err)
	}

	sourceDisplayID := fmt.Sprintf("%s-%d", projectKey, itemNumber)
	targetDisplayID := fmt.Sprintf("%s-%d", targetKey, targetItem.ItemNumber)

	s.recordEventWithMetadata(ctx, sourceItem.ID, info, "relation_added", model.VisibilityInternal, map[string]interface{}{
		"relation_id":   relation.ID.String(),
		"relation_type": input.RelationType,
		"target":        targetDisplayID,
	})

	// Re-fetch to get DB-assigned created_at
	fetched, err := s.relations.GetByID(ctx, relation.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching created relation: %w", err)
	}

	return &RelationWithDisplay{
		WorkItemRelation: *fetched,
		SourceDisplayID:  sourceDisplayID,
		SourceTitle:      sourceItem.Title,
		TargetDisplayID:  targetDisplayID,
		TargetTitle:      targetItem.Title,
	}, nil
}

// ListRelations returns all relations for a work item, enriched with display IDs.
func (s *WorkItemService) ListRelations(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) ([]RelationWithDisplay, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	relations, err := s.relations.ListByWorkItemWithDetails(ctx, item.ID)
	if err != nil {
		return nil, fmt.Errorf("listing relations: %w", err)
	}

	result := make([]RelationWithDisplay, len(relations))
	for i, rel := range relations {
		result[i] = RelationWithDisplay{
			WorkItemRelation: rel.WorkItemRelation,
			SourceDisplayID:  fmt.Sprintf("%s-%d", rel.SourceProjectKey, rel.SourceItemNumber),
			SourceTitle:      rel.SourceTitle,
			TargetDisplayID:  fmt.Sprintf("%s-%d", rel.TargetProjectKey, rel.TargetItemNumber),
			TargetTitle:      rel.TargetTitle,
		}
	}

	return result, nil
}

// DeleteRelation removes a work item relation.
func (s *WorkItemService) DeleteRelation(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, relationID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return err
	}

	relation, err := s.relations.GetByID(ctx, relationID)
	if err != nil {
		return err
	}

	// Verify relation belongs to this work item
	if relation.SourceID != item.ID && relation.TargetID != item.ID {
		return model.ErrNotFound
	}

	if err := s.relations.Delete(ctx, relationID); err != nil {
		return fmt.Errorf("deleting relation: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "relation_removed", model.VisibilityInternal, map[string]interface{}{
		"relation_id":   relationID.String(),
		"relation_type": relation.RelationType,
	})

	return nil
}

// --- Attachment methods ---

// UploadAttachment uploads a file and creates an attachment record.
func (s *WorkItemService) UploadAttachment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, input CreateAttachmentInput) (*model.Attachment, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	safeFilename := filepath.Base(strings.TrimSpace(input.Filename))
	if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
		return nil, fmt.Errorf("filename is required: %w", model.ErrValidation)
	}
	if input.Size <= 0 {
		return nil, fmt.Errorf("file is empty: %w", model.ErrValidation)
	}
	if input.Size > s.maxUploadSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes: %w", s.maxUploadSize, model.ErrValidation)
	}

	attachmentID := uuid.Must(uuid.NewV7())
	storageKey := fmt.Sprintf("%s/%s/%s/%s", project.ID, item.ID, attachmentID, safeFilename)

	if _, err := s.fileStorage.Put(ctx, storageKey, input.Reader, input.Size, input.ContentType); err != nil {
		return nil, fmt.Errorf("uploading file to storage: %w", err)
	}

	attachment := &model.Attachment{
		ID:          attachmentID,
		WorkItemID:  item.ID,
		UploaderID:  info.UserID,
		Filename:    safeFilename,
		ContentType: input.ContentType,
		SizeBytes:   input.Size,
		StorageKey:  storageKey,
		Comment:     input.Comment,
	}

	if err := s.attachments.Create(ctx, attachment); err != nil {
		// Best-effort cleanup of storage on DB failure
		_ = s.fileStorage.Delete(ctx, storageKey)
		return nil, fmt.Errorf("creating attachment record: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "attachment_added", model.VisibilityInternal, map[string]interface{}{
		"attachment_id": attachmentID.String(),
		"filename":      input.Filename,
		"size_bytes":    input.Size,
	})

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Int("item_number", itemNumber).
		Str("filename", input.Filename).
		Int64("size_bytes", input.Size).
		Msg("attachment uploaded")

	// Publish embed.index event for the new attachment
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeAttachment, attachmentID, &project.ID)

	return s.attachments.GetByID(ctx, attachmentID)
}

// ListAttachments returns all attachments for a work item.
func (s *WorkItemService) ListAttachments(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) ([]model.Attachment, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	return s.attachments.ListByWorkItem(ctx, item.ID)
}

// GetAttachmentFile returns the attachment metadata and a reader for the file content.
func (s *WorkItemService) GetAttachmentFile(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, attachmentID uuid.UUID) (*model.Attachment, io.ReadCloser, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, nil, err
	}

	attachment, err := s.attachments.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, err
	}

	if attachment.WorkItemID != item.ID {
		return nil, nil, model.ErrNotFound
	}

	reader, _, err := s.fileStorage.Get(ctx, attachment.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("retrieving file from storage: %w", err)
	}

	return attachment, reader, nil
}

// DeleteAttachment soft-deletes an attachment. Only the uploader or project owner/admin can delete.
func (s *WorkItemService) DeleteAttachment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, attachmentID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return err
	}

	attachment, err := s.attachments.GetByID(ctx, attachmentID)
	if err != nil {
		return err
	}

	if attachment.WorkItemID != item.ID {
		return model.ErrNotFound
	}

	isUploader := attachment.UploaderID == info.UserID
	if !isUploader {
		if err := s.requireRole(ctx, info, project.ID,
			model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return model.ErrForbidden
		}
	}

	if err := s.attachments.Delete(ctx, attachmentID); err != nil {
		return fmt.Errorf("deleting attachment: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "attachment_removed", model.VisibilityInternal, map[string]interface{}{
		"attachment_id": attachmentID.String(),
		"filename":      attachment.Filename,
	})

	// Delete attachment embedding
	publishEmbedDelete(ctx, s.publisher, s.embedCache, model.EntityTypeAttachment, attachmentID)

	return nil
}

// UpdateAttachmentComment updates the comment on an attachment. Only the uploader or project owner/admin can update.
func (s *WorkItemService) UpdateAttachmentComment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, attachmentID uuid.UUID, comment string) (*model.Attachment, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	attachment, err := s.attachments.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, err
	}

	if attachment.WorkItemID != item.ID {
		return nil, model.ErrNotFound
	}

	isUploader := attachment.UploaderID == info.UserID
	if !isUploader {
		if err := s.requireRole(ctx, info, project.ID,
			model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return nil, model.ErrForbidden
		}
	}

	if err := s.attachments.UpdateComment(ctx, attachmentID, comment); err != nil {
		return nil, fmt.Errorf("updating attachment comment: %w", err)
	}

	attachment.Comment = comment
	return attachment, nil
}

// --- Event methods ---

// ListEvents returns events for a work item with optional visibility filter and actor info.
func (s *WorkItemService) ListEvents(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, visibility string) ([]model.WorkItemEventWithActor, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	return s.events.ListByWorkItemFiltered(ctx, item.ID, visibility)
}

// --- Authorization helpers ---

func (s *WorkItemService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	if info.IsSystemKey() {
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

func (s *WorkItemService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	if info.IsSystemKey() {
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

// eventActorFromAuthInfo returns actor fields for a WorkItemEvent based on the auth info.
// System keys get ActorType "system_key" with key metadata; regular users get ActorType "user" with their UserID.
func eventActorFromAuthInfo(info *model.AuthInfo) (actorID *uuid.UUID, actorType string, metadata map[string]interface{}) {
	if info.IsSystemKey() {
		return nil, model.ActorTypeSystemKey, map[string]interface{}{
			"key_id":   info.KeyID.String(),
			"key_name": info.KeyName,
		}
	}
	uid := info.UserID
	return &uid, model.ActorTypeUser, nil
}

// --- Time entry methods ---

// LogTime creates a new time entry on a work item.
func (s *WorkItemService) LogTime(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, input CreateTimeEntryInput) (*model.TimeEntry, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	if input.DurationSeconds <= 0 {
		return nil, fmt.Errorf("duration_seconds must be positive: %w", model.ErrValidation)
	}

	if input.StartedAt.IsZero() {
		return nil, fmt.Errorf("started_at is required: %w", model.ErrValidation)
	}

	entry := &model.TimeEntry{
		ID:              uuid.Must(uuid.NewV7()),
		WorkItemID:      item.ID,
		UserID:          info.UserID,
		StartedAt:       input.StartedAt,
		DurationSeconds: input.DurationSeconds,
	}

	if strings.TrimSpace(input.Description) != "" {
		desc := input.Description
		entry.Description = &desc
	}

	if err := s.timeEntries.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("creating time entry: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "time_logged", model.VisibilityInternal, map[string]interface{}{
		"time_entry_id":    entry.ID.String(),
		"duration_seconds": input.DurationSeconds,
	})

	return s.timeEntries.GetByID(ctx, entry.ID)
}

// ListTimeEntries returns all time entries for a work item.
func (s *WorkItemService) ListTimeEntries(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) ([]model.TimeEntry, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	return s.timeEntries.ListByWorkItem(ctx, item.ID)
}

// GetTimeEntrySummary returns the total logged seconds for a work item.
func (s *WorkItemService) GetTimeEntrySummary(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) (int, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return 0, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return 0, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return 0, err
	}

	return s.timeEntries.SumByWorkItem(ctx, item.ID)
}

// UpdateTimeEntry updates a time entry. Only the author or project owner/admin can edit.
func (s *WorkItemService) UpdateTimeEntry(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, entryID uuid.UUID, input UpdateTimeEntryInput) (*model.TimeEntry, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	entry, err := s.timeEntries.GetByID(ctx, entryID)
	if err != nil {
		return nil, err
	}

	if entry.WorkItemID != item.ID {
		return nil, model.ErrNotFound
	}

	// Author can edit, otherwise need owner/admin
	if entry.UserID != info.UserID {
		if err := s.requireRole(ctx, info, project.ID,
			model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return nil, model.ErrForbidden
		}
	}

	if input.StartedAt != nil {
		entry.StartedAt = *input.StartedAt
	}
	if input.DurationSeconds != nil {
		if *input.DurationSeconds <= 0 {
			return nil, fmt.Errorf("duration_seconds must be positive: %w", model.ErrValidation)
		}
		entry.DurationSeconds = *input.DurationSeconds
	}
	if input.ClearDescription {
		entry.Description = nil
	} else if input.Description != nil {
		entry.Description = input.Description
	}

	if err := s.timeEntries.Update(ctx, entry); err != nil {
		return nil, fmt.Errorf("updating time entry: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "time_entry_updated", model.VisibilityInternal, map[string]interface{}{
		"time_entry_id": entryID.String(),
	})

	return s.timeEntries.GetByID(ctx, entryID)
}

// DeleteTimeEntry soft-deletes a time entry. Only the author or project owner/admin can delete.
func (s *WorkItemService) DeleteTimeEntry(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, entryID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return err
	}

	entry, err := s.timeEntries.GetByID(ctx, entryID)
	if err != nil {
		return err
	}

	if entry.WorkItemID != item.ID {
		return model.ErrNotFound
	}

	// Author can delete, otherwise need owner/admin
	if entry.UserID != info.UserID {
		if err := s.requireRole(ctx, info, project.ID,
			model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return model.ErrForbidden
		}
	}

	if err := s.timeEntries.Delete(ctx, entryID); err != nil {
		return fmt.Errorf("deleting time entry: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "time_entry_deleted", model.VisibilityInternal, map[string]interface{}{
		"time_entry_id": entryID.String(),
	})

	return nil
}

// --- Event recording helpers ---

func (s *WorkItemService) recordEvent(ctx context.Context, workItemID uuid.UUID, info *model.AuthInfo, eventType string, fieldName, oldValue, newValue *string) {
	actorID, actorType, actorMeta := eventActorFromAuthInfo(info)
	md := map[string]interface{}{}
	for k, v := range actorMeta {
		md[k] = v
	}
	event := &model.WorkItemEvent{
		ID:         uuid.Must(uuid.NewV7()),
		WorkItemID: workItemID,
		ActorID:    actorID,
		ActorType:  actorType,
		EventType:  eventType,
		FieldName:  fieldName,
		OldValue:   oldValue,
		NewValue:   newValue,
		Metadata:   md,
		Visibility: model.VisibilityInternal,
	}
	if err := s.events.Create(ctx, event); err != nil {
		log.Ctx(ctx).Error().Err(err).
			Str("event_type", eventType).
			Str("work_item_id", workItemID.String()).
			Msg("failed to record work item event")
	}
}

func (s *WorkItemService) recordEventWithMetadata(ctx context.Context, workItemID uuid.UUID, info *model.AuthInfo, eventType, visibility string, metadata map[string]interface{}) {
	actorID, actorType, actorMeta := eventActorFromAuthInfo(info)
	for k, v := range actorMeta {
		metadata[k] = v
	}
	event := &model.WorkItemEvent{
		ID:         uuid.Must(uuid.NewV7()),
		WorkItemID: workItemID,
		ActorID:    actorID,
		ActorType:  actorType,
		EventType:  eventType,
		Metadata:   metadata,
		Visibility: visibility,
	}
	if err := s.events.Create(ctx, event); err != nil {
		log.Ctx(ctx).Error().Err(err).
			Str("event_type", eventType).
			Str("work_item_id", workItemID.String()).
			Msg("failed to record work item event")
	}
}

func (s *WorkItemService) recordFieldChange(ctx context.Context, workItemID uuid.UUID, info *model.AuthInfo, fieldName, oldValue, newValue string) {
	eventType := fieldName + "_updated"
	switch fieldName {
	case "status":
		eventType = "status_changed"
	case "priority":
		eventType = "priority_changed"
	}
	s.recordEvent(ctx, workItemID, info, eventType, &fieldName, &oldValue, &newValue)
}

// publishWatcherNotification publishes a watcher notification event (best-effort).
func (s *WorkItemService) publishWatcherNotification(_ context.Context, projectKey string, projectID uuid.UUID, item *model.WorkItem, actorID uuid.UUID, eventType, fieldName, oldValue, newValue, summary string) {
	if s.publisher == nil {
		return
	}
	evt := model.WatcherEvent{
		WorkItemID: item.ID,
		ProjectKey: projectKey,
		ProjectID:  projectID,
		ItemNumber: item.ItemNumber,
		Title:      item.Title,
		ActorID:    actorID,
		EventType:  eventType,
		FieldName:  fieldName,
		OldValue:   oldValue,
		NewValue:   newValue,
		Summary:    summary,
	}
	if err := s.publisher.Publish("notification.watcher", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish watcher notification")
	}
}

// publishAssignment publishes an assignment notification event (best-effort).
func (s *WorkItemService) publishAssignment(_ context.Context, projectKey string, item *model.WorkItem, assignerID uuid.UUID) {
	if s.publisher == nil || item.AssigneeID == nil {
		return
	}
	// Don't notify when users assign to themselves
	if *item.AssigneeID == assignerID {
		return
	}
	evt := model.AssignmentEvent{
		WorkItemID: item.ID,
		ProjectKey: projectKey,
		ItemNumber: item.ItemNumber,
		Title:      item.Title,
		AssigneeID: *item.AssigneeID,
		AssignerID: assignerID,
		ProjectID:  item.ProjectID,
	}
	if err := s.publisher.Publish("notification.assignment", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish assignment notification")
	}
}

// publishNewItem publishes a new-item notification event (best-effort).
func (s *WorkItemService) publishNewItem(_ context.Context, projectKey string, projectID uuid.UUID, item *model.WorkItem, creatorID uuid.UUID) {
	if s.publisher == nil {
		return
	}
	evt := model.NewItemEvent{
		WorkItemID: item.ID,
		ProjectKey: projectKey,
		ProjectID:  projectID,
		ItemNumber: item.ItemNumber,
		Title:      item.Title,
		CreatorID:  creatorID,
		Type:       item.Type,
	}
	if err := s.publisher.Publish("notification.new_item", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish new item notification")
	}
}

// publishCommentOnAssigned publishes a comment-on-assigned notification event (best-effort).
func (s *WorkItemService) publishCommentOnAssigned(_ context.Context, projectKey string, projectID uuid.UUID, item *model.WorkItem, commenterID uuid.UUID, preview string) {
	if s.publisher == nil || item.AssigneeID == nil {
		return
	}
	// Don't notify when assignees comment on their own items
	if *item.AssigneeID == commenterID {
		return
	}
	evt := model.CommentOnAssignedEvent{
		WorkItemID:  item.ID,
		ProjectKey:  projectKey,
		ProjectID:   projectID,
		ItemNumber:  item.ItemNumber,
		Title:       item.Title,
		AssigneeID:  *item.AssigneeID,
		CommenterID: commenterID,
		Preview:     preview,
	}
	if err := s.publisher.Publish("notification.comment_assigned", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish comment on assigned notification")
	}
}

// publishStatusChange publishes a status-change notification event (best-effort).
func (s *WorkItemService) publishStatusChange(_ context.Context, projectKey string, projectID uuid.UUID, item *model.WorkItem, actorID uuid.UUID, oldStatus, newStatus, category string) {
	if s.publisher == nil || item.AssigneeID == nil {
		return
	}
	// Don't notify when assignees change their own item's status
	if *item.AssigneeID == actorID {
		return
	}
	// Only notify for in_progress, done, or cancelled categories
	if category != model.CategoryInProgress && category != model.CategoryDone && category != model.CategoryCancelled {
		return
	}
	evt := model.StatusChangeEvent{
		WorkItemID: item.ID,
		ProjectKey: projectKey,
		ProjectID:  projectID,
		ItemNumber: item.ItemNumber,
		Title:      item.Title,
		AssigneeID: *item.AssigneeID,
		ActorID:    actorID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		Category:   category,
	}
	if err := s.publisher.Publish("notification.status_change", evt); err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Msg("failed to publish status change notification")
	}
}

// resolveWorkflowID returns the workflow ID for a given project and item type.
// It checks the type-workflow mapping first, then falls back to the project's default.
func (s *WorkItemService) resolveWorkflowID(ctx context.Context, projectID uuid.UUID, itemType string, fallback *uuid.UUID) (uuid.UUID, error) {
	mapping, err := s.typeWorkflows.GetByProjectAndType(ctx, projectID, itemType)
	if err == nil {
		return mapping.WorkflowID, nil
	}
	if fallback != nil {
		return *fallback, nil
	}
	return uuid.Nil, model.ErrNotFound
}

// accumulateElapsedOnLeave computes the elapsed time for the current status
// using business-hours-aware calculation when applicable, then persists it.
func (s *WorkItemService) accumulateElapsedOnLeave(ctx context.Context, project *model.Project, item *model.WorkItem, now time.Time) error {
	// Try to resolve the SLA target to check calendar mode
	if s.slaService != nil {
		wfID, err := s.resolveWorkflowID(ctx, project.ID, item.Type, project.DefaultWorkflowID)
		if err == nil {
			target, err := s.sla.GetTarget(ctx, project.ID, item.Type, wfID, item.Status)
			if err == nil && target.CalendarMode == model.CalendarModeBusinessHours && project.BusinessHours != nil {
				// Compute business-aware elapsed seconds
				elapsed, err := s.sla.GetElapsed(ctx, item.ID, item.Status)
				if err == nil && elapsed.LastEnteredAt != nil {
					additionalSeconds := CalculateBusinessSeconds(*elapsed.LastEnteredAt, now, *project.BusinessHours)
					return s.sla.UpdateElapsedOnLeaveWithSeconds(ctx, item.ID, item.Status, additionalSeconds)
				}
			}
		}
	}

	// Fallback: wall-clock time (no SLA target, or 24x7 mode)
	return s.sla.UpdateElapsedOnLeave(ctx, item.ID, item.Status, now)
}

// --- Validation helpers ---

func isValidWorkItemType(t string) bool {
	switch t {
	case model.WorkItemTypeTask, model.WorkItemTypeTicket, model.WorkItemTypeBug,
		model.WorkItemTypeFeedback, model.WorkItemTypeEpic:
		return true
	}
	return false
}

func isValidPriority(p string) bool {
	switch p {
	case model.PriorityCritical, model.PriorityHigh, model.PriorityMedium, model.PriorityLow:
		return true
	}
	return false
}

func isValidVisibility(v string) bool {
	switch v {
	case model.VisibilityInternal, model.VisibilityPortal, model.VisibilityPublic:
		return true
	}
	return false
}

func isValidSortField(s string) bool {
	switch s {
	case "created_at", "updated_at", "priority", "due_date", "item_number", "type", "title", "status", "sla_target_at":
		return true
	}
	return false
}

func isValidRelationType(t string) bool {
	switch t {
	case model.RelationBlocks, model.RelationBlockedBy, model.RelationRelatesTo,
		model.RelationDuplicates, model.RelationCausedBy, model.RelationParentOf, model.RelationChildOf:
		return true
	}
	return false
}

// --- Watcher methods ---

// AddWatcher adds a user as a watcher on a work item.
// Members/admins/owners can add any project member. Viewers can only add themselves.
func (s *WorkItemService) AddWatcher(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, targetUserID uuid.UUID) (*model.WorkItemWatcher, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	// Viewers can only add themselves
	if info.GlobalRole != model.RoleAdmin {
		member, err := s.members.GetByProjectAndUser(ctx, project.ID, info.UserID)
		if err != nil {
			return nil, fmt.Errorf("checking membership: %w", err)
		}
		if member.Role == model.ProjectRoleViewer && targetUserID != info.UserID {
			return nil, model.ErrForbidden
		}
	}

	// Verify target user is a project member
	if _, err := s.members.GetByProjectAndUser(ctx, project.ID, targetUserID); err != nil {
		if err == model.ErrNotFound {
			return nil, fmt.Errorf("target user is not a project member: %w", model.ErrValidation)
		}
		return nil, fmt.Errorf("checking target membership: %w", err)
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	watcher := &model.WorkItemWatcher{
		ID:         uuid.Must(uuid.NewV7()),
		WorkItemID: item.ID,
		UserID:     targetUserID,
		AddedBy:    info.UserID,
	}

	if err := s.watchers.Create(ctx, watcher); err != nil {
		return nil, fmt.Errorf("adding watcher: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "watcher_added", model.VisibilityInternal, map[string]interface{}{
		"watcher_user_id": targetUserID.String(),
	})

	return watcher, nil
}

// RemoveWatcher removes a user from a work item's watchers.
// Owners/admins can remove anyone. Members/viewers can only remove themselves.
func (s *WorkItemService) RemoveWatcher(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, targetUserID uuid.UUID) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	// Only owners/admins can remove other users
	if targetUserID != info.UserID {
		if err := s.requireRole(ctx, info, project.ID,
			model.ProjectRoleOwner, model.ProjectRoleAdmin); err != nil {
			return model.ErrForbidden
		}
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return err
	}

	if err := s.watchers.Delete(ctx, item.ID, targetUserID); err != nil {
		return fmt.Errorf("removing watcher: %w", err)
	}

	s.recordEventWithMetadata(ctx, item.ID, info, "watcher_removed", model.VisibilityInternal, map[string]interface{}{
		"watcher_user_id": targetUserID.String(),
	})

	return nil
}

// WatcherListResponse holds the response for listing watchers, which varies by role.
type WatcherListResponse struct {
	Watchers   []model.WorkItemWatcherWithUser `json:"watchers,omitempty"`
	Me         *model.WorkItemWatcher          `json:"me,omitempty"`
	OtherCount int                             `json:"other_count"`
	IsViewer   bool                            `json:"-"`
}

// ListWatchers returns watchers for a work item.
// Members/admins/owners see the full list. Viewers see only themselves + count.
func (s *WorkItemService) ListWatchers(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) (*WatcherListResponse, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	// Determine if user is a viewer
	isViewer := false
	if info.GlobalRole != model.RoleAdmin {
		member, err := s.members.GetByProjectAndUser(ctx, project.ID, info.UserID)
		if err != nil {
			return nil, fmt.Errorf("checking membership: %w", err)
		}
		isViewer = member.Role == model.ProjectRoleViewer
	}

	if isViewer {
		// Viewers: return their own entry + total count
		isWatching, err := s.watchers.IsWatching(ctx, item.ID, info.UserID)
		if err != nil {
			return nil, fmt.Errorf("checking watch status: %w", err)
		}

		count, err := s.watchers.CountByWorkItem(ctx, item.ID)
		if err != nil {
			return nil, fmt.Errorf("counting watchers: %w", err)
		}

		resp := &WatcherListResponse{IsViewer: true}
		if isWatching {
			resp.Me = &model.WorkItemWatcher{
				WorkItemID: item.ID,
				UserID:     info.UserID,
			}
			resp.OtherCount = count - 1
		} else {
			resp.OtherCount = count
		}
		return resp, nil
	}

	// Non-viewers: return full list
	watchers, err := s.watchers.ListByWorkItem(ctx, item.ID)
	if err != nil {
		return nil, fmt.Errorf("listing watchers: %w", err)
	}

	return &WatcherListResponse{
		Watchers:   watchers,
		OtherCount: len(watchers),
	}, nil
}

// ToggleWatch toggles the current user's watch status on a work item.
func (s *WorkItemService) ToggleWatch(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) (bool, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return false, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return false, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return false, err
	}

	watching, err := s.watchers.IsWatching(ctx, item.ID, info.UserID)
	if err != nil {
		return false, fmt.Errorf("checking watch status: %w", err)
	}

	if watching {
		if err := s.watchers.Delete(ctx, item.ID, info.UserID); err != nil {
			return false, fmt.Errorf("removing watch: %w", err)
		}
		s.recordEventWithMetadata(ctx, item.ID, info, "watcher_removed", model.VisibilityInternal, map[string]interface{}{
			"watcher_user_id": info.UserID.String(),
		})
		return false, nil
	}

	watcher := &model.WorkItemWatcher{
		ID:         uuid.Must(uuid.NewV7()),
		WorkItemID: item.ID,
		UserID:     info.UserID,
		AddedBy:    info.UserID,
	}
	if err := s.watchers.Create(ctx, watcher); err != nil {
		return false, fmt.Errorf("adding watch: %w", err)
	}
	s.recordEventWithMetadata(ctx, item.ID, info, "watcher_added", model.VisibilityInternal, map[string]interface{}{
		"watcher_user_id": info.UserID.String(),
	})
	return true, nil
}

// ListWatchedItemIDs returns the IDs of work items the current user is watching, optionally scoped to a project.
func (s *WorkItemService) ListWatchedItemIDs(ctx context.Context, info *model.AuthInfo, projectKey string) ([]uuid.UUID, error) {
	var projectID *uuid.UUID
	if projectKey != "" {
		project, err := s.projects.GetByKey(ctx, projectKey)
		if err != nil {
			return nil, err
		}
		if err := s.requireMembership(ctx, info, project.ID); err != nil {
			return nil, err
		}
		projectID = &project.ID
	}

	return s.watchers.ListWatchedItemIDs(ctx, info.UserID, projectID)
}

// ListWatchedItems returns work items that the current user is watching, with standard filters and pagination.
// When projectKeys is empty, returns watched items across all projects the user is a member of.
// When projectKeys has one entry, scopes to that project. When multiple, scopes to those projects.
func (s *WorkItemService) ListWatchedItems(ctx context.Context, info *model.AuthInfo, projectKeys []string, filter *model.WorkItemFilter) (*model.WorkItemList, error) {
	if len(projectKeys) == 1 {
		// Single project: use original optimized path
		project, err := s.projects.GetByKey(ctx, projectKeys[0])
		if err != nil {
			return nil, err
		}
		if err := s.requireMembership(ctx, info, project.ID); err != nil {
			return nil, err
		}
		ids, err := s.watchers.ListWatchedItemIDs(ctx, info.UserID, &project.ID)
		if err != nil {
			return nil, fmt.Errorf("listing watched item IDs: %w", err)
		}
		if len(ids) == 0 {
			return &model.WorkItemList{Items: []model.WorkItem{}}, nil
		}
		filter.ItemIDs = ids
		return s.items.List(ctx, project.ID, filter)
	}

	// Multi-project or all-projects path
	var projectIDs []uuid.UUID
	if len(projectKeys) > 0 {
		for _, key := range projectKeys {
			project, err := s.projects.GetByKey(ctx, key)
			if err != nil {
				return nil, err
			}
			if err := s.requireMembership(ctx, info, project.ID); err != nil {
				return nil, err
			}
			projectIDs = append(projectIDs, project.ID)
		}
	}

	// Get watched item IDs across all or selected projects
	var allIDs []uuid.UUID
	if len(projectIDs) == 0 {
		// All projects
		ids, err := s.watchers.ListWatchedItemIDs(ctx, info.UserID, nil)
		if err != nil {
			return nil, fmt.Errorf("listing watched item IDs: %w", err)
		}
		allIDs = ids
	} else {
		for _, pid := range projectIDs {
			pid := pid
			ids, err := s.watchers.ListWatchedItemIDs(ctx, info.UserID, &pid)
			if err != nil {
				return nil, fmt.Errorf("listing watched item IDs: %w", err)
			}
			allIDs = append(allIDs, ids...)
		}
	}

	if len(allIDs) == 0 {
		return &model.WorkItemList{Items: []model.WorkItem{}}, nil
	}

	filter.ItemIDs = allIDs
	filter.SkipProjectFilter = true
	return s.items.List(ctx, uuid.Nil, filter)
}

// ResolveProjectNamespaces returns namespace info for the given project keys.
func (s *WorkItemService) ResolveProjectNamespaces(ctx context.Context, projectKeys []string) (map[string]model.ProjectNamespaceInfo, error) {
	return s.projects.ResolveNamespaces(ctx, projectKeys)
}

// parseDisplayID parses a display ID like "INFRA-38" into project key and item number.
func parseDisplayID(displayID string) (string, int, error) {
	idx := strings.LastIndex(displayID, "-")
	if idx < 1 || idx >= len(displayID)-1 {
		return "", 0, fmt.Errorf("invalid display ID format")
	}
	key := displayID[:idx]
	numStr := displayID[idx+1:]
	var num int
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil || num <= 0 {
		return "", 0, fmt.Errorf("invalid item number in display ID")
	}
	return key, num, nil
}

func validateComplexity(value int, allowed []int) error {
	if value <= 0 {
		return fmt.Errorf("complexity must be a positive integer: %w", model.ErrValidation)
	}
	if len(allowed) > 0 {
		for _, a := range allowed {
			if a == value {
				return nil
			}
		}
		return fmt.Errorf("complexity value %d is not in the allowed values for this project: %w", value, model.ErrValidation)
	}
	return nil
}

func strPtr(s string) *string {
	return &s
}
