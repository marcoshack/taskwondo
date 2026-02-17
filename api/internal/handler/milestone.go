package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/model"
	"github.com/marcoshack/trackforge/internal/service"
)

// MilestoneHandler handles milestone endpoints.
type MilestoneHandler struct {
	milestones *service.MilestoneService
}

// NewMilestoneHandler creates a new MilestoneHandler.
func NewMilestoneHandler(milestones *service.MilestoneService) *MilestoneHandler {
	return &MilestoneHandler{milestones: milestones}
}

// --- Request DTOs ---

type createMilestoneRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
}

type updateMilestoneRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// --- Response DTOs ---

type milestoneResponse struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	DueDate     *string    `json:"due_date,omitempty"`
	Status      string     `json:"status"`
	OpenCount   int        `json:"open_count"`
	ClosedCount int        `json:"closed_count"`
	TotalCount  int        `json:"total_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func toMilestoneResponse(mp *model.MilestoneWithProgress) milestoneResponse {
	resp := milestoneResponse{
		ID:          mp.ID,
		ProjectID:   mp.ProjectID,
		Name:        mp.Name,
		Description: mp.Description,
		Status:      mp.Status,
		OpenCount:   mp.OpenCount,
		ClosedCount: mp.ClosedCount,
		TotalCount:  mp.TotalCount,
		CreatedAt:   mp.CreatedAt,
		UpdatedAt:   mp.UpdatedAt,
	}
	if mp.DueDate != nil {
		d := mp.DueDate.Format(time.DateOnly)
		resp.DueDate = &d
	}
	return resp
}

// --- Handlers ---

// List handles GET /api/v1/projects/{projectKey}/milestones
func (h *MilestoneHandler) List(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	milestones, err := h.milestones.List(r.Context(), info, projectKey)
	if err != nil {
		handleMilestoneError(w, r, err, "failed to list milestones")
		return
	}

	resp := make([]milestoneResponse, len(milestones))
	for i := range milestones {
		resp[i] = toMilestoneResponse(&milestones[i])
	}

	writeData(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/projects/{projectKey}/milestones
func (h *MilestoneHandler) Create(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input := service.CreateMilestoneInput{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.DueDate != nil {
		t, err := time.Parse(time.DateOnly, *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid due_date format, expected YYYY-MM-DD")
			return
		}
		input.DueDate = &t
	}

	mp, err := h.milestones.Create(r.Context(), info, projectKey, input)
	if err != nil {
		handleMilestoneError(w, r, err, "failed to create milestone")
		return
	}

	writeData(w, http.StatusCreated, toMilestoneResponse(mp))
}

// Get handles GET /api/v1/projects/{projectKey}/milestones/{milestoneId}
func (h *MilestoneHandler) Get(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	milestoneID, err := uuid.Parse(chi.URLParam(r, "milestoneId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid milestone ID")
		return
	}

	mp, err := h.milestones.Get(r.Context(), info, projectKey, milestoneID)
	if err != nil {
		handleMilestoneError(w, r, err, "failed to get milestone")
		return
	}

	writeData(w, http.StatusOK, toMilestoneResponse(mp))
}

// Update handles PATCH /api/v1/projects/{projectKey}/milestones/{milestoneId}
func (h *MilestoneHandler) Update(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	milestoneID, err := uuid.Parse(chi.URLParam(r, "milestoneId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid milestone ID")
		return
	}

	// Decode to raw map for explicit null detection
	raw := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	var input service.UpdateMilestoneInput

	if v, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(v, &name); err == nil {
			input.Name = &name
		}
	}

	if v, ok := raw["description"]; ok {
		if string(v) == "null" {
			input.ClearDescription = true
		} else {
			var desc string
			if err := json.Unmarshal(v, &desc); err == nil {
				input.Description = &desc
			}
		}
	}

	if v, ok := raw["due_date"]; ok {
		if string(v) == "null" {
			input.ClearDueDate = true
		} else {
			var dateStr string
			if err := json.Unmarshal(v, &dateStr); err == nil {
				t, err := time.Parse(time.DateOnly, dateStr)
				if err != nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid due_date format, expected YYYY-MM-DD")
					return
				}
				input.DueDate = &t
			}
		}
	}

	if v, ok := raw["status"]; ok {
		var status string
		if err := json.Unmarshal(v, &status); err == nil {
			input.Status = &status
		}
	}

	mp, err := h.milestones.Update(r.Context(), info, projectKey, milestoneID, input)
	if err != nil {
		handleMilestoneError(w, r, err, "failed to update milestone")
		return
	}

	writeData(w, http.StatusOK, toMilestoneResponse(mp))
}

// Delete handles DELETE /api/v1/projects/{projectKey}/milestones/{milestoneId}
func (h *MilestoneHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")
	milestoneID, err := uuid.Parse(chi.URLParam(r, "milestoneId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid milestone ID")
		return
	}

	if err := h.milestones.Delete(r.Context(), info, projectKey, milestoneID); err != nil {
		handleMilestoneError(w, r, err, "failed to delete milestone")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleMilestoneError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "milestone not found")
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

	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
