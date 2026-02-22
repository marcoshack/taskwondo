package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/storage"
)

// --- Mock project type workflow repository ---

type mockTypeWorkflowRepo struct {
	mappings map[string]*model.ProjectTypeWorkflow // key: "projectID:type"
}

func newMockTypeWorkflowRepo() *mockTypeWorkflowRepo {
	return &mockTypeWorkflowRepo{mappings: make(map[string]*model.ProjectTypeWorkflow)}
}

func (m *mockTypeWorkflowRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectTypeWorkflow, error) {
	var result []model.ProjectTypeWorkflow
	for _, tw := range m.mappings {
		if tw.ProjectID == projectID {
			result = append(result, *tw)
		}
	}
	return result, nil
}

func (m *mockTypeWorkflowRepo) GetByProjectAndType(_ context.Context, projectID uuid.UUID, workItemType string) (*model.ProjectTypeWorkflow, error) {
	key := projectID.String() + ":" + workItemType
	if tw, ok := m.mappings[key]; ok {
		return tw, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockTypeWorkflowRepo) Upsert(_ context.Context, mapping *model.ProjectTypeWorkflow) error {
	now := time.Now()
	mapping.CreatedAt = now
	mapping.UpdatedAt = now
	key := mapping.ProjectID.String() + ":" + mapping.WorkItemType
	m.mappings[key] = mapping
	return nil
}

// --- Mock workflow repository ---

type mockWorkflowRepo struct {
	workflows map[uuid.UUID]*model.Workflow
}

func newMockWorkflowRepo() *mockWorkflowRepo {
	return &mockWorkflowRepo{workflows: make(map[uuid.UUID]*model.Workflow)}
}

func (m *mockWorkflowRepo) Create(_ context.Context, wf *model.Workflow) error {
	now := time.Now()
	wf.CreatedAt = now
	wf.UpdatedAt = now
	m.workflows[wf.ID] = wf
	return nil
}

func (m *mockWorkflowRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Workflow, error) {
	wf, ok := m.workflows[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return wf, nil
}

func (m *mockWorkflowRepo) List(_ context.Context) ([]model.Workflow, error) {
	var result []model.Workflow
	for _, wf := range m.workflows {
		result = append(result, *wf)
	}
	return result, nil
}

func (m *mockWorkflowRepo) Update(_ context.Context, wf *model.Workflow) error {
	if _, ok := m.workflows[wf.ID]; !ok {
		return model.ErrNotFound
	}
	wf.UpdatedAt = time.Now()
	m.workflows[wf.ID] = wf
	return nil
}

func (m *mockWorkflowRepo) GetDefaultByName(_ context.Context, name string) (*model.Workflow, error) {
	for _, wf := range m.workflows {
		if wf.Name == name && wf.IsDefault {
			return wf, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockWorkflowRepo) ValidateTransition(_ context.Context, workflowID uuid.UUID, fromStatus, toStatus string) (bool, error) {
	wf, ok := m.workflows[workflowID]
	if !ok {
		return false, model.ErrNotFound
	}
	for _, t := range wf.Transitions {
		if t.FromStatus == fromStatus && t.ToStatus == toStatus {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockWorkflowRepo) GetInitialStatus(_ context.Context, workflowID uuid.UUID) (*model.WorkflowStatus, error) {
	wf, ok := m.workflows[workflowID]
	if !ok {
		return nil, model.ErrNotFound
	}
	for i := range wf.Statuses {
		if wf.Statuses[i].Position == 0 {
			return &wf.Statuses[i], nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *mockWorkflowRepo) GetStatusCategory(_ context.Context, workflowID uuid.UUID, statusName string) (string, error) {
	wf, ok := m.workflows[workflowID]
	if !ok {
		return "", model.ErrNotFound
	}
	for _, s := range wf.Statuses {
		if s.Name == statusName {
			return s.Category, nil
		}
	}
	return "", model.ErrNotFound
}

func (m *mockWorkflowRepo) ListTransitions(_ context.Context, workflowID uuid.UUID) ([]model.WorkflowTransition, error) {
	wf, ok := m.workflows[workflowID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return wf.Transitions, nil
}

func (m *mockWorkflowRepo) ListStatuses(_ context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error) {
	wf, ok := m.workflows[workflowID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return wf.Statuses, nil
}

func (m *mockWorkflowRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.Workflow, error) {
	var result []model.Workflow
	for _, wf := range m.workflows {
		if wf.ProjectID == nil || *wf.ProjectID == projectID {
			result = append(result, *wf)
		}
	}
	return result, nil
}

func (m *mockWorkflowRepo) ListProjectOnly(_ context.Context, projectID uuid.UUID) ([]model.Workflow, error) {
	var result []model.Workflow
	for _, wf := range m.workflows {
		if wf.ProjectID != nil && *wf.ProjectID == projectID {
			result = append(result, *wf)
		}
	}
	return result, nil
}

func (m *mockWorkflowRepo) ReplaceStatusesAndTransitions(_ context.Context, wf *model.Workflow) error {
	if _, ok := m.workflows[wf.ID]; !ok {
		return model.ErrNotFound
	}
	wf.UpdatedAt = time.Now()
	m.workflows[wf.ID] = wf
	return nil
}

func (m *mockWorkflowRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.workflows[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.workflows, id)
	return nil
}

func (m *mockWorkflowRepo) ListDefaultNames(_ context.Context) ([]string, error) {
	var names []string
	for _, wf := range m.workflows {
		if wf.IsDefault {
			names = append(names, wf.Name)
		}
	}
	return names, nil
}

func (m *mockWorkflowRepo) ListAllStatuses(_ context.Context) ([]model.WorkflowStatus, error) {
	seen := make(map[string]bool)
	var result []model.WorkflowStatus
	for _, wf := range m.workflows {
		if wf.IsDefault {
			for _, s := range wf.Statuses {
				if !seen[s.Name] {
					seen[s.Name] = true
					result = append(result, s)
				}
			}
		}
	}
	return result, nil
}

// --- Mock work item repository ---

type mockWorkItemRepo struct {
	items        map[uuid.UUID]*model.WorkItem
	byProjectNum map[string]*model.WorkItem // key: "projectID:itemNumber"
	counters     map[uuid.UUID]int          // project item counters
	projectKeys  map[uuid.UUID]string       // projectID -> project key
}

func newMockWorkItemRepo() *mockWorkItemRepo {
	return &mockWorkItemRepo{
		items:        make(map[uuid.UUID]*model.WorkItem),
		byProjectNum: make(map[string]*model.WorkItem),
		counters:     make(map[uuid.UUID]int),
		projectKeys:  make(map[uuid.UUID]string),
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
	if key, ok := m.projectKeys[item.ProjectID]; ok {
		item.DisplayID = fmt.Sprintf("%s-%d", key, item.ItemNumber)
	}
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

func (m *mockWorkItemEventRepo) ListByWorkItemFiltered(_ context.Context, workItemID uuid.UUID, visibility string) ([]model.WorkItemEventWithActor, error) {
	events := m.events[workItemID]
	var result []model.WorkItemEventWithActor
	for _, e := range events {
		if visibility != "" && e.Visibility != visibility {
			continue
		}
		result = append(result, model.WorkItemEventWithActor{WorkItemEvent: e})
	}
	return result, nil
}

// --- Mock comment repository ---

type mockCommentRepo struct {
	comments map[uuid.UUID]*model.Comment
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{comments: make(map[uuid.UUID]*model.Comment)}
}

func (m *mockCommentRepo) Create(_ context.Context, comment *model.Comment) error {
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	m.comments[comment.ID] = comment
	return nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Comment, error) {
	c, ok := m.comments[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return c, nil
}

func (m *mockCommentRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID, visibility string) ([]model.Comment, error) {
	var result []model.Comment
	for _, c := range m.comments {
		if c.WorkItemID != workItemID {
			continue
		}
		if visibility != "" && c.Visibility != visibility {
			continue
		}
		result = append(result, *c)
	}
	return result, nil
}

func (m *mockCommentRepo) Update(_ context.Context, comment *model.Comment) error {
	existing, ok := m.comments[comment.ID]
	if !ok {
		return model.ErrNotFound
	}
	existing.Body = comment.Body
	existing.EditCount++
	existing.UpdatedAt = time.Now()
	return nil
}

func (m *mockCommentRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.comments[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.comments, id)
	return nil
}

// --- Mock relation repository ---

type mockRelationRepo struct {
	relations map[uuid.UUID]*model.WorkItemRelation
}

func newMockRelationRepo() *mockRelationRepo {
	return &mockRelationRepo{relations: make(map[uuid.UUID]*model.WorkItemRelation)}
}

func (m *mockRelationRepo) Create(_ context.Context, rel *model.WorkItemRelation) error {
	rel.CreatedAt = time.Now()
	// Check unique constraint
	for _, existing := range m.relations {
		if existing.SourceID == rel.SourceID && existing.TargetID == rel.TargetID && existing.RelationType == rel.RelationType {
			return fmt.Errorf("duplicate relation: %w", model.ErrConflict)
		}
	}
	m.relations[rel.ID] = rel
	return nil
}

func (m *mockRelationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.WorkItemRelation, error) {
	rel, ok := m.relations[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return rel, nil
}

func (m *mockRelationRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID) ([]model.WorkItemRelation, error) {
	var result []model.WorkItemRelation
	for _, rel := range m.relations {
		if rel.SourceID == workItemID || rel.TargetID == workItemID {
			result = append(result, *rel)
		}
	}
	return result, nil
}

func (m *mockRelationRepo) ListByWorkItemWithDetails(_ context.Context, workItemID uuid.UUID) ([]model.WorkItemRelationWithDetails, error) {
	var result []model.WorkItemRelationWithDetails
	for _, rel := range m.relations {
		if rel.SourceID == workItemID || rel.TargetID == workItemID {
			result = append(result, model.WorkItemRelationWithDetails{WorkItemRelation: *rel})
		}
	}
	return result, nil
}

func (m *mockRelationRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.relations[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.relations, id)
	return nil
}

// --- Mock queue repository ---

type mockQueueRepo struct {
	queues map[uuid.UUID]*model.Queue
}

func newMockQueueRepo() *mockQueueRepo {
	return &mockQueueRepo{queues: make(map[uuid.UUID]*model.Queue)}
}

func (m *mockQueueRepo) Create(_ context.Context, q *model.Queue) error {
	now := time.Now()
	q.CreatedAt = now
	q.UpdatedAt = now
	m.queues[q.ID] = q
	return nil
}

func (m *mockQueueRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Queue, error) {
	q, ok := m.queues[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return q, nil
}

func (m *mockQueueRepo) List(_ context.Context, projectID uuid.UUID) ([]model.Queue, error) {
	var result []model.Queue
	for _, q := range m.queues {
		if q.ProjectID == projectID {
			result = append(result, *q)
		}
	}
	return result, nil
}

func (m *mockQueueRepo) Update(_ context.Context, q *model.Queue) error {
	if _, ok := m.queues[q.ID]; !ok {
		return model.ErrNotFound
	}
	q.UpdatedAt = time.Now()
	m.queues[q.ID] = q
	return nil
}

func (m *mockQueueRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.queues[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.queues, id)
	return nil
}

// --- Mock milestone repository ---

type mockMilestoneRepo struct {
	milestones map[uuid.UUID]*model.Milestone
}

func newMockMilestoneRepo() *mockMilestoneRepo {
	return &mockMilestoneRepo{milestones: make(map[uuid.UUID]*model.Milestone)}
}

func (m *mockMilestoneRepo) Create(_ context.Context, ms *model.Milestone) error {
	now := time.Now()
	ms.CreatedAt = now
	ms.UpdatedAt = now
	m.milestones[ms.ID] = ms
	return nil
}

func (m *mockMilestoneRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Milestone, error) {
	ms, ok := m.milestones[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return ms, nil
}

func (m *mockMilestoneRepo) GetByIDWithProgress(_ context.Context, id uuid.UUID) (*model.MilestoneWithProgress, error) {
	ms, ok := m.milestones[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return &model.MilestoneWithProgress{Milestone: *ms}, nil
}

func (m *mockMilestoneRepo) List(_ context.Context, projectID uuid.UUID) ([]model.Milestone, error) {
	var result []model.Milestone
	for _, ms := range m.milestones {
		if ms.ProjectID == projectID {
			result = append(result, *ms)
		}
	}
	return result, nil
}

func (m *mockMilestoneRepo) ListWithProgress(_ context.Context, projectID uuid.UUID) ([]model.MilestoneWithProgress, error) {
	var result []model.MilestoneWithProgress
	for _, ms := range m.milestones {
		if ms.ProjectID == projectID {
			result = append(result, model.MilestoneWithProgress{Milestone: *ms})
		}
	}
	return result, nil
}

func (m *mockMilestoneRepo) Update(_ context.Context, ms *model.Milestone) error {
	if _, ok := m.milestones[ms.ID]; !ok {
		return model.ErrNotFound
	}
	ms.UpdatedAt = time.Now()
	m.milestones[ms.ID] = ms
	return nil
}

func (m *mockMilestoneRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.milestones[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.milestones, id)
	return nil
}

// --- Mock attachment repo ---

type mockAttachmentRepo struct {
	attachments map[uuid.UUID]*model.Attachment
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{attachments: make(map[uuid.UUID]*model.Attachment)}
}

func (m *mockAttachmentRepo) Create(_ context.Context, a *model.Attachment) error {
	a.CreatedAt = time.Now()
	m.attachments[a.ID] = a
	return nil
}

func (m *mockAttachmentRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Attachment, error) {
	a, ok := m.attachments[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return a, nil
}

func (m *mockAttachmentRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID) ([]model.Attachment, error) {
	var result []model.Attachment
	for _, a := range m.attachments {
		if a.WorkItemID == workItemID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockAttachmentRepo) UpdateComment(_ context.Context, id uuid.UUID, comment string) error {
	a, ok := m.attachments[id]
	if !ok {
		return model.ErrNotFound
	}
	a.Comment = comment
	return nil
}

func (m *mockAttachmentRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.attachments[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.attachments, id)
	return nil
}

// --- Mock storage ---

type mockStorage struct {
	objects map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{objects: make(map[string][]byte)}
}

// --- Mock SLA repo ---

type mockSLARepo struct{}

func newMockSLARepo() *mockSLARepo { return &mockSLARepo{} }

func (m *mockSLARepo) ListTargetsByProject(_ context.Context, _ uuid.UUID) ([]model.SLAStatusTarget, error) {
	return nil, nil
}
func (m *mockSLARepo) ListTargetsByProjectAndType(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) ([]model.SLAStatusTarget, error) {
	return nil, nil
}
func (m *mockSLARepo) GetTarget(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*model.SLAStatusTarget, error) {
	return nil, model.ErrNotFound
}
func (m *mockSLARepo) BulkUpsertTargets(_ context.Context, targets []model.SLAStatusTarget) ([]model.SLAStatusTarget, error) {
	return targets, nil
}
func (m *mockSLARepo) DeleteTarget(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockSLARepo) DeleteTargetsByTypeAndWorkflow(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) error {
	return nil
}
func (m *mockSLARepo) InitElapsedOnCreate(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *mockSLARepo) UpsertElapsedOnEnter(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *mockSLARepo) UpdateElapsedOnLeave(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *mockSLARepo) UpdateElapsedOnLeaveWithSeconds(_ context.Context, _ uuid.UUID, _ string, _ int) error {
	return nil
}
func (m *mockSLARepo) GetElapsed(_ context.Context, _ uuid.UUID, _ string) (*model.SLAElapsed, error) {
	return nil, model.ErrNotFound
}
func (m *mockSLARepo) ListElapsedByWorkItemIDs(_ context.Context, _ []uuid.UUID) ([]model.SLAElapsed, error) {
	return nil, nil
}

func (m *mockStorage) Put(_ context.Context, key string, reader io.Reader, _ int64, contentType string) (*storage.ObjectInfo, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	m.objects[key] = data
	return &storage.ObjectInfo{Key: key, Size: int64(len(data)), ContentType: contentType}, nil
}

func (m *mockStorage) Get(_ context.Context, key string) (io.ReadCloser, *storage.ObjectInfo, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, nil, fmt.Errorf("object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), &storage.ObjectInfo{Key: key, Size: int64(len(data))}, nil
}

func (m *mockStorage) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

// --- Test helpers ---

type testWorkItemSetup struct {
	svc              *WorkItemService
	itemRepo         *mockWorkItemRepo
	eventRepo        *mockWorkItemEventRepo
	commentRepo      *mockCommentRepo
	relationRepo     *mockRelationRepo
	projectRepo      *mockProjectRepo
	memberRepo       *mockProjectMemberRepo
	workflowRepo     *mockWorkflowRepo
	typeWorkflowRepo *mockTypeWorkflowRepo
	queueRepo        *mockQueueRepo
	milestoneRepo    *mockMilestoneRepo
	attachRepo       *mockAttachmentRepo
	storage          *mockStorage
}

func newTestWorkItemService() (*WorkItemService, *mockWorkItemRepo, *mockWorkItemEventRepo, *mockProjectRepo, *mockProjectMemberRepo) {
	s := newTestWorkItemSetup()
	return s.svc, s.itemRepo, s.eventRepo, s.projectRepo, s.memberRepo
}

func newTestWorkItemSetup() *testWorkItemSetup {
	itemRepo := newMockWorkItemRepo()
	eventRepo := newMockWorkItemEventRepo()
	commentRepo := newMockCommentRepo()
	relationRepo := newMockRelationRepo()
	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	workflowRepo := newMockWorkflowRepo()
	typeWorkflowRepo := newMockTypeWorkflowRepo()
	queueRepo := newMockQueueRepo()
	milestoneRepo := newMockMilestoneRepo()
	attachRepo := newMockAttachmentRepo()
	slaRepo := newMockSLARepo()
	slaService := NewSLAService(slaRepo, projectRepo, memberRepo, workflowRepo)
	store := newMockStorage()
	svc := NewWorkItemService(itemRepo, eventRepo, commentRepo, relationRepo, attachRepo, projectRepo, memberRepo, workflowRepo, typeWorkflowRepo, queueRepo, milestoneRepo, slaRepo, slaService, store, 50*1024*1024)
	return &testWorkItemSetup{
		svc:              svc,
		itemRepo:         itemRepo,
		eventRepo:        eventRepo,
		commentRepo:      commentRepo,
		relationRepo:     relationRepo,
		projectRepo:      projectRepo,
		memberRepo:       memberRepo,
		workflowRepo:     workflowRepo,
		typeWorkflowRepo: typeWorkflowRepo,
		queueRepo:        queueRepo,
		milestoneRepo:    milestoneRepo,
		attachRepo:       attachRepo,
		storage:          store,
	}
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

// --- Comment Tests ---

func TestCreateComment_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	comment, err := s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body:       "This is a comment",
		Visibility: model.VisibilityInternal,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment.Body != "This is a comment" {
		t.Fatalf("expected body 'This is a comment', got %s", comment.Body)
	}
	if comment.AuthorID == nil || *comment.AuthorID != info.UserID {
		t.Fatal("expected author_id to match user")
	}
}

func TestCreateComment_EmptyBody(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "",
	})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestCreateComment_InvalidVisibility(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body:       "test",
		Visibility: "secret",
	})
	if err == nil {
		t.Fatal("expected error for invalid visibility")
	}
}

func TestCreateComment_CreatesEvent(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "Hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	events := s.eventRepo.events[item.ID]
	// 1 "created" + 1 "comment_added"
	found := false
	for _, e := range events {
		if e.EventType == "comment_added" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected comment_added event")
	}
}

func TestListComments_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		_, err = s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
			Body: fmt.Sprintf("Comment %d", i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	comments, err := s.svc.ListComments(context.Background(), info, "TEST", item.ItemNumber, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(comments))
	}
}

func TestUpdateComment_AuthorCanEdit(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	comment, err := s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "Original",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.svc.UpdateComment(context.Background(), info, "TEST", item.ItemNumber, comment.ID, "Updated body")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Body != "Updated body" {
		t.Fatalf("expected body 'Updated body', got %s", updated.Body)
	}
}

func TestUpdateComment_NonAuthorMemberDenied(t *testing.T) {
	s := newTestWorkItemSetup()
	author := userAuthInfo()
	other := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, author, model.ProjectRoleMember)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID, UserID: other.UserID, Role: model.ProjectRoleMember,
	})

	item, err := s.svc.Create(context.Background(), author, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	comment, err := s.svc.CreateComment(context.Background(), author, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "Original",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UpdateComment(context.Background(), other, "TEST", item.ItemNumber, comment.ID, "Hacked")
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateComment_AdminCannotEdit(t *testing.T) {
	s := newTestWorkItemSetup()
	author := userAuthInfo()
	admin := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, author, model.ProjectRoleMember)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID, UserID: admin.UserID, Role: model.ProjectRoleAdmin,
	})

	item, err := s.svc.Create(context.Background(), author, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	comment, err := s.svc.CreateComment(context.Background(), author, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "Original",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UpdateComment(context.Background(), admin, "TEST", item.ItemNumber, comment.ID, "Admin edit")
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteComment_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	comment, err := s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "To delete",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = s.svc.DeleteComment(context.Background(), info, "TEST", item.ItemNumber, comment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// --- Relation Tests ---

func TestCreateRelation_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item1, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	item2, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	rel, err := s.svc.CreateRelation(context.Background(), info, "TEST", item1.ItemNumber, CreateRelationInput{
		TargetDisplayID: fmt.Sprintf("TEST-%d", item2.ItemNumber),
		RelationType:    model.RelationBlocks,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rel.RelationType != model.RelationBlocks {
		t.Fatalf("expected relation type 'blocks', got %s", rel.RelationType)
	}
	if rel.SourceDisplayID != fmt.Sprintf("TEST-%d", item1.ItemNumber) {
		t.Fatalf("expected source display ID TEST-%d, got %s", item1.ItemNumber, rel.SourceDisplayID)
	}
}

func TestCreateRelation_InvalidType(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item1, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	item2, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateRelation(context.Background(), info, "TEST", item1.ItemNumber, CreateRelationInput{
		TargetDisplayID: fmt.Sprintf("TEST-%d", item2.ItemNumber),
		RelationType:    "depends_on",
	})
	if err == nil {
		t.Fatal("expected error for invalid relation type")
	}
}

func TestCreateRelation_SelfRelation(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateRelation(context.Background(), info, "TEST", item.ItemNumber, CreateRelationInput{
		TargetDisplayID: fmt.Sprintf("TEST-%d", item.ItemNumber),
		RelationType:    model.RelationRelatesTo,
	})
	if err == nil {
		t.Fatal("expected error for self-relation")
	}
}

func TestCreateRelation_InvalidDisplayID(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateRelation(context.Background(), info, "TEST", item.ItemNumber, CreateRelationInput{
		TargetDisplayID: "invalid",
		RelationType:    model.RelationBlocks,
	})
	if err == nil {
		t.Fatal("expected error for invalid display ID")
	}
}

func TestListRelations_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item1, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	item2, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.CreateRelation(context.Background(), info, "TEST", item1.ItemNumber, CreateRelationInput{
		TargetDisplayID: fmt.Sprintf("TEST-%d", item2.ItemNumber),
		RelationType:    model.RelationBlocks,
	})
	if err != nil {
		t.Fatal(err)
	}

	relations, err := s.svc.ListRelations(context.Background(), info, "TEST", item1.ItemNumber)
	if err != nil {
		t.Fatal(err)
	}
	if len(relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(relations))
	}
}

func TestDeleteRelation_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item1, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	item2, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	rel, err := s.svc.CreateRelation(context.Background(), info, "TEST", item1.ItemNumber, CreateRelationInput{
		TargetDisplayID: fmt.Sprintf("TEST-%d", item2.ItemNumber),
		RelationType:    model.RelationBlocks,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = s.svc.DeleteRelation(context.Background(), info, "TEST", item1.ItemNumber, rel.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// --- Event Tests ---

func TestListEvents_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	events, err := s.svc.ListEvents(context.Background(), info, "TEST", item.ItemNumber, "")
	if err != nil {
		t.Fatal(err)
	}
	// At least 1 "created" event
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}
}

func TestListEvents_VisibilityFilter(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleMember)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// Add a public comment (creates a public event)
	_, err = s.svc.CreateComment(context.Background(), info, "TEST", item.ItemNumber, CreateCommentInput{
		Body:       "Public comment",
		Visibility: model.VisibilityPublic,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Filter for public events only
	events, err := s.svc.ListEvents(context.Background(), info, "TEST", item.ItemNumber, model.VisibilityPublic)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Visibility != model.VisibilityPublic {
			t.Fatalf("expected only public events, got visibility %s", e.Visibility)
		}
	}
}

// --- Update branch tests ---

func TestUpdateWorkItem_Description(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	desc := "New description"
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Description == nil || *updated.Description != "New description" {
		t.Fatalf("expected description 'New description', got %v", updated.Description)
	}
}

func TestUpdateWorkItem_EmptyTitle(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	empty := "   "
	_, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Title: &empty,
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestUpdateWorkItem_InvalidType(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	badType := "invalid_type"
	_, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Type: &badType,
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestUpdateWorkItem_InvalidVisibility(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	badVis := "secret"
	_, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Visibility: &badVis,
	})
	if err == nil {
		t.Fatal("expected error for invalid visibility")
	}
}

func TestUpdateWorkItem_ChangeType(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	newType := model.WorkItemTypeBug
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Type: &newType,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Type != model.WorkItemTypeBug {
		t.Fatalf("expected type 'bug', got %s", updated.Type)
	}
}

func TestUpdateWorkItem_ChangeVisibility(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	newVis := model.VisibilityPublic
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Visibility: &newVis,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Visibility != model.VisibilityPublic {
		t.Fatalf("expected visibility 'public', got %s", updated.Visibility)
	}
}

func TestUpdateWorkItem_AssignAndClear(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	assignee := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    assignee.UserID,
		Role:      model.ProjectRoleMember,
	})

	created, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// Assign
	updated, err := s.svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		AssigneeID: &assignee.UserID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != assignee.UserID {
		t.Fatal("expected assignee to be set")
	}

	// Clear
	updated, err = s.svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		ClearAssignee: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AssigneeID != nil {
		t.Fatal("expected assignee to be cleared")
	}
}

func TestUpdateWorkItem_AssignNonMemberDenied(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	nonMember := uuid.New()
	_, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		AssigneeID: &nonMember,
	})
	if err == nil {
		t.Fatal("expected error for assigning non-member")
	}
}

func TestUpdateWorkItem_Labels(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	labels := []string{"frontend", "urgent"}
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		Labels: &labels,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updated.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(updated.Labels))
	}
}

func TestUpdateWorkItem_DueDateSetAndClear(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// Set due date
	due := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		DueDate: &due,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DueDate == nil {
		t.Fatal("expected due date to be set")
	}

	// Clear due date
	updated, err = svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		ClearDueDate: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DueDate != nil {
		t.Fatal("expected due date to be cleared")
	}
}

func TestUpdateWorkItem_QueueAndMilestone(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	queue := &model.Queue{ID: uuid.New(), ProjectID: project.ID, Name: "Support", QueueType: "support"}
	s.queueRepo.Create(context.Background(), queue)

	milestone := &model.Milestone{ID: uuid.New(), ProjectID: project.ID, Name: "v1.0", Status: "open"}
	s.milestoneRepo.Create(context.Background(), milestone)

	created, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// Set queue and milestone
	updated, err := s.svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		QueueID:     &queue.ID,
		MilestoneID: &milestone.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.QueueID == nil || *updated.QueueID != queue.ID {
		t.Fatal("expected queue to be set")
	}
	if updated.MilestoneID == nil || *updated.MilestoneID != milestone.ID {
		t.Fatal("expected milestone to be set")
	}

	// Clear both
	updated, err = s.svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		ClearQueue:     true,
		ClearMilestone: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.QueueID != nil {
		t.Fatal("expected queue to be cleared")
	}
	if updated.MilestoneID != nil {
		t.Fatal("expected milestone to be cleared")
	}
}

func TestUpdateWorkItem_QueueWrongProject(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	otherProjectID := uuid.New()
	queue := &model.Queue{ID: uuid.New(), ProjectID: otherProjectID, Name: "Other", QueueType: "support"}
	s.queueRepo.Create(context.Background(), queue)

	created, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		QueueID: &queue.ID,
	})
	if err == nil {
		t.Fatal("expected error for queue from wrong project")
	}
}

func TestUpdateWorkItem_MilestoneWrongProject(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	otherProjectID := uuid.New()
	milestone := &model.Milestone{ID: uuid.New(), ProjectID: otherProjectID, Name: "Other", Status: "open"}
	s.milestoneRepo.Create(context.Background(), milestone)

	created, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		MilestoneID: &milestone.ID,
	})
	if err == nil {
		t.Fatal("expected error for milestone from wrong project")
	}
}

func TestUpdateWorkItem_CustomFields(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	created, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	fields := map[string]interface{}{"severity": "high", "component": "auth"}
	updated, err := svc.Update(context.Background(), info, "TEST", created.ItemNumber, UpdateWorkItemInput{
		CustomFields: fields,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CustomFields["severity"] != "high" {
		t.Fatalf("expected custom field severity=high, got %v", updated.CustomFields["severity"])
	}
}

// --- Attachment tests ---

func TestUploadAttachment_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	data := bytes.NewReader([]byte("hello world"))
	attachment, err := s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Size:        11,
		Comment:     "test file",
		Reader:      data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attachment.Filename != "test.txt" {
		t.Fatalf("expected filename 'test.txt', got %s", attachment.Filename)
	}
	if attachment.SizeBytes != 11 {
		t.Fatalf("expected size 11, got %d", attachment.SizeBytes)
	}
	if attachment.Comment != "test file" {
		t.Fatalf("expected comment 'test file', got %s", attachment.Comment)
	}
}

func TestUploadAttachment_PathTraversal(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	data := bytes.NewReader([]byte("content"))
	attachment, err := s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename:    "../../../etc/passwd",
		ContentType: "text/plain",
		Size:        7,
		Reader:      data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// filepath.Base should strip traversal
	if attachment.Filename != "passwd" {
		t.Fatalf("expected sanitized filename 'passwd', got %s", attachment.Filename)
	}
}

func TestUploadAttachment_EmptyFilename(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "",
		Size:     10,
		Reader:   bytes.NewReader([]byte("content")),
	})
	if err == nil {
		t.Fatal("expected error for empty filename")
	}
}

func TestUploadAttachment_DotDotFilename(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "..",
		Size:     10,
		Reader:   bytes.NewReader([]byte("content")),
	})
	if err == nil {
		t.Fatal("expected error for '..' filename")
	}
}

func TestUploadAttachment_EmptyFile(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "empty.txt",
		Size:     0,
		Reader:   bytes.NewReader(nil),
	})
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestUploadAttachment_TooLarge(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "huge.bin",
		Size:     100 * 1024 * 1024, // 100MB > 50MB limit
		Reader:   bytes.NewReader(nil),
	})
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

func TestListAttachments_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// Upload two files
	for _, name := range []string{"a.txt", "b.txt"} {
		_, err := s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
			Filename: name, ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	attachments, err := s.svc.ListAttachments(context.Background(), info, "TEST", item.ItemNumber)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(attachments))
	}
}

