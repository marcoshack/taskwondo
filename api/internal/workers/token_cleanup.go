package workers

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
)

// tokenCleanupRepository is the minimal interface for the token cleanup task.
type tokenCleanupRepository interface {
	DeleteExpired(ctx context.Context) (int64, error)
}

// TokenStore names a repository of expiring tokens for the cleanup task to purge.
// Name is used for logging only.
type TokenStore struct {
	Name string
	Repo tokenCleanupRepository
}

// TokenCleanupTask periodically removes expired tokens.
//
// Every store it covers holds personal data: both email verification and password
// reset rows carry an email address, including addresses of people who never
// completed the flow and so never became users. Expired rows serve no purpose and
// are purged rather than left to accumulate.
type TokenCleanupTask struct {
	stores []TokenStore
	logger zerolog.Logger
}

// NewTokenCleanupTask creates a cleanup task covering the given token stores.
func NewTokenCleanupTask(logger zerolog.Logger, stores ...TokenStore) *TokenCleanupTask {
	return &TokenCleanupTask{stores: stores, logger: logger}
}

// Run purges expired tokens from every store. A store that fails does not prevent
// the others from being cleaned; all errors are returned together.
func (t *TokenCleanupTask) Run(ctx context.Context) error {
	var errs []error

	for _, store := range t.stores {
		deleted, err := store.Repo.DeleteExpired(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", store.Name, err))
			continue
		}
		if deleted > 0 {
			t.logger.Info().
				Str("store", store.Name).
				Int64("deleted_tokens", deleted).
				Msg("expired tokens cleaned up")
		}
	}

	return errors.Join(errs...)
}
