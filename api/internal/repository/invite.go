package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// InviteRepository handles invite persistence for both project and namespace
// invites. Each invite row carries exactly one of project_id / namespace_id.
type InviteRepository struct {
	db *sql.DB
}

// NewInviteRepository creates a new InviteRepository.
func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{db: db}
}

// Create inserts a new invite. The caller must have set exactly one of
// invite.ProjectID / invite.NamespaceID; the DB CHECK constraint enforces this.
func (r *InviteRepository) Create(ctx context.Context, invite *model.Invite) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO invites (id, project_id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		invite.ID, invite.ProjectID, invite.NamespaceID, invite.Code, invite.Role,
		invite.CreatedBy, invite.InviteeEmail, invite.ExpiresAt, invite.MaxUses)
	if err != nil {
		return fmt.Errorf("inserting invite: %w", err)
	}
	return nil
}

// GetByCode returns an invite by its unique code.
func (r *InviteRepository) GetByCode(ctx context.Context, code string) (*model.Invite, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
		 FROM invites WHERE code = $1`, code)
	return scanInvite(row)
}

// GetByID returns an invite by ID.
func (r *InviteRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Invite, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
		 FROM invites WHERE id = $1`, id)
	return scanInvite(row)
}

// ListByProject returns all invites for a project, ordered by created_at desc.
func (r *InviteRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Invite, error) {
	return r.listWhere(ctx, `i.project_id = $1`, projectID)
}

// ListByNamespace returns all invites for a namespace, ordered by created_at desc.
func (r *InviteRepository) ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]model.Invite, error) {
	return r.listWhere(ctx, `i.namespace_id = $1`, namespaceID)
}

func (r *InviteRepository) listWhere(ctx context.Context, where string, arg any) ([]model.Invite, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT i.id, i.project_id, i.namespace_id, i.code, i.role, i.created_by, u.display_name,
		        i.invitee_email, i.expires_at, i.max_uses, i.use_count, i.created_at
		 FROM invites i
		 JOIN users u ON u.id = i.created_by
		 WHERE `+where+` ORDER BY i.created_at DESC`, arg)
	if err != nil {
		return nil, fmt.Errorf("querying invites: %w", err)
	}
	defer rows.Close()

	var invites []model.Invite
	for rows.Next() {
		var inv model.Invite
		var projectID, namespaceID uuid.NullUUID
		var expiresAt sql.NullTime
		var inviteeEmail sql.NullString
		if err := rows.Scan(&inv.ID, &projectID, &namespaceID, &inv.Code, &inv.Role,
			&inv.CreatedBy, &inv.CreatedByName, &inviteeEmail, &expiresAt,
			&inv.MaxUses, &inv.UseCount, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning invite row: %w", err)
		}
		if projectID.Valid {
			inv.ProjectID = &projectID.UUID
		}
		if namespaceID.Valid {
			inv.NamespaceID = &namespaceID.UUID
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
func (r *InviteRepository) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE invites SET use_count = use_count + 1
		 WHERE id = $1 AND (max_uses = 0 OR use_count < max_uses)`, id)
	if err != nil {
		return fmt.Errorf("incrementing invite use count: %w", err)
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

// Delete removes an invite by ID.
func (r *InviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM invites WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting invite: %w", err)
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

// DeleteByProject removes all invites for a project.
func (r *InviteRepository) DeleteByProject(ctx context.Context, projectID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM invites WHERE project_id = $1`, projectID)
	if err != nil {
		return fmt.Errorf("deleting project invites: %w", err)
	}
	return nil
}

func scanInvite(row *sql.Row) (*model.Invite, error) {
	var inv model.Invite
	var projectID, namespaceID uuid.NullUUID
	var expiresAt sql.NullTime
	var inviteeEmail sql.NullString
	err := row.Scan(&inv.ID, &projectID, &namespaceID, &inv.Code, &inv.Role,
		&inv.CreatedBy, &inviteeEmail, &expiresAt, &inv.MaxUses, &inv.UseCount, &inv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning invite: %w", err)
	}
	if projectID.Valid {
		inv.ProjectID = &projectID.UUID
	}
	if namespaceID.Valid {
		inv.NamespaceID = &namespaceID.UUID
	}
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
	}
	if inviteeEmail.Valid {
		inv.InviteeEmail = &inviteeEmail.String
	}
	return &inv, nil
}
