package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- In-memory mock SLA repository ---

type inMemorySLARepo struct {
	targets map[uuid.UUID]*model.SLAStatusTarget
	elapsed map[string]*model.SLAElapsed // key: "workItemID:statusName"
}

func newInMemorySLARepo() *inMemorySLARepo {
	return &inMemorySLARepo{
		targets: make(map[uuid.UUID]*model.SLAStatusTarget),
		elapsed: make(map[string]*model.SLAElapsed),
	}
}

func elapsedKey(workItemID uuid.UUID, statusName string) string {
	return workItemID.String() + ":" + statusName
}

func (m *inMemorySLARepo) ListTargetsByProject(_ context.Context, projectID uuid.UUID) ([]model.SLAStatusTarget, error) {
	var result []model.SLAStatusTarget
	for _, t := range m.targets {
		if t.ProjectID == projectID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *inMemorySLARepo) ListTargetsByProjectAndType(_ context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) ([]model.SLAStatusTarget, error) {
	var result []model.SLAStatusTarget
	for _, t := range m.targets {
		if t.ProjectID == projectID && t.WorkItemType == workItemType && t.WorkflowID == workflowID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *inMemorySLARepo) GetTarget(_ context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID, statusName string) (*model.SLAStatusTarget, error) {
	for _, t := range m.targets {
		if t.ProjectID == projectID && t.WorkItemType == workItemType && t.WorkflowID == workflowID && t.StatusName == statusName {
			return t, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *inMemorySLARepo) BulkUpsertTargets(_ context.Context, targets []model.SLAStatusTarget) ([]model.SLAStatusTarget, error) {
	now := time.Now()
	result := make([]model.SLAStatusTarget, len(targets))
	for i, t := range targets {
		t.CreatedAt = now
		t.UpdatedAt = now
		m.targets[t.ID] = &t
		result[i] = t
	}
	return result, nil
}

func (m *inMemorySLARepo) DeleteTarget(_ context.Context, id uuid.UUID) error {
	if _, ok := m.targets[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.targets, id)
	return nil
}

func (m *inMemorySLARepo) DeleteTargetsByTypeAndWorkflow(_ context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) error {
	for id, t := range m.targets {
		if t.ProjectID == projectID && t.WorkItemType == workItemType && t.WorkflowID == workflowID {
			delete(m.targets, id)
		}
	}
	return nil
}

func (m *inMemorySLARepo) InitElapsedOnCreate(_ context.Context, workItemID uuid.UUID, statusName string, enteredAt time.Time) error {
	key := elapsedKey(workItemID, statusName)
	if _, exists := m.elapsed[key]; exists {
		return nil // ON CONFLICT DO NOTHING
	}
	m.elapsed[key] = &model.SLAElapsed{
		WorkItemID:     workItemID,
		StatusName:     statusName,
		ElapsedSeconds: 0,
		LastEnteredAt:  &enteredAt,
	}
	return nil
}

func (m *inMemorySLARepo) UpsertElapsedOnEnter(_ context.Context, workItemID uuid.UUID, statusName string, now time.Time) error {
	key := elapsedKey(workItemID, statusName)
	if e, exists := m.elapsed[key]; exists {
		e.LastEnteredAt = &now
	} else {
		m.elapsed[key] = &model.SLAElapsed{
			WorkItemID:     workItemID,
			StatusName:     statusName,
			ElapsedSeconds: 0,
			LastEnteredAt:  &now,
		}
	}
	return nil
}

func (m *inMemorySLARepo) UpdateElapsedOnLeave(_ context.Context, workItemID uuid.UUID, statusName string, now time.Time) error {
	key := elapsedKey(workItemID, statusName)
	e, exists := m.elapsed[key]
	if !exists || e.LastEnteredAt == nil {
		return nil
	}
	e.ElapsedSeconds += int(now.Sub(*e.LastEnteredAt).Seconds())
	e.LastEnteredAt = nil
	return nil
}

func (m *inMemorySLARepo) UpdateElapsedOnLeaveWithSeconds(_ context.Context, workItemID uuid.UUID, statusName string, additionalSeconds int) error {
	key := elapsedKey(workItemID, statusName)
	e, exists := m.elapsed[key]
	if !exists || e.LastEnteredAt == nil {
		return nil
	}
	e.ElapsedSeconds += additionalSeconds
	e.LastEnteredAt = nil
	return nil
}

func (m *inMemorySLARepo) GetElapsed(_ context.Context, workItemID uuid.UUID, statusName string) (*model.SLAElapsed, error) {
	key := elapsedKey(workItemID, statusName)
	e, exists := m.elapsed[key]
	if !exists {
		return nil, model.ErrNotFound
	}
	return e, nil
}

func (m *inMemorySLARepo) ListElapsedByWorkItemIDs(_ context.Context, ids []uuid.UUID) ([]model.SLAElapsed, error) {
	idSet := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []model.SLAElapsed
	for _, e := range m.elapsed {
		if idSet[e.WorkItemID] {
			result = append(result, *e)
		}
	}
	return result, nil
}

// --- Test helpers ---

func newTestSLAService() (*SLAService, *inMemorySLARepo, *mockProjectRepo, *mockProjectMemberRepo, *mockWorkflowRepo) {
	slaRepo := newInMemorySLARepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	workflowRepo := newMockWorkflowRepo()
	svc := NewSLAService(slaRepo, projectRepo, memberRepo, workflowRepo)
	return svc, slaRepo, projectRepo, memberRepo, workflowRepo
}

func setupSLATestProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
	t.Helper()
	project := &model.Project{
		ID:   uuid.New(),
		Name: "Test Project",
		Key:  "TEST",
	}
	projectRepo.Create(context.Background(), project)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    info.UserID,
		Role:      role,
	})
	return project
}

func setupTestWorkflow(t *testing.T, workflowRepo *mockWorkflowRepo) *model.Workflow {
	t.Helper()
	wf := &model.Workflow{
		ID:   uuid.New(),
		Name: "Test Workflow",
		Statuses: []model.WorkflowStatus{
			{Name: "Open", Category: model.CategoryTodo},
			{Name: "In Progress", Category: model.CategoryInProgress},
			{Name: "Done", Category: model.CategoryDone},
			{Name: "Cancelled", Category: model.CategoryCancelled},
		},
	}
	workflowRepo.Create(context.Background(), wf)
	return wf
}

// --- ListTargets tests ---

func TestListTargets_Success(t *testing.T) {
	svc, slaRepo, projectRepo, memberRepo, _ := newTestSLAService()
	info := userAuthInfo()
	project := setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	// Seed a target directly
	target := model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    uuid.New(),
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}
	slaRepo.targets[target.ID] = &target

	targets, err := svc.ListTargets(context.Background(), info, "TEST")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
}

