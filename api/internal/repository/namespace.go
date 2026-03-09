package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NamespaceRepository handles namespace persistence.
type NamespaceRepository struct {
	db *sql.DB
}

// NewNamespaceRepository creates a new NamespaceRepository.
func NewNamespaceRepository(db *sql.DB) *NamespaceRepository {
	return &NamespaceRepository{db: db}
}

// Create inserts a new namespace.
func (r *NamespaceRepository) Create(ctx context.Context, ns *model.Namespace) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO namespaces (id, slug, display_name, icon, color, is_default, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		ns.ID, ns.Slug, ns.DisplayName, ns.Icon, ns.Color, ns.IsDefault, ns.CreatedBy)
	if err != nil {
		return fmt.Errorf("inserting namespace: %w", err)
	}
	return nil
}

// GetByID returns a namespace by its UUID.
func (r *NamespaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Namespace, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, slug, display_name, icon, color, is_default, created_by, created_at, updated_at
		 FROM namespaces WHERE id = $1`, id)
	return scanNamespace(row)
}

// GetBySlug returns a namespace by its slug.
func (r *NamespaceRepository) GetBySlug(ctx context.Context, slug string) (*model.Namespace, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, slug, display_name, icon, color, is_default, created_by, created_at, updated_at
		 FROM namespaces WHERE slug = $1`, slug)
	return scanNamespace(row)
}

// GetDefault returns the default namespace.
func (r *NamespaceRepository) GetDefault(ctx context.Context) (*model.Namespace, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, slug, display_name, icon, color, is_default, created_by, created_at, updated_at
		 FROM namespaces WHERE is_default = true`)
	return scanNamespace(row)
}

// List returns all namespaces ordered by slug.
func (r *NamespaceRepository) List(ctx context.Context) ([]model.Namespace, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, slug, display_name, icon, color, is_default, created_by, created_at, updated_at
		 FROM namespaces ORDER BY is_default DESC, slug`)
	if err != nil {
		return nil, fmt.Errorf("querying namespaces: %w", err)
	}
	defer rows.Close()

	return scanNamespaces(rows)
}

// ListByUser returns namespaces the user belongs to (derived from project membership).
func (r *NamespaceRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Namespace, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT n.id, n.slug, n.display_name, n.icon, n.color, n.is_default, n.created_by, n.created_at, n.updated_at
		 FROM namespaces n
		 JOIN projects p ON p.namespace_id = n.id
		 JOIN project_members pm ON pm.project_id = p.id
		 WHERE pm.user_id = $1 AND p.deleted_at IS NULL
		 ORDER BY n.is_default DESC, n.slug`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying user namespaces: %w", err)
	}
	defer rows.Close()

	return scanNamespaces(rows)
}

// Update modifies a namespace's mutable fields (slug and display_name).
func (r *NamespaceRepository) Update(ctx context.Context, ns *model.Namespace) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE namespaces SET slug = $1, display_name = $2, icon = $3, color = $4, updated_at = now()
		 WHERE id = $5`,
		ns.Slug, ns.DisplayName, ns.Icon, ns.Color, ns.ID)
	if err != nil {
		return fmt.Errorf("updating namespace: %w", err)
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

// Delete removes a namespace (hard delete). Should only be called when namespace has no projects.
func (r *NamespaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM namespaces WHERE id = $1 AND is_default = false`, id)
	if err != nil {
		return fmt.Errorf("deleting namespace: %w", err)
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

// HasProjects checks if a namespace has any non-deleted projects.
func (r *NamespaceRepository) HasProjects(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM projects WHERE namespace_id = $1 AND deleted_at IS NULL`, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("counting namespace projects: %w", err)
	}
	return count > 0, nil
}

// CountNonDefault returns the number of non-default namespaces.
func (r *NamespaceRepository) CountNonDefault(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM namespaces WHERE is_default = false`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting non-default namespaces: %w", err)
	}
	return count, nil
}

func scanNamespace(row *sql.Row) (*model.Namespace, error) {
	var ns model.Namespace
	err := row.Scan(&ns.ID, &ns.Slug, &ns.DisplayName, &ns.Icon, &ns.Color, &ns.IsDefault, &ns.CreatedBy, &ns.CreatedAt, &ns.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning namespace: %w", err)
	}
	return &ns, nil
}

func scanNamespaces(rows *sql.Rows) ([]model.Namespace, error) {
	var namespaces []model.Namespace
	for rows.Next() {
		var ns model.Namespace
		if err := rows.Scan(&ns.ID, &ns.Slug, &ns.DisplayName, &ns.Icon, &ns.Color, &ns.IsDefault, &ns.CreatedBy, &ns.CreatedAt, &ns.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning namespace row: %w", err)
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, rows.Err()
}
