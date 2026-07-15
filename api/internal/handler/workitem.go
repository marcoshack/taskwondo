package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
	"github.com/marcoshack/taskwondo/internal/weburl"
)

// WorkItemHandler handles work item endpoints.
type WorkItemHandler struct {
	items         *service.WorkItemService
	sla           *service.SLAService
	maxUploadSize int64
	baseURL       string
}

// NewWorkItemHandler creates a new WorkItemHandler. baseURL is the public web
// base (e.g. https://taskwondo.org), used to build absolute work item links.
func NewWorkItemHandler(items *service.WorkItemService, sla *service.SLAService, maxUploadSize int64, baseURL string) *WorkItemHandler {
	return &WorkItemHandler{items: items, sla: sla, maxUploadSize: maxUploadSize, baseURL: baseURL}
}

// --- Request DTOs ---

type createWorkItemRequest struct {
	Type         string                 `json:"type"`
	Title        string                 `json:"title"`
	Description  *string                `json:"description,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
	AssigneeID   *string                `json:"assignee_id,omitempty"`
	Labels       []string               `json:"labels,omitempty"`
	Complexity   *int                   `json:"complexity,omitempty"`
	ParentID     *string                `json:"parent_id,omitempty"`
	QueueID      *string                `json:"queue_id,omitempty"`
	MilestoneID  *string                `json:"milestone_id,omitempty"`
	Visibility   string                 `json:"visibility,omitempty"`
	DueDate      *string                `json:"due_date,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	WatcherIDs   []string               `json:"watcher_ids,omitempty"`
}

// --- Response DTOs ---

