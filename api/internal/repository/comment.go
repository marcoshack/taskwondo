package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// CommentRepository handles comment persistence.
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository creates a new CommentRepository.
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// commentSelect is the canonical column list returned by all comment queries.
// Anchor columns are nullable; a row with NULL anchor_revision_id is a regular
// (non-inline) comment. parent_comment_id is set only on threaded replies.
const commentSelect = `c.id, c.work_item_id, c.author_id, c.portal_contact_id, c.body,
	c.visibility, c.edit_count, c.created_at, c.updated_at,
	c.anchor_revision_id, c.anchor_start_line, c.anchor_start_col,
	c.anchor_end_line, c.anchor_end_col,
	c.anchor_snippet, c.anchor_snippet_hash, c.anchor_status,
	c.parent_comment_id, rev.revision_number`

// scanComment scans the canonical column list into a Comment, populating the
// Anchor sub-struct when the comment is inline.
func scanComment(scanner interface{ Scan(...any) error }) (*model.Comment, error) {
	var c model.Comment
	var (
		authorID        uuid.NullUUID
		portalContactID uuid.NullUUID
		anchorRevID     uuid.NullUUID
		anchorStart     sql.NullInt64
		anchorStartCol  sql.NullInt64
		anchorEnd       sql.NullInt64
		anchorEndCol    sql.NullInt64
		anchorSnippet   sql.NullString
		anchorHash      sql.NullString
		anchorStatus    sql.NullString
		parentID        uuid.NullUUID
		anchorRevNum    sql.NullInt64
	)

	if err := scanner.Scan(
		&c.ID, &c.WorkItemID, &authorID, &portalContactID, &c.Body,
		&c.Visibility, &c.EditCount, &c.CreatedAt, &c.UpdatedAt,
		&anchorRevID, &anchorStart, &anchorStartCol,
		&anchorEnd, &anchorEndCol,
		&anchorSnippet, &anchorHash, &anchorStatus,
		&parentID, &anchorRevNum,
	); err != nil {
		return nil, err
	}

	if authorID.Valid {
		c.AuthorID = &authorID.UUID
	}
	if portalContactID.Valid {
		c.PortalContactID = &portalContactID.UUID
	}
	if parentID.Valid {
		c.ParentCommentID = &parentID.UUID
	}

	if anchorRevID.Valid {
		c.Anchor = &model.CommentAnchor{
			RevisionID:     anchorRevID.UUID,
			RevisionNumber: int(anchorRevNum.Int64),
			StartLine:      int(anchorStart.Int64),
			StartCol:       int(anchorStartCol.Int64),
			EndLine:        int(anchorEnd.Int64),
			EndCol:         int(anchorEndCol.Int64),
		}
		if anchorSnippet.Valid {
			c.Anchor.Snippet = anchorSnippet.String
		}
		if anchorHash.Valid {
			c.Anchor.SnippetHash = anchorHash.String
		}
		if anchorStatus.Valid {
			c.Anchor.Status = anchorStatus.String
		}
	}

	return &c, nil
}

// ListAllIDs returns all non-deleted comment IDs with pagination (for backfill).
func (r *CommentRepository) ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error) {
	return listAllIDs(ctx, r.db, "comments", limit, offset)
}

// Create inserts a new comment, optionally with an inline anchor and/or a
// parent (threaded reply).
func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	var (
		anchorRevID    any
		anchorStart    any
		anchorStartCol any
		anchorEnd      any
		anchorEndCol   any
		anchorSnippet  any
		anchorHash     any
		anchorStatus   any
	)
	if comment.Anchor != nil {
		anchorRevID = comment.Anchor.RevisionID
		anchorStart = comment.Anchor.StartLine
		anchorStartCol = comment.Anchor.StartCol
		anchorEnd = comment.Anchor.EndLine
		anchorEndCol = comment.Anchor.EndCol
		anchorSnippet = comment.Anchor.Snippet
		anchorHash = comment.Anchor.SnippetHash
		status := comment.Anchor.Status
		if status == "" {
			status = model.AnchorStatusActive
		}
		anchorStatus = status
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (
			id, work_item_id, author_id, portal_contact_id, body, visibility,
			anchor_revision_id, anchor_start_line, anchor_start_col,
			anchor_end_line, anchor_end_col,
			anchor_snippet, anchor_snippet_hash, anchor_status, parent_comment_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		comment.ID, comment.WorkItemID, comment.AuthorID, comment.PortalContactID,
		comment.Body, comment.Visibility,
		anchorRevID, anchorStart, anchorStartCol, anchorEnd, anchorEndCol,
		anchorSnippet, anchorHash, anchorStatus, comment.ParentCommentID)
	if err != nil {
		return fmt.Errorf("inserting comment: %w", err)
	}
	return nil
}

