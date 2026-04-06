package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// --- mock embedding repo ---

type mockSearchEmbedding struct {
	results []model.SearchResult
	err     error
	// recorded args for assertions
	lastAccess model.SearchAccess
}

func (m *mockSearchEmbedding) SearchByVector(_ context.Context, _ []float32, _ *model.SearchFilter, access model.SearchAccess) ([]model.SearchResult, error) {
	m.lastAccess = access
	return m.results, m.err
}

// --- mock work item repo ---

type mockSearchWorkItems struct {
	results []model.SearchResult
	err     error
	// recorded args for assertions
	lastAccess model.SearchAccess
}

func (m *mockSearchWorkItems) SearchFTS(_ context.Context, _ string, access model.SearchAccess, _ int) ([]model.SearchResult, error) {
	m.lastAccess = access
	return m.results, m.err
}

// --- mock search members ---

type mockSearchMembers struct {
	memberships []model.ProjectMemberWithProject
	err         error
}

func (m *mockSearchMembers) ListByUser(_ context.Context, _ uuid.UUID) ([]model.ProjectMemberWithProject, error) {
	return m.memberships, m.err
}

// --- mock entity FTS repo (teams, queues, milestones) ---

type mockSearchEntityFTS struct {
	results []model.SearchResult
	err     error
}

func (m *mockSearchEntityFTS) SearchFTS(_ context.Context, _ string, _ []uuid.UUID, _ int) ([]model.SearchResult, error) {
	return m.results, m.err
}

// --- mock search settings ---

type mockSearchSettings struct {
	settings map[string]*model.SystemSetting
}

func (m *mockSearchSettings) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}

// membership is a tiny helper to build ProjectMemberWithProject for tests.
func membership(projectID uuid.UUID, role string) model.ProjectMemberWithProject {
	return model.ProjectMemberWithProject{
		ProjectMember: model.ProjectMember{ProjectID: projectID, Role: role},
	}
}

func TestSearchService_FeatureDisabled(t *testing.T) {
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	_, err := svc.Search(context.Background(), info, &model.SearchFilter{Query: "test"})
	if err != model.ErrFeatureDisabled {
		t.Fatalf("expected ErrFeatureDisabled, got %v", err)
	}
}

func TestSearchService_SemanticEnabled(t *testing.T) {
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{membership(uuid.New(), model.ProjectRoleOwner)},
		},
		&mockSearchSettings{
			settings: map[string]*model.SystemSetting{
				model.SettingFeatureSemanticSearch: {Key: model.SettingFeatureSemanticSearch, Value: []byte("true")},
			},
		},
	)

	enabled := svc.SemanticEnabled(context.Background())
	if !enabled {
		t.Fatal("expected SemanticEnabled to return true")
	}
}

func TestSearchService_SemanticDisabled(t *testing.T) {
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	enabled := svc.SemanticEnabled(context.Background())
	if enabled {
		t.Fatal("expected SemanticEnabled to return false")
	}
}

func TestSearchService_SearchFTS(t *testing.T) {
	projectID := uuid.New()
	itemID := uuid.New()
	num := 42

	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{
			results: []model.SearchResult{
				{EntityType: "work_item", EntityID: itemID, ProjectID: &projectID, Content: "[task] Fix login", ProjectKey: "TF", ItemNumber: &num},
			},
		},
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{membership(projectID, model.ProjectRoleMember)},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	results, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{Query: "fix login", Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].EntityID != itemID {
		t.Errorf("expected entity ID %s, got %s", itemID, results[0].EntityID)
	}
}

