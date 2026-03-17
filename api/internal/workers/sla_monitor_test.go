package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock implementations for SLA monitor dependencies ---

type mockSLAProjectRepo struct {
	projects []model.Project
}

func (m *mockSLAProjectRepo) ListAll(_ context.Context) ([]model.Project, error) {
	return m.projects, nil
}

type mockSLAEscalationRepo struct {
	lists    map[uuid.UUID][]model.EscalationList
	mappings map[uuid.UUID][]model.TypeEscalationMapping
}

func (m *mockSLAEscalationRepo) List(_ context.Context, projectID uuid.UUID) ([]model.EscalationList, error) {
	return m.lists[projectID], nil
}

func (m *mockSLAEscalationRepo) ListMappings(_ context.Context, projectID uuid.UUID) ([]model.TypeEscalationMapping, error) {
	return m.mappings[projectID], nil
}

type mockSLANotifRepo struct {
	sent map[string]bool // key: "workItemID:statusName:level:threshold"
}

func slaNotifKey(workItemID uuid.UUID, statusName string, level, thresholdPct int) string {
	return workItemID.String() + ":" + statusName + ":" + string(rune('0'+level)) + ":" + string(rune('0'+thresholdPct))
}

func (m *mockSLANotifRepo) HasBeenSent(_ context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) (bool, error) {
	return m.sent[slaNotifKey(workItemID, statusName, level, thresholdPct)], nil
}

type mockSLAItemRepo struct {
	items map[uuid.UUID]*model.WorkItemList
}

func (m *mockSLAItemRepo) List(_ context.Context, projectID uuid.UUID, _ *model.WorkItemFilter) (*model.WorkItemList, error) {
	if list, ok := m.items[projectID]; ok {
		return list, nil
	}
	return &model.WorkItemList{}, nil
}

type mockSLAWorkflowRepo struct {
	statuses map[uuid.UUID][]model.WorkflowStatus
}

func (m *mockSLAWorkflowRepo) ListStatuses(_ context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error) {
	return m.statuses[workflowID], nil
}

type mockSLATypeWorkflowRepo struct {
	mappings map[uuid.UUID][]model.ProjectTypeWorkflow
}

func (m *mockSLATypeWorkflowRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectTypeWorkflow, error) {
	return m.mappings[projectID], nil
}

type mockSLAPublisher struct {
	published []publishedEvent
}

type publishedEvent struct {
	subject string
	data    json.RawMessage
}

func (m *mockSLAPublisher) Publish(subject string, data any) error {
	raw, _ := json.Marshal(data)
	m.published = append(m.published, publishedEvent{subject: subject, data: raw})
	return nil
}

// mockSLAInfoComputer is a test double for slaInfoComputer that returns pre-configured SLA info.
type mockSLAInfoComputer struct {
	info map[uuid.UUID]*model.SLAInfo
}

func (m *mockSLAInfoComputer) ComputeSLAInfoBatch(_ context.Context, items []model.WorkItem, _ uuid.UUID, _ *model.BusinessHoursConfig) map[uuid.UUID]*model.SLAInfo {
	if m.info != nil {
		return m.info
	}
	return make(map[uuid.UUID]*model.SLAInfo)
}

// --- Tests ---

