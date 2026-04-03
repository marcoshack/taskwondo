package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// QueueCategoryRepository handles queue category persistence.
type QueueCategoryRepository struct {
	db *sql.DB
}

// NewQueueCategoryRepository creates a new QueueCategoryRepository.
func NewQueueCategoryRepository(db *sql.DB) *QueueCategoryRepository {
	return &QueueCategoryRepository{db: db}
}

// Create inserts a new queue category.
func (r *QueueCategoryRepository) Create(ctx context.Context, cat *model.QueueCategory) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO queue_categories (id, queue_id, name, description, position)
		 VALUES ($1, $2, $3, $4, $5)`,
		cat.ID, cat.QueueID, cat.Name, cat.Description, cat.Position)
	if err != nil {
		return fmt.Errorf("inserting queue category: %w", err)
	}
	return nil
}

// GetByID returns a queue category by ID.
func (r *QueueCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.QueueCategory, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, queue_id, name, description, position, created_at, updated_at
		 FROM queue_categories WHERE id = $1`, id)
	return scanQueueCategory(row)
}

// ListByQueue returns all categories for a queue, ordered by position then name.
func (r *QueueCategoryRepository) ListByQueue(ctx context.Context, queueID uuid.UUID) ([]model.QueueCategory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, queue_id, name, description, position, created_at, updated_at
		 FROM queue_categories WHERE queue_id = $1 ORDER BY position, name`, queueID)
	if err != nil {
		return nil, fmt.Errorf("querying queue categories: %w", err)
	}
	defer rows.Close()

	return scanQueueCategories(rows)
}

// Update modifies a queue category's mutable fields.
func (r *QueueCategoryRepository) Update(ctx context.Context, cat *model.QueueCategory) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE queue_categories SET name = $1, description = $2, position = $3, updated_at = now()
		 WHERE id = $4`,
		cat.Name, cat.Description, cat.Position, cat.ID)
	if err != nil {
		return fmt.Errorf("updating queue category: %w", err)
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

// Delete removes a queue category.
func (r *QueueCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM queue_categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting queue category: %w", err)
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

func scanQueueCategory(row *sql.Row) (*model.QueueCategory, error) {
	var cat model.QueueCategory
	var description sql.NullString

	err := row.Scan(&cat.ID, &cat.QueueID, &cat.Name, &description, &cat.Position,
		&cat.CreatedAt, &cat.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning queue category: %w", err)
	}

	if description.Valid {
		cat.Description = &description.String
	}

	return &cat, nil
}

func scanQueueCategories(rows *sql.Rows) ([]model.QueueCategory, error) {
	var categories []model.QueueCategory
	for rows.Next() {
		var cat model.QueueCategory
		var description sql.NullString

		if err := rows.Scan(&cat.ID, &cat.QueueID, &cat.Name, &description, &cat.Position,
			&cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning queue category row: %w", err)
		}

		if description.Valid {
			cat.Description = &description.String
		}

		categories = append(categories, cat)
	}
	return categories, rows.Err()
}