type workItemResponse struct {
	ID               uuid.UUID              `json:"id"`
	ProjectKey       string                 `json:"project_key"`
	NamespaceSlug    string                 `json:"namespace_slug,omitempty"`
	NamespaceName    string                 `json:"namespace_name,omitempty"`
	ItemNumber       int                    `json:"item_number"`
	DisplayID        string                 `json:"display_id"`
	URL              string                 `json:"url,omitempty"`
	Type             string                 `json:"type"`
	Title            string                 `json:"title"`
	Description      *string                `json:"description,omitempty"`
	Status           string                 `json:"status"`
	Priority         string                 `json:"priority"`
	AssigneeID       *uuid.UUID             `json:"assignee_id,omitempty"`
	ReporterID       uuid.UUID              `json:"reporter_id"`
	ReporterName     string                 `json:"reporter_name"`
	QueueID          *uuid.UUID             `json:"queue_id,omitempty"`
	MilestoneID      *uuid.UUID             `json:"milestone_id,omitempty"`
	Visibility       string                 `json:"visibility"`
	Labels           []string               `json:"labels"`
	Complexity       *int                   `json:"complexity,omitempty"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
	DueDate          *string                `json:"due_date,omitempty"`
	EstimatedSeconds *int                   `json:"estimated_seconds,omitempty"`
	SLA              *model.SLAInfo         `json:"sla,omitempty"`
	SLATargetAt      *time.Time             `json:"sla_target_at,omitempty"`
	ResolvedAt       *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// toWorkItemResponse builds the response DTO for a work item. baseURL and
// namespaceSlug are used to construct the absolute web URL to the item; pass an
// empty namespaceSlug when it must be resolved separately (e.g. cross-project
// listings) and set resp.URL afterwards.
func toWorkItemResponse(item *model.WorkItem, projectKey, baseURL, namespaceSlug string) workItemResponse {
	resp := workItemResponse{
		ID:               item.ID,
		ProjectKey:       projectKey,
		ItemNumber:       item.ItemNumber,
		DisplayID:        item.DisplayID,
		URL:              weburl.WorkItem(baseURL, namespaceSlug, projectKey, item.ItemNumber),
		Type:             item.Type,
		Title:            item.Title,
		Description:      item.Description,
		Status:           item.Status,
		Priority:         item.Priority,
		AssigneeID:       item.AssigneeID,
		ReporterID:       item.ReporterID,
		ReporterName:     item.ReporterName,
		QueueID:          item.QueueID,
		MilestoneID:      item.MilestoneID,
		Visibility:       item.Visibility,
		Labels:           item.Labels,
		Complexity:       item.Complexity,
		CustomFields:     item.CustomFields,
		EstimatedSeconds: item.EstimatedSeconds,
		SLATargetAt:      item.SLATargetAt,
		ResolvedAt:       item.ResolvedAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createWorkItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "type is required")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "title is required")
		return
	}

	input := service.CreateWorkItemInput{
		Type:         req.Type,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     req.Priority,
		Labels:       req.Labels,
		Complexity:   req.Complexity,
		Visibility:   req.Visibility,
		CustomFields: req.CustomFields,
	}

	if req.AssigneeID != nil {
		id, ok := parseOptionalUUID(w, req.AssigneeID, "invalid assignee_id format")
		if !ok {
			return
		}
		input.AssigneeID = &id
	}

	if req.ParentID != nil {
		id, ok := parseOptionalUUID(w, req.ParentID, "invalid parent_id format")
		if !ok {
			return
		}
		input.ParentID = &id
	}

	if req.QueueID != nil {
		id, ok := parseOptionalUUID(w, req.QueueID, "invalid queue_id format")
		if !ok {
			return
		}
		input.QueueID = &id
	}

	if req.MilestoneID != nil {
		id, ok := parseOptionalUUID(w, req.MilestoneID, "invalid milestone_id format")
		if !ok {
			return
		}
		input.MilestoneID = &id
	}

	if req.DueDate != nil {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid due_date format, expected YYYY-MM-DD")
			return
		}
		input.DueDate = &t
	}

	for _, idStr := range req.WatcherIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid watcher_ids format")
			return
		}
		input.WatcherIDs = append(input.WatcherIDs, id)
	}

	item, err := h.items.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to create work item")
		return
	}

	resp := toWorkItemResponse(item, projectKey, h.baseURL, chi.URLParam(r, "namespace"))
	if slaMap := h.sla.ComputeSLAForItems(r.Context(), projectKey, []model.WorkItem{*item}); slaMap != nil {
		resp.SLA = slaMap[item.ID]
	}
	writeData(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/projects/{projectKey}/items
func (h *WorkItemHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
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

	// Parse assignees (multi-value) and assignee (deprecated single-value)
	assigneeOld := q.Get("assignee")
	assigneesNew := q.Get("assignees")
	if assigneeOld != "" && assigneesNew != "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "cannot use both 'assignee' and 'assignees' parameters")
		return
	}
	assigneeRaw := assigneesNew
	if assigneeOld != "" {
		log.Ctx(r.Context()).Warn().Msg("deprecated: use 'assignees' query param instead of 'assignee'")
		assigneeRaw = assigneeOld
	}
	if assigneeRaw != "" {
		for _, part := range strings.Split(assigneeRaw, ",") {
			part = strings.TrimSpace(part)
			switch part {
			case "me":
				filter.AssigneeMe = true
			case "unassigned":
				filter.Unassigned = true
			default:
				id, err := uuid.Parse(part)
				if err != nil {
					writeError(w, http.StatusBadRequest, CodeValidationError, "invalid assignee parameter")
					return
				}
				filter.AssigneeIDs = append(filter.AssigneeIDs, id)
			}
		}
	}

	// Parse queue
	if v := q.Get("queue"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid queue parameter")
			return
		}
		filter.QueueID = &id
	}

	// Parse milestones (multi-value) and milestone (deprecated single-value)
	milestoneOld := q.Get("milestone")
	milestonesNew := q.Get("milestones")
	if milestoneOld != "" && milestonesNew != "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "cannot use both 'milestone' and 'milestones' parameters")
		return
	}
	milestoneRaw := milestonesNew
	if milestoneOld != "" {
		log.Ctx(r.Context()).Warn().Msg("deprecated: use 'milestones' query param instead of 'milestone'")
		milestoneRaw = milestoneOld
	}
	if milestoneRaw != "" {
		for _, part := range strings.Split(milestoneRaw, ",") {
			part = strings.TrimSpace(part)
			if part == "none" {
				filter.MilestoneNone = true
				continue
			}
			id, err := uuid.Parse(part)
			if err != nil {
				writeError(w, http.StatusBadRequest, CodeValidationError, "invalid milestone parameter")
				return
			}
			filter.MilestoneIDs = append(filter.MilestoneIDs, id)
		}
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
				writeError(w, http.StatusBadRequest, CodeValidationError, "invalid parent parameter")
				return
			}
			filter.ParentID = &id
		}
	}

	// Parse cursor
	if v := q.Get("cursor"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid cursor parameter")
			return
		}
		filter.Cursor = &id
	}

	// Parse limit
	if v := q.Get("limit"); v != "" {
		limit, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid limit parameter")
			return
		}
		filter.Limit = limit
	}

	result, err := h.items.List(r.Context(), info, projectKey, filter)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list work items")
		return
	}

	slaMap := h.sla.ComputeSLAForItems(r.Context(), projectKey, result.Items)
	items := make([]workItemResponse, len(result.Items))
	for i := range result.Items {
		items[i] = toWorkItemResponse(&result.Items[i], projectKey, h.baseURL, chi.URLParam(r, "namespace"))
		if slaMap != nil {
			items[i].SLA = slaMap[result.Items[i].ID]
		}
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	item, err := h.items.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to get work item")
		return
	}

	resp := toWorkItemResponse(item, projectKey, h.baseURL, chi.URLParam(r, "namespace"))
	if slaMap := h.sla.ComputeSLAForItems(r.Context(), projectKey, []model.WorkItem{*item}); slaMap != nil {
		resp.SLA = slaMap[item.ID]
	}
	writeData(w, http.StatusOK, resp)
}

// Update handles PATCH /api/v1/projects/{projectKey}/items/{itemNumber}
func (h *WorkItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	// Decode into raw JSON to detect explicit nulls
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	var input service.UpdateWorkItemInput

	if v, ok := raw["title"]; ok {
		title, ok := unmarshalField[string](w, v, "title")
		if !ok {
			return
		}
		input.Title = &title
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			empty := ""
			input.Description = &empty
		} else {
			desc, ok := unmarshalField[string](w, v, "description")
			if !ok {
				return
			}
			input.Description = &desc
		}
	}

	if v, ok := raw["status"]; ok {
		status, ok := unmarshalField[string](w, v, "status")
		if !ok {
			return
		}
		input.Status = &status
	}

	if v, ok := raw["priority"]; ok {
		priority, ok := unmarshalField[string](w, v, "priority")
		if !ok {
			return
		}
		input.Priority = &priority
	}

	if v, ok := raw["type"]; ok {
		itemType, ok := unmarshalField[string](w, v, "type")
		if !ok {
			return
		}
		input.Type = &itemType
	}

	if v, ok := raw["assignee_id"]; ok {
		id, cleared, ok := unmarshalNullableUUID(w, v, "assignee_id")
		if !ok {
			return
		}
		if cleared {
			input.ClearAssignee = true
		} else {
			input.AssigneeID = &id
		}
	}

	if v, ok := raw["labels"]; ok {
		labels, ok := unmarshalField[[]string](w, v, "labels")
		if !ok {
			return
		}
		input.Labels = &labels
	}

	if v, ok := raw["visibility"]; ok {
		visibility, ok := unmarshalField[string](w, v, "visibility")
		if !ok {
			return
		}
		input.Visibility = &visibility
	}

	if v, ok := raw["due_date"]; ok {
		t, cleared, ok := unmarshalNullableDate(w, v, "due_date")
		if !ok {
			return
		}
		if cleared {
			input.ClearDueDate = true
		} else {
			input.DueDate = &t
		}
	}

	if v, ok := raw["parent_id"]; ok {
		id, cleared, ok := unmarshalNullableUUID(w, v, "parent_id")
		if !ok {
			return
		}
		if cleared {
			input.ClearParent = true
		} else {
			input.ParentID = &id
		}
	}

	if v, ok := raw["queue_id"]; ok {
		id, cleared, ok := unmarshalNullableUUID(w, v, "queue_id")
		if !ok {
			return
		}
		if cleared {
			input.ClearQueue = true
		} else {
			input.QueueID = &id
		}
	}

	if v, ok := raw["milestone_id"]; ok {
		id, cleared, ok := unmarshalNullableUUID(w, v, "milestone_id")
		if !ok {
			return
		}
		if cleared {
			input.ClearMilestone = true
		} else {
			input.MilestoneID = &id
		}
	}

	if v, ok := raw["complexity"]; ok {
		if string(v) == "null" {
			input.ClearComplexity = true
		} else {
			complexity, ok := unmarshalField[int](w, v, "complexity")
			if !ok {
				return
			}
			input.Complexity = &complexity
		}
	}

	if v, ok := raw["estimated_seconds"]; ok {
		if string(v) == "null" {
			input.ClearEstimate = true
		} else {
			est, ok := unmarshalField[int](w, v, "estimated_seconds")
			if !ok {
				return
			}
			input.EstimatedSeconds = &est
		}
	}

	if v, ok := raw["custom_fields"]; ok {
		cf, ok := unmarshalField[map[string]interface{}](w, v, "custom_fields")
		if !ok {
			return
		}
		input.CustomFields = cf
	}

	item, err := h.items.Update(r.Context(), info, projectKey, itemNumber, input)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to update work item")
		return
	}

	resp := toWorkItemResponse(item, projectKey, h.baseURL, chi.URLParam(r, "namespace"))
	if slaMap := h.sla.ComputeSLAForItems(r.Context(), projectKey, []model.WorkItem{*item}); slaMap != nil {
		resp.SLA = slaMap[item.ID]
	}
	writeData(w, http.StatusOK, resp)
}

// Delete handles DELETE /api/v1/projects/{projectKey}/items/{itemNumber}
func (h *WorkItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
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
	Body       string                `json:"body"`
	Visibility string                `json:"visibility,omitempty"`
	Anchor     *commentAnchorRequest `json:"anchor,omitempty"`
	// ParentCommentID, when set, makes this a threaded reply to an existing
	// inline comment; the reply inherits the thread root's anchor.
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty"`
}

