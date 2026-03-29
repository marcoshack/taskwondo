package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock InboxRepository ---

type mockInboxRepo struct {
	items      map[uuid.UUID]*model.InboxItem // keyed by inbox item ID
	projectIDs map[uuid.UUID]uuid.UUID        // work_item_id → project_id
}

func newMockInboxRepo() *mockInboxRepo {
	return &mockInboxRepo{
		items:      make(map[uuid.UUID]*model.InboxItem),
		projectIDs: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockInboxRepo) Add(_ context.Context, item *model.InboxItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockInboxRepo) Remove(_ context.Context, userID, workItemID uuid.UUID) error {
	for id, item := range m.items {
		if item.UserID == userID && item.WorkItemID == workItemID {
			delete(m.items, id)
			return nil
		}
	}
	return model.ErrNotFound
}

func (m *mockInboxRepo) RemoveByID(_ context.Context, id, userID uuid.UUID) error {
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return model.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockInboxRepo) List(_ context.Context, userID uuid.UUID, _ bool, _ string, _ []string, _ *uuid.UUID, _ *uuid.UUID, limit int) (*model.InboxItemList, error) {
	var items []model.InboxItemWithWorkItem
	for _, item := range m.items {
		if item.UserID == userID {
			items = append(items, model.InboxItemWithWorkItem{InboxItem: *item})
		}
	}
	return &model.InboxItemList{
		Items:   items,
		HasMore: false,
		Total:   len(items),
	}, nil
}

func (m *mockInboxRepo) CountByUser(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, item := range m.items {
		if item.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockInboxRepo) CountAllByUser(_ context.Context, userID uuid.UUID) (int, error) {
	return m.CountByUser(context.Background(), userID)
}

func (m *mockInboxRepo) UpdatePosition(_ context.Context, id, userID uuid.UUID, position int) error {
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return model.ErrNotFound
	}
	item.Position = position
	return nil
}

func (m *mockInboxRepo) MaxPosition(_ context.Context, userID uuid.UUID) (int, error) {
	maxPos := 0
	for _, item := range m.items {
		if item.UserID == userID && item.Position > maxPos {
			maxPos = item.Position
		}
	}
	return maxPos, nil
}

func (m *mockInboxRepo) RemoveCompleted(_ context.Context, userID uuid.UUID) (int, error) {
	// In mock, just return 0 — completion detection requires workflow JOIN
	return 0, nil
}

func (m *mockInboxRepo) Exists(_ context.Context, userID, workItemID uuid.UUID) (bool, error) {
	for _, item := range m.items {
		if item.UserID == userID && item.WorkItemID == workItemID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockInboxRepo) GetWorkItemProjectID(_ context.Context, workItemID uuid.UUID) (uuid.UUID, error) {
	pid, ok := m.projectIDs[workItemID]
	if !ok {
		return uuid.Nil, model.ErrNotFound
	}
	return pid, nil
}

func (m *mockInboxRepo) RemoveByProjectID(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// --- Helpers ---

func newTestInboxService() (*InboxService, *mockInboxRepo, *mockProjectMemberRepo) {
	inboxRepo := newMockInboxRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewInboxService(inboxRepo, memberRepo)
	return svc, inboxRepo, memberRepo
}

func setupInboxProject(t *testing.T, inboxRepo *mockInboxRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	projectID := uuid.New()
	workItemID := uuid.New()

	// Register work item → project mapping
	inboxRepo.projectIDs[workItemID] = projectID

	// Add user as project member
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: projectID,
		UserID:    info.UserID,
		Role:      role,
	})

	return projectID, workItemID
}

// --- Tests ---

func TestInboxAdd_Success(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	_, workItemID := setupInboxProject(t, inboxRepo, memberRepo, info, model.ProjectRoleMember)

	err := svc.Add(context.Background(), info, workItemID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify item was added
	count, _ := inboxRepo.CountByUser(context.Background(), info.UserID)
	if count != 1 {
		t.Fatalf("expected 1 inbox item, got %d", count)
	}
}

func TestInboxAdd_Position(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	projectID := uuid.New()

	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: projectID, UserID: info.UserID, Role: model.ProjectRoleMember,
	})

	// Add 3 items and verify positions
	for i := 0; i < 3; i++ {
		wiID := uuid.New()
		inboxRepo.projectIDs[wiID] = projectID
		if err := svc.Add(context.Background(), info, wiID); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}

	// Check positions are spaced by InboxPositionGap
	positions := make(map[int]bool)
	for _, item := range inboxRepo.items {
		if item.UserID == info.UserID {
			positions[item.Position] = true
		}
	}
	for _, expected := range []int{1000, 2000, 3000} {
		if !positions[expected] {
			t.Fatalf("expected position %d, got positions: %v", expected, positions)
		}
	}
}

