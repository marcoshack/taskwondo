package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// StatsHandler handles project stats endpoints.
type StatsHandler struct {
	stats *service.StatsService
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(stats *service.StatsService) *StatsHandler {
	return &StatsHandler{stats: stats}
}

// --- Response DTOs ---

type statsTimelinePoint struct {
	CapturedAt      string `json:"captured_at"`
	TodoCount       int    `json:"todo_count"`
	InProgressCount int    `json:"in_progress_count"`
	DoneCount       int    `json:"done_count"`
	CancelledCount  int    `json:"cancelled_count"`
}

// Timeline returns time-bucketed stats for a project.
// GET /api/v1/projects/{projectKey}/stats/timeline?range=24h
func (h *StatsHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	projectKey := chi.URLParam(r, "projectKey")

	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "7d"
	}

	snapshots, err := h.stats.Timeline(r.Context(), info, projectKey, rangeStr)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "project not found")
		case errors.Is(err, model.ErrValidation):
			writeErrorFromService(w, http.StatusBadRequest, "VALIDATION_ERROR", err)
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch stats")
		}
		return
	}

	points := make([]statsTimelinePoint, len(snapshots))
	for i, s := range snapshots {
		points[i] = statsTimelinePoint{
			CapturedAt:      s.CapturedAt.Format(time.RFC3339),
			TodoCount:       s.TodoCount,
			InProgressCount: s.InProgressCount,
			DoneCount:       s.DoneCount,
			CancelledCount:  s.CancelledCount,
		}
	}

	writeData(w, http.StatusOK, points)
}
