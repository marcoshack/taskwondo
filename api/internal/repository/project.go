package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectRepository handles project persistence.
type ProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository creates a new ProjectRepository.
func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a new project.
func (r *ProjectRepository) Create(ctx context.Context, project *model.Project) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id, name, key, description, default_workflow_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		project.ID, project.Name, project.Key, project.Description, project.DefaultWorkflowID)
	if err != nil {
		return fmt.Errorf("inserting project: %w", err)
	}
	return nil
}

// GetByKey returns a project by its unique key (e.g., "INFRA").
func (r *ProjectRepository) GetByKey(ctx context.Context, key string) (*model.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, key, description, default_workflow_id, item_counter, created_at, updated_at
		 FROM projects WHERE key = $1 AND deleted_at IS NULL`, key)
	return scanProject(row)
}

// GetByID returns a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, key, description, default_workflow_id, item_counter, created_at, updated_at
		 FROM projects WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanProject(row)
}

// ListByUser returns all non-deleted projects the given user is a member of.
func (r *ProjectRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id, p.name, p.key, p.description, p.default_workflow_id, p.item_counter, p.created_at, p.updated_at
		 FROM projects p
		 INNER JOIN project_members pm ON pm.project_id = p.id
		 WHERE pm.user_id = $1 AND p.deleted_at IS NULL
		 ORDER BY p.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

// ListAll returns all non-deleted projects (for global admins).
func (r *ProjectRepository) ListAll(ctx context.Context) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, key, description, default_workflow_id, item_counter, created_at, updated_at
		 FROM projects WHERE deleted_at IS NULL
		 ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying all projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

// Update modifies a project's mutable fields.
func (r *ProjectRepository) Update(ctx context.Context, project *model.Project) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name = $1, key = $2, description = $3, default_workflow_id = $4, updated_at = now()
		 WHERE id = $5 AND deleted_at IS NULL`,
		project.Name, project.Key, project.Description, project.DefaultWorkflowID, project.ID)
	if err != nil {
		return fmt.Errorf("updating project: %w", err)
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

// Delete soft-deletes a project by setting deleted_at.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
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

func scanProject(row *sql.Row) (*model.Project, error) {
	var p model.Project
	var description sql.NullString
	var workflowID *uuid.UUID

	err := row.Scan(&p.ID, &p.Name, &p.Key, &description,
		&workflowID, &p.ItemCounter, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning project: %w", err)
	}

	if description.Valid {
		p.Description = &description.String
	}
	p.DefaultWorkflowID = workflowID

	return &p, nil
}

func scanProjects(rows *sql.Rows) ([]model.Project, error) {
	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var description sql.NullString
		var workflowID *uuid.UUID

		if err := rows.Scan(&p.ID, &p.Name, &p.Key, &description,
			&workflowID, &p.ItemCounter, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning project row: %w", err)
		}

		if description.Valid {
			p.Description = &description.String
		}
		p.DefaultWorkflowID = workflowID

		projects = append(projects, p)
	}

	return projects, rows.Err()
}