func TestInboxAdd_AlreadyExists(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	_, workItemID := setupInboxProject(t, inboxRepo, memberRepo, info, model.ProjectRoleMember)

	// First add succeeds
	if err := svc.Add(context.Background(), info, workItemID); err != nil {
		t.Fatalf("first add: %v", err)
	}

	// Second add should fail
	err := svc.Add(context.Background(), info, workItemID)
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestInboxAdd_NonMember(t *testing.T) {
	svc, inboxRepo, _ := newTestInboxService()
	info := userAuthInfo()

	workItemID := uuid.New()
	inboxRepo.projectIDs[workItemID] = uuid.New() // project exists but user is not a member

	err := svc.Add(context.Background(), info, workItemID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

func TestInboxAdd_WorkItemNotFound(t *testing.T) {
	svc, _, _ := newTestInboxService()
	info := userAuthInfo()

	err := svc.Add(context.Background(), info, uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing work item, got %v", err)
	}
}

func TestInboxAdd_LimitExceeded(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	projectID := uuid.New()

	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: projectID, UserID: info.UserID, Role: model.ProjectRoleMember,
	})

	// Fill inbox to the limit
	for i := 0; i < model.MaxInboxItems; i++ {
		wiID := uuid.New()
		inboxRepo.projectIDs[wiID] = projectID
		inboxRepo.items[uuid.New()] = &model.InboxItem{
			ID: uuid.New(), UserID: info.UserID, WorkItemID: wiID, Position: (i + 1) * model.InboxPositionGap,
		}
	}

	// Adding one more should fail
	extraWI := uuid.New()
	inboxRepo.projectIDs[extraWI] = projectID
	err := svc.Add(context.Background(), info, extraWI)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for limit exceeded, got %v", err)
	}
}

func TestInboxAdd_AdminBypass(t *testing.T) {
	svc, inboxRepo, _ := newTestInboxService()
	admin := adminAuthInfo()

	workItemID := uuid.New()
	inboxRepo.projectIDs[workItemID] = uuid.New()

	// Admin should be able to add without being a project member
	err := svc.Add(context.Background(), admin, workItemID)
	if err != nil {
		t.Fatalf("expected admin to bypass membership check, got %v", err)
	}
}

