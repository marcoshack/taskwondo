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
