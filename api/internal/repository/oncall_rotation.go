package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// OncallRotationRepository handles on-call rotation persistence.
type OncallRotationRepository struct {
	db *sql.DB
}

// NewOncallRotationRepository creates a new OncallRotationRepository.
func NewOncallRotationRepository(db *sql.DB) *OncallRotationRepository {
	return &OncallRotationRepository{db: db}
}

// Create inserts a new on-call rotation.
func (r *OncallRotationRepository) Create(ctx context.Context, rot *model.OncallRotation) (*model.OncallRotation, error) {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO oncall_rotations (id, team_id, period_days, rotation_time, timezone, start_date, current_user_id, current_position, is_override, next_rotation_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		rot.ID, rot.TeamID, rot.PeriodDays, rot.RotationTime, rot.Timezone,
		rot.StartDate, rot.CurrentUserID, rot.CurrentPosition, rot.IsOverride, rot.NextRotationAt,
	).Scan(&rot.CreatedAt, &rot.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting oncall rotation: %w", err)
	}
	return rot, nil
}

// GetByTeamID returns the on-call rotation for a team.
func (r *OncallRotationRepository) GetByTeamID(ctx context.Context, teamID uuid.UUID) (*model.OncallRotation, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, team_id, period_days, rotation_time, timezone, start_date,
		        current_user_id, current_position, is_override, next_rotation_at, created_at, updated_at
		 FROM oncall_rotations WHERE team_id = $1`, teamID)
	return scanOncallRotation(row)
}

// GetByID returns the on-call rotation by ID.
func (r *OncallRotationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.OncallRotation, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, team_id, period_days, rotation_time, timezone, start_date,
		        current_user_id, current_position, is_override, next_rotation_at, created_at, updated_at
		 FROM oncall_rotations WHERE id = $1`, id)
	return scanOncallRotation(row)
}

// Update modifies an on-call rotation.
func (r *OncallRotationRepository) Update(ctx context.Context, rot *model.OncallRotation) (*model.OncallRotation, error) {
	err := r.db.QueryRowContext(ctx,
		`UPDATE oncall_rotations
		 SET period_days = $1, rotation_time = $2, timezone = $3, start_date = $4,
		     current_user_id = $5, current_position = $6, is_override = $7, next_rotation_at = $8, updated_at = now()
		 WHERE id = $9
		 RETURNING updated_at`,
		rot.PeriodDays, rot.RotationTime, rot.Timezone, rot.StartDate,
		rot.CurrentUserID, rot.CurrentPosition, rot.IsOverride, rot.NextRotationAt, rot.ID,
	).Scan(&rot.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating oncall rotation: %w", err)
	}
	return rot, nil
}

// Delete removes the on-call rotation for a team.
func (r *OncallRotationRepository) Delete(ctx context.Context, teamID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM oncall_rotations WHERE team_id = $1`, teamID)
	if err != nil {
		return fmt.Errorf("deleting oncall rotation: %w", err)
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

// SetMembers replaces all members of a rotation (DELETE + INSERT).
func (r *OncallRotationRepository) SetMembers(ctx context.Context, rotationID uuid.UUID, members []model.OncallRotationMember) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM oncall_rotation_members WHERE rotation_id = $1`, rotationID); err != nil {
		return fmt.Errorf("deleting old rotation members: %w", err)
	}

	for _, m := range members {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO oncall_rotation_members (id, rotation_id, user_id, position)
			 VALUES ($1, $2, $3, $4)`,
			m.ID, rotationID, m.UserID, m.Position); err != nil {
			return fmt.Errorf("inserting rotation member: %w", err)
		}
	}

	return tx.Commit()
}

// ListMembers returns all members of a rotation with user details.
func (r *OncallRotationRepository) ListMembers(ctx context.Context, rotationID uuid.UUID) ([]model.OncallRotationMemberWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.rotation_id, m.user_id, m.position,
		        u.email, u.display_name, u.avatar_url
		 FROM oncall_rotation_members m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.rotation_id = $1
		 ORDER BY m.position`, rotationID)
	if err != nil {
		return nil, fmt.Errorf("querying rotation members: %w", err)
	}
	defer rows.Close()

	var members []model.OncallRotationMemberWithUser
	for rows.Next() {
		var m model.OncallRotationMemberWithUser
		var avatarURL sql.NullString
		if err := rows.Scan(&m.ID, &m.RotationID, &m.UserID, &m.Position,
			&m.Email, &m.DisplayName, &avatarURL); err != nil {
			return nil, fmt.Errorf("scanning rotation member row: %w", err)
		}
		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// CreateHistory inserts a new history entry.
func (r *OncallRotationRepository) CreateHistory(ctx context.Context, h *model.OncallRotationHistory) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO oncall_rotation_history (id, rotation_id, user_id, started_at)
		 VALUES ($1, $2, $3, $4)`,
		h.ID, h.RotationID, h.UserID, h.StartedAt)
	if err != nil {
		return fmt.Errorf("inserting rotation history: %w", err)
	}
	return nil
}

// EndCurrentHistory ends the current open history entry for a rotation.
func (r *OncallRotationRepository) EndCurrentHistory(ctx context.Context, rotationID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE oncall_rotation_history SET ended_at = now()
		 WHERE rotation_id = $1 AND ended_at IS NULL`, rotationID)
	if err != nil {
		return fmt.Errorf("ending current history: %w", err)
	}
	return nil
}

