package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// OAuthAccountRepository handles OAuth account persistence.
type OAuthAccountRepository struct {
	db *sql.DB
}

// NewOAuthAccountRepository creates a new OAuthAccountRepository.
func NewOAuthAccountRepository(db *sql.DB) *OAuthAccountRepository {
	return &OAuthAccountRepository{db: db}
}

// GetByProviderUser finds an OAuth account by provider and provider user ID.
func (r *OAuthAccountRepository) GetByProviderUser(ctx context.Context, provider, providerUserID string) (*model.OAuthAccount, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, provider, provider_user_id, provider_email,
		        provider_username, provider_avatar, created_at, updated_at
		 FROM user_oauth_accounts
		 WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUserID)

	return scanOAuthAccount(row)
}

// Create inserts a new OAuth account link.
func (r *OAuthAccountRepository) Create(ctx context.Context, account *model.OAuthAccount) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_oauth_accounts
		 (id, user_id, provider, provider_user_id, provider_email, provider_username, provider_avatar)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		account.ID, account.UserID, account.Provider, account.ProviderUserID,
		nullStr(account.ProviderEmail), nullStr(account.ProviderUsername),
		nullStr(account.ProviderAvatar))
	if err != nil {
		return fmt.Errorf("creating oauth account: %w", err)
	}
	return nil
}

// ListByUserID returns all OAuth accounts linked to a user.
func (r *OAuthAccountRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.OAuthAccount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, provider, provider_user_id, provider_email,
		        provider_username, provider_avatar, created_at, updated_at
		 FROM user_oauth_accounts
		 WHERE user_id = $1
		 ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing oauth accounts: %w", err)
	}
	defer rows.Close()

	var accounts []model.OAuthAccount
	for rows.Next() {
		var a model.OAuthAccount
		var pEmail, pUsername, pAvatar sql.NullString
		if err := rows.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderUserID,
			&pEmail, &pUsername, &pAvatar, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning oauth account: %w", err)
		}
		if pEmail.Valid {
			a.ProviderEmail = pEmail.String
		}
		if pUsername.Valid {
			a.ProviderUsername = pUsername.String
		}
		if pAvatar.Valid {
			a.ProviderAvatar = pAvatar.String
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

// Delete removes an OAuth account link.
func (r *OAuthAccountRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM user_oauth_accounts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting oauth account: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func scanOAuthAccount(row *sql.Row) (*model.OAuthAccount, error) {
	var a model.OAuthAccount
	var pEmail, pUsername, pAvatar sql.NullString
	err := row.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderUserID,
		&pEmail, &pUsername, &pAvatar, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning oauth account: %w", err)
	}
	if pEmail.Valid {
		a.ProviderEmail = pEmail.String
	}
	if pUsername.Valid {
		a.ProviderUsername = pUsername.String
	}
	if pAvatar.Valid {
		a.ProviderAvatar = pAvatar.String
	}
	return &a, nil
}
