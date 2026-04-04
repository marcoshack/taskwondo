package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// EmbeddingRepository handles embedding persistence.
type EmbeddingRepository struct {
	db *sql.DB
}

// NewEmbeddingRepository creates a new EmbeddingRepository.
func NewEmbeddingRepository(db *sql.DB) *EmbeddingRepository {
	return &EmbeddingRepository{db: db}
}

// Upsert inserts or updates an embedding for the given entity.
func (r *EmbeddingRepository) Upsert(ctx context.Context, e *model.Embedding) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO embeddings (id, entity_type, entity_id, project_id, content, embedding, indexed_at)
		 VALUES ($1, $2, $3, $4, $5, $6::vector, now())
		 ON CONFLICT (entity_type, entity_id)
		 DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding,
		               project_id = EXCLUDED.project_id, indexed_at = now()`,
		e.ID, e.EntityType, e.EntityID, e.ProjectID, e.Content, vectorToString(e.Embedding))
	if err != nil {
		return fmt.Errorf("upserting embedding: %w", err)
	}
	return nil
}

// Delete removes an embedding for the given entity.
func (r *EmbeddingRepository) Delete(ctx context.Context, entityType string, entityID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM embeddings WHERE entity_type = $1 AND entity_id = $2`,
		entityType, entityID)
	if err != nil {
		return fmt.Errorf("deleting embedding: %w", err)
	}
	return nil
}

// SearchByVector performs a cosine similarity search filtered by accessible project IDs.
//
// RBAC: for projects where the caller has the "customer" role, results are
// restricted to the caller's own portal tickets (via reporter_id match +
// visibility='portal' on the parent work item). For comments and attachments,
// the parent work item's reporter and visibility are enforced. Customer-only
// projects cannot surface project/milestone/queue embeddings.
func (r *EmbeddingRepository) SearchByVector(ctx context.Context, vector []float32, filter *model.SearchFilter, access model.SearchAccess) ([]model.SearchResult, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var conditions []string
	var args []any
	argIdx := 1

	// Vector parameter
	args = append(args, vectorToString(vector))
	vectorArg := argIdx
	argIdx++

	// RBAC: build the project-access clause. Customer projects apply
	// per-entity-type restrictions enforced via the LEFT JOINed work items.
	var accessClauses []string

	if len(access.FullProjectIDs) > 0 {
		placeholders := make([]string, len(access.FullProjectIDs))
		for i, pid := range access.FullProjectIDs {
			args = append(args, pid)
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			argIdx++
		}
		accessClauses = append(accessClauses, fmt.Sprintf("e.project_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(access.CustomerProjectIDs) > 0 {
		placeholders := make([]string, len(access.CustomerProjectIDs))
		for i, pid := range access.CustomerProjectIDs {
			args = append(args, pid)
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			argIdx++
		}
		// Bind the user id and the portal visibility once, reuse via $N.
		args = append(args, access.UserID)
		userArg := argIdx
		argIdx++
		args = append(args, model.VisibilityPortal)
		portalArg := argIdx
		argIdx++
		args = append(args, model.VisibilityPublic)
		publicArg := argIdx
		argIdx++

		customerClause := fmt.Sprintf(
			"(e.project_id IN (%s) AND ("+
				"(e.entity_type = 'work_item' AND w.reporter_id = $%d AND w.visibility = $%d)"+
				" OR (e.entity_type = 'comment' AND cw.reporter_id = $%d AND cw.visibility = $%d AND c.visibility = $%d)"+
				" OR (e.entity_type = 'attachment' AND aw.reporter_id = $%d AND aw.visibility = $%d)"+
				"))",
			strings.Join(placeholders, ","),
			userArg, portalArg,
			userArg, portalArg, publicArg,
			userArg, portalArg,
		)
		accessClauses = append(accessClauses, customerClause)
	}

	// Always allow global entities (project_id IS NULL) — they aren't sensitive.
	accessClauses = append(accessClauses, "e.project_id IS NULL")

	conditions = append(conditions, "("+strings.Join(accessClauses, " OR ")+")")

	// Entity type filter
	if len(filter.EntityTypes) > 0 {
		placeholders := make([]string, len(filter.EntityTypes))
		for i, et := range filter.EntityTypes {
			args = append(args, et)
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("e.entity_type IN (%s)", strings.Join(placeholders, ",")))
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	args = append(args, limit)
	limitArg := argIdx

	query := fmt.Sprintf(
		`SELECT e.entity_type, e.entity_id, e.project_id,
		        1 - (e.embedding <=> $%d::vector) AS score, e.content,
		        p.key AS project_key,
		        COALESCE(w.item_number, cw.item_number, aw.item_number) AS item_number,
		        COALESCE(n.slug, 'default') AS namespace_slug,
		        COALESCE(w.status, '') AS status,
		        COALESCE(ws.category, '') AS status_category
		 FROM embeddings e
		 LEFT JOIN projects p ON p.id = e.project_id
		 LEFT JOIN namespaces n ON n.id = p.namespace_id
		 LEFT JOIN work_items w ON w.id = e.entity_id AND e.entity_type = 'work_item'
		 LEFT JOIN project_type_workflows ptw ON ptw.project_id = p.id AND ptw.work_item_type = w.type AND e.entity_type = 'work_item'
		 LEFT JOIN workflow_statuses ws ON ws.workflow_id = COALESCE(ptw.workflow_id, p.default_workflow_id) AND ws.name = w.status AND e.entity_type = 'work_item'
		 LEFT JOIN comments c ON c.id = e.entity_id AND e.entity_type = 'comment'
		 LEFT JOIN work_items cw ON cw.id = c.work_item_id
		 LEFT JOIN attachments a ON a.id = e.entity_id AND e.entity_type = 'attachment'
		 LEFT JOIN work_items aw ON aw.id = a.work_item_id
		 %s
		 ORDER BY e.embedding <=> $%d::vector
		 LIMIT $%d`,
		vectorArg, where, vectorArg, limitArg)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("searching embeddings: %w", err)
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		var sr model.SearchResult
		var projectKey sql.NullString
		var itemNumber sql.NullInt64
		var namespaceSlug sql.NullString
		var status, statusCategory string
		if err := rows.Scan(&sr.EntityType, &sr.EntityID, &sr.ProjectID, &sr.Score, &sr.Content, &projectKey, &itemNumber, &namespaceSlug, &status, &statusCategory); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		if projectKey.Valid {
			sr.ProjectKey = projectKey.String
		}
		if itemNumber.Valid {
			n := int(itemNumber.Int64)
			sr.ItemNumber = &n
		}
		if namespaceSlug.Valid {
			sr.NamespaceSlug = namespaceSlug.String
		}
		sr.Status = status
		sr.StatusCategory = statusCategory
		results = append(results, sr)
	}
	return results, rows.Err()
}

// CountByEntityType returns the number of embeddings for a given entity type.
func (r *EmbeddingRepository) CountByEntityType(ctx context.Context, entityType string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM embeddings WHERE entity_type = $1`, entityType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting embeddings: %w", err)
	}
	return count, nil
}

// vectorToString formats a float32 slice as a pgvector literal: [0.1,0.2,0.3]
func vectorToString(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}