func TestSLAMonitor_Run_NoProjects(t *testing.T) {
	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: nil},
		&mockSLAInfoComputer{},
		&mockSLAEscalationRepo{lists: map[uuid.UUID][]model.EscalationList{}, mappings: map[uuid.UUID][]model.TypeEscalationMapping{}},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		&mockSLAPublisher{},
		zerolog.Nop(),
	)
	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSLAMonitor_Run_ProjectWithoutEscalationLists(t *testing.T) {
	projectID := uuid.New()
	wfID := uuid.New()
	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{{ID: projectID, Key: "TP", DefaultWorkflowID: &wfID}}},
		&mockSLAInfoComputer{},
		&mockSLAEscalationRepo{
			lists:    map[uuid.UUID][]model.EscalationList{},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{},
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		&mockSLAPublisher{},
		zerolog.Nop(),
	)
	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSLAMonitor_Run_PublishesBreachEvent(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()
	escLevelID := uuid.New()
	workItemID := uuid.New()

	wfStatuses := []model.WorkflowStatus{
		{Name: "Open", Category: model.CategoryTodo},
		{Name: "In Progress", Category: model.CategoryInProgress},
		{Name: "Done", Category: model.CategoryDone},
	}

	publisher := &mockSLAPublisher{}

	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{info: map[uuid.UUID]*model.SLAInfo{
			workItemID: {
				TargetSeconds:    3600,
				ElapsedSeconds:   3000,
				RemainingSeconds: 600,
				Percentage:       83, // above 75% threshold
				Status:           model.SLAStatusWarning,
				Paused:           false,
			},
		}},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {
					{
						ID:        escListID,
						ProjectID: projectID,
						Levels: []model.EscalationLevel{
							{
								ID:           escLevelID,
								ListID:       escListID,
								ThresholdPct: 75,
								Position:     0,
								Users:        []model.EscalationLevelUser{{UserID: uuid.New(), Email: "mgr@example.com"}},
							},
						},
					},
				},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{
				projectID: {{ProjectID: projectID, WorkItemType: "ticket", EscalationListID: escListID}},
			},
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{
			projectID: {
				Items: []model.WorkItem{
					{
						ID:         workItemID,
						ProjectID:  projectID,
						ItemNumber: 42,
						Title:      "Test ticket",
						Type:       "ticket",
						Status:     "Open",
						Priority:   "medium",
					},
				},
			},
		}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{workflowID: wfStatuses}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		publisher,
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.published))
	}

	if publisher.published[0].subject != "notification.sla_breach" {
		t.Errorf("expected subject notification.sla_breach, got %s", publisher.published[0].subject)
	}

	var evt model.SLABreachEvent
	if err := json.Unmarshal(publisher.published[0].data, &evt); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if evt.WorkItemID != workItemID {
		t.Errorf("expected work item ID %s, got %s", workItemID, evt.WorkItemID)
	}
	if evt.ThresholdPct != 75 {
		t.Errorf("expected threshold 75, got %d", evt.ThresholdPct)
	}
	if evt.EscalationLevel != 1 {
		t.Errorf("expected escalation level 1, got %d", evt.EscalationLevel)
	}
	if evt.SLAPercentage != 83 {
		t.Errorf("expected SLA percentage 83, got %d", evt.SLAPercentage)
	}
	if evt.ProjectKey != "TP" {
		t.Errorf("expected project key TP, got %s", evt.ProjectKey)
	}
	if evt.ItemNumber != 42 {
		t.Errorf("expected item number 42, got %d", evt.ItemNumber)
	}
}

func TestSLAMonitor_Run_SkipsAlreadySent(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()
	workItemID := uuid.New()

	wfStatuses := []model.WorkflowStatus{
		{Name: "Open", Category: model.CategoryTodo},
		{Name: "Done", Category: model.CategoryDone},
	}

	// Mark as already sent
	alreadySent := &mockSLANotifRepo{sent: map[string]bool{
		slaNotifKey(workItemID, "Open", 1, 75): true,
	}}

	publisher := &mockSLAPublisher{}

	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{info: map[uuid.UUID]*model.SLAInfo{
			workItemID: {
				TargetSeconds:  3600,
				ElapsedSeconds: 3000,
				Percentage:     83,
				Status:         model.SLAStatusWarning,
			},
		}},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {{
					ID:        escListID,
					ProjectID: projectID,
					Levels: []model.EscalationLevel{
						{ThresholdPct: 75, Position: 0},
					},
				}},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{
				projectID: {{ProjectID: projectID, WorkItemType: "ticket", EscalationListID: escListID}},
			},
		},
		alreadySent,
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{
			projectID: {
				Items: []model.WorkItem{{
					ID: workItemID, ProjectID: projectID, ItemNumber: 1,
					Title: "Test", Type: "ticket", Status: "Open", Priority: "medium",
				}},
			},
		}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{workflowID: wfStatuses}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		publisher,
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.published) != 0 {
		t.Fatalf("expected 0 published events (already sent), got %d", len(publisher.published))
	}
}

