package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// WorkflowRepository handles workflow persistence.
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository creates a new WorkflowRepository.
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// Create inserts a workflow with its statuses and transitions in a single transaction.
func (r *WorkflowRepository) Create(ctx context.Context, wf *model.Workflow) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO workflows (id, project_id, name, description, is_default) VALUES ($1, $2, $3, $4, $5)`,
		wf.ID, wf.ProjectID, wf.Name, wf.Description, wf.IsDefault)
	if err != nil {
		return fmt.Errorf("inserting workflow: %w", err)
	}

	for i := range wf.Statuses {
		s := &wf.Statuses[i]
		if s.ID == uuid.Nil {
			s.ID = uuid.New()
		}
		s.WorkflowID = wf.ID
		_, err = tx.ExecContext(ctx,
			`INSERT INTO workflow_statuses (id, workflow_id, name, display_name, category, position, color)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ID, s.WorkflowID, s.Name, s.DisplayName, s.Category, s.Position, s.Color)
		if err != nil {
			return fmt.Errorf("inserting workflow status %q: %w", s.Name, err)
		}
	}

	for i := range wf.Transitions {
		t := &wf.Transitions[i]
		if t.ID == uuid.Nil {
			t.ID = uuid.New()
		}
		t.WorkflowID = wf.ID
		_, err = tx.ExecContext(ctx,
			`INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
			 VALUES ($1, $2, $3, $4, $5)`,
			t.ID, t.WorkflowID, t.FromStatus, t.ToStatus, t.Name)
		if err != nil {
			return fmt.Errorf("inserting workflow transition %q->%q: %w", t.FromStatus, t.ToStatus, err)
		}
	}

	return tx.Commit()
}

// scanWorkflow scans a workflow row including project_id.
func scanWorkflow(row interface{ Scan(dest ...interface{}) error }) (*model.Workflow, error) {
	var wf model.Workflow
	var description sql.NullString
	var projectID sql.NullString

	err := row.Scan(&wf.ID, &projectID, &wf.Name, &description, &wf.IsDefault, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if description.Valid {
		wf.Description = &description.String
	}
	if projectID.Valid {
		pid, err := uuid.Parse(projectID.String)
		if err == nil {
			wf.ProjectID = &pid
		}
	}
	return &wf, nil
}

// GetByID returns a workflow by ID, including statuses and transitions.
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, is_default, created_at, updated_at
		 FROM workflows WHERE id = $1`, id)

	wf, err := scanWorkflow(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning workflow: %w", err)
	}

	statuses, err := r.listStatuses(ctx, id)
	if err != nil {
		return nil, err
	}
	wf.Statuses = statuses

	transitions, err := r.listTransitions(ctx, id)
	if err != nil {
		return nil, err
	}
	wf.Transitions = transitions

	return wf, nil
}

// List returns all system-wide workflows (project_id IS NULL).
func (r *WorkflowRepository) List(ctx context.Context) ([]model.Workflow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, is_default, created_at, updated_at
		 FROM workflows WHERE project_id IS NULL ORDER BY is_default DESC, name`)
	if err != nil {
		return nil, fmt.Errorf("querying workflows: %w", err)
	}
	defer rows.Close()

	return r.scanWorkflowRows(rows)
}

// ListByProject returns workflows available to a project (system-wide + project-specific).
func (r *WorkflowRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Workflow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, is_default, created_at, updated_at
		 FROM workflows WHERE project_id IS NULL OR project_id = $1
		 ORDER BY is_default DESC, project_id NULLS FIRST, name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying project workflows: %w", err)
	}
	defer rows.Close()

	return r.scanWorkflowRows(rows)
}

// ListProjectOnly returns only project-scoped workflows (not system-wide).
func (r *WorkflowRepository) ListProjectOnly(ctx context.Context, projectID uuid.UUID) ([]model.Workflow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, is_default, created_at, updated_at
		 FROM workflows WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying project-only workflows: %w", err)
	}
	defer rows.Close()

	return r.scanWorkflowRows(rows)
}

