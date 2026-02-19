package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/marcoshack/taskwondo/internal/model"
)

// APIKeyRepository handles API key persistence.
type APIKeyRepository struct {
	db *sql.DB
}

// NewAPIKeyRepository creates a new APIKeyRepository.
func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create inserts a new API key.
func (r *APIKeyRepository) Create(ctx context.Context, key *model.APIKey) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, permissions, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		key.ID, key.UserID, key.Name, key.KeyHash, key.KeyPrefix,
		pq.Array(key.Permissions), key.ExpiresAt)
	if err != nil {
		return fmt.Errorf("inserting api key: %w", err)
	}
	return nil
}

// GetByKeyHash returns an API key by its SHA-256 hash.
func (r *APIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var k model.APIKey
	var lastUsedAt sql.NullTime
	var expiresAt sql.NullTime
	var permissions pq.StringArray

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_at
		 FROM api_keys WHERE key_hash = $1`, keyHash).
		Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.KeyPrefix,
			&permissions, &lastUsedAt, &expiresAt, &k.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning api key: %w", err)
	}

	k.Permissions = []string(permissions)
	if lastUsedAt.Valid {
		k.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}

	return &k, nil
}

// ListByUserID returns all API keys for a user.
func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.APIKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying api keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		var lastUsedAt sql.NullTime
		var expiresAt sql.NullTime
		var permissions pq.StringArray

		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.KeyPrefix,
			&permissions, &lastUsedAt, &expiresAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning api key row: %w", err)
		}

		k.Permissions = []string(permissions)
		if lastUsedAt.Valid {
			k.LastUsedAt = &lastUsedAt.Time
		}
		if expiresAt.Valid {
			k.ExpiresAt = &expiresAt.Time
		}

		keys = append(keys, k)
	}

	return keys, rows.Err()
}

// Delete removes an API key by ID, scoped to a user.
func (r *APIKeyRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting api key: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}

	return nil
}

// UpdateLastUsed sets the last_used_at timestamp to now.
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("updating last used: %w", err)
	}
	return nil
}
