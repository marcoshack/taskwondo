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

// ListAllIDs returns all non-deleted work item IDs with pagination (for backfill).
func (r *WorkItemRepository) ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error) {
	return listAllIDs(ctx, r.db, "work_items", limit, offset)
}

// workItemSelectColumns is the standard column list returned by scanWorkItem /
// scanWorkItems. The qualifier `wi` must match the alias applied to the
// work_items table in the surrounding query.
const workItemSelectColumns = `wi.id, wi.project_id, wi.queue_id, wi.milestone_id, wi.parent_id, wi.item_number, wi.display_id,
	wi.type, wi.title, wi.description, wi.status, wi.priority,
	wi.assignee_id, wi.reporter_id, wi.portal_contact_id, wi.visibility,
	wi.labels, wi.complexity, wi.custom_fields, wi.due_date, wi.resolved_at, wi.sla_target_at, wi.estimated_seconds,
	wi.created_at, wi.updated_at, u.display_name AS reporter_name`

// GetByID returns a work item by its UUID.
func (r *WorkItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkItem, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+workItemSelectColumns+`
		 FROM work_items wi
		 LEFT JOIN users u ON u.id = wi.reporter_id
		 WHERE wi.id = $1 AND wi.deleted_at IS NULL`, id)
	return scanWorkItem(row)
}

// GetByProjectAndNumber returns a work item by project ID and item number.
func (r *WorkItemRepository) GetByProjectAndNumber(ctx context.Context, projectID uuid.UUID, itemNumber int) (*model.WorkItem, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+workItemSelectColumns+`
		 FROM work_items wi
		 LEFT JOIN users u ON u.id = wi.reporter_id
		 WHERE wi.project_id = $1 AND wi.item_number = $2 AND wi.deleted_at IS NULL`,
		projectID, itemNumber)
	return scanWorkItem(row)
}

