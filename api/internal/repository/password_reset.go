package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/marcoshack/taskwondo/internal/model"
)

// PasswordResetRepository handles password reset token persistence.
type PasswordResetRepository struct {
	db *sql.DB
}

// NewPasswordResetRepository creates a new PasswordResetRepository.
func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// Create inserts a new password reset token.
func (r *PasswordResetRepository) Create(ctx context.Context, token *model.PasswordResetToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (id, email, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		token.ID, token.Email, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("inserting password reset token: %w", err)
	}
	return nil
}

// GetByTokenHash returns a non-expired token by its hash.
func (r *PasswordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error) {
	var token model.PasswordResetToken
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, token_hash, expires_at, created_at
		 FROM password_reset_tokens
		 WHERE token_hash = $1 AND expires_at > now()`, tokenHash).Scan(
		&token.ID, &token.Email, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying password reset token: %w", err)
	}
	return &token, nil
}

// DeleteByTokenHash deletes a token by its hash.
func (r *PasswordResetRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("deleting password reset token: %w", err)
	}
	return nil
}

// DeleteByEmail deletes all tokens for a given email address.
func (r *PasswordResetRepository) DeleteByEmail(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE email = $1`, email)
	if err != nil {
		return fmt.Errorf("deleting password reset tokens by email: %w", err)
	}
	return nil
}

// DeleteExpired removes all tokens that have passed their expiration time.
func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE expires_at < now()`)
	if err != nil {
		return 0, fmt.Errorf("deleting expired password reset tokens: %w", err)
	}
	return result.RowsAffected()
}
