package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// TimeEntryRepository handles time entry persistence.
type TimeEntryRepository struct {
	db *sql.DB
}

// NewTimeEntryRepository creates a new TimeEntryRepository.
func NewTimeEntryRepository(db *sql.DB) *TimeEntryRepository {
	return &TimeEntryRepository{db: db}
}

// Create inserts a new time entry.
func (r *TimeEntryRepository) Create(ctx context.Context, entry *model.TimeEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO time_entries (id, work_item_id, user_id, started_at, duration_seconds, description)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.ID, entry.WorkItemID, entry.UserID, entry.StartedAt, entry.DurationSeconds, entry.Description)
	if err != nil {
		return fmt.Errorf("inserting time entry: %w", err)
	}
	return nil
}

// GetByID returns a time entry by its ID.
func (r *TimeEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	var description sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, user_id, started_at, duration_seconds, description, created_at, updated_at
		 FROM time_entries
		 WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&entry.ID, &entry.WorkItemID, &entry.UserID, &entry.StartedAt,
		&entry.DurationSeconds, &description, &entry.CreatedAt, &entry.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying time entry: %w", err)
	}

	if description.Valid {
		entry.Description = &description.String
	}

	return &entry, nil
}

// ListByWorkItem returns all non-deleted time entries for a work item, ordered by started_at descending.
func (r *TimeEntryRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.TimeEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, work_item_id, user_id, started_at, duration_seconds, description, created_at, updated_at
		 FROM time_entries
		 WHERE work_item_id = $1 AND deleted_at IS NULL
		 ORDER BY started_at DESC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying time entries: %w", err)
	}
	defer rows.Close()

	var entries []model.TimeEntry
	for rows.Next() {
		var entry model.TimeEntry
		var description sql.NullString

		if err := rows.Scan(
			&entry.ID, &entry.WorkItemID, &entry.UserID, &entry.StartedAt,
			&entry.DurationSeconds, &description, &entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning time entry: %w", err)
		}

		if description.Valid {
			entry.Description = &description.String
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Update modifies a time entry's mutable fields.
func (r *TimeEntryRepository) Update(ctx context.Context, entry *model.TimeEntry) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE time_entries SET started_at = $1, duration_seconds = $2, description = $3, updated_at = now()
		 WHERE id = $4 AND deleted_at IS NULL`,
		entry.StartedAt, entry.DurationSeconds, entry.Description, entry.ID)
	if err != nil {
		return fmt.Errorf("updating time entry: %w", err)
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

// Delete soft-deletes a time entry.
func (r *TimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE time_entries SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting time entry: %w", err)
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

// SumByWorkItem returns the total duration_seconds for all non-deleted time entries on a work item.
func (r *TimeEntryRepository) SumByWorkItem(ctx context.Context, workItemID uuid.UUID) (int, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(duration_seconds), 0) FROM time_entries
		 WHERE work_item_id = $1 AND deleted_at IS NULL`, workItemID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("summing time entries: %w", err)
	}
	return int(total.Int64), nil
}