func TestListTargets_NonMember(t *testing.T) {
	svc, _, projectRepo, _, _ := newTestSLAService()
	info := userAuthInfo()

	// Create project but don't add user as member
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	_, err := svc.ListTargets(context.Background(), info, "TEST")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

// --- BulkUpsertTargets tests ---

func TestBulkUpsertTargets_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	project := setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	project.BusinessHours = &model.BusinessHoursConfig{
		Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 17, Timezone: "UTC",
	}
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
			{StatusName: "In Progress", TargetSeconds: 7200, CalendarMode: model.CalendarModeBusinessHours},
		},
	}

	result, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(result))
	}
	if result[0].ProjectID != project.ID {
		t.Fatalf("expected project ID %s, got %s", project.ID, result[0].ProjectID)
	}
}

func TestBulkUpsertTargets_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden for member, got %v", err)
	}
}

func TestBulkUpsertTargets_InvalidType(t *testing.T) {
	svc, _, projectRepo, memberRepo, _ := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := BulkUpsertSLAInput{
		WorkItemType: "invalid_type",
		WorkflowID:   uuid.New(),
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestBulkUpsertTargets_TerminalStatusRejected(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Done", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for terminal status")
	}
}

func TestBulkUpsertTargets_CancelledStatusRejected(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Cancelled", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for cancelled status")
	}
}

