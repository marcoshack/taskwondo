package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
)

func newTestWorkflowService() (*WorkflowService, *mockWorkflowRepo) {
	repo := newMockWorkflowRepo()
	svc := NewWorkflowService(repo)
	return svc, repo
}

func TestWorkflowCreate_Success(t *testing.T) {
	svc, _ := newTestWorkflowService()

	desc := "A custom workflow"
	wf, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name:        "Custom",
		Description: &desc,
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "todo", ToStatus: "done", Name: strPtr("Complete")},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wf.Name != "Custom" {
		t.Fatalf("expected name 'Custom', got %s", wf.Name)
	}
	if len(wf.Statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(wf.Statuses))
	}
	if len(wf.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(wf.Transitions))
	}
}

func TestWorkflowCreate_EmptyName(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
		},
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestWorkflowCreate_NoStatuses(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name:     "Empty",
		Statuses: []model.WorkflowStatus{},
	})
	if err == nil {
		t.Fatal("expected error for empty statuses")
	}
}

func TestWorkflowCreate_NoTodoAtPosition0(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Bad",
		Statuses: []model.WorkflowStatus{
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 0},
		},
	})
	if err == nil {
		t.Fatal("expected error for no todo status at position 0")
	}
}

func TestWorkflowCreate_InvalidCategory(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Bad",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: "invalid", Position: 0},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestWorkflowCreate_TransitionReferencesUnknownStatus(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Bad",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "todo", ToStatus: "unknown"},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown to_status")
	}
}

func TestWorkflowGetByID_Success(t *testing.T) {
	svc, _ := newTestWorkflowService()

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Test",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "todo", ToStatus: "done"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wf, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wf.Name != "Test" {
		t.Fatalf("expected name 'Test', got %s", wf.Name)
	}
}

func TestWorkflowGetByID_NotFound(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.GetByID(context.Background(), uuid.New())
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWorkflowList(t *testing.T) {
	svc, _ := newTestWorkflowService()

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "WF1",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), CreateWorkflowInput{
		Name: "WF2",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	workflows, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}
}

