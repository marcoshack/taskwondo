-- Restore the global display_id unique constraint.
DROP INDEX IF EXISTS idx_work_items_display_id;
CREATE UNIQUE INDEX idx_work_items_display_id ON work_items(display_id) WHERE deleted_at IS NULL;
