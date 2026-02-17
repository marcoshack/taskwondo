package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/model"
	"github.com/marcoshack/trackforge/internal/service"
)

// WorkItemHandler handles work item endpoints.
type WorkItemHandler struct {
	items *service.WorkItemService
}

// NewWorkItemHandler creates a new WorkItemHandler.
func NewWorkItemHandler(items *service.WorkItemService) *WorkItemHandler {
	return &WorkItemHandler{items: items}
}

// --- Request DTOs ---

type createWorkItemRequest struct {
	Type         string                 `json:"type"`
	Title        string                 `json:"title"`
	Description  *string                `json:"description,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
	AssigneeID   *string                `json:"assignee_id,omitempty"`
	Labels       []string               `json:"labels,omitempty"`
	ParentID     *string                `json:"parent_id,omitempty"`
	QueueID      *string                `json:"queue_id,omitempty"`
	MilestoneID  *string                `json:"milestone_id,omitempty"`
	Visibility   string                 `json:"visibility,omitempty"`
	DueDate      *string                `json:"due_date,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// --- Response DTOs ---

type workItemResponse struct {
	ID           uuid.UUID              `json:"id"`
	ProjectKey   string                 `json:"project_key"`
	ItemNumber   int                    `json:"item_number"`
	DisplayID    string                 `json:"display_id"`
	Type         string                 `json:"type"`
	Title        string                 `json:"title"`
	Description  *string                `json:"description,omitempty"`
	Status       string                 `json:"status"`
	Priority     string                 `json:"priority"`
	AssigneeID   *uuid.UUID             `json:"assignee_id,omitempty"`
	ReporterID   uuid.UUID              `json:"reporter_id"`
	QueueID      *uuid.UUID             `json:"queue_id,omitempty"`
	MilestoneID  *uuid.UUID             `json:"milestone_id,omitempty"`
	Visibility   string                 `json:"visibility"`
	Labels       []string               `json:"labels"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	DueDate      *string                `json:"due_date,omitempty"`
	SLADeadline  *time.Time             `json:"sla_deadline,omitempty"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

func toWorkItemResponse(item *model.WorkItem, projectKey string) workItemResponse {
	resp := workItemResponse{
		ID:           item.ID,
		ProjectKey:   projectKey,
		ItemNumber:   item.ItemNumber,
		DisplayID:    fmt.Sprintf("%s-%d", projectKey, item.ItemNumber),
		Type:         item.Type,
		Title:        item.Title,
		Description:  item.Description,
		Status:       item.Status,
		Priority:     item.Priority,
		AssigneeID:   item.AssigneeID,
		ReporterID:   item.ReporterID,
		QueueID:      item.QueueID,
		MilestoneID:  item.MilestoneID,
		Visibility:   item.Visibility,
		Labels:       item.Labels,
		CustomFields: item.CustomFields,
		SLADeadline:  item.SLADeadline,
		ResolvedAt:   item.ResolvedAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
	if item.DueDate != nil {
		s := item.DueDate.Format("2006-01-02")
		resp.DueDate = &s
	}
	if resp.Labels == nil {
		resp.Labels = []string{}
	}
	if resp.CustomFields == nil {
		resp.CustomFields = map[string]interface{}{}
	}
	return resp
}

// --- Handlers ---

// Create handles POST /api/v1/projects/{projectKey}/items
func (h *WorkItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createWorkItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "type is required")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "title is required")
		return
	}

	input := service.CreateWorkItemInput{
		Type:         req.Type,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     req.Priority,
		Labels:       req.Labels,
		Visibility:   req.Visibility,
		CustomFields: req.CustomFields,
	}

	if req.AssigneeID != nil {
		id, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid assignee_id format")
			return
		}
		input.AssigneeID = &id
	}

	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent_id format")
			return
		}
		input.ParentID = &id
	}

	if req.QueueID != nil {
		id, err := uuid.Parse(*req.QueueID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid queue_id format")
			return
		}
		input.QueueID = &id
	}

	if req.MilestoneID != nil {
		id, err := uuid.Parse(*req.MilestoneID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid milestone_id format")
			return
		}
		input.MilestoneID = &id
	}

	if req.DueDate != nil {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid due_date format, expected YYYY-MM-DD")
			return
		}
		input.DueDate = &t
	}

	item, err := h.items.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to create work item")
		return
	}

	writeData(w, http.StatusCreated, toWorkItemResponse(item, projectKey))
}

