package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/model"
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
}

// CreateWorkItemInput holds the input for creating a work item.
type CreateWorkItemInput struct {
	Type         string
	Title        string
	Description  *string
	Priority     string
	AssigneeID   *uuid.UUID
	Labels       []string
	ParentID     *uuid.UUID
	QueueID      *uuid.UUID
	Visibility   string
	DueDate      *time.Time
	CustomFields map[string]interface{}
}

// UpdateWorkItemInput holds the input for updating a work item.
type UpdateWorkItemInput struct {
	Title         *string
	Description   *string
	ClearDescription bool
	Status        *string
	Priority      *string
	Type          *string
	AssigneeID    *uuid.UUID
	ClearAssignee bool
	Labels        *[]string
	Visibility    *string
	DueDate       *time.Time
	ClearDueDate  bool
	ParentID      *uuid.UUID
	ClearParent   bool
	QueueID       *uuid.UUID
	ClearQueue    bool
	CustomFields  map[string]interface{}
}

// WorkItemService handles work item business logic and authorization.
type WorkItemService struct {
	items    WorkItemRepository
	events   WorkItemEventRepository
	projects ProjectRepository
	members  ProjectMemberRepository
}

// NewWorkItemService creates a new WorkItemService.
func NewWorkItemService(
	items WorkItemRepository,
	events WorkItemEventRepository,
	projects ProjectRepository,
	members ProjectMemberRepository,
) *WorkItemService {
	return &WorkItemService{
		items:    items,
		events:   events,
		projects: projects,
		members:  members,
	}
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

	// Validate assignee is a project member
	if input.AssigneeID != nil {
		_, err := s.members.GetByProjectAndUser(ctx, project.ID, *input.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("assignee must be a project member: %w", model.ErrValidation)
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

	item := &model.WorkItem{
		ID:           uuid.Must(uuid.NewV7()),
		ProjectID:    project.ID,
		QueueID:      input.QueueID,
		ParentID:     input.ParentID,
		Type:         input.Type,
		Title:        input.Title,
		Description:  input.Description,
		Status:       "open",
		Priority:     input.Priority,
		AssigneeID:   input.AssigneeID,
		ReporterID:   info.UserID,
		Visibility:   input.Visibility,
		Labels:       labels,
		CustomFields: customFields,
		DueDate:      input.DueDate,
	}

	if err := s.items.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("creating work item: %w", err)
	}

	// Record "created" event
	s.recordEvent(ctx, item.ID, &info.UserID, "created", nil, nil, nil)

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

	// Resolve "assignee=me"
	if filter.AssigneeMe {
		filter.AssigneeID = &info.UserID
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

	// Apply changes and record events
	if input.Title != nil && *input.Title != item.Title {
		if strings.TrimSpace(*input.Title) == "" {
			return nil, fmt.Errorf("title cannot be empty: %w", model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, &info.UserID, "title", item.Title, *input.Title)
		item.Title = *input.Title
	}

	if input.Description != nil {
		oldDesc := ""
		if item.Description != nil {
			oldDesc = *item.Description
		}
		if *input.Description != oldDesc {
			s.recordFieldChange(ctx, item.ID, &info.UserID, "description", oldDesc, *input.Description)
			item.Description = input.Description
		}
	}

	if input.Status != nil && *input.Status != item.Status {
		s.recordFieldChange(ctx, item.ID, &info.UserID, "status", item.Status, *input.Status)
		item.Status = *input.Status
	}

	if input.Priority != nil && *input.Priority != item.Priority {
		if !isValidPriority(*input.Priority) {
			return nil, fmt.Errorf("invalid priority %q: %w", *input.Priority, model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, &info.UserID, "priority", item.Priority, *input.Priority)
		item.Priority = *input.Priority
	}

	if input.Type != nil && *input.Type != item.Type {
		if !isValidWorkItemType(*input.Type) {
			return nil, fmt.Errorf("invalid work item type %q: %w", *input.Type, model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, &info.UserID, "type", item.Type, *input.Type)
		item.Type = *input.Type
	}

	if input.ClearAssignee {
		if item.AssigneeID != nil {
			s.recordEvent(ctx, item.ID, &info.UserID, "unassigned", strPtr("assignee_id"), strPtr(item.AssigneeID.String()), nil)
			item.AssigneeID = nil
		}
	} else if input.AssigneeID != nil {
		oldAssignee := ""
		if item.AssigneeID != nil {
			oldAssignee = item.AssigneeID.String()
		}
		newAssignee := input.AssigneeID.String()
		if oldAssignee != newAssignee {
			// Validate assignee is a project member
			_, err := s.members.GetByProjectAndUser(ctx, project.ID, *input.AssigneeID)
			if err != nil {
				return nil, fmt.Errorf("assignee must be a project member: %w", model.ErrValidation)
			}
			s.recordEvent(ctx, item.ID, &info.UserID, "assigned", strPtr("assignee_id"), strPtr(oldAssignee), strPtr(newAssignee))
			item.AssigneeID = input.AssigneeID
		}
	}

	if input.Labels != nil {
		oldLabels := strings.Join(item.Labels, ",")
		newLabels := strings.Join(*input.Labels, ",")
		if oldLabels != newLabels {
			s.recordFieldChange(ctx, item.ID, &info.UserID, "labels", oldLabels, newLabels)
			item.Labels = *input.Labels
		}
	}

	if input.Visibility != nil && *input.Visibility != item.Visibility {
		if !isValidVisibility(*input.Visibility) {
			return nil, fmt.Errorf("invalid visibility %q: %w", *input.Visibility, model.ErrValidation)
		}
		s.recordFieldChange(ctx, item.ID, &info.UserID, "visibility", item.Visibility, *input.Visibility)
		item.Visibility = *input.Visibility
	}

	if input.ClearDueDate {
		if item.DueDate != nil {
			s.recordEvent(ctx, item.ID, &info.UserID, "due_date_cleared", strPtr("due_date"), strPtr(item.DueDate.Format(time.DateOnly)), nil)
			item.DueDate = nil
		}
	} else if input.DueDate != nil {
		oldDueDate := ""
		if item.DueDate != nil {
			oldDueDate = item.DueDate.Format(time.DateOnly)
		}
		newDueDate := input.DueDate.Format(time.DateOnly)
		if oldDueDate != newDueDate {
			s.recordEvent(ctx, item.ID, &info.UserID, "due_date_set", strPtr("due_date"), strPtr(oldDueDate), strPtr(newDueDate))
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
		item.QueueID = input.QueueID
	}

	if input.CustomFields != nil {
		item.CustomFields = input.CustomFields
	}

	if err := s.items.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("updating work item: %w", err)
	}

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

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Int("item_number", itemNumber).
		Str("user_id", info.UserID.String()).
		Msg("work item deleted")

	return nil
}

// --- Authorization helpers ---

func (s *WorkItemService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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

func (s *WorkItemService) requireRole(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID, allowedRoles ...string) error {
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

// --- Event recording helpers ---

func (s *WorkItemService) recordEvent(ctx context.Context, workItemID uuid.UUID, actorID *uuid.UUID, eventType string, fieldName, oldValue, newValue *string) {
	event := &model.WorkItemEvent{
		ID:         uuid.Must(uuid.NewV7()),
		WorkItemID: workItemID,
		ActorID:    actorID,
		EventType:  eventType,
		FieldName:  fieldName,
		OldValue:   oldValue,
		NewValue:   newValue,
		Metadata:   map[string]interface{}{},
		Visibility: model.VisibilityInternal,
	}
	if err := s.events.Create(ctx, event); err != nil {
		log.Ctx(ctx).Error().Err(err).
			Str("event_type", eventType).
			Str("work_item_id", workItemID.String()).
			Msg("failed to record work item event")
	}
}

func (s *WorkItemService) recordFieldChange(ctx context.Context, workItemID uuid.UUID, actorID *uuid.UUID, fieldName, oldValue, newValue string) {
	eventType := fieldName + "_updated"
	switch fieldName {
	case "status":
		eventType = "status_changed"
	case "priority":
		eventType = "priority_changed"
	}
	s.recordEvent(ctx, workItemID, actorID, eventType, &fieldName, &oldValue, &newValue)
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
	case "created_at", "updated_at", "priority", "due_date", "item_number":
		return true
	}
	return false
}

func strPtr(s string) *string {
	return &s
}