type updateCommentRequest struct {
	Body       string `json:"body"`
	Visibility string `json:"visibility,omitempty"`
}

// commentAnchorRequest is the optional anchor sent by clients when creating
// an inline comment. Line numbers are 1-based and inclusive; column numbers
// are 1-based character offsets within their line (end_col exclusive).
type commentAnchorRequest struct {
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
	Snippet   string `json:"snippet"`
}

type commentAnchorResponse struct {
	RevisionID     uuid.UUID `json:"revision_id"`
	RevisionNumber int       `json:"revision_number"`
	StartLine      int       `json:"start_line"`
	StartCol       int       `json:"start_col"`
	EndLine        int       `json:"end_line"`
	EndCol         int       `json:"end_col"`
	Snippet        string    `json:"snippet"`
	Status         string    `json:"status"`
}

type commentResponse struct {
	ID              uuid.UUID              `json:"id"`
	AuthorID        *uuid.UUID             `json:"author_id,omitempty"`
	Body            string                 `json:"body"`
	Visibility      string                 `json:"visibility"`
	EditCount       int                    `json:"edit_count"`
	Anchor          *commentAnchorResponse `json:"anchor,omitempty"`
	ParentCommentID *uuid.UUID             `json:"parent_comment_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func toCommentResponse(c *model.Comment) commentResponse {
	resp := commentResponse{
		ID:              c.ID,
		AuthorID:        c.AuthorID,
		Body:            c.Body,
		Visibility:      c.Visibility,
		EditCount:       c.EditCount,
		ParentCommentID: c.ParentCommentID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if c.Anchor != nil {
		resp.Anchor = &commentAnchorResponse{
			RevisionID:     c.Anchor.RevisionID,
			RevisionNumber: c.Anchor.RevisionNumber,
			StartLine:      c.Anchor.StartLine,
			StartCol:       c.Anchor.StartCol,
			EndLine:        c.Anchor.EndLine,
			EndCol:         c.Anchor.EndCol,
			Snippet:        c.Anchor.Snippet,
			Status:         c.Anchor.Status,
		}
	}
	return resp
}

type descriptionRevisionResponse struct {
	ID             uuid.UUID  `json:"id"`
	RevisionNumber int        `json:"revision_number"`
	Content        string     `json:"content"`
	ContentHash    string     `json:"content_hash"`
	AuthorID       *uuid.UUID `json:"author_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toDescriptionRevisionResponse(r *model.DescriptionRevision) descriptionRevisionResponse {
	return descriptionRevisionResponse{
		ID:             r.ID,
		RevisionNumber: r.RevisionNumber,
		Content:        r.Content,
		ContentHash:    r.ContentHash,
		AuthorID:       r.AuthorID,
		CreatedAt:      r.CreatedAt,
	}
}

