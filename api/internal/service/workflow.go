package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// WorkflowRepository defines persistence operations for workflows.
type WorkflowRepository interface {
	Create(ctx context.Context, wf *model.Workflow) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
	List(ctx context.Context) ([]model.Workflow, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Workflow, error)
	ListProjectOnly(ctx context.Context, projectID uuid.UUID) ([]model.Workflow, error)
	Update(ctx context.Context, wf *model.Workflow) error
	ReplaceStatusesAndTransitions(ctx context.Context, wf *model.Workflow) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetDefaultByName(ctx context.Context, name string) (*model.Workflow, error)
	ListDefaultNames(ctx context.Context) ([]string, error)
	ValidateTransition(ctx context.Context, workflowID uuid.UUID, fromStatus, toStatus string) (bool, error)
	GetInitialStatus(ctx context.Context, workflowID uuid.UUID) (*model.WorkflowStatus, error)
	GetStatusCategory(ctx context.Context, workflowID uuid.UUID, statusName string) (string, error)
	ListTransitions(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowTransition, error)
	ListStatuses(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error)
	ListAllStatuses(ctx context.Context) ([]model.WorkflowStatus, error)
	IsInUse(ctx context.Context, id uuid.UUID) (bool, error)
}

// WorkflowService handles workflow business logic.
type WorkflowService struct {
	workflows WorkflowRepository
}

// NewWorkflowService creates a new WorkflowService.
func NewWorkflowService(workflows WorkflowRepository) *WorkflowService {
	return &WorkflowService{workflows: workflows}
}

// List returns all system-wide workflows.
func (s *WorkflowService) List(ctx context.Context) ([]model.Workflow, error) {
	workflows, err := s.workflows.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}

	// Load statuses for each workflow
	for i := range workflows {
		statuses, err := s.workflows.ListStatuses(ctx, workflows[i].ID)
		if err != nil {
			return nil, fmt.Errorf("loading statuses for workflow %s: %w", workflows[i].ID, err)
		}
		workflows[i].Statuses = statuses
	}

	return workflows, nil
}

// ListByProject returns all workflows available to a project (system-wide + project-specific).
func (s *WorkflowService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Workflow, error) {
	workflows, err := s.workflows.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listing project workflows: %w", err)
	}

	for i := range workflows {
		statuses, err := s.workflows.ListStatuses(ctx, workflows[i].ID)
		if err != nil {
			return nil, fmt.Errorf("loading statuses for workflow %s: %w", workflows[i].ID, err)
		}
		workflows[i].Statuses = statuses
	}

	return workflows, nil
}

// GetByID returns a workflow with all statuses and transitions.
func (s *WorkflowService) GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	return s.workflows.GetByID(ctx, id)
}

// ListAllStatuses returns all distinct statuses from system workflows.
func (s *WorkflowService) ListAllStatuses(ctx context.Context) ([]model.WorkflowStatus, error) {
	return s.workflows.ListAllStatuses(ctx)
}

// CreateWorkflowInput holds input for creating a custom workflow.
type CreateWorkflowInput struct {
	ProjectID   *uuid.UUID
	Name        string
	Description *string
	Statuses    []model.WorkflowStatus
	Transitions []model.WorkflowTransition
}

