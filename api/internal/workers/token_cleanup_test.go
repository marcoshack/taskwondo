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
		task := NewTokenCleanupTask(zerolog.Nop(), TokenStore{Name: "email_verification", Repo: repo})

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
		task := NewTokenCleanupTask(zerolog.Nop(), TokenStore{Name: "email_verification", Repo: repo})

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
		task := NewTokenCleanupTask(zerolog.Nop(), TokenStore{Name: "email_verification", Repo: repo})

		err := task.Run(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected %v, got %v", repoErr, err)
		}
	})

	t.Run("cleans every configured store", func(t *testing.T) {
		verification := &mockTokenCleanupRepo{deleted: 3}
		reset := &mockTokenCleanupRepo{deleted: 7}
		task := NewTokenCleanupTask(zerolog.Nop(),
			TokenStore{Name: "email_verification", Repo: verification},
			TokenStore{Name: "password_reset", Repo: reset},
		)

		if err := task.Run(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !verification.called {
			t.Error("expected email verification store to be cleaned")
		}
		if !reset.called {
			t.Error("expected password reset store to be cleaned")
		}
	})

	t.Run("a failing store does not stop the others", func(t *testing.T) {
		repoErr := errors.New("database connection lost")
		failing := &mockTokenCleanupRepo{err: repoErr}
		healthy := &mockTokenCleanupRepo{deleted: 2}
		task := NewTokenCleanupTask(zerolog.Nop(),
			TokenStore{Name: "email_verification", Repo: failing},
			TokenStore{Name: "password_reset", Repo: healthy},
		)

		err := task.Run(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected wrapped %v, got %v", repoErr, err)
		}
		if !healthy.called {
			t.Error("expected the healthy store to be cleaned despite the earlier failure")
		}
	})

	t.Run("no stores configured", func(t *testing.T) {
		task := NewTokenCleanupTask(zerolog.Nop())

		if err := task.Run(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
