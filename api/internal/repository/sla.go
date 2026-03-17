package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// SLARepository handles SLA target and elapsed time persistence.
type SLARepository struct {
	db *sql.DB
}

// NewSLARepository creates a new SLARepository.
func NewSLARepository(db *sql.DB) *SLARepository {
	return &SLARepository{db: db}
}

// ListTargetsByProject returns all SLA targets for a project.
func (r *SLARepository) ListTargetsByProject(ctx context.Context, projectID uuid.UUID) ([]model.SLAStatusTarget, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, work_item_type, workflow_id, status_name, priority, target_seconds, calendar_mode, created_at, updated_at
		 FROM sla_status_targets WHERE project_id = $1
		 ORDER BY work_item_type, status_name, priority`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying SLA targets: %w", err)
	}
	defer rows.Close()

	var targets []model.SLAStatusTarget
	for rows.Next() {
		var t model.SLAStatusTarget
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.WorkItemType, &t.WorkflowID,
			&t.StatusName, &t.Priority, &t.TargetSeconds, &t.CalendarMode, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning SLA target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// ListTargetsByProjectAndType returns SLA targets for a specific type+workflow in a project.
func (r *SLARepository) ListTargetsByProjectAndType(ctx context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) ([]model.SLAStatusTarget, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, work_item_type, workflow_id, status_name, priority, target_seconds, calendar_mode, created_at, updated_at
		 FROM sla_status_targets WHERE project_id = $1 AND work_item_type = $2 AND workflow_id = $3
		 ORDER BY status_name, priority`, projectID, workItemType, workflowID)
	if err != nil {
		return nil, fmt.Errorf("querying SLA targets by type: %w", err)
	}
	defer rows.Close()

	var targets []model.SLAStatusTarget
	for rows.Next() {
		var t model.SLAStatusTarget
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.WorkItemType, &t.WorkflowID,
			&t.StatusName, &t.Priority, &t.TargetSeconds, &t.CalendarMode, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning SLA target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// GetTarget returns a single SLA target by its composite key (including priority).
func (r *SLARepository) GetTarget(ctx context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID, statusName string, priority string) (*model.SLAStatusTarget, error) {
	var t model.SLAStatusTarget
	err := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, work_item_type, workflow_id, status_name, priority, target_seconds, calendar_mode, created_at, updated_at
		 FROM sla_status_targets WHERE project_id = $1 AND work_item_type = $2 AND workflow_id = $3 AND status_name = $4 AND priority = $5`,
		projectID, workItemType, workflowID, statusName, priority).Scan(
		&t.ID, &t.ProjectID, &t.WorkItemType, &t.WorkflowID,
		&t.StatusName, &t.Priority, &t.TargetSeconds, &t.CalendarMode, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning SLA target: %w", err)
	}
	return &t, nil
}

// BulkUpsertTargets inserts or updates multiple SLA targets in a transaction.
func (r *SLARepository) BulkUpsertTargets(ctx context.Context, targets []model.SLAStatusTarget) ([]model.SLAStatusTarget, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	result := make([]model.SLAStatusTarget, len(targets))
	for i, t := range targets {
		err := tx.QueryRowContext(ctx,
			`INSERT INTO sla_status_targets (id, project_id, work_item_type, workflow_id, status_name, priority, target_seconds, calendar_mode)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (project_id, work_item_type, workflow_id, status_name, priority)
			 DO UPDATE SET target_seconds = EXCLUDED.target_seconds, calendar_mode = EXCLUDED.calendar_mode, updated_at = now()
			 RETURNING id, created_at, updated_at`,
			t.ID, t.ProjectID, t.WorkItemType, t.WorkflowID, t.StatusName, t.Priority, t.TargetSeconds, t.CalendarMode).
			Scan(&result[i].ID, &result[i].CreatedAt, &result[i].UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("upserting SLA target: %w", err)
		}
		result[i].ProjectID = t.ProjectID
		result[i].WorkItemType = t.WorkItemType
		result[i].WorkflowID = t.WorkflowID
		result[i].StatusName = t.StatusName
		result[i].Priority = t.Priority
		result[i].TargetSeconds = t.TargetSeconds
		result[i].CalendarMode = t.CalendarMode
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}
	return result, nil
}

// DeleteTarget deletes a single SLA target by ID.
func (r *SLARepository) DeleteTarget(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM sla_status_targets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting SLA target: %w", err)
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

// DeleteTargetsByTypeAndWorkflow deletes all SLA targets for a type+workflow combo.
func (r *SLARepository) DeleteTargetsByTypeAndWorkflow(ctx context.Context, projectID uuid.UUID, workItemType string, workflowID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sla_status_targets WHERE project_id = $1 AND work_item_type = $2 AND workflow_id = $3`,
		projectID, workItemType, workflowID)
	if err != nil {
		return fmt.Errorf("deleting SLA targets by type: %w", err)
	}
	return nil
}