func TestSearchService_FTSStatusFields(t *testing.T) {
	projectID := uuid.New()
	itemID := uuid.New()
	num := 7

	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{
			results: []model.SearchResult{
				{
					EntityType:     "work_item",
					EntityID:       itemID,
					ProjectID:      &projectID,
					Content:        "[task] Done item",
					ProjectKey:     "TF",
					ItemNumber:     &num,
					Status:         "done",
					StatusCategory: "done",
				},
			},
		},
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{membership(projectID, model.ProjectRoleMember)},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	results, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{Query: "done item", Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "done" {
		t.Errorf("expected status 'done', got %q", results[0].Status)
	}
	if results[0].StatusCategory != "done" {
		t.Errorf("expected status_category 'done', got %q", results[0].StatusCategory)
	}
}

func TestSearchService_ProjectIDFiltering(t *testing.T) {
	allowedProject := uuid.New()
	disallowedProject := uuid.New()

	workItemMock := &mockSearchWorkItems{results: []model.SearchResult{}}
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{results: []model.SearchResult{}},
		workItemMock,
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{membership(allowedProject, model.ProjectRoleMember)},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	filter := &model.SearchFilter{
		Query:      "test",
		ProjectIDs: []uuid.UUID{allowedProject, disallowedProject},
	}

	if _, err := svc.SearchFTS(context.Background(), info, filter); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The disallowed project must not leak through — only the allowed one should
	// survive the intersection.
	if got := workItemMock.lastAccess.FullProjectIDs; len(got) != 1 || got[0] != allowedProject {
		t.Errorf("expected full access to contain only %s, got %v", allowedProject, got)
	}
	if len(workItemMock.lastAccess.CustomerProjectIDs) != 0 {
		t.Errorf("expected no customer projects, got %v", workItemMock.lastAccess.CustomerProjectIDs)
	}
}

// TestSearchService_CustomerRoleSplit verifies that customer-role memberships
// are partitioned into CustomerProjectIDs (not FullProjectIDs) and the caller's
// UserID is propagated — so downstream SQL can scope customer results to their
// own portal tickets.
func TestSearchService_CustomerRoleSplit(t *testing.T) {
	ownedProject := uuid.New()
	customerProject := uuid.New()
	userID := uuid.New()

	workItemMock := &mockSearchWorkItems{}
	embeddingMock := &mockSearchEmbedding{}

	svc := NewSearchService(
		&EmbeddingService{},
		embeddingMock,
		workItemMock,
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{
				membership(ownedProject, model.ProjectRoleOwner),
				membership(customerProject, model.ProjectRoleCustomer),
			},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: userID}
	if _, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{Query: "q", Limit: 20}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	access := workItemMock.lastAccess
	if access.UserID != userID {
		t.Errorf("expected access.UserID=%s, got %s", userID, access.UserID)
	}
	if len(access.FullProjectIDs) != 1 || access.FullProjectIDs[0] != ownedProject {
		t.Errorf("expected full project %s, got %v", ownedProject, access.FullProjectIDs)
	}
	if len(access.CustomerProjectIDs) != 1 || access.CustomerProjectIDs[0] != customerProject {
		t.Errorf("expected customer project %s, got %v", customerProject, access.CustomerProjectIDs)
	}
}

// TestSearchService_CustomerOnly verifies that a user who is only a customer
// (no other memberships) ends up with zero full-access projects and only
// customer projects passed through.
func TestSearchService_CustomerOnly(t *testing.T) {
	customerProject := uuid.New()
	userID := uuid.New()

	workItemMock := &mockSearchWorkItems{}
	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		workItemMock,
		&mockSearchEntityFTS{}, &mockSearchEntityFTS{}, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{
				membership(customerProject, model.ProjectRoleCustomer),
			},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: userID}
	if _, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{Query: "q", Limit: 20}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	access := workItemMock.lastAccess
	if len(access.FullProjectIDs) != 0 {
		t.Errorf("expected no full-access projects, got %v", access.FullProjectIDs)
	}
	if len(access.CustomerProjectIDs) != 1 || access.CustomerProjectIDs[0] != customerProject {
		t.Errorf("expected customer project %s, got %v", customerProject, access.CustomerProjectIDs)
	}
	if access.UserID != userID {
		t.Errorf("expected user id %s, got %s", userID, access.UserID)
	}
}

// TestSearchService_FTSMultiEntity verifies that SearchFTS queries all entity
// types (work items, teams, queues, milestones) and merges results.
func TestSearchService_FTSMultiEntity(t *testing.T) {
	projectID := uuid.New()
	itemID := uuid.New()
	teamID := uuid.New()
	queueID := uuid.New()
	milestoneID := uuid.New()
	num := 1

	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{results: []model.SearchResult{
			{EntityType: "work_item", EntityID: itemID, ProjectID: &projectID, Content: "[task] Test", ItemNumber: &num},
		}},
		&mockSearchEntityFTS{results: []model.SearchResult{
			{EntityType: "team", EntityID: teamID, ProjectID: &projectID, Content: "Team: Backend"},
		}},
		&mockSearchEntityFTS{results: []model.SearchResult{
			{EntityType: "queue", EntityID: queueID, ProjectID: &projectID, Content: "Queue: Support"},
		}},
		&mockSearchEntityFTS{results: []model.SearchResult{
			{EntityType: "milestone", EntityID: milestoneID, ProjectID: &projectID, Content: "Milestone: v1"},
		}},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{membership(projectID, model.ProjectRoleMember)},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	results, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{Query: "test", Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results (1 per entity type), got %d", len(results))
	}

	typeSet := map[string]bool{}
	for _, r := range results {
		typeSet[r.EntityType] = true
	}
	for _, et := range []string{"work_item", "team", "queue", "milestone"} {
		if !typeSet[et] {
			t.Errorf("expected entity type %s in results", et)
		}
	}
}

// TestSearchService_FTSEntityTypeFilter verifies that entity_type filter
// restricts which entity types are queried.
func TestSearchService_FTSEntityTypeFilter(t *testing.T) {
	projectID := uuid.New()
	teamID := uuid.New()

	teamMock := &mockSearchEntityFTS{results: []model.SearchResult{
		{EntityType: "team", EntityID: teamID, ProjectID: &projectID, Content: "Team: Backend"},
	}}
	queueMock := &mockSearchEntityFTS{results: []model.SearchResult{
		{EntityType: "queue", EntityID: uuid.New(), ProjectID: &projectID, Content: "Queue: Support"},
	}}

	svc := NewSearchService(
		&EmbeddingService{},
		&mockSearchEmbedding{},
		&mockSearchWorkItems{},
		teamMock, queueMock, &mockSearchEntityFTS{},
		&mockSearchMembers{
			memberships: []model.ProjectMemberWithProject{membership(projectID, model.ProjectRoleMember)},
		},
		&mockSearchSettings{settings: map[string]*model.SystemSetting{}},
	)

	info := &model.AuthInfo{UserID: uuid.New()}
	results, err := svc.SearchFTS(context.Background(), info, &model.SearchFilter{
		Query:       "test",
		EntityTypes: []string{"team"},
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only team results should be returned (queue filtered out, work items filtered out)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].EntityType != "team" {
		t.Errorf("expected entity type 'team', got %s", results[0].EntityType)
	}
}
