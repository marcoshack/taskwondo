package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- Mock escalation repository ---

type mockEscalationRepo struct {
	lists    map[uuid.UUID]*model.EscalationList
	mappings map[string]*model.TypeEscalationMapping // key: "projectID:type"
}

func newMockEscalationRepo() *mockEscalationRepo {
	return &mockEscalationRepo{
		lists:    make(map[uuid.UUID]*model.EscalationList),
		mappings: make(map[string]*model.TypeEscalationMapping),
	}
}

func (m *mockEscalationRepo) Create(_ context.Context, el *model.EscalationList) error {
	now := time.Now()
	el.CreatedAt = now
	el.UpdatedAt = now
	for i := range el.Levels {
		el.Levels[i].CreatedAt = now
	}
	m.lists[el.ID] = el
	return nil
}

func (m *mockEscalationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.EscalationList, error) {
	el, ok := m.lists[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return el, nil
}

func (m *mockEscalationRepo) List(_ context.Context, projectID uuid.UUID) ([]model.EscalationList, error) {
	var result []model.EscalationList
	for _, el := range m.lists {
		if el.ProjectID == projectID {
			result = append(result, *el)
		}
	}
	return result, nil
}

func (m *mockEscalationRepo) Update(_ context.Context, el *model.EscalationList) error {
	if _, ok := m.lists[el.ID]; !ok {
		return model.ErrNotFound
	}
	el.UpdatedAt = time.Now()
	for i := range el.Levels {
		el.Levels[i].CreatedAt = time.Now()
	}
	m.lists[el.ID] = el
	return nil
}

func (m *mockEscalationRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.lists[id]; !ok {
		return model.ErrNotFound
	}
	// Also remove any mappings that reference this list
	for key, mapping := range m.mappings {
		if mapping.EscalationListID == id {
			delete(m.mappings, key)
		}
	}
	delete(m.lists, id)
	return nil
}

func (m *mockEscalationRepo) ListMappings(_ context.Context, projectID uuid.UUID) ([]model.TypeEscalationMapping, error) {
	var result []model.TypeEscalationMapping
	for _, mapping := range m.mappings {
		if mapping.ProjectID == projectID {
			result = append(result, *mapping)
		}
	}
	return result, nil
}

func (m *mockEscalationRepo) UpsertMapping(_ context.Context, mapping *model.TypeEscalationMapping) error {
	key := mapping.ProjectID.String() + ":" + mapping.WorkItemType
	m.mappings[key] = mapping
	return nil
}

func (m *mockEscalationRepo) DeleteMapping(_ context.Context, projectID uuid.UUID, workItemType string) error {
	key := projectID.String() + ":" + workItemType
	if _, ok := m.mappings[key]; !ok {
		return model.ErrNotFound
	}
	delete(m.mappings, key)
	return nil
}

// --- Mock user repository for escalation ---

type mockEscalationUserRepo struct {
	users map[uuid.UUID]*model.User
}

func newMockEscalationUserRepo() *mockEscalationUserRepo {
	return &mockEscalationUserRepo{users: make(map[uuid.UUID]*model.User)}
}

func (m *mockEscalationUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockEscalationUserRepo) addUser(id uuid.UUID) {
	m.users[id] = &model.User{
		ID:          id,
		Email:       id.String() + "@test.com",
		DisplayName: "User " + id.String()[:8],
		GlobalRole:  model.RoleUser,
		IsActive:    true,
	}
}

// --- Test helpers ---

func newTestEscalationService() (*EscalationService, *mockEscalationRepo, *mockProjectRepo, *mockProjectMemberRepo, *mockEscalationUserRepo) {
	escRepo := newMockEscalationRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	userRepo := newMockEscalationUserRepo()
	svc := NewEscalationService(escRepo, projectRepo, memberRepo, userRepo)
	return svc, escRepo, projectRepo, memberRepo, userRepo
}

func setupEscalationProject(t *testing.T, projectRepo *mockProjectRepo, memberRepo *mockProjectMemberRepo, info *model.AuthInfo, role string) *model.Project {
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

func validCreateEscalationInput(userIDs ...uuid.UUID) CreateEscalationListInput {
	if len(userIDs) == 0 {
		userIDs = []uuid.UUID{uuid.New()}
	}
	return CreateEscalationListInput{
		Name: "Critical Escalation",
		Levels: []EscalationLevelInput{
			{ThresholdPct: 75, UserIDs: userIDs},
			{ThresholdPct: 100, UserIDs: userIDs},
		},
	}
}

// --- Tests ---

func TestEscalationCreate_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	el, err := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if el.Name != "Critical Escalation" {
		t.Fatalf("expected name 'Critical Escalation', got %s", el.Name)
	}
	if len(el.Levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(el.Levels))
	}
}