// InitElapsedOnCreate inserts an initial elapsed record when a work item is created.
func (r *SLARepository) InitElapsedOnCreate(ctx context.Context, workItemID uuid.UUID, statusName string, enteredAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO work_item_sla_elapsed (work_item_id, status_name, elapsed_seconds, last_entered_at)
		 VALUES ($1, $2, 0, $3)
		 ON CONFLICT (work_item_id, status_name) DO NOTHING`,
		workItemID, statusName, enteredAt)
	if err != nil {
		return fmt.Errorf("initializing SLA elapsed: %w", err)
	}
	return nil
}

// UpsertElapsedOnEnter upserts an elapsed record when an item enters a status.
// If the item has been in this status before, elapsed_seconds is preserved (anti-gaming).
func (r *SLARepository) UpsertElapsedOnEnter(ctx context.Context, workItemID uuid.UUID, statusName string, now time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO work_item_sla_elapsed (work_item_id, status_name, elapsed_seconds, last_entered_at)
		 VALUES ($1, $2, 0, $3)
		 ON CONFLICT (work_item_id, status_name)
		 DO UPDATE SET last_entered_at = $3`,
		workItemID, statusName, now)
	if err != nil {
		return fmt.Errorf("upserting SLA elapsed on enter: %w", err)
	}
	return nil
}

// UpdateElapsedOnLeave accumulates elapsed time when an item leaves a status.
func (r *SLARepository) UpdateElapsedOnLeave(ctx context.Context, workItemID uuid.UUID, statusName string, now time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE work_item_sla_elapsed
		 SET elapsed_seconds = elapsed_seconds + GREATEST(0, EXTRACT(EPOCH FROM ($3::timestamptz - last_entered_at))::INT),
		     last_entered_at = NULL
		 WHERE work_item_id = $1 AND status_name = $2 AND last_entered_at IS NOT NULL`,
		workItemID, statusName, now)
	if err != nil {
		return fmt.Errorf("updating SLA elapsed on leave: %w", err)
	}
	return nil
}

// UpdateElapsedOnLeaveWithSeconds accumulates a pre-computed number of seconds
// when an item leaves a status. Used for business-hours-aware elapsed tracking
// where the caller computes business seconds instead of relying on wall-clock SQL.
func (r *SLARepository) UpdateElapsedOnLeaveWithSeconds(ctx context.Context, workItemID uuid.UUID, statusName string, additionalSeconds int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE work_item_sla_elapsed
		 SET elapsed_seconds = elapsed_seconds + $3,
		     last_entered_at = NULL
		 WHERE work_item_id = $1 AND status_name = $2 AND last_entered_at IS NOT NULL`,
		workItemID, statusName, additionalSeconds)
	if err != nil {
		return fmt.Errorf("updating SLA elapsed on leave with seconds: %w", err)
	}
	return nil
}

// GetElapsed returns the elapsed record for a work item in a specific status.
func (r *SLARepository) GetElapsed(ctx context.Context, workItemID uuid.UUID, statusName string) (*model.SLAElapsed, error) {
	var e model.SLAElapsed
	var lastEnteredAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT work_item_id, status_name, elapsed_seconds, last_entered_at
		 FROM work_item_sla_elapsed WHERE work_item_id = $1 AND status_name = $2`,
		workItemID, statusName).Scan(&e.WorkItemID, &e.StatusName, &e.ElapsedSeconds, &lastEnteredAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning SLA elapsed: %w", err)
	}
	if lastEnteredAt.Valid {
		e.LastEnteredAt = &lastEnteredAt.Time
	}
	return &e, nil
}

// ListElapsedByWorkItemIDs returns all elapsed records for multiple work items (batch load).
func (r *SLARepository) ListElapsedByWorkItemIDs(ctx context.Context, ids []uuid.UUID) ([]model.SLAElapsed, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT work_item_id, status_name, elapsed_seconds, last_entered_at
		 FROM work_item_sla_elapsed WHERE work_item_id = ANY($1)`,
		uuidArray(ids))
	if err != nil {
		return nil, fmt.Errorf("querying SLA elapsed batch: %w", err)
	}
	defer rows.Close()

	var result []model.SLAElapsed
	for rows.Next() {
		var e model.SLAElapsed
		var lastEnteredAt sql.NullTime
		if err := rows.Scan(&e.WorkItemID, &e.StatusName, &e.ElapsedSeconds, &lastEnteredAt); err != nil {
			return nil, fmt.Errorf("scanning SLA elapsed row: %w", err)
		}
		if lastEnteredAt.Valid {
			e.LastEnteredAt = &lastEnteredAt.Time
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
