package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/trackforge/internal/model"
)

// QueueRepository handles queue persistence.
type QueueRepository struct {
	db *sql.DB
}

// NewQueueRepository creates a new QueueRepository.
func NewQueueRepository(db *sql.DB) *QueueRepository {
	return &QueueRepository{db: db}
}

// Create inserts a new queue.
func (r *QueueRepository) Create(ctx context.Context, q *model.Queue) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO queues (id, project_id, name, description, queue_type, is_public, default_priority, default_assignee_id, workflow_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		q.ID, q.ProjectID, q.Name, q.Description, q.QueueType, q.IsPublic, q.DefaultPriority, q.DefaultAssigneeID, q.WorkflowID)
	if err != nil {
		return fmt.Errorf("inserting queue: %w", err)
	}
	return nil
}

// GetByID returns a queue by ID.
func (r *QueueRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Queue, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, queue_type, is_public, default_priority, default_assignee_id, workflow_id, created_at, updated_at
		 FROM queues WHERE id = $1`, id)
	return scanQueue(row)
}

// List returns all queues for a project.
func (r *QueueRepository) List(ctx context.Context, projectID uuid.UUID) ([]model.Queue, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, queue_type, is_public, default_priority, default_assignee_id, workflow_id, created_at, updated_at
		 FROM queues WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying queues: %w", err)
	}
	defer rows.Close()

	return scanQueues(rows)
}

// Update modifies a queue's mutable fields.
func (r *QueueRepository) Update(ctx context.Context, q *model.Queue) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE queues SET name = $1, description = $2, queue_type = $3, is_public = $4,
		 default_priority = $5, default_assignee_id = $6, workflow_id = $7, updated_at = now()
		 WHERE id = $8`,
		q.Name, q.Description, q.QueueType, q.IsPublic, q.DefaultPriority, q.DefaultAssigneeID, q.WorkflowID, q.ID)
	if err != nil {
		return fmt.Errorf("updating queue: %w", err)
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

// Delete removes a queue.
func (r *QueueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM queues WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting queue: %w", err)
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

func scanQueue(row *sql.Row) (*model.Queue, error) {
	var q model.Queue
	var description sql.NullString
	var defaultAssigneeID uuid.NullUUID
	var workflowID uuid.NullUUID

	err := row.Scan(&q.ID, &q.ProjectID, &q.Name, &description, &q.QueueType,
		&q.IsPublic, &q.DefaultPriority, &defaultAssigneeID, &workflowID,
		&q.CreatedAt, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning queue: %w", err)
	}

	if description.Valid {
		q.Description = &description.String
	}
	if defaultAssigneeID.Valid {
		q.DefaultAssigneeID = &defaultAssigneeID.UUID
	}
	if workflowID.Valid {
		q.WorkflowID = &workflowID.UUID
	}

	return &q, nil
}

func scanQueues(rows *sql.Rows) ([]model.Queue, error) {
	var queues []model.Queue
	for rows.Next() {
		var q model.Queue
		var description sql.NullString
		var defaultAssigneeID uuid.NullUUID
		var workflowID uuid.NullUUID

		if err := rows.Scan(&q.ID, &q.ProjectID, &q.Name, &description, &q.QueueType,
			&q.IsPublic, &q.DefaultPriority, &defaultAssigneeID, &workflowID,
			&q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning queue row: %w", err)
		}

		if description.Valid {
			q.Description = &description.String
		}
		if defaultAssigneeID.Valid {
			q.DefaultAssigneeID = &defaultAssigneeID.UUID
		}
		if workflowID.Valid {
			q.WorkflowID = &workflowID.UUID
		}

		queues = append(queues, q)
	}
	return queues, rows.Err()
}
