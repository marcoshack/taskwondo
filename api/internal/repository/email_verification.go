package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/marcoshack/taskwondo/internal/model"
)

// EmailVerificationRepository handles email verification token persistence.
type EmailVerificationRepository struct {
	db *sql.DB
}

// NewEmailVerificationRepository creates a new EmailVerificationRepository.
func NewEmailVerificationRepository(db *sql.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

// Create inserts a new email verification token.
func (r *EmailVerificationRepository) Create(ctx context.Context, token *model.EmailVerificationToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO email_verification_tokens (id, email, display_name, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		token.ID, token.Email, token.DisplayName, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("inserting email verification token: %w", err)
	}
	return nil
}

// GetByTokenHash returns a non-expired token by its hash.
func (r *EmailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.EmailVerificationToken, error) {
	var token model.EmailVerificationToken
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, token_hash, expires_at, created_at
		 FROM email_verification_tokens
		 WHERE token_hash = $1 AND expires_at > now()`, tokenHash).Scan(
		&token.ID, &token.Email, &token.DisplayName, &token.TokenHash,
		&token.ExpiresAt, &token.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying email verification token: %w", err)
	}
	return &token, nil
}

// DeleteByTokenHash deletes a token by its hash.
func (r *EmailVerificationRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_verification_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("deleting email verification token: %w", err)
	}
	return nil
}

// DeleteByEmail deletes all tokens for a given email address.
func (r *EmailVerificationRepository) DeleteByEmail(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_verification_tokens WHERE email = $1`, email)
	if err != nil {
		return fmt.Errorf("deleting email verification tokens by email: %w", err)
	}
	return nil
}
