package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// InboxRepository handles inbox item persistence.
type InboxRepository struct {
	db *sql.DB
}

// NewInboxRepository creates a new InboxRepository.
func NewInboxRepository(db *sql.DB) *InboxRepository {
	return &InboxRepository{db: db}
}

// Add inserts a new inbox item at the end of the user's inbox.
func (r *InboxRepository) Add(ctx context.Context, item *model.InboxItem) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO inbox_items (id, user_id, work_item_id, position)
		 VALUES ($1, $2, $3, $4)`,
		item.ID, item.UserID, item.WorkItemID, item.Position)
	if err != nil {
		return fmt.Errorf("inserting inbox item: %w", err)
	}
	return nil
}

// Remove deletes an inbox item by user and work item ID.
func (r *InboxRepository) Remove(ctx context.Context, userID, workItemID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM inbox_items WHERE user_id = $1 AND work_item_id = $2`,
		userID, workItemID)
	if err != nil {
		return fmt.Errorf("deleting inbox item: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// RemoveByID deletes an inbox item by its own ID, scoped to the user.
func (r *InboxRepository) RemoveByID(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM inbox_items WHERE id = $1 AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("deleting inbox item: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// List returns the user's inbox items with joined work item data.
// When excludeCompleted is true, items whose status category is 'done' or 'cancelled' are filtered out.
func (r *InboxRepository) List(ctx context.Context, userID uuid.UUID, excludeCompleted bool, search string, projectKeys []string, cursor *uuid.UUID, limit int) (*model.InboxItemList, error) {
	qb := &queryBuilder{argIndex: 0}
	qb.add("i.user_id = ?", userID)
	qb.addRaw("wi.deleted_at IS NULL")

	if excludeCompleted {
		qb.addRaw("COALESCE(ws.category, '') NOT IN ('done', 'cancelled')")
	}

	if search != "" {
		qb.add("(wi.search_vector @@ plainto_tsquery('english', ?) OR wi.search_vector @@ plainto_tsquery('simple', ?))", search, search)
	}

	if len(projectKeys) > 0 {
		placeholders := make([]string, len(projectKeys))
		for i, key := range projectKeys {
			qb.argIndex++
			placeholders[i] = fmt.Sprintf("$%d", qb.argIndex)
			qb.args = append(qb.args, key)
		}
		qb.conditions = append(qb.conditions, fmt.Sprintf("p.key IN (%s)", strings.Join(placeholders, ", ")))
	}

	whereClause := "WHERE " + strings.Join(qb.conditions, " AND ")

	joinClause := `FROM inbox_items i
		 JOIN work_items wi ON i.work_item_id = wi.id
		 JOIN projects p ON wi.project_id = p.id
		 LEFT JOIN users u ON wi.assignee_id = u.id
		 LEFT JOIN project_type_workflows ptw ON ptw.project_id = p.id AND ptw.work_item_type = wi.type
		 LEFT JOIN workflow_statuses ws ON ws.workflow_id = COALESCE(ptw.workflow_id, p.default_workflow_id) AND ws.name = wi.status`

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) %s %s", joinClause, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, qb.args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("counting inbox items: %w", err)
	}

	// Cursor pagination (ordered by position ASC)
	if cursor != nil {
		var cursorPos int
		err := r.db.QueryRowContext(ctx,
			`SELECT position FROM inbox_items WHERE id = $1 AND user_id = $2`, *cursor, userID).Scan(&cursorPos)
		if err == nil {
			qb.add("(i.position, i.id) > (?, ?)", cursorPos, *cursor)
			whereClause = "WHERE " + strings.Join(qb.conditions, " AND ")
		}
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	selectQuery := fmt.Sprintf(
		`SELECT i.id, i.user_id, i.work_item_id, i.position, i.created_at,
		        wi.display_id, wi.title, wi.type, wi.status, COALESCE(ws.category, ''), wi.priority,
		        p.key, p.name, wi.assignee_id, COALESCE(u.display_name, ''), COALESCE(wi.description, ''),
		        wi.due_date, wi.sla_target_at, wi.updated_at
		 %s %s
		 ORDER BY i.position ASC, i.id ASC
		 LIMIT %d`, joinClause, whereClause, limit+1)

	rows, err := r.db.QueryContext(ctx, selectQuery, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("querying inbox items: %w", err)
	}
	defer rows.Close()

	var items []model.InboxItemWithWorkItem
	for rows.Next() {
		var item model.InboxItemWithWorkItem
		var assigneeID uuid.NullUUID
		var dueDate sql.NullTime
		var slaTargetAt sql.NullTime

		err := rows.Scan(
			&item.ID, &item.UserID, &item.WorkItemID, &item.Position, &item.CreatedAt,
			&item.DisplayID, &item.Title, &item.Type, &item.Status, &item.StatusCategory, &item.Priority,
			&item.ProjectKey, &item.ProjectName, &assigneeID, &item.AssigneeDisplayName, &item.Description,
			&dueDate, &slaTargetAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning inbox item: %w", err)
		}
		if assigneeID.Valid {
			item.AssigneeID = &assigneeID.UUID
		}
		if dueDate.Valid {
			item.DueDate = &dueDate.Time
		}
		if slaTargetAt.Valid {
			item.SLATargetAt = &slaTargetAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating inbox items: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var cursorStr string
	if len(items) > 0 {
		cursorStr = items[len(items)-1].ID.String()
	}

	return &model.InboxItemList{
		Items:   items,
		Cursor:  cursorStr,
		HasMore: hasMore,
		Total:   total,
	}, nil
}

// CountByUser returns the number of non-completed inbox items for a user.
func (r *InboxRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM inbox_items i
		 JOIN work_items wi ON i.work_item_id = wi.id
		 JOIN projects p ON wi.project_id = p.id
		 LEFT JOIN project_type_workflows ptw ON ptw.project_id = p.id AND ptw.work_item_type = wi.type
		 LEFT JOIN workflow_statuses ws ON ws.workflow_id = COALESCE(ptw.workflow_id, p.default_workflow_id) AND ws.name = wi.status
		 WHERE i.user_id = $1 AND wi.deleted_at IS NULL
		   AND COALESCE(ws.category, '') NOT IN ('done', 'cancelled')`,
		userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting inbox items: %w", err)
	}
	return count, nil
}

// CountAllByUser returns the total number of inbox items for a user (including completed).
func (r *InboxRepository) CountAllByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inbox_items WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting all inbox items: %w", err)
	}
	return count, nil
}

// UpdatePosition updates the position of a single inbox item.
func (r *InboxRepository) UpdatePosition(ctx context.Context, id, userID uuid.UUID, position int) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE inbox_items SET position = $1 WHERE id = $2 AND user_id = $3`,
		position, id, userID)
	if err != nil {
		return fmt.Errorf("updating inbox item position: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// MaxPosition returns the highest position value in the user's inbox, or 0 if empty.
func (r *InboxRepository) MaxPosition(ctx context.Context, userID uuid.UUID) (int, error) {
	var pos sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(position) FROM inbox_items WHERE user_id = $1`, userID).Scan(&pos)
	if err != nil {
		return 0, fmt.Errorf("getting max position: %w", err)
	}
	if pos.Valid {
		return int(pos.Int64), nil
	}
	return 0, nil
}

// RemoveCompleted deletes inbox items whose work item status category is 'done' or 'cancelled'.
func (r *InboxRepository) RemoveCompleted(ctx context.Context, userID uuid.UUID) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM inbox_items i
		 USING work_items wi
		 JOIN projects p ON wi.project_id = p.id
		 LEFT JOIN project_type_workflows ptw ON ptw.project_id = p.id AND ptw.work_item_type = wi.type
		 LEFT JOIN workflow_statuses ws ON ws.workflow_id = COALESCE(ptw.workflow_id, p.default_workflow_id) AND ws.name = wi.status
		 WHERE i.work_item_id = wi.id
		   AND i.user_id = $1
		   AND ws.category IN ('done', 'cancelled')`,
		userID)
	if err != nil {
		return 0, fmt.Errorf("removing completed inbox items: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// GetWorkItemProjectID returns the project_id for a work item, used for membership checks.
func (r *InboxRepository) GetWorkItemProjectID(ctx context.Context, workItemID uuid.UUID) (uuid.UUID, error) {
	var projectID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT project_id FROM work_items WHERE id = $1 AND deleted_at IS NULL`, workItemID).Scan(&projectID)
	if err == sql.ErrNoRows {
		return uuid.Nil, model.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("getting work item project: %w", err)
	}
	return projectID, nil
}

// Exists checks if a work item is already in the user's inbox.
func (r *InboxRepository) Exists(ctx context.Context, userID, workItemID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM inbox_items WHERE user_id = $1 AND work_item_id = $2)`,
		userID, workItemID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking inbox item existence: %w", err)
	}
	return exists, nil
}
