package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// EscalationRepository handles escalation list persistence.
type EscalationRepository struct {
	db *sql.DB
}

// NewEscalationRepository creates a new EscalationRepository.
func NewEscalationRepository(db *sql.DB) *EscalationRepository {
	return &EscalationRepository{db: db}
}

// Create inserts a new escalation list with its levels and level users in a single transaction.
func (r *EscalationRepository) Create(ctx context.Context, el *model.EscalationList) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO escalation_lists (id, project_id, name)
		 VALUES ($1, $2, $3)
		 RETURNING created_at, updated_at`,
		el.ID, el.ProjectID, el.Name).Scan(&el.CreatedAt, &el.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting escalation list: %w", err)
	}

	if err := insertLevels(ctx, tx, el); err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID returns an escalation list by ID with all levels and users.
func (r *EscalationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.EscalationList, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, created_at, updated_at
		 FROM escalation_lists WHERE id = $1`, id)

	var el model.EscalationList
	err := row.Scan(&el.ID, &el.ProjectID, &el.Name, &el.CreatedAt, &el.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning escalation list: %w", err)
	}

	levels, err := r.loadLevels(ctx, el.ID)
	if err != nil {
		return nil, err
	}
	el.Levels = levels

	return &el, nil
}

// List returns all escalation lists for a project with levels and users.
func (r *EscalationRepository) List(ctx context.Context, projectID uuid.UUID) ([]model.EscalationList, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, created_at, updated_at
		 FROM escalation_lists WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying escalation lists: %w", err)
	}
	defer rows.Close()

	var lists []model.EscalationList
	for rows.Next() {
		var el model.EscalationList
		if err := rows.Scan(&el.ID, &el.ProjectID, &el.Name, &el.CreatedAt, &el.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning escalation list row: %w", err)
		}
		lists = append(lists, el)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating escalation lists: %w", err)
	}

	// Load levels for each list
	for i := range lists {
		levels, err := r.loadLevels(ctx, lists[i].ID)
		if err != nil {
			return nil, err
		}
		lists[i].Levels = levels
	}

	return lists, nil
}

