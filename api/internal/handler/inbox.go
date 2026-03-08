package handler

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// InboxHandler handles inbox endpoints.
type InboxHandler struct {
	inbox *service.InboxService
	sla   *service.SLAService
}

// NewInboxHandler creates a new InboxHandler.
func NewInboxHandler(inbox *service.InboxService, sla *service.SLAService) *InboxHandler {
	return &InboxHandler{inbox: inbox, sla: sla}
}

// --- Request DTOs ---

type addInboxItemRequest struct {
	WorkItemID string `json:"work_item_id"`
}

type reorderInboxItemRequest struct {
	Position int `json:"position"`
}

// --- Response DTOs ---

type inboxItemResponse struct {
	ID             uuid.UUID  `json:"id"`
	WorkItemID     uuid.UUID  `json:"work_item_id"`
	Position       int        `json:"position"`
	CreatedAt      time.Time  `json:"created_at"`
	DisplayID      string     `json:"display_id"`
	Title          string     `json:"title"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	StatusCategory string     `json:"status_category"`
	Priority       string     `json:"priority"`
	ProjectKey     string     `json:"project_key"`
	ProjectName    string     `json:"project_name"`
	NamespaceSlug  string     `json:"namespace_slug"`
	NamespaceName  string     `json:"namespace_name"`
	AssigneeID          *uuid.UUID     `json:"assignee_id,omitempty"`
	AssigneeDisplayName string         `json:"assignee_display_name,omitempty"`
	Description         string         `json:"description,omitempty"`
	DueDate             *time.Time     `json:"due_date,omitempty"`
	SLA                 *model.SLAInfo `json:"sla,omitempty"`
	SLATargetAt         *time.Time     `json:"sla_target_at,omitempty"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type inboxListResponse struct {
	Items   []inboxItemResponse `json:"items"`
	Cursor  string              `json:"cursor"`
	HasMore bool                `json:"has_more"`
	Total   int                 `json:"total"`
}

type inboxCountResponse struct {
	Count int `json:"count"`
}

type clearCompletedResponse struct {
	Removed int `json:"removed"`
}

func toInboxItemResponse(item model.InboxItemWithWorkItem) inboxItemResponse {
	return inboxItemResponse{
		ID:             item.ID,
		WorkItemID:     item.WorkItemID,
		Position:       item.Position,
		CreatedAt:      item.CreatedAt,
		DisplayID:      item.DisplayID,
		Title:          item.Title,
		Type:           item.Type,
		Status:         item.Status,
		StatusCategory: item.StatusCategory,
		Priority:       item.Priority,
		ProjectKey:     item.ProjectKey,
		ProjectName:    item.ProjectName,
		NamespaceSlug:  item.NamespaceSlug,
		NamespaceName:  item.NamespaceName,
		AssigneeID:          item.AssigneeID,
		AssigneeDisplayName: item.AssigneeDisplayName,
		Description:         item.Description,
		DueDate:             item.DueDate,
		SLATargetAt:         item.SLATargetAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func toInboxListResponse(list *model.InboxItemList) inboxListResponse {
	items := make([]inboxItemResponse, len(list.Items))
	for i, item := range list.Items {
		items[i] = toInboxItemResponse(item)
	}
	return inboxListResponse{
		Items:   items,
		Cursor:  list.Cursor,
		HasMore: list.HasMore,
		Total:   list.Total,
	}
}

// List handles GET /api/v1/user/inbox
func (h *InboxHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	search := r.URL.Query().Get("search")
	includeCompleted := r.URL.Query().Get("include_completed") == "true"

	var projectKeys []string
	if p := r.URL.Query().Get("project"); p != "" {
		for _, k := range strings.Split(p, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				projectKeys = append(projectKeys, k)
			}
		}
	}

	var cursor *uuid.UUID
	if c := r.URL.Query().Get("cursor"); c != "" {
		parsed, err := uuid.Parse(c)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid cursor")
			return
		}
		cursor = &parsed
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid limit")
			return
		}
		limit = parsed
	}

	list, err := h.inbox.List(r.Context(), info, !includeCompleted, search, projectKeys, cursor, limit)
	if err != nil {
		handleInboxError(w, r, err, "listing inbox items")
		return
	}

	resp := toInboxListResponse(list)

	// Compute SLA for inbox items, grouped by project
	if h.sla != nil && len(list.Items) > 0 {
		byProject := map[string][]model.WorkItem{}
		for _, item := range list.Items {
			byProject[item.ProjectKey] = append(byProject[item.ProjectKey], model.WorkItem{
				ID:     item.WorkItemID,
				Type:   item.Type,
				Status: item.Status,
			})
		}
		slaMap := map[uuid.UUID]*model.SLAInfo{}
		for projectKey, items := range byProject {
			if m := h.sla.ComputeSLAForItems(r.Context(), projectKey, items); m != nil {
				maps.Copy(slaMap, m)
			}
		}
		for i, item := range list.Items {
			if slaInfo, ok := slaMap[item.WorkItemID]; ok {
				resp.Items[i].SLA = slaInfo
			}
		}
	}

	writeData(w, http.StatusOK, resp)
}

// Add handles POST /api/v1/user/inbox
func (h *InboxHandler) Add(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	var req addInboxItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	workItemID, err := uuid.Parse(req.WorkItemID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid work_item_id")
		return
	}

	if err := h.inbox.Add(r.Context(), info, workItemID); err != nil {
		handleInboxError(w, r, err, "adding inbox item")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Remove handles DELETE /api/v1/user/inbox/{inboxItemId}
func (h *InboxHandler) Remove(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	inboxItemID, err := uuid.Parse(chi.URLParam(r, "inboxItemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid inbox item ID")
		return
	}

	if err := h.inbox.Remove(r.Context(), info, inboxItemID); err != nil {
		handleInboxError(w, r, err, "removing inbox item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Reorder handles PATCH /api/v1/user/inbox/{inboxItemId}
func (h *InboxHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	inboxItemID, err := uuid.Parse(chi.URLParam(r, "inboxItemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid inbox item ID")
		return
	}

	var req reorderInboxItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := h.inbox.Reorder(r.Context(), info, inboxItemID, req.Position); err != nil {
		handleInboxError(w, r, err, "reordering inbox item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ClearCompleted handles DELETE /api/v1/user/inbox/completed
func (h *InboxHandler) ClearCompleted(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	count, err := h.inbox.ClearCompleted(r.Context(), info)
	if err != nil {
		handleInboxError(w, r, err, "clearing completed inbox items")
		return
	}

	writeData(w, http.StatusOK, clearCompletedResponse{Removed: count})
}

// Count handles GET /api/v1/user/inbox/count
func (h *InboxHandler) Count(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())

	count, err := h.inbox.Count(r.Context(), info)
	if err != nil {
		handleInboxError(w, r, err, "counting inbox items")
		return
	}

	writeData(w, http.StatusOK, inboxCountResponse{Count: count})
}

func handleInboxError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "item not found")
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
	if errors.Is(err, model.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