func TestBulkUpsertTargets_InvalidCalendarMode(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 3600, CalendarMode: "invalid"},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for invalid calendar mode")
	}
}

func TestBulkUpsertTargets_NegativeTargetSeconds(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: -1, CalendarMode: model.CalendarMode24x7},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for negative target_seconds")
	}
}

func TestBulkUpsertTargets_StatusNotInWorkflow(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTask,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "NonExistent", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for status not in workflow")
	}
}

// --- DeleteTarget tests ---

func TestDeleteTarget_Success(t *testing.T) {
	svc, slaRepo, projectRepo, memberRepo, _ := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	targetID := uuid.New()
	slaRepo.targets[targetID] = &model.SLAStatusTarget{
		ID:            targetID,
		ProjectID:     uuid.New(),
		WorkItemType:  model.WorkItemTypeTask,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	err := svc.DeleteTarget(context.Background(), info, "TEST", targetID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, exists := slaRepo.targets[targetID]; exists {
		t.Fatal("expected target to be deleted")
	}
}

func TestDeleteTarget_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo, _ := newTestSLAService()
	info := userAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	err := svc.DeleteTarget(context.Background(), info, "TEST", uuid.New())
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// --- ComputeSLAInfo tests ---

func TestComputeSLAInfo_OnTrack(t *testing.T) {
	svc, slaRepo, _, _, _ := newTestSLAService()

	projectID := uuid.New()
	workflowID := uuid.New()
	itemID := uuid.New()

	// Set up target: 1 hour
	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     projectID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    workflowID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	// Set up elapsed: entered 10 minutes ago
	now := time.Now()
	tenMinAgo := now.Add(-10 * time.Minute)
	slaRepo.elapsed[elapsedKey(itemID, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "Open",
		ElapsedSeconds: 0,
		LastEnteredAt:  &tenMinAgo,
	}

	item := &model.WorkItem{
		ID:        itemID,
		ProjectID: projectID,
		Type:      model.WorkItemTypeTask,
		Status:    "Open",
	}

	info := svc.ComputeSLAInfo(context.Background(), item, workflowID, nil)
	if info == nil {
		t.Fatal("expected SLA info, got nil")
	}
	if info.Status != model.SLAStatusOnTrack {
		t.Fatalf("expected on_track, got %s", info.Status)
	}
	if info.Percentage >= 75 {
		t.Fatalf("expected percentage < 75, got %d", info.Percentage)
	}
}

func TestComputeSLAInfo_Warning(t *testing.T) {
	svc, slaRepo, _, _, _ := newTestSLAService()

	projectID := uuid.New()
	workflowID := uuid.New()
	itemID := uuid.New()

	// Set up target: 1 hour
	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     projectID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    workflowID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	// Set up elapsed: 50 minutes already accumulated, entered 1 minute ago = 51 min total (~85%)
	now := time.Now()
	oneMinAgo := now.Add(-1 * time.Minute)
	slaRepo.elapsed[elapsedKey(itemID, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "Open",
		ElapsedSeconds: 3000, // 50 minutes
		LastEnteredAt:  &oneMinAgo,
	}

	item := &model.WorkItem{
		ID:        itemID,
		ProjectID: projectID,
		Type:      model.WorkItemTypeTask,
		Status:    "Open",
	}

	info := svc.ComputeSLAInfo(context.Background(), item, workflowID, nil)
	if info == nil {
		t.Fatal("expected SLA info, got nil")
	}
	if info.Status != model.SLAStatusWarning {
		t.Fatalf("expected warning, got %s", info.Status)
	}
}

func TestComputeSLAInfo_Breached(t *testing.T) {
	svc, slaRepo, _, _, _ := newTestSLAService()

	projectID := uuid.New()
	workflowID := uuid.New()
	itemID := uuid.New()

	// Set up target: 1 hour
	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     projectID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    workflowID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	// Set up elapsed: 1 hour already accumulated + entered 5 minutes ago = breached
	now := time.Now()
	fiveMinAgo := now.Add(-5 * time.Minute)
	slaRepo.elapsed[elapsedKey(itemID, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "Open",
		ElapsedSeconds: 3600,
		LastEnteredAt:  &fiveMinAgo,
	}

	item := &model.WorkItem{
		ID:        itemID,
		ProjectID: projectID,
		Type:      model.WorkItemTypeTask,
		Status:    "Open",
	}

	info := svc.ComputeSLAInfo(context.Background(), item, workflowID, nil)
	if info == nil {
		t.Fatal("expected SLA info, got nil")
	}
	if info.Status != model.SLAStatusBreached {
		t.Fatalf("expected breached, got %s", info.Status)
	}
	if info.RemainingSeconds >= 0 {
		t.Fatalf("expected negative remaining, got %d", info.RemainingSeconds)
	}
}

func TestComputeSLAInfo_NoTarget(t *testing.T) {
	svc, _, _, _, _ := newTestSLAService()

	item := &model.WorkItem{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Type:      model.WorkItemTypeTask,
		Status:    "Open",
	}

	info := svc.ComputeSLAInfo(context.Background(), item, uuid.New(), nil)
	if info != nil {
		t.Fatal("expected nil for no SLA target")
	}
}

// --- CalculateBusinessSeconds tests ---

func TestCalculateBusinessSeconds_FullDay(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5}, // Mon-Fri
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}

	// Monday 9:00 to Monday 17:00 = 8 hours = 28800 seconds
	from := time.Date(2024, 1, 8, 9, 0, 0, 0, time.UTC)  // Monday
	to := time.Date(2024, 1, 8, 17, 0, 0, 0, time.UTC)    // Monday
	result := CalculateBusinessSeconds(from, to, config)

	if result != 28800 {
		t.Fatalf("expected 28800 seconds, got %d", result)
	}
}

func TestCalculateBusinessSeconds_PartialDay(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}

	// Monday 10:00 to Monday 14:00 = 4 hours = 14400 seconds
	from := time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 8, 14, 0, 0, 0, time.UTC)
	result := CalculateBusinessSeconds(from, to, config)

	if result != 14400 {
		t.Fatalf("expected 14400 seconds, got %d", result)
	}
}

func TestCalculateBusinessSeconds_SkipWeekend(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5}, // Mon-Fri
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}

	// Friday 9:00 to Monday 17:00 = 2 business days (Fri + Mon) = 57600 seconds
	from := time.Date(2024, 1, 5, 9, 0, 0, 0, time.UTC)  // Friday
	to := time.Date(2024, 1, 8, 17, 0, 0, 0, time.UTC)    // Monday
	result := CalculateBusinessSeconds(from, to, config)

	if result != 57600 {
		t.Fatalf("expected 57600 seconds (2 x 8h), got %d", result)
	}
}

