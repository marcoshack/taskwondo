package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
)

// AttachmentRepository handles attachment metadata persistence.
type AttachmentRepository struct {
	db *sql.DB
}

// NewAttachmentRepository creates a new AttachmentRepository.
func NewAttachmentRepository(db *sql.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

// Create inserts a new attachment record.
func (r *AttachmentRepository) Create(ctx context.Context, a *model.Attachment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO attachments (id, work_item_id, uploader_id, filename, content_type, size_bytes, storage_key, comment)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.WorkItemID, a.UploaderID, a.Filename, a.ContentType, a.SizeBytes, a.StorageKey, a.Comment)
	if err != nil {
		return fmt.Errorf("inserting attachment: %w", err)
	}
	return nil
}

// GetByID returns an attachment by its ID (non-deleted).
func (r *AttachmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Attachment, error) {
	var a model.Attachment
	err := r.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, uploader_id, filename, content_type, size_bytes, storage_key, comment, created_at
		 FROM attachments
		 WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&a.ID, &a.WorkItemID, &a.UploaderID, &a.Filename, &a.ContentType,
		&a.SizeBytes, &a.StorageKey, &a.Comment, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying attachment: %w", err)
	}
	return &a, nil
}

// ListByWorkItem returns all non-deleted attachments for a work item, ordered by creation time.
func (r *AttachmentRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.Attachment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, work_item_id, uploader_id, filename, content_type, size_bytes, storage_key, comment, created_at
		 FROM attachments
		 WHERE work_item_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at ASC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying attachments: %w", err)
	}
	defer rows.Close()

	var attachments []model.Attachment
	for rows.Next() {
		var a model.Attachment
		if err := rows.Scan(
			&a.ID, &a.WorkItemID, &a.UploaderID, &a.Filename, &a.ContentType,
			&a.SizeBytes, &a.StorageKey, &a.Comment, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning attachment: %w", err)
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

// Delete soft-deletes an attachment.
func (r *AttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE attachments SET deleted_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting attachment: %w", err)
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
