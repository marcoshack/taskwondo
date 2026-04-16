package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NamespaceInviteRepository handles namespace invite persistence.
type NamespaceInviteRepository struct {
	db *sql.DB
}

// NewNamespaceInviteRepository creates a new NamespaceInviteRepository.
func NewNamespaceInviteRepository(db *sql.DB) *NamespaceInviteRepository {
	return &NamespaceInviteRepository{db: db}
}

// Create inserts a new namespace invite.
func (r *NamespaceInviteRepository) Create(ctx context.Context, invite *model.NamespaceInvite) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO namespace_invites (id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		invite.ID, invite.NamespaceID, invite.Code, invite.Role, invite.CreatedBy, invite.InviteeEmail, invite.ExpiresAt, invite.MaxUses)
	if err != nil {
		return fmt.Errorf("inserting namespace invite: %w", err)
	}
	return nil
}

// GetByCode returns a namespace invite by its unique code.
func (r *NamespaceInviteRepository) GetByCode(ctx context.Context, code string) (*model.NamespaceInvite, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
		 FROM namespace_invites WHERE code = $1`, code)
	return scanNamespaceInvite(row)
}

// GetByID returns a namespace invite by ID.
func (r *NamespaceInviteRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.NamespaceInvite, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
		 FROM namespace_invites WHERE id = $1`, id)
	return scanNamespaceInvite(row)
}

// ListByNamespace returns all invites for a namespace, ordered by created_at desc.
func (r *NamespaceInviteRepository) ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]model.NamespaceInvite, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ni.id, ni.namespace_id, ni.code, ni.role, ni.created_by, u.display_name,
		        ni.invitee_email, ni.expires_at, ni.max_uses, ni.use_count, ni.created_at
		 FROM namespace_invites ni
		 JOIN users u ON u.id = ni.created_by
		 WHERE ni.namespace_id = $1 ORDER BY ni.created_at DESC`, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("querying namespace invites: %w", err)
	}
	defer rows.Close()

	var invites []model.NamespaceInvite
	for rows.Next() {
		var inv model.NamespaceInvite
		var expiresAt sql.NullTime
		var inviteeEmail sql.NullString
		if err := rows.Scan(&inv.ID, &inv.NamespaceID, &inv.Code, &inv.Role, &inv.CreatedBy, &inv.CreatedByName,
			&inviteeEmail, &expiresAt, &inv.MaxUses, &inv.UseCount, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning namespace invite row: %w", err)
		}
		if expiresAt.Valid {
			inv.ExpiresAt = &expiresAt.Time
		}
		if inviteeEmail.Valid {
			inv.InviteeEmail = &inviteeEmail.String
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// IncrementUseCount atomically increments the use count, respecting max_uses.
// Returns ErrNotFound if the invite doesn't exist or has reached max uses.
func (r *NamespaceInviteRepository) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE namespace_invites SET use_count = use_count + 1
		 WHERE id = $1 AND (max_uses = 0 OR use_count < max_uses)`, id)
	if err != nil {
		return fmt.Errorf("incrementing namespace invite use count: %w", err)
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

// Delete removes a namespace invite by ID.
func (r *NamespaceInviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM namespace_invites WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting namespace invite: %w", err)
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

func scanNamespaceInvite(row *sql.Row) (*model.NamespaceInvite, error) {
	var inv model.NamespaceInvite
	var expiresAt sql.NullTime
	var inviteeEmail sql.NullString
	err := row.Scan(&inv.ID, &inv.NamespaceID, &inv.Code, &inv.Role, &inv.CreatedBy,
		&inviteeEmail, &expiresAt, &inv.MaxUses, &inv.UseCount, &inv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning namespace invite: %w", err)
	}
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
	}
	if inviteeEmail.Valid {
		inv.InviteeEmail = &inviteeEmail.String
	}
	return &inv, nil
}
