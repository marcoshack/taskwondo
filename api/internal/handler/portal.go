package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// PortalHandler handles customer-facing portal endpoints.
type PortalHandler struct {
	workItems     *service.WorkItemService
	queues        *service.QueueService
	auth          *service.AuthService
	maxUploadSize int64
}

// NewPortalHandler creates a new PortalHandler.
func NewPortalHandler(workItems *service.WorkItemService, queues *service.QueueService, auth *service.AuthService, maxUploadSize int64) *PortalHandler {
	return &PortalHandler{
		workItems:     workItems,
		queues:        queues,
		auth:          auth,
		maxUploadSize: maxUploadSize,
	}
}

// --- Request DTOs ---

type createPortalTicketRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Priority    string  `json:"priority,omitempty"`
	CategoryID  *string `json:"category_id,omitempty"`
}

type createPortalCommentRequest struct {
	Body string `json:"body"`
}

// --- Response DTOs ---

type portalQueueResponse struct {
	ID          uuid.UUID                `json:"id"`
	Name        string                   `json:"name"`
	Description *string                  `json:"description,omitempty"`
	QueueType   string                   `json:"queue_type"`
	Categories  []portalCategoryResponse `json:"categories"`
}

type portalCategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
}

type portalTicketResponse struct {
	ID          uuid.UUID  `json:"id"`
	ItemNumber  int        `json:"item_number"`
	DisplayID   string     `json:"display_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	QueueID     *uuid.UUID `json:"queue_id,omitempty"`
	Visibility  string     `json:"visibility"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func toPortalTicketResponse(item *model.WorkItem) portalTicketResponse {
	return portalTicketResponse{
		ID:          item.ID,
		ItemNumber:  item.ItemNumber,
		DisplayID:   item.DisplayID,
		Title:       item.Title,
		Description: item.Description,
		Status:      item.Status,
		Priority:    item.Priority,
		QueueID:     item.QueueID,
		Visibility:  item.Visibility,
		ResolvedAt:  item.ResolvedAt,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

type portalCommentResponse struct {
	ID         uuid.UUID  `json:"id"`
	AuthorID   *uuid.UUID `json:"author_id,omitempty"`
	AuthorName string     `json:"author_name"`
	Body       string     `json:"body"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func toPortalCommentResponse(c *model.Comment, authorName string) portalCommentResponse {
	return portalCommentResponse{
		ID:         c.ID,
		AuthorID:   c.AuthorID,
		AuthorName: authorName,
		Body:       c.Body,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

type portalEventResponse struct {
	ID               uuid.UUID              `json:"id"`
	EventType        string                 `json:"event_type"`
	FieldName        *string                `json:"field_name,omitempty"`
	OldValue         *string                `json:"old_value,omitempty"`
	NewValue         *string                `json:"new_value,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
	ActorDisplayName *string                `json:"actor_display_name,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

func toPortalEventResponse(e *model.WorkItemEventWithActor) portalEventResponse {
	return portalEventResponse{
		ID:               e.ID,
		EventType:        e.EventType,
		FieldName:        e.FieldName,
		OldValue:         e.OldValue,
		NewValue:         e.NewValue,
		Metadata:         e.Metadata,
		ActorDisplayName: e.ActorDisplayName,
		CreatedAt:        e.CreatedAt,
	}
}

// --- Handlers ---

// ListQueues handles GET /api/v1/portal/{namespace}/projects/{projectKey}/queues
func (h *PortalHandler) ListQueues(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	queues, err := h.queues.List(r.Context(), info, projectKey)
	if err != nil {
		handlePortalError(w, r, err, "failed to list portal queues")
		return
	}

	// Filter to only public queues
	var resp []portalQueueResponse
	for i := range queues {
		if !queues[i].IsPublic {
			continue
		}

		qResp := portalQueueResponse{
			ID:          queues[i].ID,
			Name:        queues[i].Name,
			Description: queues[i].Description,
			QueueType:   queues[i].QueueType,
			Categories:  []portalCategoryResponse{},
		}

		// Fetch categories for this queue
		categories, err := h.queues.ListCategories(r.Context(), info, projectKey, queues[i].ID)
		if err != nil {
			log.Ctx(r.Context()).Warn().Err(err).
				Str("queue_id", queues[i].ID.String()).
				Msg("failed to list categories for portal queue")
		} else {
			for j := range categories {
				qResp.Categories = append(qResp.Categories, portalCategoryResponse{
					ID:          categories[j].ID,
					Name:        categories[j].Name,
					Description: categories[j].Description,
				})
			}
		}

		resp = append(resp, qResp)
	}

	writeData(w, http.StatusOK, resp)
}

// CreateTicket handles POST /api/v1/portal/{namespace}/projects/{projectKey}/tickets
func (h *PortalHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createPortalTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	// Auto-resolve the project's public queue
	publicQueue, err := h.queues.GetPublicQueue(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "project has no public queue configured")
		return
	}

	input := service.CreatePortalTicketInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		QueueID:     publicQueue.ID,
	}

	if req.CategoryID != nil {
		id, ok := parseOptionalUUID(w, req.CategoryID, "invalid category_id")
		if !ok {
			return
		}
		input.CategoryID = &id
	}

	item, err := h.workItems.CreatePortalTicket(r.Context(), info, projectKey, input)
	if err != nil {
		handlePortalError(w, r, err, "failed to create portal ticket")
		return
	}

	writeData(w, http.StatusCreated, toPortalTicketResponse(item))
}

// ListTickets handles GET /api/v1/portal/{namespace}/projects/{projectKey}/tickets
func (h *PortalHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	filter := &model.WorkItemFilter{
		ReporterID: &info.UserID,
		Types:      []string{model.WorkItemTypeTicket},
	}

	// Parse optional query params
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Statuses = []string{status}
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		id, err := uuid.Parse(cursor)
		if err == nil {
			filter.Cursor = &id
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if r.URL.Query().Get("hide_completed") == "true" {
		filter.ExcludeResolved = true
	}

	list, err := h.workItems.List(r.Context(), info, projectKey, filter)
	if err != nil {
		handlePortalError(w, r, err, "failed to list portal tickets")
		return
	}

	items := make([]portalTicketResponse, len(list.Items))
	for i := range list.Items {
		items[i] = toPortalTicketResponse(&list.Items[i])
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":     items,
		"cursor":   list.Cursor,
		"has_more": list.HasMore,
		"total":    list.Total,
	})
}

// GetTicket handles GET /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}
func (h *PortalHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
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

	item, err := h.workItems.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handlePortalError(w, r, err, "failed to get portal ticket")
		return
	}

	// Customer can only see their own tickets — return 404 to avoid information leakage
	if item.ReporterID != info.UserID {
		writeError(w, http.StatusNotFound, CodeNotFound, "ticket not found")
		return
	}

	writeData(w, http.StatusOK, toPortalTicketResponse(item))
}

type updatePortalTicketRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description"`
}

// UpdateTicket handles PATCH /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}
// Customers can only update the title and description of their own tickets.
func (h *PortalHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
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

	// Verify ownership
	item, err := h.workItems.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handlePortalError(w, r, err, "failed to get portal ticket")
		return
	}
	if item.ReporterID != info.UserID {
		writeError(w, http.StatusNotFound, CodeNotFound, "ticket not found")
		return
	}

	var req updatePortalTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	updated, err := h.workItems.UpdatePortalTicket(r.Context(), info, projectKey, itemNumber, service.UpdatePortalTicketInput{
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		handlePortalError(w, r, err, "failed to update portal ticket")
		return
	}

	writeData(w, http.StatusOK, toPortalTicketResponse(updated))
}

// ListComments handles GET /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/comments
func (h *PortalHandler) ListComments(w http.ResponseWriter, r *http.Request) {
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

	// Verify the ticket belongs to the customer
	item, err := h.workItems.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handlePortalError(w, r, err, "failed to get ticket for comments")
		return
	}
	if item.ReporterID != info.UserID {
		writeError(w, http.StatusNotFound, CodeNotFound, "ticket not found")
		return
	}

	comments, err := h.workItems.ListComments(r.Context(), info, projectKey, itemNumber, model.VisibilityPublic)
	if err != nil {
		handlePortalError(w, r, err, "failed to list portal comments")
		return
	}

	resp := make([]portalCommentResponse, len(comments))
	for i := range comments {
		authorName := h.resolveAuthorName(r.Context(), comments[i].AuthorID)
		resp[i] = toPortalCommentResponse(&comments[i], authorName)
	}

	writeData(w, http.StatusOK, resp)
}

// AddComment handles POST /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/comments
func (h *PortalHandler) AddComment(w http.ResponseWriter, r *http.Request) {
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

	// Verify the ticket belongs to the customer
	item, err := h.workItems.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handlePortalError(w, r, err, "failed to get ticket for comment")
		return
	}
	if item.ReporterID != info.UserID {
		writeError(w, http.StatusNotFound, CodeNotFound, "ticket not found")
		return
	}

	var req createPortalCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	comment, err := h.workItems.CreatePortalComment(r.Context(), info, projectKey, itemNumber, req.Body)
	if err != nil {
		handlePortalError(w, r, err, "failed to add portal comment")
		return
	}

	authorName := h.resolveAuthorName(r.Context(), comment.AuthorID)
	writeData(w, http.StatusCreated, toPortalCommentResponse(comment, authorName))
}

// resolveAuthorName looks up the display name for a user ID.
func (h *PortalHandler) resolveAuthorName(ctx context.Context, authorID *uuid.UUID) string {
	if authorID == nil || h.auth == nil {
		return ""
	}
	user, err := h.auth.GetUser(ctx, *authorID)
	if err != nil {
		return ""
	}
	return user.DisplayName
}

// ListEvents handles GET /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/events
func (h *PortalHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
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

	// Verify the ticket belongs to the customer
	item, err := h.workItems.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		handlePortalError(w, r, err, "failed to get ticket for events")
		return
	}
	if item.ReporterID != info.UserID {
		writeError(w, http.StatusNotFound, CodeNotFound, "ticket not found")
		return
	}

	events, err := h.workItems.ListEvents(r.Context(), info, projectKey, itemNumber, model.VisibilityPublic)
	if err != nil {
		handlePortalError(w, r, err, "failed to list portal events")
		return
	}

	resp := make([]portalEventResponse, len(events))
	for i := range events {
		resp[i] = toPortalEventResponse(&events[i])
	}

	writeData(w, http.StatusOK, resp)
}