func TestCalculateBusinessSeconds_BeforeBusinessHours(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}

	// Monday 6:00 to Monday 12:00 = 3 hours of business time (9:00-12:00) = 10800
	from := time.Date(2024, 1, 8, 6, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 8, 12, 0, 0, 0, time.UTC)
	result := CalculateBusinessSeconds(from, to, config)

	if result != 10800 {
		t.Fatalf("expected 10800 seconds, got %d", result)
	}
}

func TestCalculateBusinessSeconds_AfterBusinessHours(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}

	// Monday 19:00 to Tuesday 10:00 = 1 hour of business time (Tue 9:00-10:00) = 3600
	from := time.Date(2024, 1, 8, 19, 0, 0, 0, time.UTC)  // Monday evening
	to := time.Date(2024, 1, 9, 10, 0, 0, 0, time.UTC)     // Tuesday morning
	result := CalculateBusinessSeconds(from, to, config)

	if result != 3600 {
		t.Fatalf("expected 3600 seconds, got %d", result)
	}
}

func TestCalculateBusinessSeconds_ReversedTimes(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}

	from := time.Date(2024, 1, 9, 10, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC)
	result := CalculateBusinessSeconds(from, to, config)

	if result != 0 {
		t.Fatalf("expected 0 for reversed times, got %d", result)
	}
}

// --- ComputeSLAInfoBatch tests ---

