package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
)

func newTestQueueService() (*QueueService, *mockQueueRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	queueRepo := newMockQueueRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewQueueService(queueRepo, projectRepo, memberRepo)
	return svc, queueRepo, projectRepo, memberRepo
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
	svc, _, projectRepo, memberRepo := newTestQueueService()
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
	svc, _, projectRepo, memberRepo := newTestQueueService()
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
	svc, _, projectRepo, memberRepo := newTestQueueService()
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
	svc, _, projectRepo, memberRepo := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateQueueInput())
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestQueueCreate_AdminAllowed(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateQueueInput())
	if err != nil {
		t.Fatalf("expected no error for admin, got %v", err)
	}
}

func TestQueueGet_Success(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo := newTestQueueService()
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
	svc, _, projectRepo, memberRepo := newTestQueueService()
	info := userAuthInfo()
	setupQueueProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Get(context.Background(), info, "TEST", uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestQueueGet_WrongProject(t *testing.T) {
	svc, queueRepo, projectRepo, memberRepo := newTestQueueService()
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
	svc, queueRepo, projectRepo, memberRepo := newTestQueueService()
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
	svc, queueRepo, projectRepo, memberRepo := newTestQueueService()
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
	svc, queueRepo, projectRepo, memberRepo := newTestQueueService()
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
	svc, queueRepo, projectRepo, memberRepo := newTestQueueService()
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
	svc, _, projectRepo, memberRepo := newTestQueueService()
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
	svc, _, projectRepo, _ := newTestQueueService()
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
