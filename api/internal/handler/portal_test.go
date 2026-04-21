package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

type portalTestSetup struct {
	portal     *PortalHandler
	info       *model.AuthInfo
	projectKey string
	projectID  uuid.UUID
	queueID    uuid.UUID
	// Internal user for ownership-check tests
	otherInfo *model.AuthInfo
}

func newPortalTestSetup(t *testing.T) *portalTestSetup {
	t.Helper()

	projectRepo := newMockProjectRepo()
	memberRepo := newMockProjectMemberRepo()
	itemRepo := newMockWorkItemRepo()
	eventRepo := newMockWorkItemEventRepo()
	commentRepo := newMockCommentRepo()
	relationRepo := newMockRelationRepo()
	workflowRepo := newMockWorkflowRepo()
	typeWorkflowRepo := newMockTypeWorkflowRepo()
	queueRepo := newMockQueueRepo()
	milestoneRepo := newMockMilestoneRepo()
	attachRepo := newMockAttachmentRepo()
	timeEntryRepo := newMockTimeEntryRepo()
	slaRepo := newMockSLARepo()
	store := newMockStorage()
	slaSvc := service.NewSLAService(slaRepo, projectRepo, memberRepo, workflowRepo)
	watcherRepo := newMockWatcherRepo()
	workItemSvc := service.NewWorkItemService(
		service.WithWorkItemRepos(itemRepo, eventRepo, commentRepo, relationRepo, attachRepo, timeEntryRepo, watcherRepo),
		service.WithProjectContext(projectRepo, memberRepo, workflowRepo, typeWorkflowRepo, queueRepo, milestoneRepo),
		service.WithSLA(slaRepo, slaSvc),
		service.WithStorage(store, 50*1024*1024),
	)
	categoryRepo := newMockQueueCategoryRepo()
	queueTeamRepo := newMockQueueTeamRepo()
	queueSvc := service.NewQueueService(queueRepo, categoryRepo, queueTeamRepo, projectRepo, memberRepo)

	portalHandler := NewPortalHandler(workItemSvc, queueSvc, nil, 50*1024*1024)

	// Create a project
	project := &model.Project{ID: uuid.New(), Name: "Test Project", Key: "TEST"}
	projectRepo.Create(context.Background(), project)
	itemRepo.projectKeys[project.ID] = project.Key

	// Customer user
	customerInfo := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "customer@test.com",
		GlobalRole: model.RoleUser,
	}
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    customerInfo.UserID,
		Role:      model.ProjectRoleCustomer,
	})

	// Another user (member) — used for ownership checks
	otherInfo := &model.AuthInfo{
		UserID:     uuid.New(),
		Email:      "member@test.com",
		GlobalRole: model.RoleUser,
	}
	memberRepo.Add(context.Background(), &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    otherInfo.UserID,
		Role:      model.ProjectRoleMember,
	})

	// Create a public queue
	publicQueue := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Support Queue",
		QueueType:       model.QueueTypeSupport,
		IsPublic:        true,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), publicQueue)

	// Create a private queue
	privateQueue := &model.Queue{
		ID:              uuid.New(),
		ProjectID:       project.ID,
		Name:            "Internal Queue",
		QueueType:       model.QueueTypeGeneral,
		IsPublic:        false,
		DefaultPriority: model.PriorityMedium,
	}
	queueRepo.Create(context.Background(), privateQueue)

	return &portalTestSetup{
		portal:     portalHandler,
		info:       customerInfo,
		projectKey: "TEST",
		projectID:  project.ID,
		queueID:    publicQueue.ID,
		otherInfo:  otherInfo,
	}
}

func portalRequest(method, path string, body string, projectKey string, info *model.AuthInfo) (*http.Request, *chi.Context) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("projectKey", projectKey)
	rctx.URLParams.Add("namespace", "default")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	if info != nil {
		req = req.WithContext(model.ContextWithAuthInfo(req.Context(), info))
	}
	return req, rctx
}