// --- Relation DTOs ---

type createRelationRequest struct {
	TargetDisplayID string `json:"target_display_id"`
	RelationType    string `json:"relation_type"`
}

type relationResponse struct {
	ID                   uuid.UUID `json:"id"`
	SourceDisplayID      string    `json:"source_display_id"`
	SourceTitle          string    `json:"source_title"`
	SourceStatus         string    `json:"source_status"`
	SourceStatusCategory string    `json:"source_status_category"`
	SourcePriority       string    `json:"source_priority"`
	TargetDisplayID      string    `json:"target_display_id"`
	TargetTitle          string    `json:"target_title"`
	TargetStatus         string    `json:"target_status"`
	TargetStatusCategory string    `json:"target_status_category"`
	TargetPriority       string    `json:"target_priority"`
	RelationType         string    `json:"relation_type"`
	CreatedBy            uuid.UUID `json:"created_by"`
	CreatedAt            time.Time `json:"created_at"`
}

func toRelationResponse(r *service.RelationWithDisplay) relationResponse {
	return relationResponse{
		ID:                   r.ID,
		SourceDisplayID:      r.SourceDisplayID,
		SourceTitle:          r.SourceTitle,
		SourceStatus:         r.SourceStatus,
		SourceStatusCategory: r.SourceStatusCategory,
		SourcePriority:       r.SourcePriority,
		TargetDisplayID:      r.TargetDisplayID,
		TargetTitle:          r.TargetTitle,
		TargetStatus:         r.TargetStatus,
		TargetStatusCategory: r.TargetStatusCategory,
		TargetPriority:       r.TargetPriority,
		RelationType:         r.RelationType,
		CreatedBy:            r.CreatedBy,
		CreatedAt:            r.CreatedAt,
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "body is required")
		return
	}

	var comment *model.Comment
	if req.Anchor != nil || req.ParentCommentID != nil {
		var anchorInput service.InlineAnchorInput
		if req.Anchor != nil {
			anchorInput = service.InlineAnchorInput{
				StartLine: req.Anchor.StartLine,
				StartCol:  req.Anchor.StartCol,
				EndLine:   req.Anchor.EndLine,
				EndCol:    req.Anchor.EndCol,
				Snippet:   req.Anchor.Snippet,
			}
		}
		comment, err = h.items.CreateInlineComment(r.Context(), info, projectKey, itemNumber,
			req.Body, req.Visibility, anchorInput, req.ParentCommentID)
	} else {
		comment, err = h.items.CreateComment(r.Context(), info, projectKey, itemNumber, service.CreateCommentInput{
			Body:       req.Body,
			Visibility: req.Visibility,
		})
	}
	if err != nil {
		handleWorkItemError(w, r, err, "failed to create comment")
		return
	}

	writeData(w, http.StatusCreated, toCommentResponse(comment))
}