func (r *WorkflowRepository) scanWorkflowRows(rows *sql.Rows) ([]model.Workflow, error) {
	var workflows []model.Workflow
	for rows.Next() {
		wf, err := scanWorkflow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning workflow row: %w", err)
		}
		workflows = append(workflows, *wf)
	}
	return workflows, rows.Err()
}

// Update modifies a workflow's name and description.
func (r *WorkflowRepository) Update(ctx context.Context, wf *model.Workflow) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE workflows SET name = $1, description = $2, updated_at = now()
		 WHERE id = $3`,
		wf.Name, wf.Description, wf.ID)
	if err != nil {
		return fmt.Errorf("updating workflow: %w", err)
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

// ReplaceStatusesAndTransitions replaces all statuses and transitions for a workflow.
func (r *WorkflowRepository) ReplaceStatusesAndTransitions(ctx context.Context, wf *model.Workflow) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Update name/description
	_, err = tx.ExecContext(ctx,
		`UPDATE workflows SET name = $1, description = $2, updated_at = now() WHERE id = $3`,
		wf.Name, wf.Description, wf.ID)
	if err != nil {
		return fmt.Errorf("updating workflow: %w", err)
	}

	// Delete existing statuses and transitions (cascading via FK)
	_, err = tx.ExecContext(ctx, `DELETE FROM workflow_transitions WHERE workflow_id = $1`, wf.ID)
	if err != nil {
		return fmt.Errorf("deleting transitions: %w", err)
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM workflow_statuses WHERE workflow_id = $1`, wf.ID)
	if err != nil {
		return fmt.Errorf("deleting statuses: %w", err)
	}

	// Re-insert statuses
	for i := range wf.Statuses {
		s := &wf.Statuses[i]
		if s.ID == uuid.Nil {
			s.ID = uuid.New()
		}
		s.WorkflowID = wf.ID
		_, err = tx.ExecContext(ctx,
			`INSERT INTO workflow_statuses (id, workflow_id, name, display_name, category, position, color)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ID, s.WorkflowID, s.Name, s.DisplayName, s.Category, s.Position, s.Color)
		if err != nil {
			return fmt.Errorf("inserting workflow status %q: %w", s.Name, err)
		}
	}

	// Re-insert transitions
	for i := range wf.Transitions {
		t := &wf.Transitions[i]
		if t.ID == uuid.Nil {
			t.ID = uuid.New()
		}
		t.WorkflowID = wf.ID
		_, err = tx.ExecContext(ctx,
			`INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
			 VALUES ($1, $2, $3, $4, $5)`,
			t.ID, t.WorkflowID, t.FromStatus, t.ToStatus, t.Name)
		if err != nil {
			return fmt.Errorf("inserting workflow transition %q->%q: %w", t.FromStatus, t.ToStatus, err)
		}
	}

	return tx.Commit()
}

// Delete removes a workflow and all its statuses and transitions.
func (r *WorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM workflows WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting workflow: %w", err)
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

// GetDefaultByName returns a default workflow by name, or ErrNotFound.
func (r *WorkflowRepository) GetDefaultByName(ctx context.Context, name string) (*model.Workflow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, is_default, created_at, updated_at
		 FROM workflows WHERE name = $1 AND is_default = true`, name)

	wf, err := scanWorkflow(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning workflow: %w", err)
	}
	return wf, nil
}

// ListDefaultNames returns the names of all default/system workflows.
func (r *WorkflowRepository) ListDefaultNames(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT name FROM workflows WHERE is_default = true ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying default workflow names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning default workflow name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// ValidateTransition checks if a transition from fromStatus to toStatus is valid.
func (r *WorkflowRepository) ValidateTransition(ctx context.Context, workflowID uuid.UUID, fromStatus, toStatus string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM workflow_transitions
			WHERE workflow_id = $1 AND from_status = $2 AND to_status = $3
		)`, workflowID, fromStatus, toStatus).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("validating transition: %w", err)
	}
	return exists, nil
}

// GetInitialStatus returns the status with position 0 for a workflow.
func (r *WorkflowRepository) GetInitialStatus(ctx context.Context, workflowID uuid.UUID) (*model.WorkflowStatus, error) {
	var s model.WorkflowStatus
	var color sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, workflow_id, name, display_name, category, position, color
		 FROM workflow_statuses WHERE workflow_id = $1 ORDER BY position ASC LIMIT 1`,
		workflowID).
		Scan(&s.ID, &s.WorkflowID, &s.Name, &s.DisplayName, &s.Category, &s.Position, &color)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning initial status: %w", err)
	}
	if color.Valid {
		s.Color = &color.String
	}
	return &s, nil
}

