package workers

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
)

type mockSnapshotRepo struct {
	snapshotAllCalled bool
	snapshotAllErr    error
}

func (m *mockSnapshotRepo) SnapshotAll(_ context.Context) error {
	m.snapshotAllCalled = true
	return m.snapshotAllErr
}

func TestStatsSummarize_CallsSnapshotAll(t *testing.T) {
	repo := &mockSnapshotRepo{}
	task := NewStatsSummarizeTask(repo, zerolog.Nop())

	err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.snapshotAllCalled {
		t.Error("SnapshotAll was not called")
	}
}

func TestStatsSummarize_PropagatesError(t *testing.T) {
	repo := &mockSnapshotRepo{snapshotAllErr: fmt.Errorf("db error")}
	task := NewStatsSummarizeTask(repo, zerolog.Nop())

	err := task.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Errorf("error = %q, want %q", err.Error(), "db error")
	}
}
