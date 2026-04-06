package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// OncallOverrideRepository handles on-call override persistence.
type OncallOverrideRepository struct {
	db *sql.DB
}

// NewOncallOverrideRepository creates a new OncallOverrideRepository.
func NewOncallOverrideRepository(db *sql.DB) *OncallOverrideRepository {
	return &OncallOverrideRepository{db: db}
}

// Create inserts a new on-call override.
func (r *OncallOverrideRepository) Create(ctx context.Context, o *model.OncallOverride) (*model.OncallOverride, error) {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO oncall_overrides (id, rotation_id, override_user_id, start_at, end_at, reason, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		o.ID, o.RotationID, o.OverrideUserID, o.StartAt, o.EndAt, o.Reason, o.CreatedBy,
	).Scan(&o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting oncall override: %w", err)
	}
	return o, nil
}

// GetByID returns an override by ID.
func (r *OncallOverrideRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.OncallOverride, error) {
	var o model.OncallOverride
	var reason sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, rotation_id, override_user_id, start_at, end_at, reason, created_by, created_at
		 FROM oncall_overrides WHERE id = $1`, id,
	).Scan(&o.ID, &o.RotationID, &o.OverrideUserID, &o.StartAt, &o.EndAt, &reason, &o.CreatedBy, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying oncall override: %w", err)
	}
	if reason.Valid {
		o.Reason = &reason.String
	}
	return &o, nil
}

// Update modifies an existing override.
func (r *OncallOverrideRepository) Update(ctx context.Context, o *model.OncallOverride) (*model.OncallOverride, error) {
	err := r.db.QueryRowContext(ctx,
		`UPDATE oncall_overrides
		 SET override_user_id = $1, start_at = $2, end_at = $3, reason = $4
		 WHERE id = $5
		 RETURNING created_at`,
		o.OverrideUserID, o.StartAt, o.EndAt, o.Reason, o.ID,
	).Scan(&o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating oncall override: %w", err)
	}
	return o, nil
}

// Delete removes an override by ID.
func (r *OncallOverrideRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM oncall_overrides WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting oncall override: %w", err)
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

// ListByRotation returns overrides for a rotation that have not yet ended, ordered by start time.
func (r *OncallOverrideRepository) ListByRotation(ctx context.Context, rotationID uuid.UUID) ([]model.OncallOverrideWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT o.id, o.rotation_id, o.override_user_id, o.start_at, o.end_at, o.reason,
		        o.created_by, o.created_at,
		        ou.display_name, ou.avatar_url,
		        cu.display_name
		 FROM oncall_overrides o
		 JOIN users ou ON ou.id = o.override_user_id
		 JOIN users cu ON cu.id = o.created_by
		 WHERE o.rotation_id = $1 AND o.end_at > $2
		 ORDER BY o.start_at ASC`, rotationID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("querying oncall overrides: %w", err)
	}
	defer rows.Close()

	var overrides []model.OncallOverrideWithUser
	for rows.Next() {
		var o model.OncallOverrideWithUser
		var reason sql.NullString
		var avatarURL sql.NullString
		if err := rows.Scan(&o.ID, &o.RotationID, &o.OverrideUserID, &o.StartAt, &o.EndAt, &reason,
			&o.CreatedBy, &o.CreatedAt,
			&o.OverrideUserName, &avatarURL,
			&o.CreatedByName); err != nil {
			return nil, fmt.Errorf("scanning oncall override row: %w", err)
		}
		if reason.Valid {
			o.Reason = &reason.String
		}
		if avatarURL.Valid {
			o.OverrideAvatar = &avatarURL.String
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

// GetActiveOverride returns the currently active override for a rotation (latest-created wins on overlap).
func (r *OncallOverrideRepository) GetActiveOverride(ctx context.Context, rotationID uuid.UUID) (*model.OncallOverride, error) {
	var o model.OncallOverride
	var reason sql.NullString
	now := time.Now()
	err := r.db.QueryRowContext(ctx,
		`SELECT id, rotation_id, override_user_id, start_at, end_at, reason, created_by, created_at
		 FROM oncall_overrides
		 WHERE rotation_id = $1 AND start_at <= $2 AND end_at > $2
		 ORDER BY created_at DESC
		 LIMIT 1`, rotationID, now,
	).Scan(&o.ID, &o.RotationID, &o.OverrideUserID, &o.StartAt, &o.EndAt, &reason, &o.CreatedBy, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying active oncall override: %w", err)
	}
	if reason.Valid {
		o.Reason = &reason.String
	}
	return &o, nil
}

// ListOverridesInRange returns overrides for a rotation that overlap with the given time range.
func (r *OncallOverrideRepository) ListOverridesInRange(ctx context.Context, rotationID uuid.UUID, from, to time.Time) ([]model.OncallOverrideWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT o.id, o.rotation_id, o.override_user_id, o.start_at, o.end_at, o.reason,
		        o.created_by, o.created_at,
		        ou.display_name, ou.avatar_url,
		        cu.display_name
		 FROM oncall_overrides o
		 JOIN users ou ON ou.id = o.override_user_id
		 JOIN users cu ON cu.id = o.created_by
		 WHERE o.rotation_id = $1 AND o.start_at < $3 AND o.end_at > $2
		 ORDER BY o.start_at ASC`, rotationID, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying oncall overrides in range: %w", err)
	}
	defer rows.Close()

	var overrides []model.OncallOverrideWithUser
	for rows.Next() {
		var o model.OncallOverrideWithUser
		var reason sql.NullString
		var avatarURL sql.NullString
		if err := rows.Scan(&o.ID, &o.RotationID, &o.OverrideUserID, &o.StartAt, &o.EndAt, &reason,
			&o.CreatedBy, &o.CreatedAt,
			&o.OverrideUserName, &avatarURL,
			&o.CreatedByName); err != nil {
			return nil, fmt.Errorf("scanning oncall override row: %w", err)
		}
		if reason.Valid {
			o.Reason = &reason.String
		}
		if avatarURL.Valid {
			o.OverrideAvatar = &avatarURL.String
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}
