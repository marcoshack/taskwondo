package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/marcoshack/taskwondo/internal/model"
)

// WorkItemRepository handles work item persistence.
type WorkItemRepository struct {
	db *sql.DB
}

// NewWorkItemRepository creates a new WorkItemRepository.
func NewWorkItemRepository(db *sql.DB) *WorkItemRepository {
	return &WorkItemRepository{db: db}
}

// Create inserts a new work item, assigning the next sequential item_number
// within a transaction.
func (r *WorkItemRepository) Create(ctx context.Context, item *model.WorkItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Atomically increment the project's item counter and fetch the project key.
	var itemNumber int
	var projectKey string
	err = tx.QueryRowContext(ctx,
		`UPDATE projects SET item_counter = item_counter + 1 WHERE id = $1 RETURNING item_counter, key`,
		item.ProjectID).Scan(&itemNumber, &projectKey)
	if err != nil {
		return fmt.Errorf("incrementing item counter: %w", err)
	}
	item.ItemNumber = itemNumber
	item.DisplayID = fmt.Sprintf("%s-%d", projectKey, itemNumber)

	customFieldsJSON, err := json.Marshal(item.CustomFields)
	if err != nil {
		return fmt.Errorf("marshaling custom fields: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO work_items (
			id, project_id, queue_id, milestone_id, parent_id, item_number, display_id, type, title, description,
			status, priority, assignee_id, reporter_id, portal_contact_id, visibility,
			labels, complexity, custom_fields, due_date, sla_target_at, estimated_seconds
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`,
		item.ID, item.ProjectID, item.QueueID, item.MilestoneID, item.ParentID, item.ItemNumber, item.DisplayID,
		item.Type, item.Title, item.Description, item.Status, item.Priority,
		item.AssigneeID, item.ReporterID, item.PortalContactID, item.Visibility,
		pq.Array(item.Labels), item.Complexity, customFieldsJSON, item.DueDate, item.SLATargetAt, item.EstimatedSeconds)
	if err != nil {
		return fmt.Errorf("inserting work item: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetByProjectAndNumber returns a work item by project ID and item number.
func (r *WorkItemRepository) GetByProjectAndNumber(ctx context.Context, projectID uuid.UUID, itemNumber int) (*model.WorkItem, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, queue_id, milestone_id, parent_id, item_number, display_id, type, title, description,
		        status, priority, assignee_id, reporter_id, portal_contact_id, visibility,
		        labels, complexity, custom_fields, due_date, resolved_at, sla_target_at, estimated_seconds,
		        created_at, updated_at
		 FROM work_items
		 WHERE project_id = $1 AND item_number = $2 AND deleted_at IS NULL`,
		projectID, itemNumber)
	return scanWorkItem(row)
}

// List returns work items matching the given filter with cursor-based pagination.
func (r *WorkItemRepository) List(ctx context.Context, projectID uuid.UUID, filter *model.WorkItemFilter) (*model.WorkItemList, error) {
	qb := &queryBuilder{argIndex: 0}

	// Base condition
	qb.add("project_id = ?", projectID)
	qb.add("deleted_at IS NULL")

	// Type filter
	if len(filter.Types) > 0 {
		qb.add("type = ANY(?)", pq.Array(filter.Types))
	}

	// Status filter
	if len(filter.Statuses) > 0 {
		qb.add("status = ANY(?)", pq.Array(filter.Statuses))
	}

	// Priority filter
	if len(filter.Priorities) > 0 {
		qb.add("priority = ANY(?)", pq.Array(filter.Priorities))
	}

	// Assignee filter — supports combinations via OR
	// AssigneeMe is resolved to AssigneeIDs in the service layer.
	{
		var clauses []string
		if filter.Unassigned {
			clauses = append(clauses, "assignee_id IS NULL")
		}
		if filter.AssigneeID != nil {
			// deprecated single-value
			qb.argIndex++
			clauses = append(clauses, fmt.Sprintf("assignee_id = $%d", qb.argIndex))
			qb.args = append(qb.args, *filter.AssigneeID)
		}
		if len(filter.AssigneeIDs) > 0 {
			qb.argIndex++
			clauses = append(clauses, fmt.Sprintf("assignee_id = ANY($%d)", qb.argIndex))
			qb.args = append(qb.args, pq.Array(filter.AssigneeIDs))
		}
		if len(clauses) == 1 {
			qb.conditions = append(qb.conditions, clauses[0])
		} else if len(clauses) > 1 {
			qb.conditions = append(qb.conditions, "("+strings.Join(clauses, " OR ")+")")
		}
	}

	// Queue filter
	if filter.QueueID != nil {
		qb.add("queue_id = ?", *filter.QueueID)
	}

	// Milestone filter
	if filter.MilestoneNone {
		qb.addRaw("milestone_id IS NULL")
	} else if len(filter.MilestoneIDs) > 0 {
		qb.add("milestone_id = ANY(?)", pq.Array(filter.MilestoneIDs))
	}

	// Labels filter (items must contain ALL specified labels)
	if len(filter.Labels) > 0 {
		qb.add("labels @> ?", pq.Array(filter.Labels))
	}

	// Parent filter
	if filter.ParentNone {
		qb.addRaw("parent_id IS NULL")
	} else if filter.ParentID != nil {
		qb.add("parent_id = ?", *filter.ParentID)
	}

	// Full-text search (OR simple config to match display_id tokens like "TF-29")
	if filter.Search != "" {
		qb.add("(search_vector @@ plainto_tsquery('english', ?) OR search_vector @@ plainto_tsquery('simple', ?))", filter.Search, filter.Search)
	}

	whereClause := "WHERE " + strings.Join(qb.conditions, " AND ")

	// Count total (without cursor/limit)
	countQuery := "SELECT COUNT(*) FROM work_items " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, qb.args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("counting work items: %w", err)
	}

	// Determine sort column and order
	sortCol := "created_at"
	switch filter.Sort {
	case "updated_at", "due_date", "item_number", "type", "title", "status":
		sortCol = filter.Sort
	case "priority":
		// Use CASE expression for semantic ordering: critical(1) > high(2) > medium(3) > low(4)
		sortCol = "CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END"
	case "sla_target_at":
		sortCol = filter.Sort // COALESCE applied below
	}
	sortOrder := "DESC"
	if filter.Order == "asc" {
		sortOrder = "ASC"
	}

	// Push NULL sla_target_at values to the end regardless of sort direction
	if sortCol == "sla_target_at" {
		if sortOrder == "ASC" {
			sortCol = "COALESCE(sla_target_at, 'infinity'::timestamptz)"
		} else {
			sortCol = "COALESCE(sla_target_at, '-infinity'::timestamptz)"
		}
	}

	// Cursor pagination: fetch the cursor item's sort column value for tuple comparison.
	// sortCol is already sanitized by the switch above, so this Sprintf is safe.
	if filter.Cursor != nil {
		var cursorVal interface{}
		err := r.db.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT %s FROM work_items WHERE id = $1`, sortCol), *filter.Cursor).Scan(&cursorVal)
		if err == nil && cursorVal != nil {
			if sortOrder == "DESC" {
				qb.add("("+sortCol+", id) < (?, ?)", cursorVal, *filter.Cursor)
			} else {
				qb.add("("+sortCol+", id) > (?, ?)", cursorVal, *filter.Cursor)
			}
			// Rebuild WHERE clause with cursor condition
			whereClause = "WHERE " + strings.Join(qb.conditions, " AND ")
		}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	selectQuery := fmt.Sprintf(
		`SELECT id, project_id, queue_id, milestone_id, parent_id, item_number, display_id, type, title, description,
		        status, priority, assignee_id, reporter_id, portal_contact_id, visibility,
		        labels, complexity, custom_fields, due_date, resolved_at, sla_target_at, estimated_seconds,
		        created_at, updated_at
		 FROM work_items %s
		 ORDER BY %s %s, id %s
		 LIMIT %d`,
		whereClause, sortCol, sortOrder, sortOrder, limit+1)

	rows, err := r.db.QueryContext(ctx, selectQuery, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("querying work items: %w", err)
	}
	defer rows.Close()

	items, err := scanWorkItems(rows)
	if err != nil {
		return nil, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var cursor string
	if len(items) > 0 {
		cursor = items[len(items)-1].ID.String()
	}

	return &model.WorkItemList{
		Items:   items,
		Cursor:  cursor,
		HasMore: hasMore,
		Total:   total,
	}, nil
}

// Update modifies a work item's mutable fields.
func (r *WorkItemRepository) Update(ctx context.Context, item *model.WorkItem) error {
	customFieldsJSON, err := json.Marshal(item.CustomFields)
	if err != nil {
		return fmt.Errorf("marshaling custom fields: %w", err)
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE work_items SET
			title = $1, description = $2, status = $3, priority = $4,
			assignee_id = $5, visibility = $6, labels = $7, complexity = $8, custom_fields = $9,
			due_date = $10, type = $11, parent_id = $12,
			queue_id = $13, milestone_id = $14, portal_contact_id = $15, resolved_at = $16,
			sla_target_at = $17, estimated_seconds = $18, updated_at = now()
		 WHERE id = $19 AND deleted_at IS NULL`,
		item.Title, item.Description, item.Status, item.Priority,
		item.AssigneeID, item.Visibility, pq.Array(item.Labels), item.Complexity, customFieldsJSON,
		item.DueDate, item.Type, item.ParentID,
		item.QueueID, item.MilestoneID, item.PortalContactID, item.ResolvedAt,
		item.SLATargetAt, item.EstimatedSeconds, item.ID)
	if err != nil {
		return fmt.Errorf("updating work item: %w", err)
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

// Delete soft-deletes a work item.
func (r *WorkItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE work_items SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting work item: %w", err)
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

// --- Query builder ---

type queryBuilder struct {
	conditions []string
	args       []interface{}
	argIndex   int
}

// add appends a condition with parameters. Each ? is replaced with $N.
func (qb *queryBuilder) add(condition string, args ...interface{}) {
	for _, arg := range args {
		qb.argIndex++
		condition = strings.Replace(condition, "?", fmt.Sprintf("$%d", qb.argIndex), 1)
		qb.args = append(qb.args, arg)
	}
	qb.conditions = append(qb.conditions, condition)
}

// addRaw appends a condition with no parameters.
func (qb *queryBuilder) addRaw(condition string) {
	qb.conditions = append(qb.conditions, condition)
}

// --- Scan helpers ---

func scanWorkItem(row *sql.Row) (*model.WorkItem, error) {
	var item model.WorkItem
	var (
		description      sql.NullString
		queueID          uuid.NullUUID
		milestoneID      uuid.NullUUID
		parentID         uuid.NullUUID
		assigneeID       uuid.NullUUID
		portalContactID  uuid.NullUUID
		complexity       sql.NullInt64
		dueDate          sql.NullTime
		resolvedAt       sql.NullTime
		slaTargetAt      sql.NullTime
		estimatedSeconds sql.NullInt64
		labels           pq.StringArray
		customFieldsRaw  []byte
	)

	err := row.Scan(
		&item.ID, &item.ProjectID, &queueID, &milestoneID, &parentID, &item.ItemNumber, &item.DisplayID,
		&item.Type, &item.Title, &description, &item.Status, &item.Priority,
		&assigneeID, &item.ReporterID, &portalContactID, &item.Visibility,
		&labels, &complexity, &customFieldsRaw, &dueDate, &resolvedAt, &slaTargetAt, &estimatedSeconds,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning work item: %w", err)
	}

	populateWorkItem(&item, description, queueID, milestoneID, parentID, assigneeID,
		portalContactID, complexity, dueDate, resolvedAt, slaTargetAt, estimatedSeconds, labels, customFieldsRaw)

	return &item, nil
}

func scanWorkItems(rows *sql.Rows) ([]model.WorkItem, error) {
	var items []model.WorkItem
	for rows.Next() {
		var item model.WorkItem
		var (
			description      sql.NullString
			queueID          uuid.NullUUID
			milestoneID      uuid.NullUUID
			parentID         uuid.NullUUID
			assigneeID       uuid.NullUUID
			portalContactID  uuid.NullUUID
			complexity       sql.NullInt64
			dueDate          sql.NullTime
			resolvedAt       sql.NullTime
			slaTargetAt      sql.NullTime
			estimatedSeconds sql.NullInt64
			labels           pq.StringArray
			customFieldsRaw  []byte
		)

		if err := rows.Scan(
			&item.ID, &item.ProjectID, &queueID, &milestoneID, &parentID, &item.ItemNumber, &item.DisplayID,
			&item.Type, &item.Title, &description, &item.Status, &item.Priority,
			&assigneeID, &item.ReporterID, &portalContactID, &item.Visibility,
			&labels, &complexity, &customFieldsRaw, &dueDate, &resolvedAt, &slaTargetAt, &estimatedSeconds,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning work item row: %w", err)
		}

		populateWorkItem(&item, description, queueID, milestoneID, parentID, assigneeID,
			portalContactID, complexity, dueDate, resolvedAt, slaTargetAt, estimatedSeconds, labels, customFieldsRaw)

		items = append(items, item)
	}

	return items, rows.Err()
}

func populateWorkItem(
	item *model.WorkItem,
	description sql.NullString,
	queueID, milestoneID, parentID, assigneeID, portalContactID uuid.NullUUID,
	complexity sql.NullInt64,
	dueDate, resolvedAt, slaTargetAt sql.NullTime,
	estimatedSeconds sql.NullInt64,
	labels pq.StringArray,
	customFieldsRaw []byte,
) {
	if description.Valid {
		item.Description = &description.String
	}
	if queueID.Valid {
		item.QueueID = &queueID.UUID
	}
	if milestoneID.Valid {
		item.MilestoneID = &milestoneID.UUID
	}
	if parentID.Valid {
		item.ParentID = &parentID.UUID
	}
	if assigneeID.Valid {
		item.AssigneeID = &assigneeID.UUID
	}
	if portalContactID.Valid {
		item.PortalContactID = &portalContactID.UUID
	}
	if complexity.Valid {
		v := int(complexity.Int64)
		item.Complexity = &v
	}
	if dueDate.Valid {
		item.DueDate = &dueDate.Time
	}
	if resolvedAt.Valid {
		item.ResolvedAt = &resolvedAt.Time
	}
	if slaTargetAt.Valid {
		item.SLATargetAt = &slaTargetAt.Time
	}
	if estimatedSeconds.Valid {
		v := int(estimatedSeconds.Int64)
		item.EstimatedSeconds = &v
	}

	item.Labels = []string(labels)
	if item.Labels == nil {
		item.Labels = []string{}
	}

	item.CustomFields = make(map[string]interface{})
	if len(customFieldsRaw) > 0 {
		json.Unmarshal(customFieldsRaw, &item.CustomFields)
	}
}

// UpdateSLATargetAt updates only the sla_target_at column for a work item.
func (r *WorkItemRepository) UpdateSLATargetAt(ctx context.Context, id uuid.UUID, slaTargetAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE work_items SET sla_target_at = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`,
		slaTargetAt, id)
	return err
}