// List handles GET /api/v1/projects/{projectKey}/items
func (h *WorkItemHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	q := r.URL.Query()

	filter := &model.WorkItemFilter{
		Search: q.Get("q"),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
	}

	// Parse comma-separated filters
	if v := q.Get("type"); v != "" {
		filter.Types = strings.Split(v, ",")
	}
	if v := q.Get("status"); v != "" {
		filter.Statuses = strings.Split(v, ",")
	}
	if v := q.Get("priority"); v != "" {
		filter.Priorities = strings.Split(v, ",")
	}

	// Parse assignee
	if v := q.Get("assignee"); v != "" {
		switch v {
		case "me":
			filter.AssigneeMe = true
		case "unassigned":
			filter.Unassigned = true
		default:
			id, err := uuid.Parse(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid assignee parameter")
				return
			}
			filter.AssigneeID = &id
		}
	}

	// Parse queue
	if v := q.Get("queue"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid queue parameter")
			return
		}
		filter.QueueID = &id
	}

	// Parse milestone
	if v := q.Get("milestone"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid milestone parameter")
			return
		}
		filter.MilestoneID = &id
	}

	// Parse label
	if v := q.Get("label"); v != "" {
		filter.Labels = strings.Split(v, ",")
	}

	// Parse parent
	if v := q.Get("parent"); v != "" {
		if v == "none" {
			filter.ParentNone = true
		} else {
			id, err := uuid.Parse(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent parameter")
				return
			}
			filter.ParentID = &id
		}
	}

	// Parse cursor
	if v := q.Get("cursor"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid cursor parameter")
			return
		}
		filter.Cursor = &id
	}

	// Parse limit
	if v := q.Get("limit"); v != "" {
		limit, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid limit parameter")
			return
		}
		filter.Limit = limit
	}

	result, err := h.items.List(r.Context(), info, projectKey, filter)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list work items")
		return
	}

	items := make([]workItemResponse, len(result.Items))
	for i := range result.Items {
		items[i] = toWorkItemResponse(&result.Items[i], projectKey)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
		"meta": map[string]interface{}{
			"cursor":   result.Cursor,
			"has_more": result.HasMore,
			"total":    result.Total,
		},
	})
}

// Get handles GET /api/v1/projects/{projectKey}/items/{itemNumber}
func (h *WorkItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	item, err := h.items.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to get work item")
		return
	}

	writeData(w, http.StatusOK, toWorkItemResponse(item, projectKey))
}

// Update handles PATCH /api/v1/projects/{projectKey}/items/{itemNumber}
func (h *WorkItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	// Decode into raw JSON to detect explicit nulls
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	var input service.UpdateWorkItemInput

	if v, ok := raw["title"]; ok {
		var title string
		if err := json.Unmarshal(v, &title); err == nil {
			input.Title = &title
		}
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			empty := ""
			input.Description = &empty
		} else {
			var desc string
			if err := json.Unmarshal(v, &desc); err == nil {
				input.Description = &desc
			}
		}
	}

	if v, ok := raw["status"]; ok {
		var status string
		if err := json.Unmarshal(v, &status); err == nil {
			input.Status = &status
		}
	}

	if v, ok := raw["priority"]; ok {
		var priority string
		if err := json.Unmarshal(v, &priority); err == nil {
			input.Priority = &priority
		}
	}

	if v, ok := raw["type"]; ok {
		var itemType string
		if err := json.Unmarshal(v, &itemType); err == nil {
			input.Type = &itemType
		}
	}

	if v, ok := raw["assignee_id"]; ok {
		if string(v) == "null" {
			input.ClearAssignee = true
		} else {
			var idStr string
			if err := json.Unmarshal(v, &idStr); err == nil {
				id, err := uuid.Parse(idStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid assignee_id format")
					return
				}
				input.AssigneeID = &id
			}
		}
	}

	if v, ok := raw["labels"]; ok {
		var labels []string
		if err := json.Unmarshal(v, &labels); err == nil {
			input.Labels = &labels
		}
	}

	if v, ok := raw["visibility"]; ok {
		var visibility string
		if err := json.Unmarshal(v, &visibility); err == nil {
			input.Visibility = &visibility
		}
	}

	if v, ok := raw["due_date"]; ok {
		if string(v) == "null" {
			input.ClearDueDate = true
		} else {
			var dateStr string
			if err := json.Unmarshal(v, &dateStr); err == nil {
				t, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid due_date format, expected YYYY-MM-DD")
					return
				}
				input.DueDate = &t
			}
		}
	}

	if v, ok := raw["parent_id"]; ok {
		if string(v) == "null" {
			input.ClearParent = true
		} else {
			var idStr string
			if err := json.Unmarshal(v, &idStr); err == nil {
				id, err := uuid.Parse(idStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent_id format")
					return
				}
				input.ParentID = &id
			}
		}
	}

	if v, ok := raw["queue_id"]; ok {
		if string(v) == "null" {
			input.ClearQueue = true
		} else {
			var idStr string
			if err := json.Unmarshal(v, &idStr); err == nil {
				id, err := uuid.Parse(idStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid queue_id format")
					return
				}
				input.QueueID = &id
			}
		}
	}

	if v, ok := raw["milestone_id"]; ok {
		if string(v) == "null" {
			input.ClearMilestone = true
		} else {
			var idStr string
			if err := json.Unmarshal(v, &idStr); err == nil {
				id, err := uuid.Parse(idStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid milestone_id format")
					return
				}
				input.MilestoneID = &id
			}
		}
	}

	if v, ok := raw["custom_fields"]; ok {
		var cf map[string]interface{}
		if err := json.Unmarshal(v, &cf); err == nil {
			input.CustomFields = cf
		}
	}

	item, err := h.items.Update(r.Context(), info, projectKey, itemNumber, input)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to update work item")
		return
	}

	writeData(w, http.StatusOK, toWorkItemResponse(item, projectKey))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/items/{itemNumber}
