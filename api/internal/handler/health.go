package handler

import (
	"database/sql"
	"net/http"
)

// HealthHandler provides liveness and readiness check endpoints.
type HealthHandler struct {
	db     *sql.DB
	commit string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *sql.DB, commit string) *HealthHandler {
	return &HealthHandler{db: db, commit: commit}
}

// Healthz is a liveness check — returns 200 if the process is alive.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "commit": h.commit})
}

// Readyz is a readiness check — returns 200 only if the database is reachable.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"reason": "database unreachable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