func TestComputeSLAInfoBatch_MultipleItems(t *testing.T) {
	svc, slaRepo, _, _, _ := newTestSLAService()

	projectID := uuid.New()
	workflowID := uuid.New()
	itemID1 := uuid.New()
	itemID2 := uuid.New()

	// Set up target for task+Open
	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     projectID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    workflowID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	now := time.Now()
	tenMinAgo := now.Add(-10 * time.Minute)

	// Item 1: on track
	slaRepo.elapsed[elapsedKey(itemID1, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID1,
		StatusName:     "Open",
		ElapsedSeconds: 0,
		LastEnteredAt:  &tenMinAgo,
	}

	// Item 2: breached
	slaRepo.elapsed[elapsedKey(itemID2, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID2,
		StatusName:     "Open",
		ElapsedSeconds: 4000,
		LastEnteredAt:  &tenMinAgo,
	}

	items := []model.WorkItem{
		{ID: itemID1, ProjectID: projectID, Type: model.WorkItemTypeTask, Status: "Open"},
		{ID: itemID2, ProjectID: projectID, Type: model.WorkItemTypeTask, Status: "Open"},
	}

	result := svc.ComputeSLAInfoBatch(context.Background(), items, projectID, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[itemID1].Status != model.SLAStatusOnTrack {
		t.Fatalf("expected item1 on_track, got %s", result[itemID1].Status)
	}
	if result[itemID2].Status != model.SLAStatusBreached {
		t.Fatalf("expected item2 breached, got %s", result[itemID2].Status)
	}
}

func TestComputeSLAInfoBatch_EmptyItems(t *testing.T) {
	svc, _, _, _, _ := newTestSLAService()

	result := svc.ComputeSLAInfoBatch(context.Background(), nil, uuid.New(), nil)
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d items", len(result))
	}
}

// --- ComputeSLAForItems tests ---

func TestComputeSLAForItems_Success(t *testing.T) {
	svc, slaRepo, projectRepo, _, _ := newTestSLAService()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	workflowID := uuid.New()
	itemID := uuid.New()

	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    workflowID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	tenMinAgo := time.Now().Add(-10 * time.Minute)
	slaRepo.elapsed[elapsedKey(itemID, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "Open",
		ElapsedSeconds: 0,
		LastEnteredAt:  &tenMinAgo,
	}

	items := []model.WorkItem{
		{ID: itemID, ProjectID: project.ID, Type: model.WorkItemTypeTask, Status: "Open"},
	}

	result := svc.ComputeSLAForItems(context.Background(), "TEST", items)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	info := result[itemID]
	if info == nil {
		t.Fatal("expected SLA info for item")
	}
	if info.Status != model.SLAStatusOnTrack {
		t.Fatalf("expected on_track, got %s", info.Status)
	}
	if info.TargetSeconds != 3600 {
		t.Fatalf("expected target 3600, got %d", info.TargetSeconds)
	}
}

func TestComputeSLAForItems_ProjectNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestSLAService()

	items := []model.WorkItem{
		{ID: uuid.New(), ProjectID: uuid.New(), Type: model.WorkItemTypeTask, Status: "Open"},
	}

	result := svc.ComputeSLAForItems(context.Background(), "NONEXISTENT", items)
	if result != nil {
		t.Fatalf("expected nil result for unknown project, got %v", result)
	}
}

func TestComputeSLAForItems_EmptyItems(t *testing.T) {
	svc, _, _, _, _ := newTestSLAService()

	result := svc.ComputeSLAForItems(context.Background(), "TEST", nil)
	if result != nil {
		t.Fatalf("expected nil for empty items, got %v", result)
	}
}

func TestComputeSLAForItems_NoTargets(t *testing.T) {
	svc, _, projectRepo, _, _ := newTestSLAService()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	items := []model.WorkItem{
		{ID: uuid.New(), ProjectID: project.ID, Type: model.WorkItemTypeTask, Status: "Open"},
	}

	result := svc.ComputeSLAForItems(context.Background(), "TEST", items)
	if len(result) != 0 {
		t.Fatalf("expected empty map when no targets, got %d items", len(result))
	}
}

func TestComputeSLAForItems_WithBusinessHours(t *testing.T) {
	svc, slaRepo, projectRepo, _, _ := newTestSLAService()

	bh := &model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5}, // Mon-Fri
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST", BusinessHours: bh}
	projectRepo.Create(context.Background(), project)

	workflowID := uuid.New()
	itemID := uuid.New()

	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		WorkItemType:  model.WorkItemTypeTicket,
		WorkflowID:    workflowID,
		StatusName:    "new",
		TargetSeconds: 28800, // 8h
		CalendarMode:  model.CalendarModeBusinessHours,
	}

	tenMinAgo := time.Now().Add(-10 * time.Minute)
	slaRepo.elapsed[elapsedKey(itemID, "new")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "new",
		ElapsedSeconds: 0,
		LastEnteredAt:  &tenMinAgo,
	}

	items := []model.WorkItem{
		{ID: itemID, ProjectID: project.ID, Type: model.WorkItemTypeTicket, Status: "new"},
	}

	result := svc.ComputeSLAForItems(context.Background(), "TEST", items)
	info := result[itemID]
	if info == nil {
		t.Fatal("expected SLA info for item with business hours")
	}
	if info.TargetSeconds != 28800 {
		t.Fatalf("expected target 28800, got %d", info.TargetSeconds)
	}
}

