package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// MilestoneRepository handles milestone persistence.
type MilestoneRepository struct {
	db *sql.DB
}

// NewMilestoneRepository creates a new MilestoneRepository.
func NewMilestoneRepository(db *sql.DB) *MilestoneRepository {
	return &MilestoneRepository{db: db}
}

// Create inserts a new milestone.
func (r *MilestoneRepository) Create(ctx context.Context, m *model.Milestone) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO milestones (id, project_id, name, description, due_date, status)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		m.ID, m.ProjectID, m.Name, m.Description, m.DueDate, m.Status)
	if err != nil {
		return fmt.Errorf("inserting milestone: %w", err)
	}
	return nil
}

// GetByID returns a milestone by ID.
func (r *MilestoneRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Milestone, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, due_date, status, created_at, updated_at
		 FROM milestones WHERE id = $1`, id)
	return scanMilestone(row)
}

// GetByIDWithProgress returns a milestone with work item counts.
func (r *MilestoneRepository) GetByIDWithProgress(ctx context.Context, id uuid.UUID) (*model.MilestoneWithProgress, error) {
	m, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.attachProgress(ctx, m)
}

// List returns all milestones for a project.
func (r *MilestoneRepository) List(ctx context.Context, projectID uuid.UUID) ([]model.Milestone, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, due_date, status, created_at, updated_at
		 FROM milestones WHERE project_id = $1 ORDER BY due_date NULLS LAST, name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying milestones: %w", err)
	}
	defer rows.Close()

	return scanMilestones(rows)
}

// ListWithProgress returns all milestones for a project with work item counts.
func (r *MilestoneRepository) ListWithProgress(ctx context.Context, projectID uuid.UUID) ([]model.MilestoneWithProgress, error) {
	milestones, err := r.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]model.MilestoneWithProgress, 0, len(milestones))
	for i := range milestones {
		mp, err := r.attachProgress(ctx, &milestones[i])
		if err != nil {
			return nil, err
		}
		result = append(result, *mp)
	}
	return result, nil
}

// Update modifies a milestone's mutable fields.
func (r *MilestoneRepository) Update(ctx context.Context, m *model.Milestone) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE milestones SET name = $1, description = $2, due_date = $3, status = $4, updated_at = now()
		 WHERE id = $5`,
		m.Name, m.Description, m.DueDate, m.Status, m.ID)
	if err != nil {
		return fmt.Errorf("updating milestone: %w", err)
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

// Delete removes a milestone.
func (r *MilestoneRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM milestones WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting milestone: %w", err)
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

func (r *MilestoneRepository) attachProgress(ctx context.Context, m *model.Milestone) (*model.MilestoneWithProgress, error) {
	mp := &model.MilestoneWithProgress{Milestone: *m}

	err := r.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE ws.category IS NULL OR ws.category NOT IN ('done', 'cancelled')) AS open_count,
			COUNT(*) FILTER (WHERE ws.category IN ('done', 'cancelled')) AS closed_count,
			COUNT(*) AS total_count,
			COALESCE(SUM(wi.estimated_seconds), 0) AS total_estimated_seconds,
			COALESCE((
				SELECT SUM(te.duration_seconds)
				FROM time_entries te
				WHERE te.work_item_id IN (
					SELECT wi2.id FROM work_items wi2
					WHERE wi2.milestone_id = $1 AND wi2.deleted_at IS NULL
				) AND te.deleted_at IS NULL
			), 0) AS total_spent_seconds
		 FROM work_items wi
		 LEFT JOIN LATERAL (
			SELECT ws2.category FROM workflow_statuses ws2
			WHERE ws2.name = wi.status
			  AND ws2.workflow_id = COALESCE(
				(SELECT ptw.workflow_id FROM project_type_workflows ptw
				 WHERE ptw.project_id = wi.project_id AND ptw.work_item_type = wi.type),
				(SELECT p.default_workflow_id FROM projects p WHERE p.id = wi.project_id)
			  )
			LIMIT 1
		 ) ws ON true
		 WHERE wi.milestone_id = $1 AND wi.deleted_at IS NULL`, m.ID).
		Scan(&mp.OpenCount, &mp.ClosedCount, &mp.TotalCount, &mp.TotalEstimatedSeconds, &mp.TotalSpentSeconds)
	if err != nil {
		return nil, fmt.Errorf("counting milestone progress: %w", err)
	}

	return mp, nil
}

func scanMilestone(row *sql.Row) (*model.Milestone, error) {
	var m model.Milestone
	var description sql.NullString
	var dueDate sql.NullTime

	err := row.Scan(&m.ID, &m.ProjectID, &m.Name, &description, &dueDate, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning milestone: %w", err)
	}

	if description.Valid {
		m.Description = &description.String
	}
	if dueDate.Valid {
		m.DueDate = &dueDate.Time
	}

	return &m, nil
}

func scanMilestones(rows *sql.Rows) ([]model.Milestone, error) {
	var milestones []model.Milestone
	for rows.Next() {
		var m model.Milestone
		var description sql.NullString
		var dueDate sql.NullTime

		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Name, &description, &dueDate, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning milestone row: %w", err)
		}

		if description.Valid {
			m.Description = &description.String
		}
		if dueDate.Valid {
			m.DueDate = &dueDate.Time
		}

		milestones = append(milestones, m)
	}
	return milestones, rows.Err()
}