// ListDescriptionRevisions handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/description-revisions
func (h *WorkItemHandler) ListDescriptionRevisions(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}
	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	revs, err := h.items.ListDescriptionRevisions(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list description revisions")
		return
	}
	resp := make([]descriptionRevisionResponse, len(revs))
	for i := range revs {
		resp[i] = toDescriptionRevisionResponse(&revs[i])
	}
	writeData(w, http.StatusOK, resp)
}

// GetDescriptionRevision handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/description-revisions/{revId}
func (h *WorkItemHandler) GetDescriptionRevision(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}
	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}
	revID, ok := parseUUIDParam(w, r, "revId", "invalid revision ID")
	if !ok {
		return
	}
	rev, err := h.items.GetDescriptionRevision(r.Context(), info, projectKey, itemNumber, revID)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to get description revision")
		return
	}
	writeData(w, http.StatusOK, toDescriptionRevisionResponse(rev))
}

// ListComments handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/comments
func (h *WorkItemHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	commentID, ok := parseUUIDParam(w, r, "commentId", "invalid comment ID")
	if !ok {
		return
	}

	var req updateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "body is required")
		return
	}

	comment, err := h.items.UpdateComment(r.Context(), info, projectKey, itemNumber, commentID, req.Body, req.Visibility)
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	commentID, ok := parseUUIDParam(w, r, "commentId", "invalid comment ID")
	if !ok {
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	var req createRelationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.TargetDisplayID == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "target_display_id is required")
		return
	}
	if req.RelationType == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "relation_type is required")
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	relationID, ok := parseUUIDParam(w, r, "relationId", "invalid relation ID")
	if !ok {
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
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
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

// --- Attachment DTOs ---

type attachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	UploaderID  uuid.UUID `json:"uploader_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Comment     string    `json:"comment"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func toAttachmentResponse(a *model.Attachment, namespace, projectKey string, itemNumber int) attachmentResponse {
	if namespace == "" {
		namespace = "default"
	}
	return attachmentResponse{
		ID:          a.ID,
		UploaderID:  a.UploaderID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		Comment:     a.Comment,
		DownloadURL: fmt.Sprintf("/api/v1/%s/projects/%s/items/%d/attachments/%s", namespace, projectKey, itemNumber, a.ID),
		CreatedAt:   a.CreatedAt,
	}
}

// --- Attachment Handlers ---

// UploadAttachment handles POST /api/v1/projects/{projectKey}/items/{itemNumber}/attachments
func (h *WorkItemHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	// Limit request body size (maxUploadSize + 1MB overhead for multipart headers)
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize+1024*1024)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusRequestEntityTooLarge, CodeFileTooLarge, "file exceeds maximum upload size")
			return
		}
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "file field is required")
		return
	}
	defer file.Close()

	comment := r.FormValue("comment")

	contentType := sanitizeContentType(header.Header.Get("Content-Type"))

	attachment, err := h.items.UploadAttachment(r.Context(), info, projectKey, itemNumber, service.CreateAttachmentInput{
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
		Comment:     comment,
		Reader:      file,
	})
	if err != nil {
		handleWorkItemError(w, r, err, "failed to upload attachment")
		return
	}

	writeData(w, http.StatusCreated, toAttachmentResponse(attachment, chi.URLParam(r, "namespace"), projectKey, itemNumber))
}

// ListAttachments handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/attachments
func (h *WorkItemHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	attachments, err := h.items.ListAttachments(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list attachments")
		return
	}

	ns := chi.URLParam(r, "namespace")
	resp := make([]attachmentResponse, len(attachments))
	for i := range attachments {
		resp[i] = toAttachmentResponse(&attachments[i], ns, projectKey, itemNumber)
	}

	writeData(w, http.StatusOK, resp)
}

