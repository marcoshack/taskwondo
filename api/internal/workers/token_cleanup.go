package workers

import (
	"context"

	"github.com/rs/zerolog"
)

// tokenCleanupRepository is the minimal interface for the token cleanup task.
type tokenCleanupRepository interface {
	DeleteExpired(ctx context.Context) (int64, error)
}

// TokenCleanupTask periodically removes expired email verification tokens.
type TokenCleanupTask struct {
	repo   tokenCleanupRepository
	logger zerolog.Logger
}

// NewTokenCleanupTask creates a new token cleanup task.
func NewTokenCleanupTask(repo tokenCleanupRepository, logger zerolog.Logger) *TokenCleanupTask {
	return &TokenCleanupTask{repo: repo, logger: logger}
}

// Run executes the expired token cleanup.
func (t *TokenCleanupTask) Run(ctx context.Context) error {
	deleted, err := t.repo.DeleteExpired(ctx)
	if err != nil {
		return err
	}
	if deleted > 0 {
		t.logger.Info().Int64("deleted_tokens", deleted).Msg("expired email verification tokens cleaned up")
	}
	return nil
}
