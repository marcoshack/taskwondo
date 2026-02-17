package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
)

// --- Mock work item repository ---

type mockWorkItemRepo struct {
	items       map[uuid.UUID]*model.WorkItem
	byProjectNum map[string]*model.WorkItem // key: "projectID:itemNumber"
	counters    map[uuid.UUID]int           // project item counters
}

func newMockWorkItemRepo() *mockWorkItemRepo {
	return &mockWorkItemRepo{
		items:        make(map[uuid.UUID]*model.WorkItem),
		byProjectNum: make(map[string]*model.WorkItem),
		counters:     make(map[uuid.UUID]int),
	}
}

func itemKey(projectID uuid.UUID, itemNumber int) string {
	return projectID.String() + ":" + itoa(itemNumber)
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func (m *mockWorkItemRepo) Create(_ context.Context, item *model.WorkItem) error {
	m.counters[item.ProjectID]++
	item.ItemNumber = m.counters[item.ProjectID]
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	m.items[item.ID] = item
	m.byProjectNum[itemKey(item.ProjectID, item.ItemNumber)] = item
	return nil
}

func (m *mockWorkItemRepo) GetByProjectAndNumber(_ context.Context, projectID uuid.UUID, itemNumber int) (*model.WorkItem, error) {
	key := itemKey(projectID, itemNumber)
	item, ok := m.byProjectNum[key]
	if !ok || item.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	return item, nil
}

func (m *mockWorkItemRepo) List(_ context.Context, projectID uuid.UUID, filter *model.WorkItemFilter) (*model.WorkItemList, error) {
	var matched []model.WorkItem
	for _, item := range m.items {
		if item.ProjectID != projectID || item.DeletedAt != nil {
			continue
		}
		if len(filter.Types) > 0 && !contains(filter.Types, item.Type) {
			continue
		}
		if len(filter.Statuses) > 0 && !contains(filter.Statuses, item.Status) {
			continue
		}
		if len(filter.Priorities) > 0 && !contains(filter.Priorities, item.Priority) {
			continue
		}
		if filter.Unassigned && item.AssigneeID != nil {
			continue
		}
		if filter.AssigneeID != nil && (item.AssigneeID == nil || *item.AssigneeID != *filter.AssigneeID) {
			continue
		}
		matched = append(matched, *item)
	}

	total := len(matched)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	hasMore := len(matched) > limit
	if hasMore {
		matched = matched[:limit]
	}

	var cursor string
	if len(matched) > 0 {
		cursor = matched[len(matched)-1].ID.String()
	}

	return &model.WorkItemList{
		Items:   matched,
		Cursor:  cursor,
		HasMore: hasMore,
		Total:   total,
	}, nil
}

func (m *mockWorkItemRepo) Update(_ context.Context, item *model.WorkItem) error {
	existing, ok := m.items[item.ID]
	if !ok || existing.DeletedAt != nil {
		return model.ErrNotFound
	}
	now := time.Now()
	item.UpdatedAt = now
	m.items[item.ID] = item
	m.byProjectNum[itemKey(item.ProjectID, item.ItemNumber)] = item
	return nil
}

func (m *mockWorkItemRepo) Delete(_ context.Context, id uuid.UUID) error {
	item, ok := m.items[id]
	if !ok || item.DeletedAt != nil {
		return model.ErrNotFound
	}
	now := time.Now()
	item.DeletedAt = &now
	return nil
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- Mock work item event repository ---

type mockWorkItemEventRepo struct {
	events map[uuid.UUID][]model.WorkItemEvent // keyed by work_item_id
}

func newMockWorkItemEventRepo() *mockWorkItemEventRepo {
	return &mockWorkItemEventRepo{
		events: make(map[uuid.UUID][]model.WorkItemEvent),
	}
}

func (m *mockWorkItemEventRepo) Create(_ context.Context, event *model.WorkItemEvent) error {
	event.CreatedAt = time.Now()
	m.events[event.WorkItemID] = append(m.events[event.WorkItemID], *event)
	return nil
}

func (m *mockWorkItemEventRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID) ([]model.WorkItemEvent, error) {
	return m.events[workItemID], nil
}

// --- Test helpers ---

func newTestWorkItemService() (*WorkItemService, *mockWorkItemRepo, *mockWorkItemEventRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	itemRepo := newMockWorkItemRepo()
	eventRepo := newMockWorkItemEventRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewWorkItemService(itemRepo, eventRepo, projectRepo, memberRepo)
	return svc, itemRepo, eventRepo, projectRepo, memberRepo
}

func setupProjectWithMember(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
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

func validCreateInput() CreateWorkItemInput {
	return CreateWorkItemInput{
		Type:     model.WorkItemTypeTask,
		Title:    "Test work item",
		Priority: model.PriorityMedium,
	}
}

// --- Tests ---

func TestCreateWorkItem_Success(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	item, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.ItemNumber != 1 {
		t.Fatalf("expected item_number 1, got %d", item.ItemNumber)
	}
	if item.Title != "Test work item" {
		t.Fatalf("expected title 'Test work item', got %s", item.Title)
	}
	if item.Type != model.WorkItemTypeTask {
		t.Fatalf("expected type 'task', got %s", item.Type)
	}
	if item.Status != "open" {
		t.Fatalf("expected status 'open', got %s", item.Status)
	}
	if item.ReporterID != info.UserID {
		t.Fatalf("expected reporter_id %s, got %s", info.UserID, item.ReporterID)
	}
}

func TestCreateWorkItem_SequentialNumbers(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	item1, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	item2, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	if item1.ItemNumber != 1 || item2.ItemNumber != 2 {
		t.Fatalf("expected sequential numbers 1,2 got %d,%d", item1.ItemNumber, item2.ItemNumber)
	}
}

func TestCreateWorkItem_MissingTitle(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateInput()
	input.Title = ""
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestCreateWorkItem_InvalidType(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateInput()
	input.Type = "invalid"
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestCreateWorkItem_InvalidPriority(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateInput()
	input.Priority = "urgent"
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestCreateWorkItem_DefaultValues(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := CreateWorkItemInput{
		Type:  model.WorkItemTypeBug,
		Title: "A bug",
	}
	item, err := svc.Create(context.Background(), info, "TEST", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Priority != model.PriorityMedium {
		t.Fatalf("expected default priority 'medium', got %s", item.Priority)
	}
	if item.Visibility != model.VisibilityInternal {
		t.Fatalf("expected default visibility 'internal', got %s", item.Visibility)
	}
	if item.Status != "open" {
		t.Fatalf("expected default status 'open', got %s", item.Status)
	}
}

func TestCreateWorkItem_NonMemberDenied(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	owner := userAuthInfo()
	other := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, owner, model.ProjectRoleOwner)

	_, err := svc.Create(context.Background(), other, "TEST", validCreateInput())
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

func TestCreateWorkItem_ViewerDenied(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleViewer)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden for viewer, got %v", err)
	}
}

func TestCreateWorkItem_GlobalAdminAllowed(t *testing.T) {
	svc, _, _, projectRepo, _ := newTestWorkItemService()
	admin := adminAuthInfo()

	// Create project without adding admin as member
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	item, err := svc.Create(context.Background(), admin, "TEST", validCreateInput())
	if err != nil {
		t.Fatalf("expected global admin to create item, got %v", err)
	}
	if item.ItemNumber != 1 {
		t.Fatalf("expected item_number 1, got %d", item.ItemNumber)
	}
}

func TestCreateWorkItem_CreatesEvent(t *testing.T) {
	svc, _, eventRepo, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	item, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	events := eventRepo.events[item.ID]
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "created" {
		t.Fatalf("expected event type 'created', got %s", events[0].EventType)
	}
}

func TestGetWorkItem_Success(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	item, err := svc.Get(context.Background(), info, "TEST", created.ItemNumber)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.ID != created.ID {
		t.Fatalf("expected item ID %s, got %s", created.ID, item.ID)
	}
}

func TestGetWorkItem_NotFound(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	_, err := svc.Get(context.Background(), info, "TEST", 999)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetWorkItem_NonMemberDenied(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	owner := userAuthInfo()
	other := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, owner, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), owner, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Get(context.Background(), other, "TEST", created.ItemNumber)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

func TestListWorkItems_Empty(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	result, err := svc.List(context.Background(), info, "TEST", &model.WorkItemFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

func TestListWorkItems_WithItems(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
		if err != nil {
			t.Fatal(err)
		}
	}

	result, err := svc.List(context.Background(), info, "TEST", &model.WorkItemFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}
}

func TestListWorkItems_FilterByType(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	taskInput := validCreateInput()
	taskInput.Type = model.WorkItemTypeTask
	svc.Create(context.Background(), info, "TEST", taskInput)

	bugInput := validCreateInput()
	bugInput.Type = model.WorkItemTypeBug
	svc.Create(context.Background(), info, "TEST", bugInput)

	result, err := svc.List(context.Background(), info, "TEST", &model.WorkItemFilter{
		Types: []string{model.WorkItemTypeBug},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 bug, got %d items", len(result.Items))
	}
	if result.Items[0].Type != model.WorkItemTypeBug {
		t.Fatalf("expected type 'bug', got %s", result.Items[0].Type)
	}
}

func TestListWorkItems_FilterByStatus(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// All items start as "open"
	result, err := svc.List(context.Background(), info, "TEST", &model.WorkItemFilter{
		Statuses: []string{"closed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items with status 'closed', got %d", len(result.Items))
	}

	result, err = svc.List(context.Background(), info, "TEST", &model.WorkItemFilter{
		Statuses: []string{"open"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item with status 'open', got %d", len(result.Items))
	}
}

func TestListWorkItems_Pagination(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	for i := 0; i < 5; i++ {
		_, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
		if err != nil {
			t.Fatal(err)
		}
	}

	result, err := svc.List(context.Background(), info, "TEST", &model.WorkItemFilter{
		Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	if !result.HasMore {
		t.Fatal("expected has_more to be true")
	}
	if result.Total != 5 {
		t.Fatalf("expected total 5, got %d", result.Total)
	}
}

func TestUpdateWorkItem_Success(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Updated title"
	newStatus := "in_progress"
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Title:  &newTitle,
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Title != "Updated title" {
		t.Fatalf("expected title 'Updated title', got %s", updated.Title)
	}
	if updated.Status != "in_progress" {
		t.Fatalf("expected status 'in_progress', got %s", updated.Status)
	}
}

func TestUpdateWorkItem_RecordsEvents(t *testing.T) {
	svc, _, eventRepo, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Updated title"
	newStatus := "in_progress"
	_, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Title:  &newTitle,
		Status: &newStatus,
	})
	if err != nil {
		t.Fatal(err)
	}

	events := eventRepo.events[created.ID]
	// 1 "created" event + 2 field change events
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Check event types (order: created, title_updated, status_changed)
	expectedTypes := []string{"created", "title_updated", "status_changed"}
	for i, expected := range expectedTypes {
		if events[i].EventType != expected {
			t.Fatalf("expected event[%d] type %q, got %q", i, expected, events[i].EventType)
		}
	}
}

func TestUpdateWorkItem_ViewerDenied(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	owner := userAuthInfo()
	viewer := userAuthInfo()
	project := setupProjectWithMember(t, projectRepo, memberRepo, owner, model.ProjectRoleOwner)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    viewer.UserID,
		Role:      model.ProjectRoleViewer,
	})

	created, err := svc.Create(context.Background(), owner, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Hacked"
	_, err = svc.Update(context.Background(), viewer, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Title: &newTitle,
	})
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden for viewer, got %v", err)
	}
}

func TestUpdateWorkItem_InvalidPriority(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	badPriority := "urgent"
	_, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Priority: &badPriority,
	})
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestDeleteWorkItem_Success(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(context.Background(), info, "TEST", created.ItemNumber)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not be found after deletion
	_, err = svc.Get(context.Background(), info, "TEST", created.ItemNumber)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteWorkItem_NotFound(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	err := svc.Delete(context.Background(), info, "TEST", 999)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteWorkItem_ViewerDenied(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	owner := userAuthInfo()
	viewer := userAuthInfo()
	project := setupProjectWithMember(t, projectRepo, memberRepo, owner, model.ProjectRoleOwner)
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    viewer.UserID,
		Role:      model.ProjectRoleViewer,
	})

	created, err := svc.Create(context.Background(), owner, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(context.Background(), viewer, "TEST", created.ItemNumber)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden for viewer, got %v", err)
	}
}