// DownloadAttachment handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/attachments/{attachmentId}
func (h *WorkItemHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	attachmentID, ok := parseUUIDParam(w, r, "attachmentId", "invalid attachment ID")
	if !ok {
		return
	}

	attachment, reader, err := h.items.GetAttachmentFile(r.Context(), info, projectKey, itemNumber, attachmentID)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to download attachment")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", safeDownloadContentType(attachment.ContentType))
	w.Header().Set("Content-Disposition", safeContentDisposition(attachment.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(attachment.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)

	io.Copy(w, reader)
}

// DeleteAttachment handles DELETE /api/v1/projects/{projectKey}/items/{itemNumber}/attachments/{attachmentId}
func (h *WorkItemHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	attachmentID, ok := parseUUIDParam(w, r, "attachmentId", "invalid attachment ID")
	if !ok {
		return
	}

	if err := h.items.DeleteAttachment(r.Context(), info, projectKey, itemNumber, attachmentID); err != nil {
		handleWorkItemError(w, r, err, "failed to delete attachment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateAttachmentComment handles PATCH /api/v1/projects/{projectKey}/items/{itemNumber}/attachments/{attachmentId}
func (h *WorkItemHandler) UpdateAttachmentComment(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	attachmentID, ok := parseUUIDParam(w, r, "attachmentId", "invalid attachment ID")
	if !ok {
		return
	}

	var body struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid JSON body")
		return
	}

	attachment, err := h.items.UpdateAttachmentComment(r.Context(), info, projectKey, itemNumber, attachmentID, body.Comment)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to update attachment comment")
		return
	}

	writeData(w, http.StatusOK, toAttachmentResponse(attachment, chi.URLParam(r, "namespace"), projectKey, itemNumber))
}

// handleWorkItemError maps service errors to HTTP responses.
func handleWorkItemError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "resource not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, CodeForbidden, "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrInvalidTransition) {
		writeErrorFromService(w, http.StatusConflict, CodeInvalidTransition, err)
		return
	}
	if errors.Is(err, model.ErrStatusIncompatible) {
		writeErrorFromService(w, http.StatusConflict, CodeStatusIncompatible, err)
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeErrorFromService(w, http.StatusBadRequest, CodeValidationError, err)
		return
	}
	if errors.Is(err, model.ErrConflict) {
		writeErrorFromService(w, http.StatusBadRequest, CodeValidationError, err)
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}

// dangerousContentTypes are MIME types that browsers may execute as code.
var dangerousContentTypes = []string{
	"text/html",
	"text/javascript",
	"application/javascript",
	"application/xhtml+xml",
	"image/svg+xml",
}

// safeDownloadPrefixes are Content-Type prefixes considered safe for inline display.
var safeDownloadPrefixes = []string{
	"image/",
	"audio/",
	"video/",
	"text/plain",
	"application/pdf",
}

// sanitizeContentType returns a safe content type for storage.
// Dangerous types that browsers could execute are replaced with application/octet-stream.
func sanitizeContentType(ct string) string {
	if ct == "" {
		return "application/octet-stream"
	}
	mediaType, _, _ := mime.ParseMediaType(ct)
	if mediaType == "" {
		return "application/octet-stream"
	}
	lower := strings.ToLower(mediaType)
	for _, dangerous := range dangerousContentTypes {
		if lower == dangerous {
			return "application/octet-stream"
		}
	}
	return ct
}

// safeDownloadContentType returns a content type safe for browser download.
// Types not in the safe allowlist are forced to application/octet-stream.
func safeDownloadContentType(ct string) string {
	lower := strings.ToLower(ct)
	for _, prefix := range safeDownloadPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return ct
		}
	}
	return "application/octet-stream"
}

// --- Time entry DTOs ---

type createTimeEntryRequest struct {
	StartedAt       string `json:"started_at"`
	DurationSeconds int    `json:"duration_seconds"`
	Description     string `json:"description,omitempty"`
}

type updateTimeEntryRequest struct {
	StartedAt       *string `json:"started_at,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	Description     *string `json:"description,omitempty"`
}

type timeEntryResponse struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	StartedAt       time.Time `json:"started_at"`
	DurationSeconds int       `json:"duration_seconds"`
	Description     *string   `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func toTimeEntryResponse(e *model.TimeEntry) timeEntryResponse {
	return timeEntryResponse{
		ID:              e.ID,
		UserID:          e.UserID,
		StartedAt:       e.StartedAt,
		DurationSeconds: e.DurationSeconds,
		Description:     e.Description,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

type timeEntrySummaryResponse struct {
	Entries            []timeEntryResponse `json:"entries"`
	TotalLoggedSeconds int                 `json:"total_logged_seconds"`
}

// --- Time entry handlers ---

// CreateTimeEntry handles POST /api/v1/projects/{projectKey}/items/{itemNumber}/time-entries
func (h *WorkItemHandler) CreateTimeEntry(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	var req createTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.DurationSeconds <= 0 {
		writeError(w, http.StatusBadRequest, CodeValidationError, "duration_seconds must be positive")
		return
	}

	startedAt, err := time.Parse(time.RFC3339, req.StartedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid started_at format, expected RFC3339")
		return
	}

	entry, err := h.items.LogTime(r.Context(), info, projectKey, itemNumber, service.CreateTimeEntryInput{
		StartedAt:       startedAt,
		DurationSeconds: req.DurationSeconds,
		Description:     req.Description,
	})
	if err != nil {
		handleWorkItemError(w, r, err, "failed to create time entry")
		return
	}

	writeData(w, http.StatusCreated, toTimeEntryResponse(entry))
}

// ListTimeEntries handles GET /api/v1/projects/{projectKey}/items/{itemNumber}/time-entries
func (h *WorkItemHandler) ListTimeEntries(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	entries, err := h.items.ListTimeEntries(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list time entries")
		return
	}

	totalLogged, err := h.items.GetTimeEntrySummary(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to get time entry summary")
		return
	}

	resp := make([]timeEntryResponse, len(entries))
	for i := range entries {
		resp[i] = toTimeEntryResponse(&entries[i])
	}

	writeData(w, http.StatusOK, timeEntrySummaryResponse{
		Entries:            resp,
		TotalLoggedSeconds: totalLogged,
	})
}

// UpdateTimeEntry handles PATCH /api/v1/projects/{projectKey}/items/{itemNumber}/time-entries/{timeEntryId}
func (h *WorkItemHandler) UpdateTimeEntry(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	entryID, ok := parseUUIDParam(w, r, "timeEntryId", "invalid time entry ID")
	if !ok {
		return
	}

	// Decode into raw JSON to detect explicit nulls
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	var input service.UpdateTimeEntryInput

	if v, ok := raw["started_at"]; ok {
		s, ok := unmarshalField[string](w, v, "started_at")
		if !ok {
			return
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationError, "invalid started_at format, expected RFC3339")
			return
		}
		input.StartedAt = &t
	}

	if v, ok := raw["duration_seconds"]; ok {
		dur, ok := unmarshalField[int](w, v, "duration_seconds")
		if !ok {
			return
		}
		input.DurationSeconds = &dur
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			input.ClearDescription = true
		} else {
			desc, ok := unmarshalField[string](w, v, "description")
			if !ok {
				return
			}
			input.Description = &desc
		}
	}

	entry, err := h.items.UpdateTimeEntry(r.Context(), info, projectKey, itemNumber, entryID, input)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to update time entry")
		return
	}

	writeData(w, http.StatusOK, toTimeEntryResponse(entry))
}

