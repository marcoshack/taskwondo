package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// SavedSearchRepository handles saved search persistence.
type SavedSearchRepository struct {
	db *sql.DB
}

// NewSavedSearchRepository creates a new SavedSearchRepository.
func NewSavedSearchRepository(db *sql.DB) *SavedSearchRepository {
	return &SavedSearchRepository{db: db}
}

// Create inserts a new saved search.
func (r *SavedSearchRepository) Create(ctx context.Context, s *model.SavedSearch) error {
	filtersJSON, err := json.Marshal(s.Filters)
	if err != nil {
		return fmt.Errorf("marshaling filters: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO saved_searches (id, project_id, user_id, name, filters, view_mode, position)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.ProjectID, s.UserID, s.Name, filtersJSON, s.ViewMode, s.Position)
	if err != nil {
		return fmt.Errorf("inserting saved search: %w", err)
	}
	return nil
}

// GetByID returns a saved search by ID.
func (r *SavedSearchRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SavedSearch, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, name, filters, view_mode, position, created_at, updated_at
		 FROM saved_searches WHERE id = $1`, id)
	return scanSavedSearch(row)
}

// ListByProjectAndUser returns the user's personal searches plus all shared searches for a project.
func (r *SavedSearchRepository) ListByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) ([]model.SavedSearch, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, user_id, name, filters, view_mode, position, created_at, updated_at
		 FROM saved_searches
		 WHERE project_id = $1 AND (user_id = $2 OR user_id IS NULL)
		 ORDER BY (user_id IS NULL) ASC, position ASC, created_at ASC`, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("querying saved searches: %w", err)
	}
	defer rows.Close()

	return scanSavedSearches(rows)
}

// Update modifies a saved search's mutable fields.
func (r *SavedSearchRepository) Update(ctx context.Context, s *model.SavedSearch) error {
	filtersJSON, err := json.Marshal(s.Filters)
	if err != nil {
		return fmt.Errorf("marshaling filters: %w", err)
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE saved_searches SET name = $1, filters = $2, view_mode = $3, position = $4, updated_at = now()
		 WHERE id = $5`,
		s.Name, filtersJSON, s.ViewMode, s.Position, s.ID)
	if err != nil {
		return fmt.Errorf("updating saved search: %w", err)
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

// Delete removes a saved search.
func (r *SavedSearchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM saved_searches WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting saved search: %w", err)
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

func scanSavedSearch(row *sql.Row) (*model.SavedSearch, error) {
	var s model.SavedSearch
	var userID uuid.NullUUID
	var filtersJSON []byte

	err := row.Scan(&s.ID, &s.ProjectID, &userID, &s.Name, &filtersJSON, &s.ViewMode, &s.Position, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning saved search: %w", err)
	}

	if userID.Valid {
		s.UserID = &userID.UUID
	}
	if err := json.Unmarshal(filtersJSON, &s.Filters); err != nil {
		return nil, fmt.Errorf("unmarshaling filters: %w", err)
	}

	return &s, nil
}

func scanSavedSearches(rows *sql.Rows) ([]model.SavedSearch, error) {
	var searches []model.SavedSearch
	for rows.Next() {
		var s model.SavedSearch
		var userID uuid.NullUUID
		var filtersJSON []byte

		if err := rows.Scan(&s.ID, &s.ProjectID, &userID, &s.Name, &filtersJSON, &s.ViewMode, &s.Position, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning saved search row: %w", err)
		}

		if userID.Valid {
			s.UserID = &userID.UUID
		}
		if err := json.Unmarshal(filtersJSON, &s.Filters); err != nil {
			return nil, fmt.Errorf("unmarshaling filters: %w", err)
		}

		searches = append(searches, s)
	}
	return searches, rows.Err()
}