func TestComputeSLAForItems_MultipleItemsMixedTypes(t *testing.T) {
	svc, slaRepo, projectRepo, _, _ := newTestSLAService()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	workflowID := uuid.New()
	taskID := uuid.New()
	bugID := uuid.New()

	// Only task type has SLA target
	slaRepo.targets[uuid.New()] = &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     project.ID,
		WorkItemType:  model.WorkItemTypeTask,
		WorkflowID:    workflowID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}

	tenMinAgo := time.Now().Add(-10 * time.Minute)
	slaRepo.elapsed[elapsedKey(taskID, "Open")] = &model.SLAElapsed{
		WorkItemID:     taskID,
		StatusName:     "Open",
		ElapsedSeconds: 0,
		LastEnteredAt:  &tenMinAgo,
	}

	items := []model.WorkItem{
		{ID: taskID, ProjectID: project.ID, Type: model.WorkItemTypeTask, Status: "Open"},
		{ID: bugID, ProjectID: project.ID, Type: model.WorkItemTypeBug, Status: "Open"},
	}

	result := svc.ComputeSLAForItems(context.Background(), "TEST", items)
	if result[taskID] == nil {
		t.Fatal("expected SLA info for task")
	}
	if result[bugID] != nil {
		t.Fatal("expected nil SLA info for bug (no target)")
	}
}

// --- AddBusinessSeconds tests ---