// --- Error handling ---

func handlePortalError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, CodeForbidden, "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeErrorFromService(w, http.StatusBadRequest, CodeValidationError, err)
		return
	}
	if errors.Is(err, model.ErrAlreadyExists) || errors.Is(err, model.ErrConflict) {
		writeErrorFromService(w, http.StatusConflict, CodeConflict, err)
		return
	}

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}

// --- Portal Attachment Handlers ---

// portalElevatedInfo returns an AuthInfo with admin role to bypass requireRole checks
// in the work item service. The caller must have already verified ticket ownership.
func portalElevatedInfo(info *model.AuthInfo) *model.AuthInfo {
	return &model.AuthInfo{
		UserID:     info.UserID,
		GlobalRole: model.RoleAdmin,
	}
}

// verifyTicketOwnership checks the ticket belongs to the authenticated customer.
func (h *PortalHandler) verifyTicketOwnership(r *http.Request, info *model.AuthInfo, projectKey string, itemNumber int) (*model.WorkItem, error) {
	item, err := h.workItems.Get(r.Context(), info, projectKey, itemNumber)
	if err != nil {
		return nil, err
	}
	if item.ReporterID != info.UserID {
		return nil, model.ErrNotFound
	}
	return item, nil
}

// ListAttachments handles GET /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/attachments
func (h *PortalHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.verifyTicketOwnership(r, info, projectKey, itemNumber); err != nil {
		handlePortalError(w, r, err, "failed to verify ticket ownership")
		return
	}

	attachments, err := h.workItems.ListAttachments(r.Context(), portalElevatedInfo(info), projectKey, itemNumber)
	if err != nil {
		handlePortalError(w, r, err, "failed to list attachments")
		return
	}

	ns := chi.URLParam(r, "namespace")
	resp := make([]portalAttachmentResponse, len(attachments))
	for i := range attachments {
		resp[i] = toPortalAttachmentResponse(&attachments[i], ns, projectKey, itemNumber)
	}

	writeData(w, http.StatusOK, resp)
}