func TestGetAttachmentFile_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, err := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	fileContent := []byte("file contents here")
	uploaded, err := s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "doc.txt", ContentType: "text/plain", Size: int64(len(fileContent)), Reader: bytes.NewReader(fileContent),
	})
	if err != nil {
		t.Fatal(err)
	}

	attachment, reader, err := s.svc.GetAttachmentFile(context.Background(), info, "TEST", item.ItemNumber, uploaded.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	if attachment.Filename != "doc.txt" {
		t.Fatalf("expected filename 'doc.txt', got %s", attachment.Filename)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "file contents here" {
		t.Fatalf("expected file content 'file contents here', got %s", string(data))
	}
}

func TestGetAttachmentFile_WrongWorkItem(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item1, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	item2, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())

	uploaded, _ := s.svc.UploadAttachment(context.Background(), info, "TEST", item1.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	// Try to access item1's attachment via item2's URL
	_, _, err := s.svc.GetAttachmentFile(context.Background(), info, "TEST", item2.ItemNumber, uploaded.ID)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteAttachment_UploaderCanDelete(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	uploaded, _ := s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	err := s.svc.DeleteAttachment(context.Background(), info, "TEST", item.ItemNumber, uploaded.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAttachment_NonUploaderMemberDenied(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	member := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    member.UserID,
		Role:      model.ProjectRoleMember,
	})

	item, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())
	uploaded, _ := s.svc.UploadAttachment(context.Background(), owner, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	// Member who didn't upload should be denied
	err := s.svc.DeleteAttachment(context.Background(), member, "TEST", item.ItemNumber, uploaded.ID)
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteAttachment_WrongWorkItem(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item1, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	item2, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())

	uploaded, _ := s.svc.UploadAttachment(context.Background(), info, "TEST", item1.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	err := s.svc.DeleteAttachment(context.Background(), info, "TEST", item2.ItemNumber, uploaded.ID)
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateAttachmentComment_Success(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	uploaded, _ := s.svc.UploadAttachment(context.Background(), info, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Comment: "old", Reader: bytes.NewReader([]byte("hello")),
	})

	updated, err := s.svc.UpdateAttachmentComment(context.Background(), info, "TEST", item.ItemNumber, uploaded.ID, "new comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Comment != "new comment" {
		t.Fatalf("expected comment 'new comment', got %s", updated.Comment)
	}
}

func TestUpdateAttachmentComment_NonUploaderMemberDenied(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	member := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    member.UserID,
		Role:      model.ProjectRoleMember,
	})

	item, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())
	uploaded, _ := s.svc.UploadAttachment(context.Background(), owner, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	_, err := s.svc.UpdateAttachmentComment(context.Background(), member, "TEST", item.ItemNumber, uploaded.ID, "hacked")
	if err != model.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateAttachmentComment_WrongWorkItem(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	item1, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())
	item2, _ := s.svc.Create(context.Background(), info, "TEST", validCreateInput())

	uploaded, _ := s.svc.UploadAttachment(context.Background(), info, "TEST", item1.ItemNumber, CreateAttachmentInput{
		Filename: "file.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	_, err := s.svc.UpdateAttachmentComment(context.Background(), info, "TEST", item2.ItemNumber, uploaded.ID, "wrong")
	if err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Admin bypass tests ---

func TestListComments_AdminBypass(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)

	item, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())
	s.svc.CreateComment(context.Background(), owner, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "a comment", Visibility: model.VisibilityInternal,
	})

	comments, err := s.svc.ListComments(context.Background(), admin, "TEST", item.ItemNumber, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
}

func TestListAttachments_AdminBypass(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)

	item, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())
	s.svc.UploadAttachment(context.Background(), owner, "TEST", item.ItemNumber, CreateAttachmentInput{
		Filename: "f.txt", ContentType: "text/plain", Size: 5, Reader: bytes.NewReader([]byte("hello")),
	})

	attachments, err := s.svc.ListAttachments(context.Background(), admin, "TEST", item.ItemNumber)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
}