// Update replaces an escalation list's name, levels, and level users.
func (r *EscalationRepository) Update(ctx context.Context, el *model.EscalationList) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Update name
	result, err := tx.ExecContext(ctx,
		`UPDATE escalation_lists SET name = $1, updated_at = now() WHERE id = $2`,
		el.Name, el.ID)
	if err != nil {
		return fmt.Errorf("updating escalation list: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}

	// Full-replace levels: delete existing, insert new
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM escalation_levels WHERE escalation_list_id = $1`, el.ID); err != nil {
		return fmt.Errorf("deleting old levels: %w", err)
	}

	if err := insertLevels(ctx, tx, el); err != nil {
		return err
	}

	// Re-read updated_at
	if err := tx.QueryRowContext(ctx,
		`SELECT updated_at FROM escalation_lists WHERE id = $1`, el.ID).Scan(&el.UpdatedAt); err != nil {
		return fmt.Errorf("reading updated_at: %w", err)
	}

	return tx.Commit()
}

// Delete removes an escalation list (CASCADE deletes levels and level users).
func (r *EscalationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM escalation_lists WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting escalation list: %w", err)
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

// ListMappings returns all type-escalation-list mappings for a project.
func (r *EscalationRepository) ListMappings(ctx context.Context, projectID uuid.UUID) ([]model.TypeEscalationMapping, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT project_id, work_item_type, escalation_list_id
		 FROM type_escalation_lists WHERE project_id = $1 ORDER BY work_item_type`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying type escalation mappings: %w", err)
	}
	defer rows.Close()

	var mappings []model.TypeEscalationMapping
	for rows.Next() {
		var m model.TypeEscalationMapping
		if err := rows.Scan(&m.ProjectID, &m.WorkItemType, &m.EscalationListID); err != nil {
			return nil, fmt.Errorf("scanning type escalation mapping row: %w", err)
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

// UpsertMapping inserts or updates a type-escalation-list mapping.
func (r *EscalationRepository) UpsertMapping(ctx context.Context, m *model.TypeEscalationMapping) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO type_escalation_lists (project_id, work_item_type, escalation_list_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (project_id, work_item_type)
		 DO UPDATE SET escalation_list_id = EXCLUDED.escalation_list_id`,
		m.ProjectID, m.WorkItemType, m.EscalationListID)
	if err != nil {
		return fmt.Errorf("upserting type escalation mapping: %w", err)
	}
	return nil
}

// DeleteMapping removes a type-escalation-list mapping.
func (r *EscalationRepository) DeleteMapping(ctx context.Context, projectID uuid.UUID, workItemType string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM type_escalation_lists WHERE project_id = $1 AND work_item_type = $2`,
		projectID, workItemType)
	if err != nil {
		return fmt.Errorf("deleting type escalation mapping: %w", err)
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

// loadLevels returns all levels (with users) for an escalation list.
func (r *EscalationRepository) loadLevels(ctx context.Context, listID uuid.UUID) ([]model.EscalationLevel, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT el.id, el.escalation_list_id, el.threshold_pct, el.position, el.created_at
		 FROM escalation_levels el
		 WHERE el.escalation_list_id = $1
		 ORDER BY el.position, el.threshold_pct`, listID)
	if err != nil {
		return nil, fmt.Errorf("querying escalation levels: %w", err)
	}
	defer rows.Close()

	var levels []model.EscalationLevel
	for rows.Next() {
		var lv model.EscalationLevel
		if err := rows.Scan(&lv.ID, &lv.ListID, &lv.ThresholdPct, &lv.Position, &lv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning escalation level row: %w", err)
		}
		levels = append(levels, lv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating escalation levels: %w", err)
	}

	// Load users for each level
	for i := range levels {
		users, err := r.loadLevelUsers(ctx, levels[i].ID)
		if err != nil {
			return nil, err
		}
		levels[i].Users = users
	}

	return levels, nil
}

// loadLevelUsers returns all users for an escalation level, joining with users table for display info.
func (r *EscalationRepository) loadLevelUsers(ctx context.Context, levelID uuid.UUID) ([]model.EscalationLevelUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.display_name, u.email
		 FROM escalation_level_users elu
		 JOIN users u ON u.id = elu.user_id
		 WHERE elu.escalation_level_id = $1
		 ORDER BY u.display_name`, levelID)
	if err != nil {
		return nil, fmt.Errorf("querying escalation level users: %w", err)
	}
	defer rows.Close()

	var users []model.EscalationLevelUser
	for rows.Next() {
		var u model.EscalationLevelUser
		if err := rows.Scan(&u.UserID, &u.DisplayName, &u.Email); err != nil {
			return nil, fmt.Errorf("scanning escalation level user row: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// insertLevels inserts levels and their users into the database within a transaction.
func insertLevels(ctx context.Context, tx *sql.Tx, el *model.EscalationList) error {
	for i := range el.Levels {
		lv := &el.Levels[i]
		lv.ListID = el.ID
		lv.Position = i

		err := tx.QueryRowContext(ctx,
			`INSERT INTO escalation_levels (id, escalation_list_id, threshold_pct, position)
			 VALUES ($1, $2, $3, $4)
			 RETURNING created_at`,
			lv.ID, lv.ListID, lv.ThresholdPct, lv.Position).Scan(&lv.CreatedAt)
		if err != nil {
			return fmt.Errorf("inserting escalation level: %w", err)
		}

		for _, u := range lv.Users {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO escalation_level_users (escalation_level_id, user_id)
				 VALUES ($1, $2)`,
				lv.ID, u.UserID); err != nil {
				return fmt.Errorf("inserting escalation level user: %w", err)
			}
		}
	}
	return nil
}
