package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
)

// CommentRepository handles comment persistence.
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository creates a new CommentRepository.
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create inserts a new comment.
func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (id, work_item_id, author_id, portal_contact_id, body, visibility)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		comment.ID, comment.WorkItemID, comment.AuthorID, comment.PortalContactID,
		comment.Body, comment.Visibility)
	if err != nil {
		return fmt.Errorf("inserting comment: %w", err)
	}
	return nil
}

// GetByID returns a comment by its ID.
func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	var comment model.Comment
	var (
		authorID        uuid.NullUUID
		portalContactID uuid.NullUUID
	)

	err := r.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, author_id, portal_contact_id, body, visibility, edit_count, created_at, updated_at
		 FROM comments
		 WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&comment.ID, &comment.WorkItemID, &authorID, &portalContactID,
		&comment.Body, &comment.Visibility, &comment.EditCount, &comment.CreatedAt, &comment.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying comment: %w", err)
	}

	if authorID.Valid {
		comment.AuthorID = &authorID.UUID
	}
	if portalContactID.Valid {
		comment.PortalContactID = &portalContactID.UUID
	}

	return &comment, nil
}

// ListByWorkItem returns all non-deleted comments for a work item, ordered by creation time.
// If visibility is non-empty, only comments matching that visibility are returned.
func (r *CommentRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID, visibility string) ([]model.Comment, error) {
	query := `SELECT id, work_item_id, author_id, portal_contact_id, body, visibility, edit_count, created_at, updated_at
		 FROM comments
		 WHERE work_item_id = $1 AND deleted_at IS NULL`
	args := []interface{}{workItemID}

	if visibility != "" {
		query += ` AND visibility = $2`
		args = append(args, visibility)
	}

	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying comments: %w", err)
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		var (
			authorID        uuid.NullUUID
			portalContactID uuid.NullUUID
		)

		if err := rows.Scan(
			&c.ID, &c.WorkItemID, &authorID, &portalContactID,
			&c.Body, &c.Visibility, &c.EditCount, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}

		if authorID.Valid {
			c.AuthorID = &authorID.UUID
		}
		if portalContactID.Valid {
			c.PortalContactID = &portalContactID.UUID
		}

		comments = append(comments, c)
	}

	return comments, rows.Err()
}

// Update modifies a comment's body.
func (r *CommentRepository) Update(ctx context.Context, comment *model.Comment) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE comments SET body = $1, edit_count = edit_count + 1, updated_at = now()
		 WHERE id = $2 AND deleted_at IS NULL`,
		comment.Body, comment.ID)
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

// Delete soft-deletes a comment.
func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE comments SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
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
