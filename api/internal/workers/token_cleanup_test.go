package workers

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

type mockTokenCleanupRepo struct {
	deleted int64
	err     error
	called  bool
}

func (m *mockTokenCleanupRepo) DeleteExpired(_ context.Context) (int64, error) {
	m.called = true
	return m.deleted, m.err
}

func TestTokenCleanupTask_Run(t *testing.T) {
	t.Run("deletes expired tokens", func(t *testing.T) {
		repo := &mockTokenCleanupRepo{deleted: 5}
		task := NewTokenCleanupTask(repo, zerolog.Nop())

		err := task.Run(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !repo.called {
			t.Fatal("expected DeleteExpired to be called")
		}
	})

	t.Run("no expired tokens", func(t *testing.T) {
		repo := &mockTokenCleanupRepo{deleted: 0}
		task := NewTokenCleanupTask(repo, zerolog.Nop())

		err := task.Run(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !repo.called {
			t.Fatal("expected DeleteExpired to be called")
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repoErr := errors.New("database connection lost")
		repo := &mockTokenCleanupRepo{err: repoErr}
		task := NewTokenCleanupTask(repo, zerolog.Nop())

		err := task.Run(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected %v, got %v", repoErr, err)
		}
	})
}
