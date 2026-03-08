package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectRepository handles project persistence.
type ProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository creates a new ProjectRepository.
// ListAllIDs returns all non-deleted project IDs with pagination (for backfill).
func (r *ProjectRepository) ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error) {
	return listAllIDs(ctx, r.db, "projects", limit, offset)
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a new project.
func (r *ProjectRepository) Create(ctx context.Context, project *model.Project) error {
	businessHoursJSON, err := json.Marshal(project.BusinessHours)
	if err != nil {
		return fmt.Errorf("marshaling business hours: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO projects (id, name, key, description, default_workflow_id, allowed_complexity_values, business_hours, namespace_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		project.ID, project.Name, project.Key, project.Description, project.DefaultWorkflowID, pq.Array(project.AllowedComplexityValues), businessHoursJSON, project.NamespaceID)
	if err != nil {
		return fmt.Errorf("inserting project: %w", err)
	}
	return nil
}

// GetByKey returns a project by its unique key (e.g., "INFRA").
// When namespace context is present, scopes the lookup to that namespace.
// Otherwise falls back to a global search for backward compatibility.
func (r *ProjectRepository) GetByKey(ctx context.Context, key string) (*model.Project, error) {
	namespaceID := model.NamespaceIDFromContext(ctx)
	if namespaceID != uuid.Nil {
		return r.GetByKeyAndNamespace(ctx, namespaceID, key)
	}
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, key, description, namespace_id, default_workflow_id, allowed_complexity_values, business_hours, item_counter, created_at, updated_at
		 FROM projects WHERE key = $1 AND deleted_at IS NULL`, key)
	return scanProject(row)
}

// GetByKeyAndNamespace returns a project by its key scoped to a specific namespace.
func (r *ProjectRepository) GetByKeyAndNamespace(ctx context.Context, namespaceID uuid.UUID, key string) (*model.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, key, description, namespace_id, default_workflow_id, allowed_complexity_values, business_hours, item_counter, created_at, updated_at
		 FROM projects WHERE namespace_id = $1 AND key = $2 AND deleted_at IS NULL`, namespaceID, key)
	return scanProject(row)
}

// GetByID returns a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, key, description, namespace_id, default_workflow_id, allowed_complexity_values, business_hours, item_counter, created_at, updated_at
		 FROM projects WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanProject(row)
}

// SetNamespaceID updates a project's namespace_id. Used for backfill and migration.
func (r *ProjectRepository) SetNamespaceID(ctx context.Context, projectID, namespaceID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET namespace_id = $1, updated_at = now()
		 WHERE id = $2 AND deleted_at IS NULL`,
		namespaceID, projectID)
	if err != nil {
		return fmt.Errorf("setting project namespace: %w", err)
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

// ListByUser returns all non-deleted projects the given user is a member of.
func (r *ProjectRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id, p.name, p.key, p.description, p.namespace_id, p.default_workflow_id, p.allowed_complexity_values, p.business_hours, p.item_counter, p.created_at, p.updated_at
		 FROM projects p
		 INNER JOIN project_members pm ON pm.project_id = p.id
		 WHERE pm.user_id = $1 AND p.deleted_at IS NULL
		 ORDER BY p.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

// ListAll returns all non-deleted projects (for global admins).
func (r *ProjectRepository) ListAll(ctx context.Context) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, key, description, namespace_id, default_workflow_id, allowed_complexity_values, business_hours, item_counter, created_at, updated_at
		 FROM projects WHERE deleted_at IS NULL
		 ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying all projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

// Update modifies a project's mutable fields.
func (r *ProjectRepository) Update(ctx context.Context, project *model.Project) error {
	businessHoursJSON, err := json.Marshal(project.BusinessHours)
	if err != nil {
		return fmt.Errorf("marshaling business hours: %w", err)
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name = $1, key = $2, description = $3, default_workflow_id = $4, allowed_complexity_values = $5, business_hours = $6, updated_at = now()
		 WHERE id = $7 AND deleted_at IS NULL`,
		project.Name, project.Key, project.Description, project.DefaultWorkflowID, pq.Array(project.AllowedComplexityValues), businessHoursJSON, project.ID)
	if err != nil {
		return fmt.Errorf("updating project: %w", err)
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

// Delete soft-deletes a project by setting deleted_at.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
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

// CountByOwner returns the number of non-deleted projects where the given user is the owner.
func (r *ProjectRepository) CountByOwner(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM projects p
		 INNER JOIN project_members pm ON pm.project_id = p.id
		 WHERE pm.user_id = $1 AND pm.role = 'owner' AND p.deleted_at IS NULL`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting projects by user: %w", err)
	}
	return count, nil
}

// GetSummaries returns aggregate counts (members, open, in_progress) for the given project IDs.
// Open counts items whose status belongs to the "todo" category; in_progress counts the "in_progress" category.
func (r *ProjectRepository) GetSummaries(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectSummary, error) {
	if len(projectIDs) == 0 {
		return map[uuid.UUID]model.ProjectSummary{}, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT
			p.id,
			COALESCE(pm_counts.member_count, 0),
			COALESCE(wi_counts.open_count, 0),
			COALESCE(wi_counts.in_progress_count, 0)
		 FROM unnest($1::uuid[]) AS p(id)
		 LEFT JOIN LATERAL (
			SELECT COUNT(*) AS member_count
			FROM project_members WHERE project_id = p.id
		 ) pm_counts ON true
		 LEFT JOIN LATERAL (
			SELECT
				COUNT(*) FILTER (WHERE wi.status IN (SELECT DISTINCT ws.name FROM workflow_statuses ws WHERE ws.category = 'todo')) AS open_count,
				COUNT(*) FILTER (WHERE wi.status IN (SELECT DISTINCT ws.name FROM workflow_statuses ws WHERE ws.category = 'in_progress')) AS in_progress_count
			FROM work_items wi WHERE wi.project_id = p.id AND wi.deleted_at IS NULL
		 ) wi_counts ON true`,
		uuidArray(projectIDs))
	if err != nil {
		return nil, fmt.Errorf("querying project summaries: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]model.ProjectSummary, len(projectIDs))
	for rows.Next() {
		var id uuid.UUID
		var s model.ProjectSummary
		if err := rows.Scan(&id, &s.MemberCount, &s.OpenCount, &s.InProgressCount); err != nil {
			return nil, fmt.Errorf("scanning project summary: %w", err)
		}
		result[id] = s
	}
	return result, rows.Err()
}

// uuidArray converts a slice of UUIDs to a PostgreSQL-compatible array literal.
func uuidArray(ids []uuid.UUID) string {
	s := "{"
	for i, id := range ids {
		if i > 0 {
			s += ","
		}
		s += id.String()
	}
	s += "}"
	return s
}

func scanProject(row *sql.Row) (*model.Project, error) {
	var p model.Project
	var description sql.NullString
	var namespaceID *uuid.UUID
	var workflowID *uuid.UUID
	var allowedComplexity pq.Int64Array
	var businessHoursRaw []byte

	err := row.Scan(&p.ID, &p.Name, &p.Key, &description,
		&namespaceID, &workflowID, &allowedComplexity, &businessHoursRaw, &p.ItemCounter, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning project: %w", err)
	}

	if description.Valid {
		p.Description = &description.String
	}
	p.NamespaceID = namespaceID
	p.DefaultWorkflowID = workflowID
	p.AllowedComplexityValues = int64ArrayToIntSlice(allowedComplexity)
	if len(businessHoursRaw) > 0 && string(businessHoursRaw) != "null" {
		var bh model.BusinessHoursConfig
		if err := json.Unmarshal(businessHoursRaw, &bh); err == nil {
			p.BusinessHours = &bh
		}
	}

	return &p, nil
}

func scanProjects(rows *sql.Rows) ([]model.Project, error) {
	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var description sql.NullString
		var namespaceID *uuid.UUID
		var workflowID *uuid.UUID
		var allowedComplexity pq.Int64Array
		var businessHoursRaw []byte

		if err := rows.Scan(&p.ID, &p.Name, &p.Key, &description,
			&namespaceID, &workflowID, &allowedComplexity, &businessHoursRaw, &p.ItemCounter, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning project row: %w", err)
		}

		if description.Valid {
			p.Description = &description.String
		}
		p.NamespaceID = namespaceID
		p.DefaultWorkflowID = workflowID
		p.AllowedComplexityValues = int64ArrayToIntSlice(allowedComplexity)
		if len(businessHoursRaw) > 0 && string(businessHoursRaw) != "null" {
			var bh model.BusinessHoursConfig
			if err := json.Unmarshal(businessHoursRaw, &bh); err == nil {
				p.BusinessHours = &bh
			}
		}

		projects = append(projects, p)
	}

	return projects, rows.Err()
}

func int64ArrayToIntSlice(arr pq.Int64Array) []int {
	if arr == nil {
		return []int{}
	}
	result := make([]int, len(arr))
	for i, v := range arr {
		result[i] = int(v)
	}
	return result
}
