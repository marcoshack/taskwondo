package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock queue category repository ---

type mockQueueCategoryRepo struct {
	categories map[uuid.UUID]*model.QueueCategory
}

func newMockQueueCategoryRepo() *mockQueueCategoryRepo {
	return &mockQueueCategoryRepo{categories: make(map[uuid.UUID]*model.QueueCategory)}
}

func (m *mockQueueCategoryRepo) Create(_ context.Context, cat *model.QueueCategory) error {
	now := time.Now()
	cat.CreatedAt = now
	cat.UpdatedAt = now
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockQueueCategoryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.QueueCategory, error) {
	cat, ok := m.categories[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return cat, nil
}

func (m *mockQueueCategoryRepo) ListByQueue(_ context.Context, queueID uuid.UUID) ([]model.QueueCategory, error) {
	var result []model.QueueCategory
	for _, cat := range m.categories {
		if cat.QueueID == queueID {
			result = append(result, *cat)
		}
	}
	return result, nil
}

func (m *mockQueueCategoryRepo) Update(_ context.Context, cat *model.QueueCategory) error {
	if _, ok := m.categories[cat.ID]; !ok {
		return model.ErrNotFound
	}
	cat.UpdatedAt = time.Now()
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockQueueCategoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.categories[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.categories, id)
	return nil
}

// --- Mock queue team repository ---

type mockQueueTeamRepo struct {
	assignments map[string]bool // "queueID:teamID"
	teams       map[uuid.UUID]*model.Team
}

func newMockQueueTeamRepo() *mockQueueTeamRepo {
	return &mockQueueTeamRepo{
		assignments: make(map[string]bool),
		teams:       make(map[uuid.UUID]*model.Team),
	}
}

func qtKey(queueID, teamID uuid.UUID) string {
	return queueID.String() + ":" + teamID.String()
}

func (m *mockQueueTeamRepo) Assign(_ context.Context, queueID, teamID uuid.UUID) error {
	m.assignments[qtKey(queueID, teamID)] = true
	return nil
}

func (m *mockQueueTeamRepo) Unassign(_ context.Context, queueID, teamID uuid.UUID) error {
	key := qtKey(queueID, teamID)
	if !m.assignments[key] {
		return model.ErrNotFound
	}
	delete(m.assignments, key)
	return nil
}

func (m *mockQueueTeamRepo) ListTeamsByQueue(_ context.Context, queueID uuid.UUID) ([]model.Team, error) {
	var result []model.Team
	for key := range m.assignments {
		qID := key[:36]
		tID := key[37:]
		if qID == queueID.String() {
			teamID, _ := uuid.Parse(tID)
			if team, ok := m.teams[teamID]; ok {
				result = append(result, *team)
			}
		}
	}
	return result, nil
}

func (m *mockQueueTeamRepo) ListQueuesByTeam(_ context.Context, _ uuid.UUID) ([]model.Queue, error) {
	return nil, nil
}

func (m *mockQueueTeamRepo) AddTeam(team *model.Team) {
	m.teams[team.ID] = team
}

func newTestQueueService() (*QueueService, *mockQueueRepo, *mockProjectRepo, *mockProjectMemberRepo, *mockQueueCategoryRepo, *mockQueueTeamRepo) {
	queueRepo := newMockQueueRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	categoryRepo := newMockQueueCategoryRepo()
	queueTeamRepo := newMockQueueTeamRepo()
	svc := NewQueueService(queueRepo, categoryRepo, queueTeamRepo, projectRepo, memberRepo)
	return svc, queueRepo, projectRepo, memberRepo, categoryRepo, queueTeamRepo
}

func setupQueueProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
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

func validCreateQueueInput() CreateQueueInput {
	return CreateQueueInput{
		Name:            "Support Queue",
		QueueType:       model.QueueTypeSupport,
		DefaultPriority: model.PriorityMedium,
	}
}

// --- Tests ---

func TestQueueCreate_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q, err := svc.Create(context.Background(), info, "TEST", validCreateQueueInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if q.Name != "Support Queue" {
		t.Fatalf("expected name 'Support Queue', got %s", q.Name)
	}
	if q.QueueType != model.QueueTypeSupport {
		t.Fatalf("expected queue type 'support', got %s", q.QueueType)
	}
}

func TestQueueCreate_EmptyName(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateQueueInput()
	input.Name = ""
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestQueueCreate_InvalidType(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateQueueInput()
	input.QueueType = "invalid"
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected validation error for invalid type")
	}
}

func TestQueueCreate_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateQueueInput())
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestQueueCreate_AdminAllowed(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateQueueInput())
	if err != nil {
		t.Fatalf("expected no error for admin, got %v", err)
	}
}