// List returns work items matching the given filter with cursor-based pagination.
func (r *WorkItemRepository) List(ctx context.Context, projectID uuid.UUID, filter *model.WorkItemFilter) (*model.WorkItemList, error) {
	qb := &queryBuilder{argIndex: 0}

	// Base condition
	if !filter.SkipProjectFilter {
		qb.add("project_id = ?", projectID)
	}
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

	// Reporter filter
	if filter.ReporterID != nil {
		qb.add("reporter_id = ?", *filter.ReporterID)
	}

	// Exclude resolved (hide completed)
	if filter.ExcludeResolved {
		qb.addRaw("resolved_at IS NULL")
	}

	// Milestone filter
	if filter.MilestoneNone && len(filter.MilestoneIDs) > 0 {
		qb.add("(milestone_id IS NULL OR milestone_id = ANY(?))", pq.Array(filter.MilestoneIDs))
	} else if filter.MilestoneNone {
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

	// Item IDs filter (for watchlist). Qualified because the outer query joins
	// users, whose table also has an `id` column.
	if len(filter.ItemIDs) > 0 {
		qb.add("wi.id = ANY(?)", pq.Array(filter.ItemIDs))
	}

	// Full-text search (OR simple config to match display_id tokens like "TF-29")
	if filter.Search != "" {
		qb.add("(search_vector @@ plainto_tsquery('english', ?) OR search_vector @@ plainto_tsquery('simple', ?))", filter.Search, filter.Search)
	}

	whereClause := qb.whereClause()

	// Count total (without cursor/limit). Aliased so `wi.id` in qb.conditions
	// (added above for watchlist) resolves here.
	countQuery := "SELECT COUNT(*) FROM work_items wi " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, qb.args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("counting work items: %w", err)
	}

	// Determine sort column and order. The column name is qualified with `wi.`
	// so it's unambiguous under the `LEFT JOIN users u` in the SELECT query;
	// ambiguity would otherwise bite on created_at / updated_at / id.
	sortCol := "wi.created_at"
	switch filter.Sort {
	case "updated_at", "due_date", "item_number", "type", "title", "status":
		sortCol = "wi." + filter.Sort
	case "priority":
		// Use CASE expression for semantic ordering: critical(1) > high(2) > medium(3) > low(4)
		sortCol = "CASE wi.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END"
	case "sla_target_at":
		sortCol = "wi.sla_target_at" // COALESCE applied below
	}
	sortOrder := "DESC"
	if filter.Order == "asc" {
		sortOrder = "ASC"
	}

	// Push NULL sla_target_at values to the end regardless of sort direction.
	// Use extreme but parseable timestamps instead of PostgreSQL infinity/-infinity,
	// which lib/pq cannot scan into time.Time (breaking cursor pagination).
	if sortCol == "wi.sla_target_at" {
		if sortOrder == "ASC" {
			sortCol = "COALESCE(wi.sla_target_at, '9999-12-31T23:59:59Z'::timestamptz)"
		} else {
			sortCol = "COALESCE(wi.sla_target_at, '0001-01-01T00:00:00Z'::timestamptz)"
		}
	}

	// Cursor pagination: fetch the cursor item's sort column value for tuple comparison.
	// sortCol is already sanitized by the switch above, so this Sprintf is safe.
	if filter.Cursor != nil {
		var cursorVal interface{}
		err := r.db.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT %s FROM work_items wi WHERE wi.id = $1`, sortCol), *filter.Cursor).Scan(&cursorVal)
		if err == nil && cursorVal != nil {
			if sortOrder == "DESC" {
				qb.add("("+sortCol+", wi.id) < (?, ?)", cursorVal, *filter.Cursor)
			} else {
				qb.add("("+sortCol+", wi.id) > (?, ?)", cursorVal, *filter.Cursor)
			}
			// Rebuild WHERE clause with cursor condition
			whereClause = qb.whereClause()
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
		`SELECT `+workItemSelectColumns+`
		 FROM work_items wi
		 LEFT JOIN users u ON u.id = wi.reporter_id
		 %s
		 ORDER BY %s %s, wi.id %s
		 LIMIT %d`,
		whereClause, sortCol, sortOrder, sortOrder, limit+1)

	rows, err := r.db.QueryContext(ctx, selectQuery, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("querying work items: %w", err)
	}
	defer rows.Close()

	items, err := scanWorkItems(rows, limit+1)
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

// TouchUpdatedAt bumps the updated_at timestamp on a work item.
func (r *WorkItemRepository) TouchUpdatedAt(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE work_items SET updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

// --- Scan helpers ---

// workItemScanCols holds the nullable columns scanned from work_items queries.
// It lets a single Scan args slice feed both *sql.Row and *sql.Rows callers.
type workItemScanCols struct {
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
	reporterName     sql.NullString
}

// scanArgs returns the pointer list for the standard work_items SELECT column
// order used by GetByID, List, and related queries.
func (c *workItemScanCols) scanArgs(item *model.WorkItem) []interface{} {
	return []interface{}{
		&item.ID, &item.ProjectID, &c.queueID, &c.milestoneID, &c.parentID, &item.ItemNumber, &item.DisplayID,
		&item.Type, &item.Title, &c.description, &item.Status, &item.Priority,
		&c.assigneeID, &item.ReporterID, &c.portalContactID, &item.Visibility,
		&c.labels, &c.complexity, &c.customFieldsRaw, &c.dueDate, &c.resolvedAt, &c.slaTargetAt, &c.estimatedSeconds,
		&item.CreatedAt, &item.UpdatedAt,
		&c.reporterName,
	}
}

// apply copies scanned nullable values into the item. Returns an error if the
// custom_fields blob cannot be unmarshaled.
func (c *workItemScanCols) apply(item *model.WorkItem) error {
	if c.reporterName.Valid {
		item.ReporterName = c.reporterName.String
	}
	return populateWorkItem(item, c.description, c.queueID, c.milestoneID, c.parentID, c.assigneeID,
		c.portalContactID, c.complexity, c.dueDate, c.resolvedAt, c.slaTargetAt, c.estimatedSeconds, c.labels, c.customFieldsRaw)
}

func scanWorkItem(row *sql.Row) (*model.WorkItem, error) {
	var item model.WorkItem
	var cols workItemScanCols
	err := row.Scan(cols.scanArgs(&item)...)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning work item: %w", err)
	}
	if err := cols.apply(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

// scanWorkItems scans rows into a slice. capacityHint should be the expected
// number of rows (e.g. the query limit) to pre-size the slice and avoid
// repeated growth; pass 0 if unknown.
func scanWorkItems(rows *sql.Rows, capacityHint int) ([]model.WorkItem, error) {
	items := make([]model.WorkItem, 0, capacityHint)
	for rows.Next() {
		var item model.WorkItem
		var cols workItemScanCols
		if err := rows.Scan(cols.scanArgs(&item)...); err != nil {
			return nil, fmt.Errorf("scanning work item row: %w", err)
		}
		if err := cols.apply(&item); err != nil {
			return nil, err
		}
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
) error {
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
		if err := json.Unmarshal(customFieldsRaw, &item.CustomFields); err != nil {
			return fmt.Errorf("unmarshaling custom_fields for work item %s: %w", item.ID, err)
		}
	}
	return nil
}

// SearchFTS performs a cross-project full-text search across all accessible projects.
// It uses the existing search_vector tsvector column with ts_rank for ordering.
// Display ID queries (e.g. "TF-42") are boosted to rank highest.
//
// RBAC: for projects where the caller has the "customer" role, results are
// restricted to the caller's own portal tickets (reporter_id=UserID AND
// visibility='portal'). No internal work items are ever leaked to customers.
func (r *WorkItemRepository) SearchFTS(ctx context.Context, query string, access model.SearchAccess, limit int) ([]model.SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// If the caller has no reachable projects, short-circuit.
	if !access.HasAny() {
		return nil, nil
	}

	qb := &queryBuilder{argIndex: 0}
	qb.add("w.deleted_at IS NULL")
	qb.add("(w.search_vector @@ plainto_tsquery('english', ?) OR w.search_vector @@ plainto_tsquery('simple', ?))", query, query)

	// RBAC: full-access projects OR (customer projects AND own portal tickets)
	switch {
	case len(access.FullProjectIDs) > 0 && len(access.CustomerProjectIDs) > 0:
		qb.add(
			"(w.project_id = ANY(?) OR (w.project_id = ANY(?) AND w.reporter_id = ? AND w.visibility = ?))",
			pq.Array(access.FullProjectIDs), pq.Array(access.CustomerProjectIDs), access.UserID, model.VisibilityPortal,
		)
	case len(access.FullProjectIDs) > 0:
		qb.add("w.project_id = ANY(?)", pq.Array(access.FullProjectIDs))
	case len(access.CustomerProjectIDs) > 0:
		qb.add(
			"w.project_id = ANY(?) AND w.reporter_id = ? AND w.visibility = ?",
			pq.Array(access.CustomerProjectIDs), access.UserID, model.VisibilityPortal,
		)
	}

	whereClause := qb.whereClause()

	// Boost display ID exact matches: if the query looks like a display ID (e.g. "TF-42"),
	// add a large bonus so it sorts first.
	qb.argIndex++
	queryArgIdx := qb.argIndex
	qb.args = append(qb.args, query)

	qb.argIndex++
	limitArgIdx := qb.argIndex
	qb.args = append(qb.args, limit)

	sqlQuery := fmt.Sprintf(
		`SELECT w.id, w.project_id, w.item_number, w.display_id, w.type, w.title,
		        p.key AS project_key,
		        COALESCE(n.slug, 'default') AS namespace_slug,
		        w.status,
		        COALESCE(ws.category, '') AS status_category,
		        ts_rank(w.search_vector, plainto_tsquery('english', $%d)) +
		        ts_rank(w.search_vector, plainto_tsquery('simple', $%d)) +
		        CASE WHEN UPPER(w.display_id) = UPPER($%d) THEN 1000 ELSE 0 END AS rank
		 FROM work_items w
		 JOIN projects p ON p.id = w.project_id
		 LEFT JOIN namespaces n ON n.id = p.namespace_id
		 LEFT JOIN project_type_workflows ptw ON ptw.project_id = p.id AND ptw.work_item_type = w.type
		 LEFT JOIN workflow_statuses ws ON ws.workflow_id = COALESCE(ptw.workflow_id, p.default_workflow_id) AND ws.name = w.status
		 %s
		 ORDER BY rank DESC, w.updated_at DESC
		 LIMIT $%d`,
		queryArgIdx, queryArgIdx, queryArgIdx, whereClause, limitArgIdx)

	rows, err := r.db.QueryContext(ctx, sqlQuery, qb.args...)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		var (
			id             uuid.UUID
			projectID      uuid.UUID
			itemNumber     int
			displayID      string
			itemType       string
			title          string
			projectKey     string
			namespaceSlug  string
			status         string
			statusCategory string
			rank           float64
		)
		if err := rows.Scan(&id, &projectID, &itemNumber, &displayID, &itemType, &title, &projectKey, &namespaceSlug, &status, &statusCategory, &rank); err != nil {
			return nil, fmt.Errorf("scanning fts result: %w", err)
		}
		results = append(results, model.SearchResult{
			EntityType:     model.EntityTypeWorkItem,
			EntityID:       id,
			ProjectID:      &projectID,
			Score:          0, // FTS results don't have similarity scores
			Content:        fmt.Sprintf("[%s] %s", itemType, title),
			ProjectKey:     projectKey,
			ItemNumber:     &itemNumber,
			NamespaceSlug:  namespaceSlug,
			Status:         status,
			StatusCategory: statusCategory,
		})
	}
	return results, rows.Err()
}

// ListByProjectTypeNullSLA returns non-deleted work items in a project with a
// given type that have sla_target_at IS NULL. Used for backfilling SLA deadlines.
func (r *WorkItemRepository) ListByProjectTypeNullSLA(ctx context.Context, projectID uuid.UUID, workItemType string) ([]model.WorkItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+workItemSelectColumns+`
		 FROM work_items wi
		 LEFT JOIN users u ON u.id = wi.reporter_id
		 WHERE wi.project_id = $1 AND wi.type = $2 AND wi.sla_target_at IS NULL AND wi.deleted_at IS NULL`,
		projectID, workItemType)
	if err != nil {
		return nil, fmt.Errorf("listing items for SLA backfill: %w", err)
	}
	defer rows.Close()
	return scanWorkItems(rows, 0)
}

// UpdateSLATargetAt updates only the sla_target_at column for a work item.
func (r *WorkItemRepository) UpdateSLATargetAt(ctx context.Context, id uuid.UUID, slaTargetAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE work_items SET sla_target_at = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`,
		slaTargetAt, id)
	return err
}