// UploadAttachment handles POST /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/attachments
func (h *PortalHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.verifyTicketOwnership(r, info, projectKey, itemNumber); err != nil {
		handlePortalError(w, r, err, "failed to verify ticket ownership")
		return
	}

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

	attachment, err := h.workItems.UploadAttachment(r.Context(), portalElevatedInfo(info), projectKey, itemNumber, service.CreateAttachmentInput{
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
		Comment:     comment,
		Reader:      file,
	})
	if err != nil {
		handlePortalError(w, r, err, "failed to upload attachment")
		return
	}

	writeData(w, http.StatusCreated, toPortalAttachmentResponse(attachment, chi.URLParam(r, "namespace"), projectKey, itemNumber))
}

// DownloadAttachment handles GET /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/attachments/{attachmentId}
func (h *PortalHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.verifyTicketOwnership(r, info, projectKey, itemNumber); err != nil {
		handlePortalError(w, r, err, "failed to verify ticket ownership")
		return
	}

	attachment, reader, err := h.workItems.GetAttachmentFile(r.Context(), portalElevatedInfo(info), projectKey, itemNumber, attachmentID)
	if err != nil {
		handlePortalError(w, r, err, "failed to download attachment")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", safeDownloadContentType(attachment.ContentType))
	w.Header().Set("Content-Disposition", safeContentDisposition(attachment.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(attachment.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

// DeleteAttachment handles DELETE /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/attachments/{attachmentId}
func (h *PortalHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.verifyTicketOwnership(r, info, projectKey, itemNumber); err != nil {
		handlePortalError(w, r, err, "failed to verify ticket ownership")
		return
	}

	if err := h.workItems.DeleteAttachment(r.Context(), portalElevatedInfo(info), projectKey, itemNumber, attachmentID); err != nil {
		handlePortalError(w, r, err, "failed to delete attachment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateAttachmentComment handles PATCH /api/v1/portal/{namespace}/projects/{projectKey}/tickets/{itemNumber}/attachments/{attachmentId}
func (h *PortalHandler) UpdateAttachmentComment(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.verifyTicketOwnership(r, info, projectKey, itemNumber); err != nil {
		handlePortalError(w, r, err, "failed to verify ticket ownership")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	attachment, err := h.workItems.UpdateAttachmentComment(r.Context(), portalElevatedInfo(info), projectKey, itemNumber, attachmentID, req.Comment)
	if err != nil {
		handlePortalError(w, r, err, "failed to update attachment comment")
		return
	}

	writeData(w, http.StatusOK, toPortalAttachmentResponse(attachment, chi.URLParam(r, "namespace"), projectKey, itemNumber))
}

type portalAttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	UploaderID  uuid.UUID `json:"uploader_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Comment     string    `json:"comment"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func toPortalAttachmentResponse(a *model.Attachment, namespace, projectKey string, itemNumber int) portalAttachmentResponse {
	if namespace == "" {
		namespace = "default"
	}
	return portalAttachmentResponse{
		ID:          a.ID,
		UploaderID:  a.UploaderID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		Comment:     a.Comment,
		DownloadURL: fmt.Sprintf("/api/v1/portal/%s/projects/%s/tickets/%d/attachments/%s", namespace, projectKey, itemNumber, a.ID),
		CreatedAt:   a.CreatedAt,
	}
}