func TestSLAMonitor_Run_SkipsBelowThreshold(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()
	workItemID := uuid.New()

	wfStatuses := []model.WorkflowStatus{
		{Name: "Open", Category: model.CategoryTodo},
		{Name: "Done", Category: model.CategoryDone},
	}

	publisher := &mockSLAPublisher{}

	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{info: map[uuid.UUID]*model.SLAInfo{
			workItemID: {
				TargetSeconds:  3600,
				ElapsedSeconds: 1000,
				Percentage:     27, // below 75% threshold
				Status:         model.SLAStatusOnTrack,
			},
		}},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {{
					ID:        escListID,
					ProjectID: projectID,
					Levels: []model.EscalationLevel{
						{ThresholdPct: 75, Position: 0},
					},
				}},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{
				projectID: {{ProjectID: projectID, WorkItemType: "ticket", EscalationListID: escListID}},
			},
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{
			projectID: {
				Items: []model.WorkItem{{
					ID: workItemID, ProjectID: projectID, ItemNumber: 1,
					Title: "Test", Type: "ticket", Status: "Open", Priority: "medium",
				}},
			},
		}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{workflowID: wfStatuses}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		publisher,
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.published) != 0 {
		t.Fatalf("expected 0 published events (below threshold), got %d", len(publisher.published))
	}
}

func TestSLAMonitor_Run_MultipleThresholds(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()
	workItemID := uuid.New()

	wfStatuses := []model.WorkflowStatus{
		{Name: "Open", Category: model.CategoryTodo},
		{Name: "Done", Category: model.CategoryDone},
	}

	publisher := &mockSLAPublisher{}

	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{info: map[uuid.UUID]*model.SLAInfo{
			workItemID: {
				TargetSeconds:  3600,
				ElapsedSeconds: 3700,
				Percentage:     102, // above both 75% and 100%
				Status:         model.SLAStatusBreached,
			},
		}},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {{
					ID:        escListID,
					ProjectID: projectID,
					Levels: []model.EscalationLevel{
						{ThresholdPct: 75, Position: 0},
						{ThresholdPct: 100, Position: 1},
					},
				}},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{
				projectID: {{ProjectID: projectID, WorkItemType: "ticket", EscalationListID: escListID}},
			},
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{
			projectID: {
				Items: []model.WorkItem{{
					ID: workItemID, ProjectID: projectID, ItemNumber: 1,
					Title: "Test", Type: "ticket", Status: "Open", Priority: "medium",
				}},
			},
		}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{workflowID: wfStatuses}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		publisher,
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.published) != 2 {
		t.Fatalf("expected 2 published events (both thresholds crossed), got %d", len(publisher.published))
	}

	var evt1, evt2 model.SLABreachEvent
	json.Unmarshal(publisher.published[0].data, &evt1)
	json.Unmarshal(publisher.published[1].data, &evt2)

	if evt1.EscalationLevel != 1 || evt1.ThresholdPct != 75 {
		t.Errorf("first event should be level 1 threshold 75, got level %d threshold %d", evt1.EscalationLevel, evt1.ThresholdPct)
	}
	if evt2.EscalationLevel != 2 || evt2.ThresholdPct != 100 {
		t.Errorf("second event should be level 2 threshold 100, got level %d threshold %d", evt2.EscalationLevel, evt2.ThresholdPct)
	}
}

