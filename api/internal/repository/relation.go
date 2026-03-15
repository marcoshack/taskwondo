package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// WorkItemRelationRepository handles work item relation persistence.
type WorkItemRelationRepository struct {
	db *sql.DB
}

// NewWorkItemRelationRepository creates a new WorkItemRelationRepository.
func NewWorkItemRelationRepository(db *sql.DB) *WorkItemRelationRepository {
	return &WorkItemRelationRepository{db: db}
}

// Create inserts a new work item relation.
func (r *WorkItemRelationRepository) Create(ctx context.Context, relation *model.WorkItemRelation) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO work_item_relations (id, source_id, target_id, relation_type, created_by)
		 VALUES ($1, $2, $3, $4, $5)`,
		relation.ID, relation.SourceID, relation.TargetID, relation.RelationType, relation.CreatedBy)
	if err != nil {
		return fmt.Errorf("inserting work item relation: %w", err)
	}
	return nil
}

// GetByID returns a relation by its ID.
func (r *WorkItemRelationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkItemRelation, error) {
	var rel model.WorkItemRelation
	err := r.db.QueryRowContext(ctx,
		`SELECT id, source_id, target_id, relation_type, created_by, created_at
		 FROM work_item_relations WHERE id = $1`, id).Scan(
		&rel.ID, &rel.SourceID, &rel.TargetID, &rel.RelationType,
		&rel.CreatedBy, &rel.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying relation: %w", err)
	}
	return &rel, nil
}

// ListByWorkItem returns all relations where the given work item is source or target.
func (r *WorkItemRelationRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemRelation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, source_id, target_id, relation_type, created_by, created_at
		 FROM work_item_relations
		 WHERE source_id = $1 OR target_id = $1
		 ORDER BY created_at ASC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying relations: %w", err)
	}
	defer rows.Close()

	var relations []model.WorkItemRelation
	for rows.Next() {
		var rel model.WorkItemRelation
		if err := rows.Scan(
			&rel.ID, &rel.SourceID, &rel.TargetID, &rel.RelationType,
			&rel.CreatedBy, &rel.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning relation: %w", err)
		}
		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// ListByWorkItemWithDetails returns all relations with display info for both sides.
func (r *WorkItemRelationRepository) ListByWorkItemWithDetails(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemRelationWithDetails, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.source_id, r.target_id, r.relation_type, r.created_by, r.created_at,
		        sp.key, sw.item_number, sw.title, sw.status, COALESCE(sws.category, ''),
		        tp.key, tw.item_number, tw.title, tw.status, COALESCE(tws.category, '')
		 FROM work_item_relations r
		 JOIN work_items sw ON sw.id = r.source_id
		 JOIN projects sp ON sp.id = sw.project_id
		 LEFT JOIN project_type_workflows sptw ON sptw.project_id = sp.id AND sptw.work_item_type = sw.type
		 LEFT JOIN workflow_statuses sws ON sws.workflow_id = COALESCE(sptw.workflow_id, sp.default_workflow_id) AND sws.name = sw.status
		 JOIN work_items tw ON tw.id = r.target_id
		 JOIN projects tp ON tp.id = tw.project_id
		 LEFT JOIN project_type_workflows tptw ON tptw.project_id = tp.id AND tptw.work_item_type = tw.type
		 LEFT JOIN workflow_statuses tws ON tws.workflow_id = COALESCE(tptw.workflow_id, tp.default_workflow_id) AND tws.name = tw.status
		 WHERE r.source_id = $1 OR r.target_id = $1
		 ORDER BY r.created_at ASC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying relations with details: %w", err)
	}
	defer rows.Close()

	var relations []model.WorkItemRelationWithDetails
	for rows.Next() {
		var rel model.WorkItemRelationWithDetails
		if err := rows.Scan(
			&rel.ID, &rel.SourceID, &rel.TargetID, &rel.RelationType,
			&rel.CreatedBy, &rel.CreatedAt,
			&rel.SourceProjectKey, &rel.SourceItemNumber, &rel.SourceTitle,
			&rel.SourceStatus, &rel.SourceStatusCategory,
			&rel.TargetProjectKey, &rel.TargetItemNumber, &rel.TargetTitle,
			&rel.TargetStatus, &rel.TargetStatusCategory,
		); err != nil {
			return nil, fmt.Errorf("scanning relation with details: %w", err)
		}
		relations = append(relations, rel)
	}

	return relations, rows.Err()
}

// Delete hard-deletes a relation.
func (r *WorkItemRelationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM work_item_relations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting relation: %w", err)
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