// GetStatusCategory returns the category of a named status in a workflow.
func (r *WorkflowRepository) GetStatusCategory(ctx context.Context, workflowID uuid.UUID, statusName string) (string, error) {
	var category string
	err := r.db.QueryRowContext(ctx,
		`SELECT category FROM workflow_statuses
		 WHERE workflow_id = $1 AND name = $2`,
		workflowID, statusName).Scan(&category)
	if err == sql.ErrNoRows {
		return "", model.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("getting status category: %w", err)
	}
	return category, nil
}

// ListTransitions returns transitions for a workflow.
func (r *WorkflowRepository) ListTransitions(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowTransition, error) {
	return r.listTransitions(ctx, workflowID)
}

// ListStatuses returns statuses for a workflow.
func (r *WorkflowRepository) ListStatuses(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error) {
	return r.listStatuses(ctx, workflowID)
}

// ListAllStatuses returns all distinct statuses across all system workflows.
func (r *WorkflowRepository) ListAllStatuses(ctx context.Context) ([]model.WorkflowStatus, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ON (ws.name) ws.id, ws.workflow_id, ws.name, ws.display_name, ws.category, ws.position, ws.color
		 FROM workflow_statuses ws
		 JOIN workflows w ON ws.workflow_id = w.id
		 WHERE w.is_default = true
		 ORDER BY ws.name, ws.position`)
	if err != nil {
		return nil, fmt.Errorf("querying all statuses: %w", err)
	}
	defer rows.Close()

	var statuses []model.WorkflowStatus
	for rows.Next() {
		var s model.WorkflowStatus
		var color sql.NullString
		if err := rows.Scan(&s.ID, &s.WorkflowID, &s.Name, &s.DisplayName, &s.Category, &s.Position, &color); err != nil {
			return nil, fmt.Errorf("scanning status row: %w", err)
		}
		if color.Valid {
			s.Color = &color.String
		}
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

func (r *WorkflowRepository) listStatuses(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowStatus, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, workflow_id, name, display_name, category, position, color
		 FROM workflow_statuses WHERE workflow_id = $1 ORDER BY position`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("querying workflow statuses: %w", err)
	}
	defer rows.Close()

	var statuses []model.WorkflowStatus
	for rows.Next() {
		var s model.WorkflowStatus
		var color sql.NullString
		if err := rows.Scan(&s.ID, &s.WorkflowID, &s.Name, &s.DisplayName, &s.Category, &s.Position, &color); err != nil {
			return nil, fmt.Errorf("scanning status row: %w", err)
		}
		if color.Valid {
			s.Color = &color.String
		}
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

func (r *WorkflowRepository) listTransitions(ctx context.Context, workflowID uuid.UUID) ([]model.WorkflowTransition, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, workflow_id, from_status, to_status, name
		 FROM workflow_transitions WHERE workflow_id = $1
		 ORDER BY from_status, to_status`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("querying workflow transitions: %w", err)
	}
	defer rows.Close()

	var transitions []model.WorkflowTransition
	for rows.Next() {
		var t model.WorkflowTransition
		var name sql.NullString
		if err := rows.Scan(&t.ID, &t.WorkflowID, &t.FromStatus, &t.ToStatus, &name); err != nil {
			return nil, fmt.Errorf("scanning transition row: %w", err)
		}
		if name.Valid {
			t.Name = &name.String
		}
		transitions = append(transitions, t)
	}
	return transitions, rows.Err()
}