func TestWorkflowUpdate_Success(t *testing.T) {
	svc, _ := newTestWorkflowService()

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Old Name",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	newName := "New Name"
	wf, err := svc.Update(context.Background(), created.ID, UpdateWorkflowInput{Name: &newName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wf.Name != "New Name" {
		t.Fatalf("expected name 'New Name', got %s", wf.Name)
	}
}

func TestWorkflowUpdate_EmptyName(t *testing.T) {
	svc, _ := newTestWorkflowService()

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Test",
		Statuses: []model.WorkflowStatus{
			{Name: "todo", DisplayName: "To Do", Category: model.CategoryTodo, Position: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	empty := ""
	_, err = svc.Update(context.Background(), created.ID, UpdateWorkflowInput{Name: &empty})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestWorkflowGetTransitionsMap(t *testing.T) {
	svc, _ := newTestWorkflowService()

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		Name: "Test",
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "wip", DisplayName: "WIP", Category: model.CategoryInProgress, Position: 1},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 2},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "wip"},
			{FromStatus: "open", ToStatus: "done"},
			{FromStatus: "wip", ToStatus: "done"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	transMap, err := svc.GetTransitionsMap(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(transMap["open"]) != 2 {
		t.Fatalf("expected 2 transitions from 'open', got %d", len(transMap["open"]))
	}
	if len(transMap["wip"]) != 1 {
		t.Fatalf("expected 1 transition from 'wip', got %d", len(transMap["wip"]))
	}
}

func TestSeedDefaultWorkflows(t *testing.T) {
	svc, _ := newTestWorkflowService()

	err := svc.SeedDefaultWorkflows(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have created two workflows
	workflows, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 seeded workflows, got %d", len(workflows))
	}

	// Seed again should be idempotent
	err = svc.SeedDefaultWorkflows(context.Background())
	if err != nil {
		t.Fatalf("expected idempotent seed, got %v", err)
	}

	workflows, err = svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected still 2 workflows after re-seed, got %d", len(workflows))
	}
}

func TestSeedDefaultWorkflows_TaskWorkflowID(t *testing.T) {
	svc, _ := newTestWorkflowService()

	err := svc.SeedDefaultWorkflows(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	id, err := svc.GetDefaultTaskWorkflowID(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("expected non-nil task workflow ID")
	}
}

// --- Test work item transition validation ---

func TestWorkItemUpdate_ValidTransition(t *testing.T) {
	setup := newTestWorkItemSetup()

	// Create a workflow with open -> in_progress transition
	wfID := uuid.New()
	wf := &model.Workflow{
		ID:        wfID,
		Name:      "Test Workflow",
		IsDefault: true,
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "in_progress", DisplayName: "In Progress", Category: model.CategoryInProgress, Position: 1},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 2},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "in_progress"},
			{FromStatus: "in_progress", ToStatus: "done"},
		},
	}
	setup.svc.workflows.(*mockWorkflowRepo).workflows[wfID] = wf

	info := userAuthInfo()
	project := setupProjectWithMember(t, setup.projectRepo, setup.memberRepo, info, model.ProjectRoleMember)
	project.DefaultWorkflowID = &wfID

	item, err := setup.svc.Create(context.Background(), info, project.Key, CreateWorkItemInput{
		Type:  model.WorkItemTypeTask,
		Title: "Test Item",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Valid transition: open -> in_progress
	status := "in_progress"
	updated, err := setup.svc.Update(context.Background(), info, project.Key, item.ItemNumber, UpdateWorkItemInput{
		Status: &status,
	})
	if err != nil {
		t.Fatalf("expected valid transition to succeed, got %v", err)
	}
	if updated.Status != "in_progress" {
		t.Fatalf("expected status 'in_progress', got %s", updated.Status)
	}
}

func TestWorkItemUpdate_InvalidTransition(t *testing.T) {
	setup := newTestWorkItemSetup()

	wfID := uuid.New()
	wf := &model.Workflow{
		ID:        wfID,
		Name:      "Test Workflow",
		IsDefault: true,
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "done"},
		},
	}
	setup.svc.workflows.(*mockWorkflowRepo).workflows[wfID] = wf

	info := userAuthInfo()
	project := setupProjectWithMember(t, setup.projectRepo, setup.memberRepo, info, model.ProjectRoleMember)
	project.DefaultWorkflowID = &wfID

	item, err := setup.svc.Create(context.Background(), info, project.Key, CreateWorkItemInput{
		Type:  model.WorkItemTypeTask,
		Title: "Test Item",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Move to done first (valid)
	doneStatus := "done"
	_, err = setup.svc.Update(context.Background(), info, project.Key, item.ItemNumber, UpdateWorkItemInput{
		Status: &doneStatus,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Try done -> open (no such transition)
	openStatus := "open"
	_, err = setup.svc.Update(context.Background(), info, project.Key, item.ItemNumber, UpdateWorkItemInput{
		Status: &openStatus,
	})
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	if !isErrInvalidTransition(err) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestWorkItemUpdate_ResolvedAtSetOnDone(t *testing.T) {
	setup := newTestWorkItemSetup()

	wfID := uuid.New()
	wf := &model.Workflow{
		ID:        wfID,
		Name:      "Test Workflow",
		IsDefault: true,
		Statuses: []model.WorkflowStatus{
			{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
			{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "open", ToStatus: "done"},
			{FromStatus: "done", ToStatus: "open"},
		},
	}
	setup.svc.workflows.(*mockWorkflowRepo).workflows[wfID] = wf

	info := userAuthInfo()
	project := setupProjectWithMember(t, setup.projectRepo, setup.memberRepo, info, model.ProjectRoleMember)
	project.DefaultWorkflowID = &wfID

	item, err := setup.svc.Create(context.Background(), info, project.Key, CreateWorkItemInput{
		Type:  model.WorkItemTypeTask,
		Title: "Test Item",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Transition to done -> resolved_at should be set
	doneStatus := "done"
	updated, err := setup.svc.Update(context.Background(), info, project.Key, item.ItemNumber, UpdateWorkItemInput{
		Status: &doneStatus,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set when transitioning to done")
	}

	// Transition back to open -> resolved_at should be cleared
	openStatus := "open"
	updated, err = setup.svc.Update(context.Background(), info, project.Key, item.ItemNumber, UpdateWorkItemInput{
		Status: &openStatus,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResolvedAt != nil {
		t.Fatal("expected resolved_at to be cleared when reopening")
	}
}

func TestWorkItemCreate_UsesWorkflowInitialStatus(t *testing.T) {
	setup := newTestWorkItemSetup()

	wfID := uuid.New()
	wf := &model.Workflow{
		ID:        wfID,
		Name:      "Ticket Workflow",
		IsDefault: true,
		Statuses: []model.WorkflowStatus{
			{Name: "new", DisplayName: "New", Category: model.CategoryTodo, Position: 0},
			{Name: "closed", DisplayName: "Closed", Category: model.CategoryDone, Position: 1},
		},
		Transitions: []model.WorkflowTransition{
			{FromStatus: "new", ToStatus: "closed"},
		},
	}
	setup.svc.workflows.(*mockWorkflowRepo).workflows[wfID] = wf

	info := userAuthInfo()
	project := setupProjectWithMember(t, setup.projectRepo, setup.memberRepo, info, model.ProjectRoleMember)
	project.DefaultWorkflowID = &wfID

	item, err := setup.svc.Create(context.Background(), info, project.Key, CreateWorkItemInput{
		Type:  model.WorkItemTypeTicket,
		Title: "Test Ticket",
	})
	if err != nil {
		t.Fatal(err)
	}

	if item.Status != "new" {
		t.Fatalf("expected initial status 'new' from workflow, got %s", item.Status)
	}
}

func isErrInvalidTransition(err error) bool {
	for err != nil {
		if err == model.ErrInvalidTransition {
			return true
		}
		unwrapped := interface{ Unwrap() error }(nil)
		if u, ok := err.(interface{ Unwrap() error }); ok {
			unwrapped = u
			err = unwrapped.Unwrap()
		} else {
			break
		}
	}
	return false
}
