-- TF-350 rollback: remove sub-line anchor columns and reply threading.

DROP INDEX IF EXISTS idx_comments_parent;

ALTER TABLE comments
    DROP COLUMN IF EXISTS parent_comment_id,
    DROP COLUMN IF EXISTS anchor_end_col,
    DROP COLUMN IF EXISTS anchor_start_col;