// DeleteTimeEntry handles DELETE /api/v1/projects/{projectKey}/items/{itemNumber}/time-entries/{timeEntryId}
func (h *WorkItemHandler) DeleteTimeEntry(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	entryID, ok := parseUUIDParam(w, r, "timeEntryId", "invalid time entry ID")
	if !ok {
		return
	}

	if err := h.items.DeleteTimeEntry(r.Context(), info, projectKey, itemNumber, entryID); err != nil {
		handleWorkItemError(w, r, err, "failed to delete time entry")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Watcher DTOs ---

type addWatcherRequest struct {
	UserID string `json:"user_id"`
}

type watcherResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	AddedBy     uuid.UUID `json:"added_by"`
	AddedByName string    `json:"added_by_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type viewerWatcherResponse struct {
	Me         *viewerWatcherMe `json:"me,omitempty"`
	OtherCount int              `json:"other_count"`
}

type viewerWatcherMe struct {
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type toggleWatchResponse struct {
	IsWatching bool `json:"is_watching"`
}

// --- Watcher Handlers ---

// ListWatchers returns all watchers for a work item.
func (h *WorkItemHandler) ListWatchers(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	result, err := h.items.ListWatchers(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list watchers")
		return
	}

	if result.IsViewer {
		resp := viewerWatcherResponse{
			OtherCount: result.OtherCount,
		}
		if result.Me != nil {
			resp.Me = &viewerWatcherMe{
				UserID:    result.Me.UserID,
				CreatedAt: result.Me.CreatedAt,
			}
		}
		writeData(w, http.StatusOK, resp)
		return
	}

	watchers := make([]watcherResponse, 0, len(result.Watchers))
	for _, w := range result.Watchers {
		watchers = append(watchers, watcherResponse{
			ID:          w.ID,
			UserID:      w.UserID,
			DisplayName: w.DisplayName,
			Email:       w.Email,
			AvatarURL:   avatarURL(w.AvatarURL, w.UserID, 0),
			AddedBy:     w.AddedBy,
			AddedByName: w.AddedByName,
			CreatedAt:   w.CreatedAt,
		})
	}
	writeData(w, http.StatusOK, watchers)
}

// AddWatcher adds a user as a watcher on a work item.
func (h *WorkItemHandler) AddWatcher(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	var req addWatcherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "user_id is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid user_id")
		return
	}

	watcher, err := h.items.AddWatcher(r.Context(), info, projectKey, itemNumber, userID)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to add watcher")
		return
	}

	writeData(w, http.StatusCreated, watcherResponse{
		ID:        watcher.ID,
		UserID:    watcher.UserID,
		AddedBy:   watcher.AddedBy,
		CreatedAt: watcher.CreatedAt,
	})
}

// RemoveWatcher removes a user from a work item's watchers.
func (h *WorkItemHandler) RemoveWatcher(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	userID, ok := parseUUIDParam(w, r, "userId", "invalid user ID")
	if !ok {
		return
	}

	if err := h.items.RemoveWatcher(r.Context(), info, projectKey, itemNumber, userID); err != nil {
		handleWorkItemError(w, r, err, "failed to remove watcher")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ToggleWatch toggles the current user's watch status on a work item.
func (h *WorkItemHandler) ToggleWatch(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	itemNumber, err := strconv.Atoi(chi.URLParam(r, "itemNumber"))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid item number")
		return
	}

	isWatching, err := h.items.ToggleWatch(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to toggle watch")
		return
	}

	writeData(w, http.StatusOK, toggleWatchResponse{IsWatching: isWatching})
}

// ListWatchedItemIDs returns the IDs of work items the current user is watching.
// When ?project=KEY is provided without ?mode=list, returns just IDs.
// When ?mode=list&project=KEY is provided, returns full work items with filters.
func (h *WorkItemHandler) ListWatchedItemIDs(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	q := r.URL.Query()
	projectKey := q.Get("project")
	mode := q.Get("mode")

	// Full list mode: return work items with standard filters
	// Supports comma-separated project keys or empty for all projects.
	if mode == "list" {
		var projectKeys []string
		if projectKey != "" {
			projectKeys = strings.Split(projectKey, ",")
		}

		filter := &model.WorkItemFilter{
			Search: q.Get("q"),
			Sort:   q.Get("sort"),
			Order:  q.Get("order"),
		}
		if v := q.Get("type"); v != "" {
			filter.Types = strings.Split(v, ",")
		}
		if v := q.Get("status"); v != "" {
			filter.Statuses = strings.Split(v, ",")
		}
		if v := q.Get("priority"); v != "" {
			filter.Priorities = strings.Split(v, ",")
		}
		// Parse assignees
		if assigneeRaw := q.Get("assignees"); assigneeRaw != "" {
			for _, part := range strings.Split(assigneeRaw, ",") {
				part = strings.TrimSpace(part)
				switch part {
				case "me":
					filter.AssigneeMe = true
				case "unassigned":
					filter.Unassigned = true
				default:
					id, err := uuid.Parse(part)
					if err != nil {
						writeError(w, http.StatusBadRequest, CodeValidationError, "invalid assignee parameter")
						return
					}
					filter.AssigneeIDs = append(filter.AssigneeIDs, id)
				}
			}
		}
		// Parse milestones
		if milestoneRaw := q.Get("milestones"); milestoneRaw != "" {
			for _, part := range strings.Split(milestoneRaw, ",") {
				part = strings.TrimSpace(part)
				if part == "none" {
					filter.MilestoneNone = true
					continue
				}
				id, err := uuid.Parse(part)
				if err != nil {
					writeError(w, http.StatusBadRequest, CodeValidationError, "invalid milestone parameter")
					return
				}
				filter.MilestoneIDs = append(filter.MilestoneIDs, id)
			}
		}
		if v := q.Get("cursor"); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, CodeValidationError, "invalid cursor")
				return
			}
			filter.Cursor = &id
		}
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err == nil && n > 0 {
				filter.Limit = n
			}
		}

		result, err := h.items.ListWatchedItems(r.Context(), info, projectKeys, filter)
		if err != nil {
			handleWorkItemError(w, r, err, "failed to list watched items")
			return
		}

		// For cross-project responses, resolve project_key from display_id and compute SLA
		// Collect unique project keys and resolve their namespaces
		pkSet := map[string]struct{}{}
		items := make([]workItemResponse, len(result.Items))
		for i := range result.Items {
			pk := projectKey
			if len(projectKeys) != 1 {
				if idx := strings.LastIndex(result.Items[i].DisplayID, "-"); idx > 0 {
					pk = result.Items[i].DisplayID[:idx]
				}
			}
			pkSet[pk] = struct{}{}
			// Namespace is resolved below (items may span namespaces), so the URL
			// is filled in once the slug is known.
			items[i] = toWorkItemResponse(&result.Items[i], pk, h.baseURL, "")
			if slaMap := h.sla.ComputeSLAForItems(r.Context(), pk, []model.WorkItem{result.Items[i]}); slaMap != nil {
				items[i].SLA = slaMap[result.Items[i].ID]
			}
		}
		// Resolve namespace info for all project keys
		keys := make([]string, 0, len(pkSet))
		for k := range pkSet {
			keys = append(keys, k)
		}
		if nsMap, err := h.items.ResolveProjectNamespaces(r.Context(), keys); err == nil {
			for i := range items {
				if info, ok := nsMap[items[i].ProjectKey]; ok {
					items[i].NamespaceSlug = info.NamespaceSlug
					items[i].NamespaceName = info.NamespaceName
					items[i].URL = weburl.WorkItem(h.baseURL, info.NamespaceSlug, items[i].ProjectKey, items[i].ItemNumber)
				}
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data": items,
			"meta": map[string]interface{}{
				"cursor":   result.Cursor,
				"has_more": result.HasMore,
				"total":    result.Total,
			},
		})
		return
	}

	// Default: return just IDs
	ids, err := h.items.ListWatchedItemIDs(r.Context(), info, projectKey)
	if err != nil {
		handleWorkItemError(w, r, err, "failed to list watched items")
		return
	}

	if ids == nil {
		ids = []uuid.UUID{}
	}

	writeData(w, http.StatusOK, ids)
}

// safeContentDisposition builds a sanitized Content-Disposition header value.
func safeContentDisposition(filename string) string {
	safe := filepath.Base(filename)
	safe = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\\', '\r', '\n':
			return '_'
		}
		return r
	}, safe)
	if safe == "" || safe == "." || safe == ".." {
		safe = "download"
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": safe})
}