func TestEscalationCreate_EmptyName(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	input := validCreateEscalationInput(userID)
	input.Name = ""
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestEscalationCreate_NoLevels(t *testing.T) {
	svc, _, projectRepo, memberRepo, _ := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := CreateEscalationListInput{Name: "Empty", Levels: nil}
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestEscalationCreate_DuplicateThreshold(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	input := CreateEscalationListInput{
		Name: "Dup",
		Levels: []EscalationLevelInput{
			{ThresholdPct: 75, UserIDs: []uuid.UUID{userID}},
			{ThresholdPct: 75, UserIDs: []uuid.UUID{userID}},
		},
	}
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for duplicate thresholds, got %v", err)
	}
}

func TestEscalationCreate_ZeroThreshold(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	input := CreateEscalationListInput{
		Name: "Zero",
		Levels: []EscalationLevelInput{
			{ThresholdPct: 0, UserIDs: []uuid.UUID{userID}},
		},
	}
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for zero threshold, got %v", err)
	}
}

func TestEscalationCreate_InvalidUser(t *testing.T) {
	svc, _, projectRepo, memberRepo, _ := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	bogusUserID := uuid.New()
	// Don't add to userRepo — should fail validation
	input := validCreateEscalationInput(bogusUserID)
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid user, got %v", err)
	}
}

func TestEscalationCreate_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	userID := uuid.New()
	userRepo.addUser(userID)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestEscalationCreate_AdminAllowed(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleAdmin)

	userID := uuid.New()
	userRepo.addUser(userID)

	_, err := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))
	if err != nil {
		t.Fatalf("expected no error for admin, got %v", err)
	}
}

func TestEscalationGet_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	userID := uuid.New()
	userRepo.addUser(userID)

	// Create as admin first
	adminInfo := adminAuthInfo()
	el, err := svc.Create(context.Background(), adminInfo, "TEST", validCreateEscalationInput(userID))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	result, err := svc.Get(context.Background(), info, "TEST", el.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Critical Escalation" {
		t.Fatalf("expected name 'Critical Escalation', got %s", result.Name)
	}
}

func TestEscalationGet_WrongProject(t *testing.T) {
	svc, escRepo, projectRepo, memberRepo, _ := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	el := &model.EscalationList{
		ID:        uuid.Must(uuid.NewV7()),
		ProjectID: uuid.New(), // different project
		Name:      "Other",
	}
	escRepo.Create(context.Background(), el)

	_, err := svc.Get(context.Background(), info, "TEST", el.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong project, got %v", err)
	}
}

func TestEscalationList_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	svc.Create(context.Background(), info, "TEST", CreateEscalationListInput{
		Name:   "List A",
		Levels: []EscalationLevelInput{{ThresholdPct: 50, UserIDs: []uuid.UUID{userID}}},
	})
	svc.Create(context.Background(), info, "TEST", CreateEscalationListInput{
		Name:   "List B",
		Levels: []EscalationLevelInput{{ThresholdPct: 100, UserIDs: []uuid.UUID{userID}}},
	})

	lists, err := svc.List(context.Background(), info, "TEST")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(lists))
	}
}