func TestQueueGet_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	q := &model.Queue{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Test Queue",
		QueueType: model.QueueTypeGeneral,
	}
	queueRepo.Create(context.Background(), q)

	result, err := svc.Get(context.Background(), info, "TEST", q.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Test Queue" {
		t.Fatalf("expected name 'Test Queue', got %s", result.Name)
	}
}

func TestQueueGet_NotFound(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Get(context.Background(), info, "TEST", uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestQueueGet_WrongProject(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	q := &model.Queue{
		ID:        uuid.New(),
		ProjectID: uuid.New(), // different project
		Name:      "Other Queue",
		QueueType: model.QueueTypeGeneral,
	}
	queueRepo.Create(context.Background(), q)

	_, err := svc.Get(context.Background(), info, "TEST", q.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong project, got %v", err)
	}
}

func TestQueueList_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	queueRepo.Create(context.Background(), &model.Queue{
		ID: uuid.New(), ProjectID: project.ID, Name: "Q1", QueueType: model.QueueTypeSupport,
	})
	queueRepo.Create(context.Background(), &model.Queue{
		ID: uuid.New(), ProjectID: project.ID, Name: "Q2", QueueType: model.QueueTypeAlerts,
	})

	queues, err := svc.List(context.Background(), info, "TEST")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(queues) != 2 {
		t.Fatalf("expected 2 queues, got %d", len(queues))
	}
}

func TestQueueUpdate_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Old Name",
		QueueType:       model.QueueTypeGeneral,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), q)

	newName := "New Name"
	updated, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("expected name 'New Name', got %s", updated.Name)
	}
}

func TestQueueDelete_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Delete Me",
		QueueType: model.QueueTypeGeneral,
	}
	queueRepo.Create(context.Background(), q)

	err := svc.Delete(context.Background(), info, "TEST", q.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = queueRepo.GetByID(context.Background(), q.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected queue to be deleted")
	}
}

func TestQueueDelete_MemberForbidden(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	q := &model.Queue{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Queue",
		QueueType: model.QueueTypeGeneral,
	}
	queueRepo.Create(context.Background(), q)

	err := svc.Delete(context.Background(), info, "TEST", q.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestQueueCreate_InvalidPriority(t *testing.T) {
	svc, _, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateQueueInput()
	input.DefaultPriority = "urgent"
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if err == nil {
		t.Fatal("expected validation error for invalid priority")
	}
}

func TestQueueCreate_NonMemberNotFound(t *testing.T) {
	svc, _, projectRepo, _, _, _ := newTestQueueService()
	info := userAuthInfo()
	// Create project but don't add user as member
	projectRepo.Create(context.Background(), &model.Project{
		ID:   uuid.New(),
		Name: "Test Project",
		Key:  "TEST",
	})

	_, err := svc.Create(context.Background(), info, "TEST", validCreateQueueInput())
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-member, got %v", err)
	}
}

func TestQueueUpdate_AllFields(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Original",
		QueueType:       model.QueueTypeGeneral,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), q)

	newName := "Updated"
	desc := "A description"
	newType := model.QueueTypeSupport
	isPublic := true
	newPriority := model.PriorityHigh
	assignee := uuid.New()
	wfID := uuid.New()

	updated, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		Name:              &newName,
		Description:       &desc,
		QueueType:         &newType,
		IsPublic:          &isPublic,
		DefaultPriority:   &newPriority,
		DefaultAssigneeID: &assignee,
		WorkflowID:        &wfID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %s", updated.Name)
	}
	if updated.QueueType != model.QueueTypeSupport {
		t.Fatalf("expected type support, got %s", updated.QueueType)
	}
	if !updated.IsPublic {
		t.Fatal("expected IsPublic true")
	}
	if updated.DefaultPriority != model.PriorityHigh {
		t.Fatalf("expected priority high, got %s", updated.DefaultPriority)
	}
}

