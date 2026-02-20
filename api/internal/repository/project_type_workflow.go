package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectTypeWorkflowRepository handles project type-workflow mapping persistence.
type ProjectTypeWorkflowRepository struct {
	db *sql.DB
}

// NewProjectTypeWorkflowRepository creates a new ProjectTypeWorkflowRepository.
func NewProjectTypeWorkflowRepository(db *sql.DB) *ProjectTypeWorkflowRepository {
	return &ProjectTypeWorkflowRepository{db: db}
}

// ListByProject returns all type-workflow mappings for a project.
func (r *ProjectTypeWorkflowRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectTypeWorkflow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, work_item_type, workflow_id, created_at, updated_at
		 FROM project_type_workflows WHERE project_id = $1 ORDER BY work_item_type`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying project type workflows: %w", err)
	}
	defer rows.Close()

	var mappings []model.ProjectTypeWorkflow
	for rows.Next() {
		var m model.ProjectTypeWorkflow
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.WorkItemType, &m.WorkflowID, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning project type workflow row: %w", err)
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

// GetByProjectAndType returns the workflow mapping for a specific project and type.
func (r *ProjectTypeWorkflowRepository) GetByProjectAndType(ctx context.Context, projectID uuid.UUID, workItemType string) (*model.ProjectTypeWorkflow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, work_item_type, workflow_id, created_at, updated_at
		 FROM project_type_workflows WHERE project_id = $1 AND work_item_type = $2`,
		projectID, workItemType)

	var m model.ProjectTypeWorkflow
	err := row.Scan(&m.ID, &m.ProjectID, &m.WorkItemType, &m.WorkflowID, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning project type workflow: %w", err)
	}
	return &m, nil
}

// Upsert inserts or updates a type-workflow mapping.
func (r *ProjectTypeWorkflowRepository) Upsert(ctx context.Context, m *model.ProjectTypeWorkflow) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO project_type_workflows (id, project_id, work_item_type, workflow_id)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (project_id, work_item_type)
		 DO UPDATE SET workflow_id = EXCLUDED.workflow_id, updated_at = now()
		 RETURNING id, created_at, updated_at`,
		m.ID, m.ProjectID, m.WorkItemType, m.WorkflowID).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upserting project type workflow: %w", err)
	}
	return nil
}