// Create creates a custom workflow.
func (s *WorkflowService) Create(ctx context.Context, input CreateWorkflowInput) (*model.Workflow, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required: %w", model.ErrValidation)
	}
	if len(input.Statuses) == 0 {
		return nil, fmt.Errorf("at least one status is required: %w", model.ErrValidation)
	}
	if len(input.Statuses) > 50 {
		return nil, fmt.Errorf("workflow cannot have more than 50 statuses: %w", model.ErrValidation)
	}
	if len(input.Transitions) > 500 {
		return nil, fmt.Errorf("workflow cannot have more than 500 transitions: %w", model.ErrValidation)
	}

	// Validate name does not conflict with default workflow names
	if err := s.validateNameNotDefault(ctx, input.Name); err != nil {
		return nil, err
	}

	// Validate at least one todo-category status at position 0
	hasTodo := false
	for _, s := range input.Statuses {
		if s.Category == model.CategoryTodo && s.Position == 0 {
			hasTodo = true
		}
		if !isValidCategory(s.Category) {
			return nil, fmt.Errorf("invalid status category %q: %w", s.Category, model.ErrValidation)
		}
	}
	if !hasTodo {
		return nil, fmt.Errorf("workflow must have a todo-category status at position 0: %w", model.ErrValidation)
	}

	// Validate transitions reference existing status names and are unique
	statusNames := make(map[string]bool)
	for _, s := range input.Statuses {
		statusNames[s.Name] = true
	}
	seenTransitions := make(map[string]bool)
	for _, t := range input.Transitions {
		if !statusNames[t.FromStatus] {
			return nil, fmt.Errorf("transition references unknown from_status %q: %w", t.FromStatus, model.ErrValidation)
		}
		if !statusNames[t.ToStatus] {
			return nil, fmt.Errorf("transition references unknown to_status %q: %w", t.ToStatus, model.ErrValidation)
		}
		key := t.FromStatus + "->" + t.ToStatus
		if seenTransitions[key] {
			return nil, fmt.Errorf("duplicate transition %s: %w", key, model.ErrValidation)
		}
		seenTransitions[key] = true
	}

	wf := &model.Workflow{
		ID:          uuid.New(),
		ProjectID:   input.ProjectID,
		Name:        input.Name,
		Description: input.Description,
		IsDefault:   false,
		Statuses:    input.Statuses,
		Transitions: input.Transitions,
	}

	if err := s.workflows.Create(ctx, wf); err != nil {
		return nil, fmt.Errorf("creating workflow: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("workflow_id", wf.ID.String()).
		Str("workflow_name", wf.Name).
		Msg("workflow created")

	return s.workflows.GetByID(ctx, wf.ID)
}

// UpdateWorkflowInput holds input for updating a workflow.
type UpdateWorkflowInput struct {
	Name        *string
	Description *string
	Statuses    []model.WorkflowStatus
	Transitions []model.WorkflowTransition
}

// Update modifies a workflow's name and/or description.
func (s *WorkflowService) Update(ctx context.Context, id uuid.UUID, input UpdateWorkflowInput) (*model.Workflow, error) {
	wf, err := s.workflows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, fmt.Errorf("name cannot be empty: %w", model.ErrValidation)
		}
		// Validate name does not conflict with default workflow names (unless it's the same workflow)
		if !wf.IsDefault {
			if err := s.validateNameNotDefault(ctx, *input.Name); err != nil {
				return nil, err
			}
		}
		wf.Name = *input.Name
	}
	if input.Description != nil {
		wf.Description = input.Description
	}

	// If statuses and transitions are provided, do a full replace
	if input.Statuses != nil {
		if len(input.Statuses) > 50 {
			return nil, fmt.Errorf("workflow cannot have more than 50 statuses: %w", model.ErrValidation)
		}
		if len(input.Transitions) > 500 {
			return nil, fmt.Errorf("workflow cannot have more than 500 transitions: %w", model.ErrValidation)
		}

		// Validate transitions reference valid statuses and are unique
		statusNames := make(map[string]bool)
		for _, s := range input.Statuses {
			statusNames[s.Name] = true
		}
		seenTransitions := make(map[string]bool)
		for _, t := range input.Transitions {
			if !statusNames[t.FromStatus] {
				return nil, fmt.Errorf("transition references unknown from_status %q: %w", t.FromStatus, model.ErrValidation)
			}
			if !statusNames[t.ToStatus] {
				return nil, fmt.Errorf("transition references unknown to_status %q: %w", t.ToStatus, model.ErrValidation)
			}
			key := t.FromStatus + "->" + t.ToStatus
			if seenTransitions[key] {
				return nil, fmt.Errorf("duplicate transition %s: %w", key, model.ErrValidation)
			}
			seenTransitions[key] = true
		}

		wf.Statuses = input.Statuses
		wf.Transitions = input.Transitions

		if err := s.workflows.ReplaceStatusesAndTransitions(ctx, wf); err != nil {
			return nil, fmt.Errorf("replacing workflow statuses and transitions: %w", err)
		}
	} else {
		if err := s.workflows.Update(ctx, wf); err != nil {
			return nil, fmt.Errorf("updating workflow: %w", err)
		}
	}

	return s.workflows.GetByID(ctx, id)
}