func TestQueueUpdate_ClearFields(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	desc := "some desc"
	assignee := uuid.New()
	wfID := uuid.New()
	q := &model.Queue{
		ID:                uuid.New(),
		ProjectID:         project.ID,
		Name:              "Queue",
		Description:       &desc,
		QueueType:         model.QueueTypeGeneral,
		DefaultPriority:   model.PriorityMedium,
		DefaultAssigneeID: &assignee,
		WorkflowID:        &wfID,
	}
	queueRepo.Create(context.Background(), q)

	updated, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		ClearDescription:     true,
		ClearDefaultAssignee: true,
		ClearWorkflow:        true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Description != nil {
		t.Fatal("expected description to be cleared")
	}
	if updated.DefaultAssigneeID != nil {
		t.Fatal("expected default assignee to be cleared")
	}
	if updated.WorkflowID != nil {
		t.Fatal("expected workflow to be cleared")
	}
}

func TestQueueUpdate_EmptyName(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Queue",
		QueueType:       model.QueueTypeGeneral,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), q)

	empty := "   "
	_, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		Name: &empty,
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestQueueUpdate_InvalidType(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Queue",
		QueueType:       model.QueueTypeGeneral,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), q)

	badType := "invalid"
	_, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		QueueType: &badType,
	})
	if err == nil {
		t.Fatal("expected error for invalid queue type")
	}
}

func TestQueueUpdate_InvalidPriority(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	project := setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Queue",
		QueueType:       model.QueueTypeGeneral,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), q)

	badPriority := "urgent"
	_, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		DefaultPriority: &badPriority,
	})
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestQueueUpdate_WrongProject(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	q := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       uuid.New(), // different project
		Name:            "Queue",
		QueueType:       model.QueueTypeGeneral,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), q)

	newName := "Hacked"
	_, err := svc.Update(context.Background(), info, "TEST", q.ID, UpdateQueueInput{
		Name: &newName,
	})
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestQueueList_AdminBypass(t *testing.T) {
	svc, queueRepo, projectRepo, _, _, _ := newTestQueueService()
	admin := adminAuthInfo()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	queueRepo.Create(context.Background(), &model.Queue{
		ID: uuid.New(), ProjectID: project.ID, Name: "Q1", QueueType: model.QueueTypeGeneral,
	})

	queues, err := svc.List(context.Background(), admin, "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queues) != 1 {
		t.Fatalf("expected 1 queue, got %d", len(queues))
	}
}

// --- Category Tests ---

func setupQueueForCategory(t *testing.T, svc *QueueService, queueRepo *mockQueueRepo, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) (*model.Project, *model.Queue) {
	t.Helper()
	project := setupQueueProject(t, projectRepo, memberRepo, info, role)
	q := &model.Queue{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Test Queue",
		QueueType: model.QueueTypeGeneral,
	}
	queueRepo.Create(context.Background(), q)
	return project, q
}