func TestAddBusinessSeconds_BasicSameDay(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5}, // Mon-Fri
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	// Monday 9:00 + 3600s = Monday 10:00
	from := time.Date(2025, 6, 2, 9, 0, 0, 0, time.UTC) // Monday
	result := AddBusinessSeconds(from, 3600, config)
	expected := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestAddBusinessSeconds_CrossDay(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	// Monday 16:00 + 7200s (2h) = 1h left on Monday + 1h on Tuesday = Tuesday 10:00
	from := time.Date(2025, 6, 2, 16, 0, 0, 0, time.UTC)
	result := AddBusinessSeconds(from, 7200, config)
	expected := time.Date(2025, 6, 3, 10, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestAddBusinessSeconds_SkipWeekend(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	// Friday 16:00 + 7200s (2h) = 1h left on Friday + skip Sat/Sun + 1h on Monday = Monday 10:00
	from := time.Date(2025, 5, 30, 16, 0, 0, 0, time.UTC) // Friday
	result := AddBusinessSeconds(from, 7200, config)
	expected := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC) // Monday
	if !result.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestAddBusinessSeconds_BeforeBusinessHours(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	// Monday 7:00 (before business hours) + 3600s = snaps to 9:00 + 1h = Monday 10:00
	from := time.Date(2025, 6, 2, 7, 0, 0, 0, time.UTC)
	result := AddBusinessSeconds(from, 3600, config)
	expected := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestAddBusinessSeconds_AfterBusinessHours(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	// Monday 18:00 (after business hours) + 3600s = advance to Tuesday 9:00 + 1h = Tuesday 10:00
	from := time.Date(2025, 6, 2, 18, 0, 0, 0, time.UTC)
	result := AddBusinessSeconds(from, 3600, config)
	expected := time.Date(2025, 6, 3, 10, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestAddBusinessSeconds_ZeroSeconds(t *testing.T) {
	config := model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	from := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	result := AddBusinessSeconds(from, 0, config)
	if !result.Equal(from) {
		t.Fatalf("expected %v, got %v", from, result)
	}
}

// --- ComputeSLATargetAt tests ---

func TestComputeSLATargetAt_24x7(t *testing.T) {
	repo := newInMemorySLARepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	workflowRepo := newMockWorkflowRepo()
	svc := NewSLAService(repo, projectRepo, memberRepo, workflowRepo)

	projectID := uuid.New()
	wfID := uuid.New()
	itemID := uuid.Must(uuid.NewV7())

	// Set up a target: 3600s for "Open" status
	target := &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     projectID,
		WorkItemType:  "task",
		WorkflowID:    wfID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}
	repo.targets[target.ID] = target

	// Set up elapsed record: 600s already elapsed
	now := time.Now()
	repo.elapsed[elapsedKey(itemID, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "Open",
		ElapsedSeconds: 600,
		LastEnteredAt:  &now,
	}

	item := &model.WorkItem{
		ID:        itemID,
		ProjectID: projectID,
		Type:      "task",
		Status:    "Open",
	}

	result := svc.ComputeSLATargetAt(context.Background(), item, wfID, nil)
	if result == nil {
		t.Fatal("expected non-nil SLA target time")
	}
	// Remaining = 3600 - 600 = 3000s, but live elapsed since LastEnteredAt
	// will have passed a tiny amount. Just check it's roughly 3000s from now.
	diff := result.Sub(time.Now()).Seconds()
	if diff < 2900 || diff > 3100 {
		t.Fatalf("expected ~3000s from now, got %.0fs", diff)
	}
}

func TestComputeSLATargetAt_NoTarget(t *testing.T) {
	repo := newInMemorySLARepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	workflowRepo := newMockWorkflowRepo()
	svc := NewSLAService(repo, projectRepo, memberRepo, workflowRepo)

	item := &model.WorkItem{
		ID:        uuid.Must(uuid.NewV7()),
		ProjectID: uuid.New(),
		Type:      "task",
		Status:    "Open",
	}

	result := svc.ComputeSLATargetAt(context.Background(), item, uuid.New(), nil)
	if result != nil {
		t.Fatal("expected nil for item with no SLA target")
	}
}

func TestComputeSLATargetAt_Breached(t *testing.T) {
	repo := newInMemorySLARepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	workflowRepo := newMockWorkflowRepo()
	svc := NewSLAService(repo, projectRepo, memberRepo, workflowRepo)

	projectID := uuid.New()
	wfID := uuid.New()
	itemID := uuid.Must(uuid.NewV7())

	target := &model.SLAStatusTarget{
		ID:            uuid.New(),
		ProjectID:     projectID,
		WorkItemType:  "task",
		WorkflowID:    wfID,
		StatusName:    "Open",
		TargetSeconds: 3600,
		CalendarMode:  model.CalendarMode24x7,
	}
	repo.targets[target.ID] = target

	// Already spent 4000s (breached by 400s)
	repo.elapsed[elapsedKey(itemID, "Open")] = &model.SLAElapsed{
		WorkItemID:     itemID,
		StatusName:     "Open",
		ElapsedSeconds: 4000,
	}

	item := &model.WorkItem{
		ID:        itemID,
		ProjectID: projectID,
		Type:      "task",
		Status:    "Open",
	}

	result := svc.ComputeSLATargetAt(context.Background(), item, wfID, nil)
	if result == nil {
		t.Fatal("expected non-nil SLA target time for breached item")
	}
	if !result.Before(time.Now()) {
		t.Fatal("expected target time to be in the past for breached SLA")
	}
}

func TestComputeSLATargetAtSimple_24x7(t *testing.T) {
	before := time.Now()
	result := ComputeSLATargetAtSimple(7200, model.CalendarMode24x7, nil)
	after := time.Now()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	expectedMin := before.Add(7200 * time.Second)
	expectedMax := after.Add(7200 * time.Second)
	if result.Before(expectedMin) || result.After(expectedMax) {
		t.Fatalf("expected result between %v and %v, got %v", expectedMin, expectedMax, *result)
	}
}

func TestComputeSLATargetAtSimple_BusinessHours(t *testing.T) {
	config := &model.BusinessHoursConfig{
		Days:      []int{1, 2, 3, 4, 5},
		StartHour: 9,
		EndHour:   17,
		Timezone:  "UTC",
	}
	result := ComputeSLATargetAtSimple(3600, model.CalendarModeBusinessHours, config)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Can't easily test exact time since it depends on current time, but verify it's in the future
	if !result.After(time.Now()) {
		t.Fatal("expected result to be in the future")
	}
}

func TestBulkUpsertTargets_BusinessHoursRequiresProjectConfig(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := adminAuthInfo()
	project := setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	// Project has no business hours configured (nil)
	project.BusinessHours = nil

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTicket,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 1800, CalendarMode: model.CalendarModeBusinessHours},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error when using business_hours without project config")
	}
}

func TestBulkUpsertTargets_BusinessHoursAllowedWithProjectConfig(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := adminAuthInfo()
	project := setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	project.BusinessHours = &model.BusinessHoursConfig{
		Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 17, Timezone: "UTC",
	}
	wf := setupTestWorkflow(t, workflowRepo)

	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTicket,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 1800, CalendarMode: model.CalendarModeBusinessHours},
		},
	}

	result, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 target, got %d", len(result))
	}
	if result[0].CalendarMode != model.CalendarModeBusinessHours {
		t.Fatalf("expected business_hours, got %s", result[0].CalendarMode)
	}
}