func TestInboxRemove_Success(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	_, workItemID := setupInboxProject(t, inboxRepo, memberRepo, info, model.ProjectRoleMember)

	if err := svc.Add(context.Background(), info, workItemID); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Find the inbox item ID
	var inboxItemID uuid.UUID
	for id, item := range inboxRepo.items {
		if item.WorkItemID == workItemID {
			inboxItemID = id
			break
		}
	}

	err := svc.Remove(context.Background(), info, inboxItemID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	count, _ := inboxRepo.CountByUser(context.Background(), info.UserID)
	if count != 0 {
		t.Fatalf("expected 0 items, got %d", count)
	}
}

func TestInboxRemove_NotFound(t *testing.T) {
	svc, _, _ := newTestInboxService()
	info := userAuthInfo()

	err := svc.Remove(context.Background(), info, uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestInboxRemove_WrongUser(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	_, workItemID := setupInboxProject(t, inboxRepo, memberRepo, info, model.ProjectRoleMember)

	if err := svc.Add(context.Background(), info, workItemID); err != nil {
		t.Fatalf("add: %v", err)
	}

	var inboxItemID uuid.UUID
	for id := range inboxRepo.items {
		inboxItemID = id
		break
	}

	// Another user tries to remove
	otherUser := userAuthInfo()
	err := svc.Remove(context.Background(), otherUser, inboxItemID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong user, got %v", err)
	}
}

func TestInboxList_Success(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	projectID := uuid.New()

	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: projectID, UserID: info.UserID, Role: model.ProjectRoleMember,
	})

	for i := 0; i < 3; i++ {
		wiID := uuid.New()
		inboxRepo.projectIDs[wiID] = projectID
		if err := svc.Add(context.Background(), info, wiID); err != nil {
			t.Fatalf("add: %v", err)
		}
	}

	list, err := svc.List(context.Background(), info, false, "", nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if list.Total != 3 {
		t.Fatalf("expected 3 items, got %d", list.Total)
	}
}

func TestInboxList_IsolatedPerUser(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	user1 := userAuthInfo()
	user2 := userAuthInfo()
	projectID := uuid.New()

	for _, u := range []*model.AuthInfo{user1, user2} {
		memberRepo.Add(context.Background(), &model.ProjectMember{
			ID: uuid.New(), ProjectID: projectID, UserID: u.UserID, Role: model.ProjectRoleMember,
		})
	}

	// User1 adds 2 items
	for i := 0; i < 2; i++ {
		wiID := uuid.New()
		inboxRepo.projectIDs[wiID] = projectID
		svc.Add(context.Background(), user1, wiID)
	}

	// User2 adds 1 item
	wiID := uuid.New()
	inboxRepo.projectIDs[wiID] = projectID
	svc.Add(context.Background(), user2, wiID)

	list1, _ := svc.List(context.Background(), user1, false, "", nil, nil, nil, 50)
	list2, _ := svc.List(context.Background(), user2, false, "", nil, nil, nil, 50)

	if list1.Total != 2 {
		t.Fatalf("user1: expected 2, got %d", list1.Total)
	}
	if list2.Total != 1 {
		t.Fatalf("user2: expected 1, got %d", list2.Total)
	}
}

func TestInboxReorder_Success(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	_, workItemID := setupInboxProject(t, inboxRepo, memberRepo, info, model.ProjectRoleMember)

	svc.Add(context.Background(), info, workItemID)

	var inboxItemID uuid.UUID
	for id := range inboxRepo.items {
		inboxItemID = id
		break
	}

	err := svc.Reorder(context.Background(), info, inboxItemID, 500)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if inboxRepo.items[inboxItemID].Position != 500 {
		t.Fatalf("expected position 500, got %d", inboxRepo.items[inboxItemID].Position)
	}
}

func TestInboxReorder_NegativePosition(t *testing.T) {
	svc, _, _ := newTestInboxService()
	info := userAuthInfo()

	err := svc.Reorder(context.Background(), info, uuid.New(), -1)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for negative position, got %v", err)
	}
}

func TestInboxCount_Success(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	projectID := uuid.New()

	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: projectID, UserID: info.UserID, Role: model.ProjectRoleMember,
	})

	for i := 0; i < 5; i++ {
		wiID := uuid.New()
		inboxRepo.projectIDs[wiID] = projectID
		svc.Add(context.Background(), info, wiID)
	}

	count, err := svc.Count(context.Background(), info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5, got %d", count)
	}
}

func TestInboxClearCompleted_Success(t *testing.T) {
	svc, _, _ := newTestInboxService()
	info := userAuthInfo()

	// ClearCompleted with empty inbox should return 0
	count, err := svc.ClearCompleted(context.Background(), info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 removed, got %d", count)
	}
}

func TestInboxRemoveByWorkItem_Success(t *testing.T) {
	svc, inboxRepo, memberRepo := newTestInboxService()
	info := userAuthInfo()
	_, workItemID := setupInboxProject(t, inboxRepo, memberRepo, info, model.ProjectRoleMember)

	svc.Add(context.Background(), info, workItemID)

	err := svc.RemoveByWorkItem(context.Background(), info, workItemID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	count, _ := inboxRepo.CountByUser(context.Background(), info.UserID)
	if count != 0 {
		t.Fatalf("expected 0 items, got %d", count)
	}
}
