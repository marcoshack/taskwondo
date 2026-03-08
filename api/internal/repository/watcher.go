package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// WatcherRepository handles work item watcher persistence.
type WatcherRepository struct {
	db *sql.DB
}

// NewWatcherRepository creates a new WatcherRepository.
func NewWatcherRepository(db *sql.DB) *WatcherRepository {
	return &WatcherRepository{db: db}
}

// Create inserts a new watcher. Uses ON CONFLICT DO NOTHING for idempotency.
func (r *WatcherRepository) Create(ctx context.Context, watcher *model.WorkItemWatcher) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO work_item_watchers (id, work_item_id, user_id, added_by)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (work_item_id, user_id) DO NOTHING
		 RETURNING id, created_at`,
		watcher.ID, watcher.WorkItemID, watcher.UserID, watcher.AddedBy,
	).Scan(&watcher.ID, &watcher.CreatedAt)
	if err == sql.ErrNoRows {
		// Already exists — fetch the existing record
		return r.db.QueryRowContext(ctx,
			`SELECT id, created_at FROM work_item_watchers WHERE work_item_id = $1 AND user_id = $2`,
			watcher.WorkItemID, watcher.UserID,
		).Scan(&watcher.ID, &watcher.CreatedAt)
	}
	if err != nil {
		return fmt.Errorf("inserting watcher: %w", err)
	}
	return nil
}

// Delete hard-deletes a watcher by work item and user.
func (r *WatcherRepository) Delete(ctx context.Context, workItemID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM work_item_watchers WHERE work_item_id = $1 AND user_id = $2`,
		workItemID, userID)
	if err != nil {
		return fmt.Errorf("deleting watcher: %w", err)
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

// ListByWorkItem returns all watchers for a work item with user display info.
func (r *WatcherRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemWatcherWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT w.id, w.work_item_id, w.user_id, w.added_by, w.created_at,
		        u.display_name, u.email, u.avatar_url,
		        ab.display_name
		 FROM work_item_watchers w
		 JOIN users u ON u.id = w.user_id
		 JOIN users ab ON ab.id = w.added_by
		 WHERE w.work_item_id = $1
		 ORDER BY w.created_at ASC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying watchers: %w", err)
	}
	defer rows.Close()

	var watchers []model.WorkItemWatcherWithUser
	for rows.Next() {
		var w model.WorkItemWatcherWithUser
		if err := rows.Scan(
			&w.ID, &w.WorkItemID, &w.UserID, &w.AddedBy, &w.CreatedAt,
			&w.DisplayName, &w.Email, &w.AvatarURL,
			&w.AddedByName,
		); err != nil {
			return nil, fmt.Errorf("scanning watcher: %w", err)
		}
		watchers = append(watchers, w)
	}

	return watchers, rows.Err()
}

// CountByWorkItem returns the total number of watchers for a work item.
func (r *WatcherRepository) CountByWorkItem(ctx context.Context, workItemID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM work_item_watchers WHERE work_item_id = $1`, workItemID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting watchers: %w", err)
	}
	return count, nil
}

// IsWatching checks if a user is watching a work item.
func (r *WatcherRepository) IsWatching(ctx context.Context, workItemID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM work_item_watchers WHERE work_item_id = $1 AND user_id = $2)`,
		workItemID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking watcher: %w", err)
	}
	return exists, nil
}

// RemoveByProjectID deletes all watchers for work items belonging to a project.
func (r *WatcherRepository) RemoveByProjectID(ctx context.Context, projectID uuid.UUID) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM work_item_watchers
		 WHERE work_item_id IN (SELECT id FROM work_items WHERE project_id = $1)`,
		projectID)
	if err != nil {
		return 0, fmt.Errorf("removing watchers by project: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// ListWatchedItemIDs returns the work item IDs that a user is watching, optionally scoped to a project.
func (r *WatcherRepository) ListWatchedItemIDs(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID) ([]uuid.UUID, error) {
	var rows *sql.Rows
	var err error
	if projectID != nil {
		rows, err = r.db.QueryContext(ctx,
			`SELECT w.work_item_id FROM work_item_watchers w
			 JOIN work_items wi ON wi.id = w.work_item_id
			 WHERE w.user_id = $1 AND wi.project_id = $2 AND wi.deleted_at IS NULL`,
			userID, *projectID)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT w.work_item_id FROM work_item_watchers w
			 JOIN work_items wi ON wi.id = w.work_item_id
			 WHERE w.user_id = $1 AND wi.deleted_at IS NULL`,
			userID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying watched item IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning watched item ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
