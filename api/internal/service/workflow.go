package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// WorkflowRepository defines persistence operations for workflows.
type WorkflowRepository interface {
	Create(ctx context.Context, wf *model.Workflow) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
	List(ctx context.Context) ([]model.Workflow, error)
	Update(ctx context.Context, wf *model.Workflow) error
	GetDefaultByName(ctx context.Context, name string) (*model.Workflow, error)
	ValidateTransition(ctx context.Context, workflowID uuid.UUID, fromStatus, toStatus string) (bool, error)
	GetInitialStatus(ctx context.Context, workflowID uuid.UUID) (*model.WorkflowStatus, error)
	GetStatusCategory(ctx context.Context, workflowID uuid.UUID, statusName string) (string, error)
	ListTransitions(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowTransition, error)
	ListStatuses(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error)
}

// WorkflowService handles workflow business logic.
type WorkflowService struct {
	workflows WorkflowRepository
}

// NewWorkflowService creates a new WorkflowService.
func NewWorkflowService(workflows WorkflowRepository) *WorkflowService {
	return &WorkflowService{workflows: workflows}
}

// List returns all workflows.
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

// GetByID returns a workflow with all statuses and transitions.
func (s *WorkflowService) GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	return s.workflows.GetByID(ctx, id)
}

// CreateWorkflowInput holds input for creating a custom workflow.
type CreateWorkflowInput struct {
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

	// Validate transitions reference existing status names
	statusNames := make(map[string]bool)
	for _, s := range input.Statuses {
		statusNames[s.Name] = true
	}
	for _, t := range input.Transitions {
		if !statusNames[t.FromStatus] {
			return nil, fmt.Errorf("transition references unknown from_status %q: %w", t.FromStatus, model.ErrValidation)
		}
		if !statusNames[t.ToStatus] {
			return nil, fmt.Errorf("transition references unknown to_status %q: %w", t.ToStatus, model.ErrValidation)
		}
	}

	wf := &model.Workflow{
		ID:          uuid.New(),
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
		wf.Name = *input.Name
	}
	if input.Description != nil {
		wf.Description = input.Description
	}

	if err := s.workflows.Update(ctx, wf); err != nil {
		return nil, fmt.Errorf("updating workflow: %w", err)
	}

	return s.workflows.GetByID(ctx, id)
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
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "in_progress", DisplayName: "In Progress", Category: model.CategoryInProgress, Position: 1},
			{Name: "in_review", DisplayName: "In Review", Category: model.CategoryInProgress, Position: 2},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 3},
			{Name: "cancelled", DisplayName: "Cancelled", Category: model.CategoryCancelled, Position: 4},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "in_progress", Name: strPtr("Start Work")},
			{FromStatus: "open", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "in_progress", ToStatus: "in_review", Name: strPtr("Submit for Review")},
			{FromStatus: "in_progress", ToStatus: "open", Name: strPtr("Move to Backlog")},
			{FromStatus: "in_progress", ToStatus: "cancelled", Name: strPtr("Cancel")},
			{FromStatus: "in_review", ToStatus: "done", Name: strPtr("Approve")},
			{FromStatus: "in_review", ToStatus: "in_progress", Name: strPtr("Request Changes")},
			{FromStatus: "done", ToStatus: "open", Name: strPtr("Reopen")},
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