// DeleteProjectWorkflow deletes a project-scoped workflow.
func (s *WorkflowService) DeleteProjectWorkflow(ctx context.Context, id uuid.UUID) error {
	wf, err := s.workflows.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if wf.IsDefault {
		return fmt.Errorf("cannot delete a system workflow: %w", model.ErrForbidden)
	}
	if wf.ProjectID == nil {
		return fmt.Errorf("cannot delete a system workflow: %w", model.ErrForbidden)
	}

	return s.workflows.Delete(ctx, id)
}

// DeleteSystemWorkflow deletes a system-scoped workflow if it is not in use by any project.
func (s *WorkflowService) DeleteSystemWorkflow(ctx context.Context, id uuid.UUID) error {
	wf, err := s.workflows.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if wf.ProjectID != nil {
		return fmt.Errorf("use project endpoint to delete project workflows: %w", model.ErrValidation)
	}

	if wf.IsDefault {
		return fmt.Errorf("cannot delete a default workflow: %w", model.ErrForbidden)
	}

	inUse, err := s.workflows.IsInUse(ctx, id)
	if err != nil {
		return fmt.Errorf("checking workflow usage: %w", err)
	}
	if inUse {
		return fmt.Errorf("workflow is in use by one or more projects: %w", model.ErrValidation)
	}

	return s.workflows.Delete(ctx, id)
}

// GetTransitionsMap returns transitions grouped by from_status.
func (s *WorkflowService) GetTransitionsMap(ctx context.Context, id uuid.UUID) (map[string][]model.WorkflowTransition, error) {
	_, err := s.workflows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	transitions, err := s.workflows.ListTransitions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listing transitions: %w", err)
	}

	result := make(map[string][]model.WorkflowTransition)
	for _, t := range transitions {
		result[t.FromStatus] = append(result[t.FromStatus], t)
	}
	return result, nil
}

// GetDefaultTaskWorkflowID returns the ID of the default Task Workflow.
func (s *WorkflowService) GetDefaultTaskWorkflowID(ctx context.Context) (uuid.UUID, error) {
	wf, err := s.workflows.GetDefaultByName(ctx, "Task Workflow")
	if err != nil {
		return uuid.Nil, err
	}
	return wf.ID, nil
}

// SeedDefaultWorkflows creates the Task and Ticket workflows if they don't exist.
func (s *WorkflowService) SeedDefaultWorkflows(ctx context.Context) error {
	if err := s.seedTaskWorkflow(ctx); err != nil {
		return fmt.Errorf("seeding task workflow: %w", err)
	}
	if err := s.seedTicketWorkflow(ctx); err != nil {
		return fmt.Errorf("seeding ticket workflow: %w", err)
	}
	return nil
}

// validateNameNotDefault checks that the given name doesn't match any default workflow name (case-insensitive).
func (s *WorkflowService) validateNameNotDefault(ctx context.Context, name string) error {
	defaultNames, err := s.workflows.ListDefaultNames(ctx)
	if err != nil {
		return fmt.Errorf("checking default workflow names: %w", err)
	}
	for _, dn := range defaultNames {
		if strings.EqualFold(name, dn) {
			return fmt.Errorf("workflow name %q conflicts with a system workflow: %w", name, model.ErrValidation)
		}
	}
	return nil
}