func TestListEvents_AdminBypass(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)

	item, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())

	events, err := s.svc.ListEvents(context.Background(), admin, "TEST", item.ItemNumber, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// At least the "created" event
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}
}

func TestDeleteComment_AdminCanDelete(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID, UserID: admin.UserID, Role: model.ProjectRoleAdmin,
	})

	item, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())
	comment, _ := s.svc.CreateComment(context.Background(), owner, "TEST", item.ItemNumber, CreateCommentInput{
		Body: "delete me", Visibility: model.VisibilityInternal,
	})

	err := s.svc.DeleteComment(context.Background(), admin, "TEST", item.ItemNumber, comment.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteRelation_AdminCanDelete(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)
	s.memberRepo.Add(context.Background(), &model.ProjectMember{
		ID: uuid.New(), ProjectID: project.ID, UserID: admin.UserID, Role: model.ProjectRoleAdmin,
	})

	item1, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())
	item2, _ := s.svc.Create(context.Background(), owner, "TEST", validCreateInput())

	rel, err := s.svc.CreateRelation(context.Background(), owner, "TEST", item1.ItemNumber, CreateRelationInput{
		TargetDisplayID: fmt.Sprintf("TEST-%d", item2.ItemNumber),
		RelationType:    model.RelationBlocks,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = s.svc.DeleteRelation(context.Background(), admin, "TEST", item1.ItemNumber, rel.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkItemList_AdminBypass(t *testing.T) {
	s := newTestWorkItemSetup()
	owner := userAuthInfo()
	admin := adminAuthInfo()
	setupProjectWithMember(t, s.projectRepo, s.memberRepo, owner, model.ProjectRoleOwner)

	s.svc.Create(context.Background(), owner, "TEST", validCreateInput())

	list, err := s.svc.List(context.Background(), admin, "TEST", &model.WorkItemFilter{Limit: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Total < 1 {
		t.Fatal("expected at least 1 item")
	}
}

// --- Type-Workflow Resolution Tests ---

func createTestWorkflow(name string, statuses []model.WorkflowStatus, transitions []model.WorkflowTransition) *model.Workflow {
	return &model.Workflow{
		ID:          uuid.New(),
		Name:        name,
		IsDefault:   true,
		Statuses:    statuses,
		Transitions: transitions,
	}
}

func TestCreate_UsesTypeSpecificWorkflow(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	// Create two different workflows
	taskWf := createTestWorkflow("Task Workflow", []model.WorkflowStatus{
		{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{
		{FromStatus: "open", ToStatus: "done"},
	})
	ticketWf := createTestWorkflow("Ticket Workflow", []model.WorkflowStatus{
		{Name: "new", DisplayName: "New", Category: model.CategoryTodo, Position: 0},
		{Name: "resolved", DisplayName: "Resolved", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{
		{FromStatus: "new", ToStatus: "resolved"},
	})

	s.workflowRepo.Create(context.Background(), taskWf)
	s.workflowRepo.Create(context.Background(), ticketWf)

	// Map task → taskWf, ticket → ticketWf
	ctx := context.Background()
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: taskWf.ID,
	})
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTicket, WorkflowID: ticketWf.ID,
	})

	// Create a task — should get "open" as initial status
	taskItem, err := s.svc.Create(ctx, info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTask, Title: "A task", Priority: model.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("expected no error creating task, got %v", err)
	}
	if taskItem.Status != "open" {
		t.Errorf("expected task initial status 'open', got %q", taskItem.Status)
	}

	// Create a ticket — should get "new" as initial status
	ticketItem, err := s.svc.Create(ctx, info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTicket, Title: "A ticket", Priority: model.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("expected no error creating ticket, got %v", err)
	}
	if ticketItem.Status != "new" {
		t.Errorf("expected ticket initial status 'new', got %q", ticketItem.Status)
	}
}

func TestCreate_FallsBackToDefaultWorkflow(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)

	// Create a workflow and set it as project default, but don't add type mappings
	wf := createTestWorkflow("Default Workflow", []model.WorkflowStatus{
		{Name: "pending", DisplayName: "Pending", Category: model.CategoryTodo, Position: 0},
		{Name: "closed", DisplayName: "Closed", Category: model.CategoryDone, Position: 1},
	}, nil)
	s.workflowRepo.Create(context.Background(), wf)
	project.DefaultWorkflowID = &wf.ID
	s.projectRepo.Update(context.Background(), project)

	// Create an item — no type mapping exists, should fall back to project default
	item, err := s.svc.Create(context.Background(), info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTask, Title: "Fallback test", Priority: model.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Status != "pending" {
		t.Errorf("expected fallback initial status 'pending', got %q", item.Status)
	}
}

func TestUpdate_TypeChange_StatusCompatible(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)
	ctx := context.Background()

	// Both workflows share "open" status
	taskWf := createTestWorkflow("Task Workflow", []model.WorkflowStatus{
		{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{{FromStatus: "open", ToStatus: "done"}})
	bugWf := createTestWorkflow("Bug Workflow", []model.WorkflowStatus{
		{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		{Name: "fixed", DisplayName: "Fixed", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{{FromStatus: "open", ToStatus: "fixed"}})

	s.workflowRepo.Create(ctx, taskWf)
	s.workflowRepo.Create(ctx, bugWf)
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: taskWf.ID,
	})
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeBug, WorkflowID: bugWf.ID,
	})

	// Create as task with "open"
	item, _ := s.svc.Create(ctx, info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTask, Title: "Type change test", Priority: model.PriorityMedium,
	})

	// Change type to bug — "open" exists in both, should succeed
	newType := model.WorkItemTypeBug
	updated, err := s.svc.Update(ctx, info, "TEST", item.ItemNumber, UpdateWorkItemInput{Type: &newType})
	if err != nil {
		t.Fatalf("expected no error for compatible type change, got %v", err)
	}
	if updated.Type != model.WorkItemTypeBug {
		t.Errorf("expected type 'bug', got %q", updated.Type)
	}
}

func TestUpdate_TypeChange_StatusIncompatible(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)
	ctx := context.Background()

	// Task workflow has "open", ticket workflow does NOT
	taskWf := createTestWorkflow("Task Workflow", []model.WorkflowStatus{
		{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{{FromStatus: "open", ToStatus: "done"}})
	ticketWf := createTestWorkflow("Ticket Workflow", []model.WorkflowStatus{
		{Name: "new", DisplayName: "New", Category: model.CategoryTodo, Position: 0},
		{Name: "resolved", DisplayName: "Resolved", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{{FromStatus: "new", ToStatus: "resolved"}})

	s.workflowRepo.Create(ctx, taskWf)
	s.workflowRepo.Create(ctx, ticketWf)
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: taskWf.ID,
	})
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTicket, WorkflowID: ticketWf.ID,
	})

	// Create as task with "open"
	item, _ := s.svc.Create(ctx, info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTask, Title: "Incompatible test", Priority: model.PriorityMedium,
	})

	// Change type to ticket — "open" doesn't exist in ticket workflow → ErrStatusIncompatible
	newType := model.WorkItemTypeTicket
	_, err := s.svc.Update(ctx, info, "TEST", item.ItemNumber, UpdateWorkItemInput{Type: &newType})
	if err == nil {
		t.Fatal("expected ErrStatusIncompatible, got nil")
	}
	if !errors.Is(err, model.ErrStatusIncompatible) {
		t.Errorf("expected ErrStatusIncompatible, got %v", err)
	}
}

func TestUpdate_TypeChange_WithNewStatus(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)
	ctx := context.Background()

	taskWf := createTestWorkflow("Task Workflow", []model.WorkflowStatus{
		{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{{FromStatus: "open", ToStatus: "done"}})
	ticketWf := createTestWorkflow("Ticket Workflow", []model.WorkflowStatus{
		{Name: "new", DisplayName: "New", Category: model.CategoryTodo, Position: 0},
		{Name: "resolved", DisplayName: "Resolved", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{{FromStatus: "new", ToStatus: "resolved"}})

	s.workflowRepo.Create(ctx, taskWf)
	s.workflowRepo.Create(ctx, ticketWf)
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: taskWf.ID,
	})
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTicket, WorkflowID: ticketWf.ID,
	})

	// Create as task with "open"
	item, _ := s.svc.Create(ctx, info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTask, Title: "Type+status test", Priority: model.PriorityMedium,
	})

	// Change type to ticket AND provide a valid new status — should succeed
	newType := model.WorkItemTypeTicket
	newStatus := "new"
	updated, err := s.svc.Update(ctx, info, "TEST", item.ItemNumber, UpdateWorkItemInput{
		Type:   &newType,
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("expected no error for type+status change, got %v", err)
	}
	if updated.Type != model.WorkItemTypeTicket {
		t.Errorf("expected type 'ticket', got %q", updated.Type)
	}
	if updated.Status != "new" {
		t.Errorf("expected status 'new', got %q", updated.Status)
	}
}

func TestUpdate_StatusValidatedAgainstTypeWorkflow(t *testing.T) {
	s := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, s.projectRepo, s.memberRepo, info, model.ProjectRoleOwner)
	ctx := context.Background()

	wf := createTestWorkflow("Task Workflow", []model.WorkflowStatus{
		{Name: "open", DisplayName: "Open", Category: model.CategoryTodo, Position: 0},
		{Name: "done", DisplayName: "Done", Category: model.CategoryDone, Position: 1},
	}, []model.WorkflowTransition{
		{FromStatus: "open", ToStatus: "done"},
	})

	s.workflowRepo.Create(ctx, wf)
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: project.ID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: wf.ID,
	})

	item, _ := s.svc.Create(ctx, info, "TEST", CreateWorkItemInput{
		Type: model.WorkItemTypeTask, Title: "Transition test", Priority: model.PriorityMedium,
	})

	// Valid transition: open → done
	status := "done"
	updated, err := s.svc.Update(ctx, info, "TEST", item.ItemNumber, UpdateWorkItemInput{Status: &status})
	if err != nil {
		t.Fatalf("expected no error for valid transition, got %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("expected status 'done', got %q", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Error("expected resolved_at to be set when transitioning to done")
	}

	// Invalid transition: done → open (not defined)
	invalidStatus := "open"
	_, err = s.svc.Update(ctx, info, "TEST", item.ItemNumber, UpdateWorkItemInput{Status: &invalidStatus})
	if err == nil {
		t.Fatal("expected error for invalid transition, got nil")
	}
	if !errors.Is(err, model.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestResolveWorkflowID_Priority(t *testing.T) {
	s := newTestWorkItemSetup()
	ctx := context.Background()

	projectID := uuid.New()
	typeWfID := uuid.New()
	fallbackID := uuid.New()

	// Add a type-specific mapping
	s.typeWorkflowRepo.Upsert(ctx, &model.ProjectTypeWorkflow{
		ID: uuid.New(), ProjectID: projectID,
		WorkItemType: model.WorkItemTypeTask, WorkflowID: typeWfID,
	})

	// Type-specific mapping should take priority
	result, err := s.svc.resolveWorkflowID(ctx, projectID, model.WorkItemTypeTask, &fallbackID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != typeWfID {
		t.Errorf("expected type-specific workflow %s, got %s", typeWfID, result)
	}

	// No mapping for bug → should fall back
	result, err = s.svc.resolveWorkflowID(ctx, projectID, model.WorkItemTypeBug, &fallbackID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != fallbackID {
		t.Errorf("expected fallback workflow %s, got %s", fallbackID, result)
	}

	// No mapping, no fallback → should return error
	_, err = s.svc.resolveWorkflowID(ctx, projectID, model.WorkItemTypeBug, nil)
	if err == nil {
		t.Fatal("expected error when no mapping and no fallback")
	}
}

// --- Complexity tests ---

func TestCreateWorkItem_WithComplexity(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	complexity := 5
	input := validCreateInput()
	input.Complexity = &complexity

	item, err := svc.Create(context.Background(), info, "TEST", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Complexity == nil || *item.Complexity != 5 {
		t.Fatalf("expected complexity 5, got %v", item.Complexity)
	}
}

func TestCreateWorkItem_ComplexityValidatedAgainstProject(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	project := setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	// Set allowed values on the project
	project.AllowedComplexityValues = []int{1, 2, 3, 5, 8}
	projectRepo.Update(context.Background(), project)

	// Valid value should succeed
	complexity := 3
	input := validCreateInput()
	input.Complexity = &complexity
	item, err := svc.Create(context.Background(), info, "TEST", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Complexity == nil || *item.Complexity != 3 {
		t.Fatalf("expected complexity 3, got %v", item.Complexity)
	}

	// Invalid value should fail
	invalidComplexity := 4
	input2 := validCreateInput()
	input2.Complexity = &invalidComplexity
	_, err = svc.Create(context.Background(), info, "TEST", input2)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for disallowed complexity, got %v", err)
	}
}

func TestCreateWorkItem_ComplexityMustBePositive(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	zero := 0
	input := validCreateInput()
	input.Complexity = &zero
	_, err := svc.Create(context.Background(), info, "TEST", input)
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for zero complexity, got %v", err)
	}
}

func TestUpdateWorkItem_SetAndClearComplexity(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	item, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if item.Complexity != nil {
		t.Fatal("expected nil complexity on creation")
	}

	// Set complexity
	complexity := 8
	updated, err := svc.Update(context.Background(), info, "TEST", item.ItemNumber, UpdateWorkItemInput{Complexity: &complexity})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Complexity == nil || *updated.Complexity != 8 {
		t.Fatalf("expected complexity 8, got %v", updated.Complexity)
	}

	// Clear complexity
	updated, err = svc.Update(context.Background(), info, "TEST", item.ItemNumber, UpdateWorkItemInput{ClearComplexity: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Complexity != nil {
		t.Fatalf("expected nil complexity after clear, got %v", updated.Complexity)
	}
}

func TestUpdateWorkItem_ComplexityValidatedAgainstProject(t *testing.T) {
	svc, _, _, projectRepo, memberRepo := newTestWorkItemService()
	info := userAuthInfo()
	project := setupProjectWithMember(t, projectRepo, memberRepo, info, model.ProjectRoleOwner)

	project.AllowedComplexityValues = []int{1, 2, 3, 5, 8}
	projectRepo.Update(context.Background(), project)

	item, err := svc.Create(context.Background(), info, "TEST", validCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	// Valid value should succeed
	complexity := 5
	_, err = svc.Update(context.Background(), info, "TEST", item.ItemNumber, UpdateWorkItemInput{Complexity: &complexity})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Invalid value should fail
	badComplexity := 4
	_, err = svc.Update(context.Background(), info, "TEST", item.ItemNumber, UpdateWorkItemInput{Complexity: &badComplexity})
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected ErrValidation for disallowed complexity, got %v", err)
	}
}
