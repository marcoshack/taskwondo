package metrics

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

func TestResourceCollector_Describe(t *testing.T) {
	db, err := sql.Open("postgres", "host=localhost dbname=nonexistent sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}

	collector := NewResourceCollector(db)

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

	expectedNames := map[string]bool{
		"taskwondo_users_total":      false,
		"taskwondo_namespaces_total": false,
		"taskwondo_projects_total":   false,
		"taskwondo_work_items_total": false,
		"taskwondo_milestones_total": false,
	}

	for _, d := range descs {
		desc := d.String()
		for name := range expectedNames {
			if contains(desc, name) {
				expectedNames[name] = true
			}
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected metric %s not found in descriptors", name)
		}
	}
}

func TestResourceCollector_CollectHandlesDBErrors(t *testing.T) {
	// sql.Open with an unreachable host — queries will fail but should not panic
	db, err := sql.Open("postgres", "host=localhost dbname=nonexistent sslmode=disable connect_timeout=1")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}

	collector := NewResourceCollector(db)

	metricCh := make(chan prometheus.Metric, 50)
	// Should not panic — errors are logged and skipped
	collector.Collect(metricCh)
	close(metricCh)

	// With a non-connected DB, we expect zero metrics (all queries fail gracefully)
	var mets []prometheus.Metric
	for m := range metricCh {
		mets = append(mets, m)
	}
	if len(mets) != 0 {
		t.Errorf("expected 0 metrics from disconnected DB, got %d", len(mets))
	}
}
