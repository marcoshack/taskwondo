package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestRegistryContainsGoAndProcessCollectors(t *testing.T) {
	families, err := Registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	nameSet := make(map[string]bool)
	for _, mf := range families {
		nameSet[mf.GetName()] = true
	}

	// Go runtime collector should provide go_goroutines
	if !nameSet["go_goroutines"] {
		t.Error("expected go_goroutines metric from Go collector")
	}

	// Process collector should provide process_open_fds (on Linux)
	if !nameSet["process_open_fds"] {
		t.Error("expected process_open_fds metric from process collector")
	}
}

func TestHTTPRequestsTotal_Registered(t *testing.T) {
	// Verify the counter can be used
	HTTPRequestsTotal.WithLabelValues("GET", "/healthz", "200").Inc()

	families, err := Registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	var found *dto.MetricFamily
	for _, mf := range families {
		if mf.GetName() == "taskwondo_http_requests_total" {
			found = mf
			break
		}
	}

	if found == nil {
		t.Fatal("expected taskwondo_http_requests_total metric")
	}
}

func TestHTTPRequestDuration_Registered(t *testing.T) {
	HTTPRequestDuration.WithLabelValues("GET", "/healthz").Observe(0.05)

	families, err := Registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	var found *dto.MetricFamily
	for _, mf := range families {
		if mf.GetName() == "taskwondo_http_request_duration_seconds" {
			found = mf
			break
		}
	}

	if found == nil {
		t.Fatal("expected taskwondo_http_request_duration_seconds metric")
	}
}

func TestRegisterCollector_PanicsOnDuplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when registering duplicate collector")
		}
	}()

	dup := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "duplicate",
	})
	RegisterCollector(dup)
}
