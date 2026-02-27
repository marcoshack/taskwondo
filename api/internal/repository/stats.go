package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// StatsRepository handles project stats snapshot persistence.
type StatsRepository struct {
	db *sql.DB
}

// NewStatsRepository creates a new StatsRepository.
func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// SnapshotAll creates a point-in-time snapshot of work item counts for all
// active projects. It inserts both project-level aggregates (user_id IS NULL)
// and per-assignee breakdowns (user_id IS NOT NULL) in a single transaction.
func (r *StatsRepository) SnapshotAll(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Project-level aggregates
	_, err = tx.ExecContext(ctx, `
		INSERT INTO project_stats_snapshots (project_id, todo_count, in_progress_count, done_count, cancelled_count)
		SELECT
			p.id,
			COUNT(*) FILTER (WHERE ws.category = 'todo'),
			COUNT(*) FILTER (WHERE ws.category = 'in_progress'),
			COUNT(*) FILTER (WHERE ws.category = 'done'),
			COUNT(*) FILTER (WHERE ws.category = 'cancelled')
		FROM work_items wi
		JOIN projects p          ON p.id = wi.project_id
		JOIN workflows w         ON w.id = p.default_workflow_id
		JOIN workflow_statuses ws ON ws.workflow_id = w.id AND ws.name = wi.status
		WHERE wi.deleted_at IS NULL AND p.deleted_at IS NULL
		GROUP BY p.id
	`)
	if err != nil {
		return fmt.Errorf("inserting project-level snapshots: %w", err)
	}

	// Per-assignee breakdowns
	_, err = tx.ExecContext(ctx, `
		INSERT INTO project_stats_snapshots (project_id, user_id, todo_count, in_progress_count, done_count, cancelled_count)
		SELECT
			p.id,
			wi.assignee_id,
			COUNT(*) FILTER (WHERE ws.category = 'todo'),
			COUNT(*) FILTER (WHERE ws.category = 'in_progress'),
			COUNT(*) FILTER (WHERE ws.category = 'done'),
			COUNT(*) FILTER (WHERE ws.category = 'cancelled')
		FROM work_items wi
		JOIN projects p          ON p.id = wi.project_id
		JOIN workflows w         ON w.id = p.default_workflow_id
		JOIN workflow_statuses ws ON ws.workflow_id = w.id AND ws.name = wi.status
		WHERE wi.deleted_at IS NULL AND p.deleted_at IS NULL AND wi.assignee_id IS NOT NULL
		GROUP BY p.id, wi.assignee_id
	`)
	if err != nil {
		return fmt.Errorf("inserting per-assignee snapshots: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing snapshot transaction: %w", err)
	}

	log.Ctx(ctx).Debug().Msg("stats snapshot completed")
	return nil
}

// CompactOlderThan rolls up fine-grained snapshots older than threshold to
// hourly granularity. It keeps the last snapshot per (project_id, user_id, hour)
// and deletes the rest. Returns the number of deleted rows.
func (r *StatsRepository) CompactOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert hourly rollups for project-level rows (user_id IS NULL).
	// DISTINCT ON keeps the last snapshot per (project_id, hour).
	_, err = tx.ExecContext(ctx, `
		INSERT INTO project_stats_snapshots (project_id, todo_count, in_progress_count, done_count, cancelled_count, captured_at)
		SELECT DISTINCT ON (project_id, date_trunc('hour', captured_at))
			project_id, todo_count, in_progress_count, done_count, cancelled_count,
			date_trunc('hour', captured_at) AS captured_at
		FROM project_stats_snapshots
		WHERE captured_at < $1 AND user_id IS NULL
			AND captured_at != date_trunc('hour', captured_at)
		ORDER BY project_id, date_trunc('hour', captured_at), captured_at DESC
	`, threshold)
	if err != nil {
		return 0, fmt.Errorf("inserting project-level hourly rollups: %w", err)
	}

	// Insert hourly rollups for per-user rows (user_id IS NOT NULL).
	_, err = tx.ExecContext(ctx, `
		INSERT INTO project_stats_snapshots (project_id, user_id, todo_count, in_progress_count, done_count, cancelled_count, captured_at)
		SELECT DISTINCT ON (project_id, user_id, date_trunc('hour', captured_at))
			project_id, user_id, todo_count, in_progress_count, done_count, cancelled_count,
			date_trunc('hour', captured_at) AS captured_at
		FROM project_stats_snapshots
		WHERE captured_at < $1 AND user_id IS NOT NULL
			AND captured_at != date_trunc('hour', captured_at)
		ORDER BY project_id, user_id, date_trunc('hour', captured_at), captured_at DESC
	`, threshold)
	if err != nil {
		return 0, fmt.Errorf("inserting per-user hourly rollups: %w", err)
	}

	// Delete fine-grained rows that were compacted (not on hour boundaries).
	result, err := tx.ExecContext(ctx, `
		DELETE FROM project_stats_snapshots
		WHERE captured_at < $1
			AND captured_at != date_trunc('hour', captured_at)
	`, threshold)
	if err != nil {
		return 0, fmt.Errorf("deleting compacted rows: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing compaction transaction: %w", err)
	}

	return deleted, nil
}