func TestEscalationUpdate_Success(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	el, _ := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))

	updated, err := svc.Update(context.Background(), info, "TEST", el.ID, CreateEscalationListInput{
		Name: "Updated Name",
		Levels: []EscalationLevelInput{
			{ThresholdPct: 50, UserIDs: []uuid.UUID{userID}},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %s", updated.Name)
	}
	if len(updated.Levels) != 1 {
		t.Fatalf("expected 1 level after update, got %d", len(updated.Levels))
	}
}

func TestEscalationDelete_Success(t *testing.T) {
	svc, escRepo, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	el, _ := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))

	err := svc.Delete(context.Background(), info, "TEST", el.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = escRepo.GetByID(context.Background(), el.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected list to be deleted")
	}
}

func TestEscalationDelete_MemberForbidden(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleMember)

	userID := uuid.New()
	userRepo.addUser(userID)

	// Create as admin
	adminInfo := adminAuthInfo()
	el, _ := svc.Create(context.Background(), adminInfo, "TEST", validCreateEscalationInput(userID))

	err := svc.Delete(context.Background(), info, "TEST", el.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestEscalationMapping_UpdateAndList(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	el, _ := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))

	_, err := svc.UpdateMapping(context.Background(), info, "TEST", "bug", el.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mappings, err := svc.ListMappings(context.Background(), info, "TEST")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(mappings))
	}
	if mappings[0].WorkItemType != "bug" {
		t.Fatalf("expected type 'bug', got %s", mappings[0].WorkItemType)
	}
}

func TestEscalationMapping_Delete(t *testing.T) {
	svc, _, projectRepo, memberRepo, userRepo := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	userID := uuid.New()
	userRepo.addUser(userID)

	el, _ := svc.Create(context.Background(), info, "TEST", validCreateEscalationInput(userID))
	svc.UpdateMapping(context.Background(), info, "TEST", "bug", el.ID)

	err := svc.DeleteMapping(context.Background(), info, "TEST", "bug")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mappings, _ := svc.ListMappings(context.Background(), info, "TEST")
	if len(mappings) != 0 {
		t.Fatalf("expected 0 mappings after delete, got %d", len(mappings))
	}
}

func TestEscalationMapping_DeleteNotFound(t *testing.T) {
	svc, _, projectRepo, memberRepo, _ := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	err := svc.DeleteMapping(context.Background(), info, "TEST", "nonexistent")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestEscalationCreate_LevelWithNoUsers(t *testing.T) {
	svc, _, projectRepo, memberRepo, _ := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	input := CreateEscalationListInput{
		Name: "No Users",
		Levels: []EscalationLevelInput{
			{ThresholdPct: 50, UserIDs: nil},
		},
	}
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for empty users, got %v", err)
	}
}

func TestEscalationList_AdminBypass(t *testing.T) {
	svc, _, projectRepo, _, userRepo := newTestEscalationService()
	admin := adminAuthInfo()

	project := &model.Project{ID: uuid.New(), Name: "Test", Key: "TEST"}
	projectRepo.Create(context.Background(), project)

	userID := uuid.New()
	userRepo.addUser(userID)

	svc.Create(context.Background(), admin, "TEST", CreateEscalationListInput{
		Name:   "Admin List",
		Levels: []EscalationLevelInput{{ThresholdPct: 50, UserIDs: []uuid.UUID{userID}}},
	})

	lists, err := svc.List(context.Background(), admin, "TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}
}

func TestEscalationMapping_WrongProjectList(t *testing.T) {
	svc, escRepo, projectRepo, memberRepo, _ := newTestEscalationService()
	info := userAuthInfo()
	setupEscalationProject(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	// Create a list in a different project
	otherList := &model.EscalationList{
		ID:        uuid.Must(uuid.NewV7()),
		ProjectID: uuid.New(),
		Name:      "Other",
	}
	escRepo.Create(context.Background(), otherList)

	_, err := svc.UpdateMapping(context.Background(), info, "TEST", "bug", otherList.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong project list, got %v", err)
	}
}
