-- Phase: File Attachments

CREATE TABLE attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id UUID NOT NULL REFERENCES work_items(id),
    uploader_id  UUID NOT NULL REFERENCES users(id),
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes   BIGINT NOT NULL,
    storage_key  TEXT NOT NULL,
    comment      TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_attachments_work_item ON attachments(work_item_id, created_at)
    WHERE deleted_at IS NULL;
