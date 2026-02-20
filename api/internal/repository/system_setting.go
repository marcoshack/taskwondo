package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/marcoshack/taskwondo/internal/model"
)

// SystemSettingRepository handles system setting persistence.
type SystemSettingRepository struct {
	db *sql.DB
}

// NewSystemSettingRepository creates a new SystemSettingRepository.
func NewSystemSettingRepository(db *sql.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// Upsert creates or updates a system setting.
func (r *SystemSettingRepository) Upsert(ctx context.Context, s *model.SystemSetting) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO system_settings (key, value, updated_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (key)
		 DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		s.Key, s.Value)
	if err != nil {
		return fmt.Errorf("upserting system setting: %w", err)
	}
	return nil
}

// Get returns a single system setting by key.
func (r *SystemSettingRepository) Get(ctx context.Context, key string) (*model.SystemSetting, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT key, value, updated_at FROM system_settings WHERE key = $1`, key)

	var s model.SystemSetting
	err := row.Scan(&s.Key, &s.Value, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning system setting: %w", err)
	}
	return &s, nil
}

// List returns all system settings.
func (r *SystemSettingRepository) List(ctx context.Context) ([]model.SystemSetting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, value, updated_at FROM system_settings ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("querying system settings: %w", err)
	}
	defer rows.Close()

	var settings []model.SystemSetting
	for rows.Next() {
		var s model.SystemSetting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning system setting row: %w", err)
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

// Delete removes a system setting by key.
func (r *SystemSettingRepository) Delete(ctx context.Context, key string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM system_settings WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("deleting system setting: %w", err)
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
