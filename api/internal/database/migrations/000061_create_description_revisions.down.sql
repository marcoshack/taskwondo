-- TF-350 rollback: remove inline-comment anchor columns and revisions table.

DROP INDEX IF EXISTS idx_comments_anchor_active;

ALTER TABLE comments
    DROP COLUMN IF EXISTS anchor_status,
    DROP COLUMN IF EXISTS anchor_snippet_hash,
    DROP COLUMN IF EXISTS anchor_snippet,
    DROP COLUMN IF EXISTS anchor_end_line,
    DROP COLUMN IF EXISTS anchor_start_line,
    DROP COLUMN IF EXISTS anchor_revision_id;

DROP TABLE IF EXISTS work_item_description_revisions;
