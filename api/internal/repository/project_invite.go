package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectInviteRepository handles project invite persistence.
type ProjectInviteRepository struct {
	db *sql.DB
}

// NewProjectInviteRepository creates a new ProjectInviteRepository.
func NewProjectInviteRepository(db *sql.DB) *ProjectInviteRepository {
	return &ProjectInviteRepository{db: db}
}

// Create inserts a new project invite.
func (r *ProjectInviteRepository) Create(ctx context.Context, invite *model.ProjectInvite) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO project_invites (id, project_id, code, role, created_by, expires_at, max_uses)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		invite.ID, invite.ProjectID, invite.Code, invite.Role, invite.CreatedBy, invite.ExpiresAt, invite.MaxUses)
	if err != nil {
		return fmt.Errorf("inserting project invite: %w", err)
	}
	return nil
}

// GetByCode returns a project invite by its unique code.
func (r *ProjectInviteRepository) GetByCode(ctx context.Context, code string) (*model.ProjectInvite, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, code, role, created_by, expires_at, max_uses, use_count, created_at
		 FROM project_invites WHERE code = $1`, code)
	return scanProjectInvite(row)
}

// GetByID returns a project invite by ID.
func (r *ProjectInviteRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ProjectInvite, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, code, role, created_by, expires_at, max_uses, use_count, created_at
		 FROM project_invites WHERE id = $1`, id)
	return scanProjectInvite(row)
}

// ListByProject returns all invites for a project, ordered by created_at desc.
func (r *ProjectInviteRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectInvite, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pi.id, pi.project_id, pi.code, pi.role, pi.created_by, u.display_name,
		        pi.expires_at, pi.max_uses, pi.use_count, pi.created_at
		 FROM project_invites pi
		 JOIN users u ON u.id = pi.created_by
		 WHERE pi.project_id = $1 ORDER BY pi.created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying project invites: %w", err)
	}
	defer rows.Close()

	var invites []model.ProjectInvite
	for rows.Next() {
		var inv model.ProjectInvite
		var expiresAt sql.NullTime
		if err := rows.Scan(&inv.ID, &inv.ProjectID, &inv.Code, &inv.Role, &inv.CreatedBy, &inv.CreatedByName,
			&expiresAt, &inv.MaxUses, &inv.UseCount, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning project invite row: %w", err)
		}
		if expiresAt.Valid {
			inv.ExpiresAt = &expiresAt.Time
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// IncrementUseCount atomically increments the use count, respecting max_uses.
// Returns ErrNotFound if the invite doesn't exist or has reached max uses.
func (r *ProjectInviteRepository) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE project_invites SET use_count = use_count + 1
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

// Delete removes a project invite by ID.
func (r *ProjectInviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM project_invites WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting project invite: %w", err)
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
func (r *ProjectInviteRepository) DeleteByProject(ctx context.Context, projectID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM project_invites WHERE project_id = $1`, projectID)
	if err != nil {
		return fmt.Errorf("deleting project invites: %w", err)
	}
	return nil
}

func scanProjectInvite(row *sql.Row) (*model.ProjectInvite, error) {
	var inv model.ProjectInvite
	var expiresAt sql.NullTime
	err := row.Scan(&inv.ID, &inv.ProjectID, &inv.Code, &inv.Role, &inv.CreatedBy,
		&expiresAt, &inv.MaxUses, &inv.UseCount, &inv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning project invite: %w", err)
	}
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
	}
	return &inv, nil
}