// Backfill generates historical stats snapshots at hourly intervals from the
// earliest work item in the database to now. It reconstructs state at each
// time point using the work_item_events audit trail. Time points that already
// have snapshots are skipped (idempotent). Returns the number of snapshots inserted.
func (r *StatsRepository) Backfill(ctx context.Context) (int64, error) {
	logger := log.Ctx(ctx)

	// Find the earliest work item creation time
	var earliest time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT MIN(created_at) FROM work_items WHERE deleted_at IS NULL
	`).Scan(&earliest)
	if err != nil || earliest.IsZero() {
		logger.Info().Msg("no work items found, nothing to backfill")
		return 0, nil
	}

	// Truncate to the hour
	earliest = earliest.Truncate(time.Hour)
	now := time.Now()

	logger.Info().Time("from", earliest).Time("to", now).Msg("starting stats backfill")

	var totalInserted int64

	for t := earliest; t.Before(now); t = t.Add(time.Hour) {
		// Check if we already have snapshots for this hour
		var count int
		err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM project_stats_snapshots
			WHERE captured_at = $1
		`, t).Scan(&count)
		if err != nil {
			return totalInserted, fmt.Errorf("checking existing snapshots at %v: %w", t, err)
		}
		if count > 0 {
			continue // already backfilled
		}

		inserted, err := r.backfillAtTime(ctx, t)
		if err != nil {
			return totalInserted, fmt.Errorf("backfilling at %v: %w", t, err)
		}
		totalInserted += inserted
	}

	logger.Info().Int64("snapshots_inserted", totalInserted).Msg("stats backfill completed")
	return totalInserted, nil
}

// backfillAtTime reconstructs work item counts at a specific point in time
// and inserts project-level and per-assignee snapshots.
func (r *StatsRepository) backfillAtTime(ctx context.Context, t time.Time) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Reconstruct the status of each work item at time t using the events table.
	// For each item that existed at time t:
	//   - Find the latest status_changed event before t → that's the status at t
	//   - If no status event exists, use the item's initial status from creation
	//
	// Then map status → category via workflows and count.

	// Project-level aggregates
	result, err := tx.ExecContext(ctx, `
		INSERT INTO project_stats_snapshots (project_id, todo_count, in_progress_count, done_count, cancelled_count, captured_at)
		WITH item_status_at_t AS (
			SELECT
				wi.id,
				wi.project_id,
				COALESCE(
					(SELECT e.new_value
					 FROM work_item_events e
					 WHERE e.work_item_id = wi.id
					   AND e.field_name = 'status'
					   AND e.created_at <= $1
					 ORDER BY e.created_at DESC
					 LIMIT 1),
					'open'
				) AS status_at_t
			FROM work_items wi
			WHERE wi.created_at <= $1
			  AND (wi.deleted_at IS NULL OR wi.deleted_at > $1)
		)
		SELECT
			ist.project_id,
			COUNT(*) FILTER (WHERE ws.category = 'todo'),
			COUNT(*) FILTER (WHERE ws.category = 'in_progress'),
			COUNT(*) FILTER (WHERE ws.category = 'done'),
			COUNT(*) FILTER (WHERE ws.category = 'cancelled'),
			$1
		FROM item_status_at_t ist
		JOIN projects p          ON p.id = ist.project_id
		JOIN workflows w         ON w.id = p.default_workflow_id
		JOIN workflow_statuses ws ON ws.workflow_id = w.id AND ws.name = ist.status_at_t
		WHERE p.deleted_at IS NULL
		GROUP BY ist.project_id
	`, t)
	if err != nil {
		return 0, fmt.Errorf("inserting project-level backfill: %w", err)
	}
	projectRows, _ := result.RowsAffected()

	// Per-assignee breakdowns — also reconstruct assignee at time t
	result, err = tx.ExecContext(ctx, `
		INSERT INTO project_stats_snapshots (project_id, user_id, todo_count, in_progress_count, done_count, cancelled_count, captured_at)
		WITH item_state_at_t AS (
			SELECT
				wi.id,
				wi.project_id,
				COALESCE(
					(SELECT e.new_value
					 FROM work_item_events e
					 WHERE e.work_item_id = wi.id
					   AND e.field_name = 'status'
					   AND e.created_at <= $1
					 ORDER BY e.created_at DESC
					 LIMIT 1),
					'open'
				) AS status_at_t,
				COALESCE(
					(SELECT e.new_value
					 FROM work_item_events e
					 WHERE e.work_item_id = wi.id
					   AND e.event_type = 'assigned'
					   AND e.created_at <= $1
					 ORDER BY e.created_at DESC
					 LIMIT 1),
					wi.assignee_id::text
				) AS assignee_at_t
			FROM work_items wi
			WHERE wi.created_at <= $1
			  AND (wi.deleted_at IS NULL OR wi.deleted_at > $1)
		)
		SELECT
			ist.project_id,
			ist.assignee_at_t::uuid,
			COUNT(*) FILTER (WHERE ws.category = 'todo'),
			COUNT(*) FILTER (WHERE ws.category = 'in_progress'),
			COUNT(*) FILTER (WHERE ws.category = 'done'),
			COUNT(*) FILTER (WHERE ws.category = 'cancelled'),
			$1
		FROM item_state_at_t ist
		JOIN projects p          ON p.id = ist.project_id
		JOIN workflows w         ON w.id = p.default_workflow_id
		JOIN workflow_statuses ws ON ws.workflow_id = w.id AND ws.name = ist.status_at_t
		WHERE p.deleted_at IS NULL
		  AND ist.assignee_at_t IS NOT NULL
		  AND ist.assignee_at_t != ''
		GROUP BY ist.project_id, ist.assignee_at_t
	`, t)
	if err != nil {
		return 0, fmt.Errorf("inserting per-assignee backfill: %w", err)
	}
	userRows, _ := result.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing backfill transaction: %w", err)
	}

	return projectRows + userRows, nil
}

// DeleteBefore removes all snapshots older than the given time.
// Returns the number of deleted rows.
func (r *StatsRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM project_stats_snapshots WHERE captured_at < $1
	`, before)
	if err != nil {
		return 0, fmt.Errorf("deleting old snapshots: %w", err)
	}
	return result.RowsAffected()
}