// GetByID returns a comment by its ID.
func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+commentSelect+`
		 FROM comments c
		 LEFT JOIN work_item_description_revisions rev ON rev.id = c.anchor_revision_id
		 WHERE c.id = $1 AND c.deleted_at IS NULL`, id)
	c, err := scanComment(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying comment: %w", err)
	}
	return c, nil
}

// ListByWorkItem returns all non-deleted comments for a work item, ordered by
// creation time. If visibility is non-empty, only comments matching that
// visibility are returned. Inline anchor fields are populated when present.
func (r *CommentRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID, visibility string) ([]model.Comment, error) {
	query := `SELECT ` + commentSelect + `
		FROM comments c
		LEFT JOIN work_item_description_revisions rev ON rev.id = c.anchor_revision_id
		WHERE c.work_item_id = $1 AND c.deleted_at IS NULL`
	args := []any{workItemID}

	if visibility != "" {
		query += ` AND c.visibility = $2`
		args = append(args, visibility)
	}

	query += ` ORDER BY c.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying comments: %w", err)
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		comments = append(comments, *c)
	}

	return comments, rows.Err()
}

// ListInlineByWorkItem returns inline comments (anchor IS NOT NULL) for a
// work item, regardless of visibility. Used by the re-anchor pass.
func (r *CommentRepository) ListInlineByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.Comment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+commentSelect+`
		 FROM comments c
		 LEFT JOIN work_item_description_revisions rev ON rev.id = c.anchor_revision_id
		 WHERE c.work_item_id = $1 AND c.deleted_at IS NULL
		   AND c.anchor_revision_id IS NOT NULL
		 ORDER BY c.created_at ASC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying inline comments: %w", err)
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		comments = append(comments, *c)
	}
	return comments, rows.Err()
}

// Update modifies a comment's body and visibility.
func (r *CommentRepository) Update(ctx context.Context, comment *model.Comment) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE comments SET body = $1, visibility = $2, edit_count = edit_count + 1, updated_at = now()
		 WHERE id = $3 AND deleted_at IS NULL`,
		comment.Body, comment.Visibility, comment.ID)
	if err != nil {
		return fmt.Errorf("updating comment: %w", err)
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

// UpdateAnchor rewrites only the anchor fields of an existing comment. Used
// by the re-anchor pass after a description save. If anchor is nil, the
// comment's anchor is cleared (rare — used to convert an inline comment back
// to a regular one).
func (r *CommentRepository) UpdateAnchor(ctx context.Context, commentID uuid.UUID, anchor *model.CommentAnchor) error {
	if anchor == nil {
		_, err := r.db.ExecContext(ctx,
			`UPDATE comments SET
				anchor_revision_id = NULL,
				anchor_start_line  = NULL,
				anchor_start_col   = NULL,
				anchor_end_line    = NULL,
				anchor_end_col     = NULL,
				anchor_snippet     = NULL,
				anchor_snippet_hash = NULL,
				anchor_status      = NULL
			 WHERE id = $1 AND deleted_at IS NULL`, commentID)
		if err != nil {
			return fmt.Errorf("clearing anchor: %w", err)
		}
		return nil
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE comments SET
			anchor_revision_id = $1,
			anchor_start_line  = $2,
			anchor_start_col   = $3,
			anchor_end_line    = $4,
			anchor_end_col     = $5,
			anchor_snippet     = $6,
			anchor_snippet_hash = $7,
			anchor_status      = $8
		 WHERE id = $9 AND deleted_at IS NULL`,
		anchor.RevisionID, anchor.StartLine, anchor.StartCol,
		anchor.EndLine, anchor.EndCol,
		anchor.Snippet, anchor.SnippetHash, anchor.Status, commentID)
	if err != nil {
		return fmt.Errorf("updating anchor: %w", err)
	}
	return nil
}

// Delete soft-deletes a comment. If the comment is a thread root, its replies
// are soft-deleted along with it so a thread never outlives its anchor.
func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE comments SET deleted_at = now(), updated_at = now()
		 WHERE (id = $1 OR parent_comment_id = $1) AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)
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
