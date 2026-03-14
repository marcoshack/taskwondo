package metrics

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestDBCollector_DescribeAndCollect(t *testing.T) {
	// sql.Open doesn't actually connect; we just need a *sql.DB whose Stats() returns zeros
	db, err := sql.Open("postgres", "host=localhost dbname=nonexistent sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}

	collector := NewDBCollector(db)

	// Test Describe
	descCh := make(chan *prometheus.Desc, 10)
	collector.Describe(descCh)
	close(descCh)

	var descs []*prometheus.Desc
	for d := range descCh {
		descs = append(descs, d)
	}
	if len(descs) != 5 {
		t.Fatalf("expected 5 descriptors, got %d", len(descs))
	}

	// Test Collect
	metricCh := make(chan prometheus.Metric, 10)
	collector.Collect(metricCh)
	close(metricCh)

	var mets []prometheus.Metric
	for m := range metricCh {
		mets = append(mets, m)
	}
	if len(mets) != 5 {
		t.Fatalf("expected 5 metrics, got %d", len(mets))
	}

	// Verify metric names by writing to dto
	expectedNames := map[string]bool{
		"taskwondo_db_connections_open":                         false,
		"taskwondo_db_connections_idle":                         false,
		"taskwondo_db_connections_in_use":                       false,
		"taskwondo_db_connections_wait_total":                   false,
		"taskwondo_db_connections_wait_duration_seconds_total":  false,
	}

	for _, m := range mets {
		var d dto.Metric
		if err := m.Write(&d); err != nil {
			t.Fatalf("failed to write metric: %v", err)
		}
		desc := m.Desc().String()
		for name := range expectedNames {
			if contains(desc, name) {
				expectedNames[name] = true
			}
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected metric %s not found", name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
