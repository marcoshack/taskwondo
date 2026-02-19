package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

func newTestMilestoneService() (*MilestoneService, *mockMilestoneRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	milestoneRepo := newMockMilestoneRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewMilestoneService(milestoneRepo, projectRepo, memberRepo)
	return svc, milestoneRepo, projectRepo, memberRepo
}

func setupMilestoneProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
	t.Helper()
	project := &model.Project{
		ID:   uuid.New(),
		Name: "Test Project",
		Key:  "MILE",
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

func validCreateMilestoneInput() CreateMilestoneInput {
	return CreateMilestoneInput{
		Name: "v1.0",
	}
}

// --- Tests ---

func TestMilestoneCreate_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	mp, err := svc.Create(context.Background(), info, "MILE", validCreateMilestoneInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mp.Name != "v1.0" {
		t.Fatalf("expected name 'v1.0', got %s", mp.Name)
	}
	if mp.Status != model.MilestoneStatusOpen {
		t.Fatalf("expected status 'open', got %s", mp.Status)
	}
}

func TestMilestoneCreate_WithDueDate(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	dueDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	input := CreateMilestoneInput{
		Name:    "v2.0",
		DueDate: &dueDate,
	}

	mp, err := svc.Create(context.Background(), info, "MILE", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mp.DueDate == nil {
		t.Fatal("expected due date to be set")
	}
}

func TestMilestoneCreate_EmptyName(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := CreateMilestoneInput{Name: ""}
	_, err := svc.Create(context.Background(), info, "MILE", input)
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestMilestoneCreate_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	_, err := svc.Create(context.Background(), info, "MILE", validCreateMilestoneInput())
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestMilestoneCreate_AdminAllowed(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	_, err := svc.Create(context.Background(), info, "MILE", validCreateMilestoneInput())
	if err != nil {
		t.Fatalf("expected no error for admin, got %v", err)
	}
}

func TestMilestoneGet_Success(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	result, err := svc.Get(context.Background(), info, "MILE", m.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "v1.0" {
		t.Fatalf("expected name 'v1.0', got %s", result.Name)
	}
}

func TestMilestoneGet_WrongProject(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: uuid.New(), // different project
		Name:      "Other",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	_, err := svc.Get(context.Background(), info, "MILE", m.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong project, got %v", err)
	}
}

func TestMilestoneList_Success(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	milestoneRepo.Create(context.Background(), &model.Milestone{
		ID: uuid.New(), ProjectID: project.ID, Name: "v1.0", Status: model.MilestoneStatusOpen,
	})
	milestoneRepo.Create(context.Background(), &model.Milestone{
		ID: uuid.New(), ProjectID: project.ID, Name: "v2.0", Status: model.MilestoneStatusOpen,
	})

	milestones, err := svc.List(context.Background(), info, "MILE")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(milestones) != 2 {
		t.Fatalf("expected 2 milestones, got %d", len(milestones))
	}
}

func TestMilestoneUpdate_Success(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	newName := "v1.1"
	updated, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "v1.1" {
		t.Fatalf("expected name 'v1.1', got %s", updated.Name)
	}
}

func TestMilestoneUpdate_CloseStatus(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	closed := model.MilestoneStatusClosed
	updated, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		Status: &closed,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != model.MilestoneStatusClosed {
		t.Fatalf("expected status 'closed', got %s", updated.Status)
	}
}

func TestMilestoneUpdate_InvalidStatus(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	invalid := "invalid"
	_, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		Status: &invalid,
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestMilestoneDelete_Success(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	err := svc.Delete(context.Background(), info, "MILE", m.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = milestoneRepo.GetByID(context.Background(), m.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected milestone to be deleted")
	}
}

func TestMilestoneDelete_MemberForbidden(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	err := svc.Delete(context.Background(), info, "MILE", m.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestMilestoneUpdate_AllFields(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	newName := "v2.0"
	desc := "Release milestone"
	due := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	updated, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		Name:        &newName,
		Description: &desc,
		DueDate:     &due,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "v2.0" {
		t.Fatalf("expected name 'v2.0', got %s", updated.Name)
	}
	if updated.Description == nil || *updated.Description != "Release milestone" {
		t.Fatal("expected description to be set")
	}
}

func TestMilestoneUpdate_ClearFields(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	desc := "desc"
	due := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	m := &model.Milestone{
		ID:          uuid.New(),
		ProjectID:   project.ID,
		Name:        "v1.0",
		Description: &desc,
		DueDate:     &due,
		Status:      model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	updated, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		ClearDescription: true,
		ClearDueDate:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Description != nil {
		t.Fatal("expected description to be cleared")
	}
	if updated.DueDate != nil {
		t.Fatal("expected due date to be cleared")
	}
}

func TestMilestoneUpdate_EmptyName(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	project := setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	empty := "   "
	_, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		Name: &empty,
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestMilestoneUpdate_WrongProject(t *testing.T) {
	svc, milestoneRepo, projectRepo, memberRepo := newTestMilestoneService()
	info := userAuthInfo()
	setupMilestoneProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	m := &model.Milestone{
		ID:        uuid.New(),
		ProjectID: uuid.New(), // different project
		Name:      "v1.0",
		Status:    model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	newName := "Hacked"
	_, err := svc.Update(context.Background(), info, "MILE", m.ID, UpdateMilestoneInput{
		Name: &newName,
	})
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMilestoneList_AdminBypass(t *testing.T) {
	svc, milestoneRepo, projectRepo, _ := newTestMilestoneService()
	admin := adminAuthInfo()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "MILE"}
	projectRepo.Create(context.Background(), project)

	milestoneRepo.Create(context.Background(), &model.Milestone{
		ID: uuid.New(), ProjectID: project.ID, Name: "v1", Status: model.MilestoneStatusOpen,
	})

	milestones, err := svc.List(context.Background(), admin, "MILE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(milestones) != 1 {
		t.Fatalf("expected 1 milestone, got %d", len(milestones))
	}
}

func TestMilestoneGet_AdminBypass(t *testing.T) {
	svc, milestoneRepo, projectRepo, _ := newTestMilestoneService()
	admin := adminAuthInfo()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "MILE"}
	projectRepo.Create(context.Background(), project)

	m := &model.Milestone{
		ID: uuid.New(), ProjectID: project.ID, Name: "v1", Status: model.MilestoneStatusOpen,
	}
	milestoneRepo.Create(context.Background(), m)

	result, err := svc.Get(context.Background(), admin, "MILE", m.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "v1" {
		t.Fatalf("expected name 'v1', got %s", result.Name)
	}
}