func TestQueueCreateCategory_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	cat, err := svc.CreateCategory(context.Background(), info, "TEST", q.ID, CreateCategoryInput{
		Name:     "Bug Reports",
		Position: 1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cat.Name != "Bug Reports" {
		t.Fatalf("expected name 'Bug Reports', got %s", cat.Name)
	}
	if cat.QueueID != q.ID {
		t.Fatalf("expected queue_id %s, got %s", q.ID, cat.QueueID)
	}
	if cat.Position != 1 {
		t.Fatalf("expected position 1, got %d", cat.Position)
	}
}

func TestQueueCreateCategory_EmptyName(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	_, err := svc.CreateCategory(context.Background(), info, "TEST", q.ID, CreateCategoryInput{
		Name: "",
	})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestQueueCreateCategory_MemberForbidden(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.CreateCategory(context.Background(), info, "TEST", q.ID, CreateCategoryInput{
		Name: "Bug Reports",
	})
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestQueueListCategories_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, categoryRepo, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleMember)

	// Add categories directly to the mock
	categoryRepo.Create(context.Background(), &model.QueueCategory{
		ID: uuid.New(), QueueID: q.ID, Name: "Cat1", Position: 0,
	})
	categoryRepo.Create(context.Background(), &model.QueueCategory{
		ID: uuid.New(), QueueID: q.ID, Name: "Cat2", Position: 1,
	})

	cats, err := svc.ListCategories(context.Background(), info, "TEST", q.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
}

func TestQueueUpdateCategory_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, categoryRepo, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	cat := &model.QueueCategory{
		ID: uuid.New(), QueueID: q.ID, Name: "Original", Position: 0,
	}
	categoryRepo.Create(context.Background(), cat)

	newName := "Updated"
	updated, err := svc.UpdateCategory(context.Background(), info, "TEST", q.ID, cat.ID, UpdateCategoryInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %s", updated.Name)
	}
}

func TestQueueDeleteCategory_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, categoryRepo, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	cat := &model.QueueCategory{
		ID: uuid.New(), QueueID: q.ID, Name: "ToDelete", Position: 0,
	}
	categoryRepo.Create(context.Background(), cat)

	err := svc.DeleteCategory(context.Background(), info, "TEST", q.ID, cat.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = categoryRepo.GetByID(context.Background(), cat.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatal("expected category to be deleted")
	}
}

// --- Queue Team Tests ---

func TestQueueAssignTeam_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, queueTeamRepo := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	teamID := uuid.New()
	err := svc.AssignTeam(context.Background(), info, "TEST", q.ID, teamID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify assignment exists
	key := qtKey(q.ID, teamID)
	if !queueTeamRepo.assignments[key] {
		t.Fatal("expected team to be assigned to queue")
	}
}

func TestQueueAssignTeam_MemberForbidden(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, _ := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleMember)

	teamID := uuid.New()
	err := svc.AssignTeam(context.Background(), info, "TEST", q.ID, teamID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestQueueUnassignTeam_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, queueTeamRepo := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	teamID := uuid.New()
	queueTeamRepo.assignments[qtKey(q.ID, teamID)] = true

	err := svc.UnassignTeam(context.Background(), info, "TEST", q.ID, teamID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if queueTeamRepo.assignments[qtKey(q.ID, teamID)] {
		t.Fatal("expected team to be unassigned from queue")
	}
}

func TestQueueListQueueTeams_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo, _, queueTeamRepo := newTestQueueService()
	info := userAuthInfo()
	_, q := setupQueueForCategory(t, svc, queueRepo, projectRepo, memberRepo, info, model.ProjectRoleMember)

	team1 := &model.Team{ID: uuid.New(), ProjectID: uuid.New(), Name: "Team Alpha"}
	team2 := &model.Team{ID: uuid.New(), ProjectID: uuid.New(), Name: "Team Beta"}
	queueTeamRepo.AddTeam(team1)
	queueTeamRepo.AddTeam(team2)
	queueTeamRepo.assignments[qtKey(q.ID, team1.ID)] = true
	queueTeamRepo.assignments[qtKey(q.ID, team2.ID)] = true

	teams, err := svc.ListQueueTeams(context.Background(), info, "TEST", q.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}
}