// ListHistory returns paginated history for a rotation with user details.
func (r *OncallRotationRepository) ListHistory(ctx context.Context, rotationID uuid.UUID, limit, offset int) ([]model.OncallRotationHistoryWithUser, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM oncall_rotation_history WHERE rotation_id = $1`, rotationID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting history: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT h.id, h.rotation_id, h.user_id, h.started_at, h.ended_at, h.created_at,
		        u.display_name, u.avatar_url
		 FROM oncall_rotation_history h
		 JOIN users u ON u.id = h.user_id
		 WHERE h.rotation_id = $1
		 ORDER BY h.started_at DESC
		 LIMIT $2 OFFSET $3`, rotationID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying rotation history: %w", err)
	}
	defer rows.Close()

	var history []model.OncallRotationHistoryWithUser
	for rows.Next() {
		var h model.OncallRotationHistoryWithUser
		var endedAt sql.NullTime
		var avatarURL sql.NullString
		if err := rows.Scan(&h.ID, &h.RotationID, &h.UserID, &h.StartedAt, &endedAt, &h.CreatedAt,
			&h.DisplayName, &avatarURL); err != nil {
			return nil, 0, fmt.Errorf("scanning rotation history row: %w", err)
		}
		if endedAt.Valid {
			h.EndedAt = &endedAt.Time
		}
		if avatarURL.Valid {
			h.AvatarURL = &avatarURL.String
		}
		history = append(history, h)
	}
	return history, total, rows.Err()
}

// ListDueRotations returns all rotations where next_rotation_at <= now().
func (r *OncallRotationRepository) ListDueRotations(ctx context.Context) ([]model.OncallRotation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, team_id, period_days, rotation_time, timezone, start_date,
		        current_user_id, current_position, is_override, next_rotation_at, created_at, updated_at
		 FROM oncall_rotations
		 WHERE next_rotation_at IS NOT NULL AND next_rotation_at <= $1`, time.Now())
	if err != nil {
		return nil, fmt.Errorf("querying due rotations: %w", err)
	}
	defer rows.Close()

	var rotations []model.OncallRotation
	for rows.Next() {
		rot, err := scanOncallRotationRow(rows)
		if err != nil {
			return nil, err
		}
		rotations = append(rotations, *rot)
	}
	return rotations, rows.Err()
}

func scanOncallRotation(row *sql.Row) (*model.OncallRotation, error) {
	var rot model.OncallRotation
	var currentUserID sql.NullString
	var nextRotationAt sql.NullTime

	err := row.Scan(&rot.ID, &rot.TeamID, &rot.PeriodDays, &rot.RotationTime, &rot.Timezone,
		&rot.StartDate, &currentUserID, &rot.CurrentPosition, &rot.IsOverride, &nextRotationAt,
		&rot.CreatedAt, &rot.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning oncall rotation: %w", err)
	}

	if currentUserID.Valid {
		id, _ := uuid.Parse(currentUserID.String)
		rot.CurrentUserID = &id
	}
	if nextRotationAt.Valid {
		rot.NextRotationAt = &nextRotationAt.Time
	}

	return &rot, nil
}

func scanOncallRotationRow(rows *sql.Rows) (*model.OncallRotation, error) {
	var rot model.OncallRotation
	var currentUserID sql.NullString
	var nextRotationAt sql.NullTime

	err := rows.Scan(&rot.ID, &rot.TeamID, &rot.PeriodDays, &rot.RotationTime, &rot.Timezone,
		&rot.StartDate, &currentUserID, &rot.CurrentPosition, &rot.IsOverride, &nextRotationAt,
		&rot.CreatedAt, &rot.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning oncall rotation row: %w", err)
	}

	if currentUserID.Valid {
		id, _ := uuid.Parse(currentUserID.String)
		rot.CurrentUserID = &id
	}
	if nextRotationAt.Valid {
		rot.NextRotationAt = &nextRotationAt.Time
	}

	return &rot, nil
}
