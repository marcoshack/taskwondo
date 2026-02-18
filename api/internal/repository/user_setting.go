package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/trackforge/internal/model"
)

// UserSettingRepository handles user setting persistence.
type UserSettingRepository struct {
	db *sql.DB
}

// NewUserSettingRepository creates a new UserSettingRepository.
func NewUserSettingRepository(db *sql.DB) *UserSettingRepository {
	return &UserSettingRepository{db: db}
}

// Upsert creates or updates a user setting.
func (r *UserSettingRepository) Upsert(ctx context.Context, s *model.UserSetting) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_settings (user_id, project_id, key, value, updated_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (user_id, COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid), key)
		 DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		s.UserID, s.ProjectID, s.Key, s.Value)
	if err != nil {
		return fmt.Errorf("upserting user setting: %w", err)
	}
	return nil
}

// Get returns a single user setting by key.
func (r *UserSettingRepository) Get(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) (*model.UserSetting, error) {
	var row *sql.Row
	if projectID != nil {
		row = r.db.QueryRowContext(ctx,
			`SELECT user_id, project_id, key, value, updated_at
			 FROM user_settings WHERE user_id = $1 AND project_id = $2 AND key = $3`,
			userID, projectID, key)
	} else {
		row = r.db.QueryRowContext(ctx,
			`SELECT user_id, project_id, key, value, updated_at
			 FROM user_settings WHERE user_id = $1 AND project_id IS NULL AND key = $2`,
			userID, key)
	}
	return scanUserSetting(row)
}

// ListByProject returns all settings for a user in a project.
func (r *UserSettingRepository) ListByProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]model.UserSetting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, project_id, key, value, updated_at
		 FROM user_settings WHERE user_id = $1 AND project_id = $2 ORDER BY key`,
		userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying user settings: %w", err)
	}
	defer rows.Close()

	return scanUserSettings(rows)
}

// Delete removes a user setting.
func (r *UserSettingRepository) Delete(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) error {
	var result sql.Result
	var err error
	if projectID != nil {
		result, err = r.db.ExecContext(ctx,
			`DELETE FROM user_settings WHERE user_id = $1 AND project_id = $2 AND key = $3`,
			userID, projectID, key)
	} else {
		result, err = r.db.ExecContext(ctx,
			`DELETE FROM user_settings WHERE user_id = $1 AND project_id IS NULL AND key = $2`,
			userID, key)
	}
	if err != nil {
		return fmt.Errorf("deleting user setting: %w", err)
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

// ListGlobal returns all global (non-project-scoped) settings for a user.
func (r *UserSettingRepository) ListGlobal(ctx context.Context, userID uuid.UUID) ([]model.UserSetting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, project_id, key, value, updated_at
		 FROM user_settings WHERE user_id = $1 AND project_id IS NULL ORDER BY key`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("querying global user settings: %w", err)
	}
	defer rows.Close()

	return scanUserSettings(rows)
}

func scanUserSetting(row *sql.Row) (*model.UserSetting, error) {
	var s model.UserSetting
	var projectID uuid.NullUUID

	err := row.Scan(&s.UserID, &projectID, &s.Key, &s.Value, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user setting: %w", err)
	}

	if projectID.Valid {
		s.ProjectID = &projectID.UUID
	}
	return &s, nil
}

func scanUserSettings(rows *sql.Rows) ([]model.UserSetting, error) {
	var settings []model.UserSetting
	for rows.Next() {
		var s model.UserSetting
		var projectID uuid.NullUUID

		if err := rows.Scan(&s.UserID, &projectID, &s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user setting row: %w", err)
		}

		if projectID.Valid {
			s.ProjectID = &projectID.UUID
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}
