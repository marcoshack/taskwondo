package handler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/marcoshack/taskwondo/internal/metrics"
)

// MetricsHandler serves Prometheus metrics.
type MetricsHandler struct {
	handler http.Handler
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		handler: promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}),
	}
}

// ServeHTTP serves the Prometheus metrics endpoint.
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}