func TestSLAMonitor_Run_SkipsPausedItems(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()
	workItemID := uuid.New()

	wfStatuses := []model.WorkflowStatus{
		{Name: "Open", Category: model.CategoryTodo},
		{Name: "Done", Category: model.CategoryDone},
	}

	publisher := &mockSLAPublisher{}

	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{info: map[uuid.UUID]*model.SLAInfo{
			workItemID: {
				TargetSeconds:  3600,
				ElapsedSeconds: 3000,
				Percentage:     83,
				Status:         model.SLAStatusWarning,
				Paused:         true, // outside business hours
			},
		}},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {{
					ID:        escListID,
					ProjectID: projectID,
					Levels: []model.EscalationLevel{
						{ThresholdPct: 75, Position: 0},
					},
				}},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{
				projectID: {{ProjectID: projectID, WorkItemType: "ticket", EscalationListID: escListID}},
			},
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{
			projectID: {
				Items: []model.WorkItem{{
					ID: workItemID, ProjectID: projectID, ItemNumber: 1,
					Title: "Test", Type: "ticket", Status: "Open", Priority: "medium",
				}},
			},
		}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{workflowID: wfStatuses}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		publisher,
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.published) != 0 {
		t.Fatalf("expected 0 published events (item paused), got %d", len(publisher.published))
	}
}

func TestSLAMonitor_Run_SkipsProjectWithoutMappings(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()

	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {{ID: escListID, ProjectID: projectID}},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{}, // no mappings
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		&mockSLAPublisher{},
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSLAMonitor_Run_SkipsItemsWithNoEscalationList(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	escListID := uuid.New()
	workItemID := uuid.New()

	wfStatuses := []model.WorkflowStatus{
		{Name: "Open", Category: model.CategoryTodo},
		{Name: "Done", Category: model.CategoryDone},
	}

	publisher := &mockSLAPublisher{}

	// Escalation mapping is for "bug" type but the work item is "ticket"
	task := NewSLAMonitorTask(
		&mockSLAProjectRepo{projects: []model.Project{
			{ID: projectID, Key: "TP", DefaultWorkflowID: &workflowID},
		}},
		&mockSLAInfoComputer{info: map[uuid.UUID]*model.SLAInfo{
			workItemID: {Percentage: 90, TargetSeconds: 3600, ElapsedSeconds: 3240},
		}},
		&mockSLAEscalationRepo{
			lists: map[uuid.UUID][]model.EscalationList{
				projectID: {{
					ID:        escListID,
					ProjectID: projectID,
					Levels:    []model.EscalationLevel{{ThresholdPct: 75, Position: 0}},
				}},
			},
			mappings: map[uuid.UUID][]model.TypeEscalationMapping{
				projectID: {{ProjectID: projectID, WorkItemType: "bug", EscalationListID: escListID}}, // wrong type
			},
		},
		&mockSLANotifRepo{sent: map[string]bool{}},
		&mockSLAItemRepo{items: map[uuid.UUID]*model.WorkItemList{
			projectID: {
				Items: []model.WorkItem{{
					ID: workItemID, ProjectID: projectID, ItemNumber: 1,
					Title: "Test", Type: "ticket", Status: "Open", Priority: "medium",
				}},
			},
		}},
		&mockSLAWorkflowRepo{statuses: map[uuid.UUID][]model.WorkflowStatus{workflowID: wfStatuses}},
		&mockSLATypeWorkflowRepo{mappings: map[uuid.UUID][]model.ProjectTypeWorkflow{}},
		publisher,
		zerolog.Nop(),
	)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.published) != 0 {
		t.Fatalf("expected 0 published events (type mismatch), got %d", len(publisher.published))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "0m"},
		{30, "0m"},
		{60, "1m"},
		{3600, "1h"},
		{3660, "1h 1m"},
		{86400, "1d"},
		{90000, "1d 1h"},
		{-3600, "overdue by 1h"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.expected {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, got, tt.expected)
		}
	}
}
