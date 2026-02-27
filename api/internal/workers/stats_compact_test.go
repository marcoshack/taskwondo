package workers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type mockCompactRepo struct {
	compactCalled    bool
	compactThreshold time.Time
	compactDeleted   int64
	compactErr       error
}

func (m *mockCompactRepo) CompactOlderThan(_ context.Context, threshold time.Time) (int64, error) {
	m.compactCalled = true
	m.compactThreshold = threshold
	return m.compactDeleted, m.compactErr
}

func TestStatsCompact_CallsCompactWithCorrectThreshold(t *testing.T) {
	repo := &mockCompactRepo{compactDeleted: 42}
	retention := 7 * 24 * time.Hour
	task := NewStatsCompactTask(repo, retention, zerolog.Nop())

	before := time.Now().Add(-retention)
	err := task.Run(context.Background())
	after := time.Now().Add(-retention)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.compactCalled {
		t.Fatal("CompactOlderThan was not called")
	}

	// Threshold should be approximately now - 7 days
	if repo.compactThreshold.Before(before.Add(-time.Second)) || repo.compactThreshold.After(after.Add(time.Second)) {
		t.Errorf("threshold = %v, want between %v and %v", repo.compactThreshold, before, after)
	}
}

func TestStatsCompact_PropagatesError(t *testing.T) {
	repo := &mockCompactRepo{compactErr: fmt.Errorf("compact error")}
	task := NewStatsCompactTask(repo, 7*24*time.Hour, zerolog.Nop())

	err := task.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "compact error" {
		t.Errorf("error = %q, want %q", err.Error(), "compact error")
	}
}