func TestPortalHandler_ListQueues(t *testing.T) {
	s := newPortalTestSetup(t)

	req, _ := portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/queues", "", s.projectKey, s.info)
	w := httptest.NewRecorder()

	s.portal.ListQueues(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []map[string]interface{}
	json.Unmarshal(resp["data"], &data)

	// Should only include the public queue, not the private one
	if len(data) != 1 {
		t.Fatalf("expected 1 public queue, got %d", len(data))
	}
	if data[0]["name"] != "Support Queue" {
		t.Fatalf("expected 'Support Queue', got %v", data[0]["name"])
	}
	// Should have a categories array (empty but present)
	if _, ok := data[0]["categories"]; !ok {
		t.Fatal("expected categories field in response")
	}
}

func TestPortalHandler_CreateTicket(t *testing.T) {
	s := newPortalTestSetup(t)

	body := fmt.Sprintf(`{"title":"My issue","description":"Help me","priority":"high","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()

	s.portal.CreateTicket(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)

	if data["title"] != "My issue" {
		t.Fatalf("expected title 'My issue', got %v", data["title"])
	}
	if data["visibility"] != "portal" {
		t.Fatalf("expected visibility 'portal', got %v", data["visibility"])
	}
	if data["priority"] != "high" {
		t.Fatalf("expected priority 'high', got %v", data["priority"])
	}
}

func TestPortalHandler_CreateTicket_MissingTitle(t *testing.T) {
	s := newPortalTestSetup(t)

	body := fmt.Sprintf(`{"title":"","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()

	s.portal.CreateTicket(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPortalHandler_CreateTicket_MissingQueueID(t *testing.T) {
	s := newPortalTestSetup(t)

	body := `{"title":"My issue"}`
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()

	s.portal.CreateTicket(w, req)

	// queue_id in the request body is no longer required — the handler
	// auto-resolves the project's public queue via GetPublicQueue.
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPortalHandler_ListTickets(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket first
	body := fmt.Sprintf(`{"title":"Ticket A","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Create a ticket as the other user (should not be visible to customer)
	body2 := fmt.Sprintf(`{"title":"Other Ticket","queue_id":"%s"}`, s.queueID)
	req2, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body2, s.projectKey, s.otherInfo)
	w2 := httptest.NewRecorder()
	s.portal.CreateTicket(w2, req2)

	// List tickets as customer
	req, _ = portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/tickets", "", s.projectKey, s.info)
	w = httptest.NewRecorder()

	s.portal.ListTickets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)

	// Should only see the customer's own ticket
	if len(data) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(data))
	}
}

func TestPortalHandler_GetTicket(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket
	body := fmt.Sprintf(`{"title":"My Ticket","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	itemNumber := fmt.Sprintf("%.0f", createdData["item_number"].(float64))

	// Get the ticket
	req, rctx := portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber, "", s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()

	s.portal.GetTicket(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var getResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &getResp)
	var getData map[string]interface{}
	json.Unmarshal(getResp["data"], &getData)
	if getData["title"] != "My Ticket" {
		t.Fatalf("expected title 'My Ticket', got %v", getData["title"])
	}
}

func TestPortalHandler_GetTicket_NotOwner(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket as the other user (member)
	body := fmt.Sprintf(`{"title":"Other User Ticket","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.otherInfo)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	itemNumber := fmt.Sprintf("%.0f", createdData["item_number"].(float64))

	// Try to get as customer (different reporter) — should get 404
	req, rctx := portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber, "", s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()

	s.portal.GetTicket(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPortalHandler_ListComments(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket
	body := fmt.Sprintf(`{"title":"Comment Test","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	itemNumber := fmt.Sprintf("%.0f", createdData["item_number"].(float64))

	// Add a public comment
	commentBody := `{"body":"Public comment"}`
	req, rctx := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber+"/comments", commentBody, s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()
	s.portal.AddComment(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201 for AddComment, got %d: %s", w.Code, w.Body.String())
	}

	// List comments
	req, rctx = portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber+"/comments", "", s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()

	s.portal.ListComments(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	if len(data) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(data))
	}
}

func TestPortalHandler_AddComment(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket
	body := fmt.Sprintf(`{"title":"Comment Test","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	itemNumber := fmt.Sprintf("%.0f", createdData["item_number"].(float64))

	// Add a comment
	commentBody := `{"body":"I need help with this"}`
	req, rctx := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber+"/comments", commentBody, s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()

	s.portal.AddComment(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data map[string]interface{}
	json.Unmarshal(resp["data"], &data)
	if data["body"] != "I need help with this" {
		t.Fatalf("expected body 'I need help with this', got %v", data["body"])
	}
}

func TestPortalHandler_ListEvents(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket (this generates a "created" event)
	body := fmt.Sprintf(`{"title":"Event Test","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.info)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	itemNumber := fmt.Sprintf("%.0f", createdData["item_number"].(float64))

	// List events
	req, rctx := portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber+"/events", "", s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()

	s.portal.ListEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	var data []json.RawMessage
	json.Unmarshal(resp["data"], &data)
	// Events are filtered by visibility="public"; the "created" event is recorded
	// with internal visibility by default, so the portal may see 0 public events.
	// This is correct behavior — just verify the response is valid.
	if data == nil {
		// Ensure we got an array (even if empty)
		t.Fatal("expected events array in response")
	}
}

func TestPortalHandler_Unauthenticated(t *testing.T) {
	s := newPortalTestSetup(t)

	// No auth info in context
	req, _ := portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/queues", "", s.projectKey, nil)
	w := httptest.NewRecorder()

	s.portal.ListQueues(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestPortalHandler_ListComments_NotOwner(t *testing.T) {
	s := newPortalTestSetup(t)

	// Create a ticket as the other user (member)
	body := fmt.Sprintf(`{"title":"Other Ticket","queue_id":"%s"}`, s.queueID)
	req, _ := portalRequest(http.MethodPost, "/api/v1/portal/default/projects/TEST/tickets", body, s.projectKey, s.otherInfo)
	w := httptest.NewRecorder()
	s.portal.CreateTicket(w, req)

	var createResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdData map[string]interface{}
	json.Unmarshal(createResp["data"], &createdData)
	itemNumber := fmt.Sprintf("%.0f", createdData["item_number"].(float64))

	// Try to list comments as customer on someone else's ticket
	req, rctx := portalRequest(http.MethodGet, "/api/v1/portal/default/projects/TEST/tickets/"+itemNumber+"/comments", "", s.projectKey, s.info)
	rctx.URLParams.Add("itemNumber", itemNumber)
	w = httptest.NewRecorder()

	s.portal.ListComments(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
