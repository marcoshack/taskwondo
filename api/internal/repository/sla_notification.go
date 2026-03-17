package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// SLANotificationRepository handles SLA notification deduplication persistence.
type SLANotificationRepository struct {
	db *sql.DB
}

// NewSLANotificationRepository creates a new SLANotificationRepository.
func NewSLANotificationRepository(db *sql.DB) *SLANotificationRepository {
	return &SLANotificationRepository{db: db}
}

// RecordSent records that an SLA breach notification was sent for a specific
// (work_item_id, status_name, escalation_level, threshold_pct) combination.
func (r *SLANotificationRepository) RecordSent(ctx context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sla_notifications_sent (work_item_id, status_name, escalation_level, threshold_pct)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (work_item_id, status_name, escalation_level, threshold_pct) DO NOTHING`,
		workItemID, statusName, level, thresholdPct)
	if err != nil {
		return fmt.Errorf("recording SLA notification sent: %w", err)
	}
	return nil
}

// HasBeenSent checks whether an SLA breach notification has already been sent
// for the given combination.
func (r *SLANotificationRepository) HasBeenSent(ctx context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM sla_notifications_sent
			WHERE work_item_id = $1 AND status_name = $2 AND escalation_level = $3 AND threshold_pct = $4
		)`,
		workItemID, statusName, level, thresholdPct).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking SLA notification sent: %w", err)
	}
	return exists, nil
}

// ClearForStatus deletes all SLA notification records for a work item in a
// specific status. Called when a work item transitions away from a status so
// that re-entering the status will re-trigger escalation notifications.
func (r *SLANotificationRepository) ClearForStatus(ctx context.Context, workItemID uuid.UUID, statusName string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sla_notifications_sent WHERE work_item_id = $1 AND status_name = $2`,
		workItemID, statusName)
	if err != nil {
		return fmt.Errorf("clearing SLA notifications for status: %w", err)
	}
	return nil
}
