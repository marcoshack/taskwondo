package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- mock ---

type mockSavedSearchRepo struct {
	items map[uuid.UUID]*model.SavedSearch
}

func newMockSavedSearchRepo() *mockSavedSearchRepo {
	return &mockSavedSearchRepo{items: make(map[uuid.UUID]*model.SavedSearch)}
}

func (m *mockSavedSearchRepo) Create(_ context.Context, s *model.SavedSearch) error {
	m.items[s.ID] = s
	return nil
}

func (m *mockSavedSearchRepo) GetByID(_ context.Context, id uuid.UUID) (*model.SavedSearch, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *mockSavedSearchRepo) ListByProjectAndUser(_ context.Context, projectID, userID uuid.UUID) ([]model.SavedSearch, error) {
	var result []model.SavedSearch
	for _, s := range m.items {
		if s.ProjectID == projectID && (s.UserID == nil || *s.UserID == userID) {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSavedSearchRepo) Update(_ context.Context, s *model.SavedSearch) error {
	if _, ok := m.items[s.ID]; !ok {
		return model.ErrNotFound
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockSavedSearchRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

// --- helpers ---

func newTestSavedSearchService() (*SavedSearchService, *mockSavedSearchRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	ssRepo := newMockSavedSearchRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	svc := NewSavedSearchService(ssRepo, projectRepo, memberRepo)
	return svc, ssRepo, projectRepo, memberRepo
}

func setupSSProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
	t.Helper()
	project := &model.Project{
		ID:   uuid.New(),
		Name: "Test Project",
		Key:  "SS",
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

func validCreateSSInput() CreateSavedSearchInput {
	return CreateSavedSearchInput{
		Name:     "My open bugs",
		Filters:  model.SavedSearchFilters{Type: []string{"bug"}, Status: []string{"open"}},
		ViewMode: "list",
		Shared:   false,
	}
}

// --- Tests ---

func TestSavedSearchCreate_UserScope_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	ss, err := svc.Create(context.Background(), info, "SS", validCreateSSInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ss.Name != "My open bugs" {
		t.Fatalf("expected name 'My open bugs', got %s", ss.Name)
	}
	if ss.UserID == nil || *ss.UserID != info.UserID {
		t.Fatal("expected user_id to be set for personal search")
	}
	if ss.Scope() != model.SavedSearchScopeUser {
		t.Fatalf("expected scope 'user', got %s", ss.Scope())
	}
}

func TestSavedSearchCreate_SharedScope_OwnerSuccess(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := validCreateSSInput()
	input.Shared = true

	ss, err := svc.Create(context.Background(), info, "SS", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ss.UserID != nil {
		t.Fatal("expected user_id to be nil for shared search")
	}
	if ss.Scope() != model.SavedSearchScopeShared {
		t.Fatalf("expected scope 'shared', got %s", ss.Scope())
	}
}

func TestSavedSearchCreate_SharedScope_AdminSuccess(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	input := validCreateSSInput()
	input.Shared = true

	_, err := svc.Create(context.Background(), info, "SS", input)
	if err != nil {
		t.Fatalf("expected no error for admin, got %v", err)
	}
}

func TestSavedSearchCreate_SharedScope_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	input := validCreateSSInput()
	input.Shared = true

	_, err := svc.Create(context.Background(), info, "SS", input)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSavedSearchCreate_EmptyName_Validation(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	input := validCreateSSInput()
	input.Name = ""
	_, err := svc.Create(context.Background(), info, "SS", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestSavedSearchCreate_InvalidViewMode_Validation(t *testing.T) {
	svc, _, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	input := validCreateSSInput()
	input.ViewMode = "kanban"
	_, err := svc.Create(context.Background(), info, "SS", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestSavedSearchList_ReturnsUserAndShared(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	// Create a personal search
	personal := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    &info.UserID,
		Name:      "My search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), personal)

	// Create a shared search
	shared := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Team search",
		ViewMode:  "board",
	}
	ssRepo.Create(context.Background(), shared)

	// Create another user's personal search (should NOT be returned)
	otherUserID := uuid.New()
	otherPersonal := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    &otherUserID,
		Name:      "Other user search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), otherPersonal)

	results, err := svc.List(context.Background(), info, "SS")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (personal + shared), got %d", len(results))
	}
}

func TestSavedSearchList_NonMemberNotFound(t *testing.T) {
	svc, _, projectRepo, _ := newTestSavedSearchService()
	info := userAuthInfo()
	// Create project but don't add user as member
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "SS"}
	projectRepo.Create(context.Background(), project)

	_, err := svc.List(context.Background(), info, "SS")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSavedSearchUpdate_OwnSearch_Success(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    &info.UserID,
		Name:      "Old name",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	newName := "New name"
	updated, err := svc.Update(context.Background(), info, "SS", ss.ID, UpdateSavedSearchInput{Name: &newName})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New name" {
		t.Fatalf("expected name 'New name', got %s", updated.Name)
	}
}

func TestSavedSearchUpdate_OtherUserSearch_Forbidden(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	otherUserID := uuid.New()
	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    &otherUserID,
		Name:      "Other user search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	newName := "Hacked"
	_, err := svc.Update(context.Background(), info, "SS", ss.ID, UpdateSavedSearchInput{Name: &newName})
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSavedSearchUpdate_SharedSearch_AdminSuccess(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Shared search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	newName := "Updated shared"
	_, err := svc.Update(context.Background(), info, "SS", ss.ID, UpdateSavedSearchInput{Name: &newName})
	if err != nil {
		t.Fatalf("expected no error for admin, got %v", err)
	}
}

func TestSavedSearchUpdate_SharedSearch_MemberForbidden(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Shared search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	newName := "Updated"
	_, err := svc.Update(context.Background(), info, "SS", ss.ID, UpdateSavedSearchInput{Name: &newName})
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSavedSearchDelete_OwnSearch_Success(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    &info.UserID,
		Name:      "My search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	err := svc.Delete(context.Background(), info, "SS", ss.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSavedSearchDelete_SharedSearch_MemberForbidden(t *testing.T) {
	svc, ssRepo, projectRepo, memberRepo := newTestSavedSearchService()
	info := userAuthInfo()
	project := setupSSProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Shared search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	err := svc.Delete(context.Background(), info, "SS", ss.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSavedSearchDelete_GlobalAdminBypass(t *testing.T) {
	svc, ssRepo, projectRepo, _ := newTestSavedSearchService()
	info := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "admin@test.com",
		GlobalRole: model.RoleAdmin,
	}
	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "SS"}
	projectRepo.Create(context.Background(), project)

	ss := &model.SavedSearch{
		ID:        uuid.New(),
		ProjectID: project.ID,
		Name:      "Shared search",
		ViewMode:  "list",
	}
	ssRepo.Create(context.Background(), ss)

	err := svc.Delete(context.Background(), info, "SS", ss.ID)
	if err != nil {
		t.Fatalf("expected no error for global admin, got %v", err)
	}
}