func TestBulkUpsertTargets_24x7AllowedWithoutProjectConfig(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := adminAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	// 24x7 mode should work even without business hours configured
	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTicket,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
		},
	}

	result, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err != nil {
		t.Fatalf("expected no error for 24x7 mode without business hours, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 target, got %d", len(result))
	}
}

func TestBulkUpsertTargets_MixedModesRequiresProjectConfig(t *testing.T) {
	svc, _, projectRepo, memberRepo, workflowRepo := newTestSLAService()
	info := adminAuthInfo()
	setupSLATestProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)
	wf := setupTestWorkflow(t, workflowRepo)

	// Mixed modes: one 24x7, one business_hours — should fail without project config
	input := BulkUpsertSLAInput{
		WorkItemType: model.WorkItemTypeTicket,
		WorkflowID:   wf.ID,
		Targets: []SLATargetInput{
			{StatusName: "Open", TargetSeconds: 3600, CalendarMode: model.CalendarMode24x7},
			{StatusName: "In Progress", TargetSeconds: 7200, CalendarMode: model.CalendarModeBusinessHours},
		},
	}

	_, err := svc.BulkUpsertTargets(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for business_hours without project config in mixed mode")
	}
}

func TestUpdateElapsedOnLeaveWithSeconds(t *testing.T) {
	repo := newInMemorySLARepo()
	itemID := uuid.New()
	status := "Open"
	entered := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC) // Friday 2pm

	// Init elapsed
	_ = repo.InitElapsedOnCreate(context.Background(), itemID, status, entered)

	// Leave with pre-computed business seconds (e.g., 3 hours of business time)
	err := repo.UpdateElapsedOnLeaveWithSeconds(context.Background(), itemID, status, 10800)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed, err := repo.GetElapsed(context.Background(), itemID, status)
	if err != nil {
		t.Fatalf("unexpected error getting elapsed: %v", err)
	}
	if elapsed.ElapsedSeconds != 10800 {
		t.Fatalf("expected 10800 elapsed seconds, got %d", elapsed.ElapsedSeconds)
	}
	if elapsed.LastEnteredAt != nil {
		t.Fatal("expected LastEnteredAt to be nil after leaving")
	}
}