func (h *WorkItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	if err := h.items.Delete(r.Context(), info, projectKey, itemNumber); err != nil {
		handleWorkItemError(w, r, err, "failed to delete work item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Comment DTOs ---

type createCommentRequest struct {
	Body       string `json:"body"`
	Visibility string `json:"visibility,omitempty"`
}

type updateCommentRequest struct {
	Body string `json:"body"`
}

type commentResponse struct {
	ID         uuid.UUID  `json:"id"`
	AuthorID   *uuid.UUID `json:"author_id,omitempty"`
	Body       string     `json:"body"`
	Visibility string     `json:"visibility"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func toCommentResponse(c *model.Comment) commentResponse {
	return commentResponse{
		ID:         c.ID,
		AuthorID:   c.AuthorID,
		Body:       c.Body,
		Visibility: c.Visibility,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

// --- Relation DTOs ---

type createRelationRequest struct {
	TargetDisplayID string `json:"target_display_id"`
	RelationType    string `json:"relation_type"`
}

type relationResponse struct {
	ID              uuid.UUID `json:"id"`
	SourceDisplayID string    `json:"source_display_id"`
	TargetDisplayID string    `json:"target_display_id"`
	RelationType    string    `json:"relation_type"`
	CreatedBy       uuid.UUID `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}

func toRelationResponse(r *service.RelationWithDisplay) relationResponse {
	return relationResponse{
		ID:              r.ID,
		SourceDisplayID: r.SourceDisplayID,
		TargetDisplayID: r.TargetDisplayID,
		RelationType:    r.RelationType,
		CreatedBy:       r.CreatedBy,
		CreatedAt:       r.CreatedAt,
	}
}

// --- Event DTOs ---

type eventActorResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
}

type eventResponse struct {
	ID         uuid.UUID              `json:"id"`
	EventType  string                 `json:"event_type"`
	Actor      *eventActorResponse    `json:"actor,omitempty"`
	FieldName  *string                `json:"field_name,omitempty"`
	OldValue   *string                `json:"old_value,omitempty"`
	NewValue   *string                `json:"new_value,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Visibility string                 `json:"visibility"`
	CreatedAt  time.Time              `json:"created_at"`
}

func toEventResponse(e *model.WorkItemEventWithActor) eventResponse {
	resp := eventResponse{
		ID:         e.ID,
		EventType:  e.EventType,
		FieldName:  e.FieldName,
		OldValue:   e.OldValue,
		NewValue:   e.NewValue,
		Metadata:   e.Metadata,
		Visibility: e.Visibility,
		CreatedAt:  e.CreatedAt,
	}
	if resp.Metadata == nil {
		resp.Metadata = map[string]interface{}{}
	}
	if e.ActorID != nil {
		actor := &eventActorResponse{ID: *e.ActorID}
		if e.ActorDisplayName != nil {
			actor.DisplayName = *e.ActorDisplayName
		}
		resp.Actor = actor
	}
	return resp
}

// --- Comment Handlers ---

// CreateComment handles POST /api/v1/projects/{projectKey}/items/{itemNumber}/comments
func (h *WorkItemHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "body is required")
		return
	}

	comment, err := h.items.CreateComment(r.Context(), info, projectKey, itemNumber, service.CreateCommentInput{
		Body:       req.Body,
		Visibility: req.Visibility,
	})
	if err != nil {
		handleWorkItemError(w, r, err, "failed to create comment")
		return
	}

	writeData(w, http.StatusCreated, toCommentResponse(comment))
}

// ListComments handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/comments
func (h *WorkItemHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	visibility := r.URL.Query().Get("visibility")

	comments, err := h.items.ListComments(r.Context(), info, projectKey, itemNumber, visibility)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list comments")
		return
	}

	resp := make([]commentResponse, len(comments))
	for i := range comments {
		resp[i] = toCommentResponse(&comments[i])
	}

	writeData(w, http.StatusOK, resp)
}

// UpdateComment handles PATCH /api/v1/projects/{projectKey}/items/{itemNumber}/comments/{commentId}
func (h *WorkItemHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	commentID, err := uuid.Parse(chi.URLParam(r, "commentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid comment ID")
		return
	}

	var req updateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "body is required")
		return
	}

	comment, err := h.items.UpdateComment(r.Context(), info, projectKey, itemNumber, commentID, req.Body)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to update comment")
		return
	}

	writeData(w, http.StatusOK, toCommentResponse(comment))
}

// DeleteComment handles DELETE /api/v1/projects/{projectKey}/items/{itemNumber}/comments/{commentId}
func (h *WorkItemHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	commentID, err := uuid.Parse(chi.URLParam(r, "commentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid comment ID")
		return
	}

	if err := h.items.DeleteComment(r.Context(), info, projectKey, itemNumber, commentID); err != nil {
		handleWorkItemError(w, r, err, "failed to delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Relation Handlers ---

// CreateRelation handles POST /api/v1/projects/{projectKey}/items/{itemNumber}/relations
func (h *WorkItemHandler) CreateRelation(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	var req createRelationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.TargetDisplayID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "target_display_id is required")
		return
	}
	if req.RelationType == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "relation_type is required")
		return
	}

	rel, err := h.items.CreateRelation(r.Context(), info, projectKey, itemNumber, service.CreateRelationInput{
		TargetDisplayID: req.TargetDisplayID,
		RelationType:    req.RelationType,
	})
	if err != nil {
		handleWorkItemError(w, r, err, "failed to create relation")
		return
	}

	writeData(w, http.StatusCreated, toRelationResponse(rel))
}

// ListRelations handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/relations
func (h *WorkItemHandler) ListRelations(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	relations, err := h.items.ListRelations(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list relations")
		return
	}

	resp := make([]relationResponse, len(relations))
	for i := range relations {
		resp[i] = toRelationResponse(&relations[i])
	}

	writeData(w, http.StatusOK, resp)
}

// DeleteRelation handles DELETE /api/v1/projects/{projectKey}/items/{itemNumber}/relations/{relationId}
func (h *WorkItemHandler) DeleteRelation(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	relationID, err := uuid.Parse(chi.URLParam(r, "relationId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid relation ID")
		return
	}

	if err := h.items.DeleteRelation(r.Context(), info, projectKey, itemNumber, relationID); err != nil {
		handleWorkItemError(w, r, err, "failed to delete relation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Event Handlers ---

// ListEvents handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/events
func (h *WorkItemHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid item number")
		return
	}

	visibility := r.URL.Query().Get("visibility")

	events, err := h.items.ListEvents(r.Context(), info, projectKey, itemNumber, visibility)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list events")
		return
	}

	resp := make([]eventResponse, len(events))
	for i := range events {
		resp[i] = toEventResponse(&events[i])
	}

	writeData(w, http.StatusOK, resp)
}

// handleWorkItemError maps service errors to HTTP responses.
func handleWorkItemError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrInvalidTransition) {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if errors.Is(err, model.ErrConflict) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
