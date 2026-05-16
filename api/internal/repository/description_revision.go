package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// DescriptionRevisionRepository persists work item description revisions.
type DescriptionRevisionRepository struct {
	db *sql.DB
}

// NewDescriptionRevisionRepository creates a new DescriptionRevisionRepository.
func NewDescriptionRevisionRepository(db *sql.DB) *DescriptionRevisionRepository {
	return &DescriptionRevisionRepository{db: db}
}

// Create inserts a new revision and assigns the next revision_number for the
// work item atomically.
func (r *DescriptionRevisionRepository) Create(ctx context.Context, rev *model.DescriptionRevision) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	var nextRev int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(revision_number), 0) + 1
		 FROM work_item_description_revisions
		 WHERE work_item_id = $1`, rev.WorkItemID).Scan(&nextRev)
	if err != nil {
		return fmt.Errorf("computing next revision number: %w", err)
	}
	rev.RevisionNumber = nextRev

	err = tx.QueryRowContext(ctx,
		`INSERT INTO work_item_description_revisions
			(id, work_item_id, revision_number, content, content_hash, author_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		rev.ID, rev.WorkItemID, rev.RevisionNumber, rev.Content, rev.ContentHash, rev.AuthorID,
	).Scan(&rev.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting revision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// GetByID fetches a single revision.
func (r *DescriptionRevisionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DescriptionRevision, error) {
	var rev model.DescriptionRevision
	var authorID uuid.NullUUID

	err := r.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, revision_number, content, content_hash, author_id, created_at
		 FROM work_item_description_revisions
		 WHERE id = $1`, id).Scan(
		&rev.ID, &rev.WorkItemID, &rev.RevisionNumber, &rev.Content, &rev.ContentHash,
		&authorID, &rev.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying revision: %w", err)
	}
	if authorID.Valid {
		rev.AuthorID = &authorID.UUID
	}
	return &rev, nil
}

// GetLatest returns the most recent revision (highest revision_number) for
// a work item, or ErrNotFound if there are none.
func (r *DescriptionRevisionRepository) GetLatest(ctx context.Context, workItemID uuid.UUID) (*model.DescriptionRevision, error) {
	var rev model.DescriptionRevision
	var authorID uuid.NullUUID

	err := r.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, revision_number, content, content_hash, author_id, created_at
		 FROM work_item_description_revisions
		 WHERE work_item_id = $1
		 ORDER BY revision_number DESC
		 LIMIT 1`, workItemID).Scan(
		&rev.ID, &rev.WorkItemID, &rev.RevisionNumber, &rev.Content, &rev.ContentHash,
		&authorID, &rev.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying latest revision: %w", err)
	}
	if authorID.Valid {
		rev.AuthorID = &authorID.UUID
	}
	return &rev, nil
}

// ListByWorkItem returns revisions for a work item, newest first.
func (r *DescriptionRevisionRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.DescriptionRevision, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, work_item_id, revision_number, content, content_hash, author_id, created_at
		 FROM work_item_description_revisions
		 WHERE work_item_id = $1
		 ORDER BY revision_number DESC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying revisions: %w", err)
	}
	defer rows.Close()

	var out []model.DescriptionRevision
	for rows.Next() {
		var rev model.DescriptionRevision
		var authorID uuid.NullUUID
		if err := rows.Scan(
			&rev.ID, &rev.WorkItemID, &rev.RevisionNumber, &rev.Content, &rev.ContentHash,
			&authorID, &rev.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning revision: %w", err)
		}
		if authorID.Valid {
			rev.AuthorID = &authorID.UUID
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}
