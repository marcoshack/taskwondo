package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcoshack/taskwondo/internal/metrics"
)

func TestMetricsHandler_ServesPrometheusFormat(t *testing.T) {
	h := NewMetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should contain Go runtime metrics
	if !strings.Contains(body, "go_goroutines") {
		t.Error("expected go_goroutines in metrics output")
	}

	// Should contain process metrics
	if !strings.Contains(body, "process_") {
		t.Error("expected process_* metrics in output")
	}

	// Content-Type should be text-based (Prometheus exposition format)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") && !strings.Contains(ct, "application/openmetrics") {
		t.Errorf("expected text content type, got %s", ct)
	}
}

func TestMetricsHandler_ContainsHTTPMetrics(t *testing.T) {
	// Initialize at least one label set so the metrics appear in output
	metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	metrics.HTTPRequestDuration.WithLabelValues("GET", "/test").Observe(0.01)

	h := NewMetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "taskwondo_http_requests_total") {
		t.Error("expected taskwondo_http_requests_total in metrics output")
	}
	if !strings.Contains(body, "taskwondo_http_request_duration_seconds") {
		t.Error("expected taskwondo_http_request_duration_seconds in metrics output")
	}
}