func (s *WorkflowService) seedTaskWorkflow(ctx context.Context) error {
	_, err := s.workflows.GetDefaultByName(ctx, "Task Workflow")
	if err == nil {
		log.Ctx(ctx).Debug().Msg("task workflow already exists, skipping seed")
		return nil
	}
	if err != model.ErrNotFound {
		return err
	}

	desc := "For tasks, bugs, and epics"
	wf := &model.Workflow{
		ID:          uuid.New(),
		Name:        "Task Workflow",
		Description: &desc,
		IsDefault:   true,
		Statuses: []model.WorkflowStatus{
			{Name: "backlog", DisplayName: "Backlog", Category: model.CategoryTodo, Position: 0},
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 1},
			{Name: "in_progress", DisplayName: "In Progress", Category: model.CategoryInProgress, Position: 2},
			{Name: "in_review", DisplayName: "In Review", Category: model.CategoryInProgress, Position: 3},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 4},
			{Name: "cancelled", DisplayName: "Cancelled", Category: model.CategoryCancelled, Position: 5},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "backlog", ToStatus: "open", Name: strPtr("Prioritize")},
			{FromStatus: "backlog", ToStatus: "in_progress", Name: strPtr("Start Work")},
			{FromStatus: "backlog", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "open", ToStatus: "in_progress", Name: strPtr("Start Work")},
			{FromStatus: "open", ToStatus: "backlog", Name: strPtr("Deprioritize")},
			{FromStatus: "open", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "in_progress", ToStatus: "in_review", Name: strPtr("Submit for Review")},
			{FromStatus: "in_progress", ToStatus: "backlog", Name: strPtr("Move to Backlog")},
			{FromStatus: "in_progress", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "in_review", ToStatus: "done", Name: strPtr("Approve")},
			{FromStatus: "in_review", ToStatus: "in_progress", Name: strPtr("Request Changes")},
			{FromStatus: "done", ToStatus: "backlog", Name: strPtr("Reopen")},
		},
	}

	if err := s.workflows.Create(ctx, wf); err != nil {
		return err
	}

	log.Ctx(ctx).Info().Str("workflow_id", wf.ID.String()).Msg("seeded task workflow")
	return nil
}

func (s *WorkflowService) seedTicketWorkflow(ctx context.Context) error {
	_, err := s.workflows.GetDefaultByName(ctx, "Ticket Workflow")
	if err == nil {
		log.Ctx(ctx).Debug().Msg("ticket workflow already exists, skipping seed")
		return nil
	}
	if err != model.ErrNotFound {
		return err
	}

	desc := "For support tickets and incidents"
	wf := &model.Workflow{
		ID:          uuid.New(),
		Name:        "Ticket Workflow",
		Description: &desc,
		IsDefault:   true,
		Statuses: []model.WorkflowStatus{
			{Name: "new", DisplayName: "New", Category: model.CategoryTodo, Position: 0},
			{Name: "triaged", DisplayName: "Triaged", Category: model.CategoryTodo, Position: 1},
			{Name: "investigating", DisplayName: "Investigating", Category: model.CategoryInProgress, Position: 2},
			{Name: "waiting_on_customer", DisplayName: "Waiting on Customer", Category: model.CategoryInProgress, Position: 3},
			{Name: "resolved", DisplayName: "Resolved", Category: model.CategoryDone, Position: 4},
			{Name: "closed", DisplayName: "Closed", Category: model.CategoryDone, Position: 5},
			{Name: "cancelled", DisplayName: "Cancelled", Category: model.CategoryCancelled, Position: 6},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "new", ToStatus: "triaged", Name: strPtr("Triage")},
			{FromStatus: "new", ToStatus: "investigating", Name: strPtr("Start Investigation")},
			{FromStatus: "new", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "triaged", ToStatus: "investigating", Name: strPtr("Start Investigation")},
			{FromStatus: "triaged", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "investigating", ToStatus: "waiting_on_customer", Name: strPtr("Waiting on Customer")},
			{FromStatus: "investigating", ToStatus: "resolved", Name: strPtr("Resolve")},
			{FromStatus: "investigating", ToStatus: "triaged", Name: strPtr("Back to Triage")},
			{FromStatus: "waiting_on_customer", ToStatus: "investigating", Name: strPtr("Customer Responded")},
			{FromStatus: "waiting_on_customer", ToStatus: "resolved", Name: strPtr("Resolve")},
			{FromStatus: "resolved", ToStatus: "closed", Name: strPtr("Close")},
			{FromStatus: "resolved", ToStatus: "investigating", Name: strPtr("Reopen")},
			{FromStatus: "closed", ToStatus: "investigating", Name: strPtr("Reopen")},
		},
	}

	if err := s.workflows.Create(ctx, wf); err != nil {
		return err
	}

	log.Ctx(ctx).Info().Str("workflow_id", wf.ID.String()).Msg("seeded ticket workflow")
	return nil
}

func isValidCategory(c string) bool {
	switch c {
	case model.CategoryTodo, model.CategoryInProgress, model.CategoryDone, model.CategoryCancelled:
		return true
	}
	return false
}
