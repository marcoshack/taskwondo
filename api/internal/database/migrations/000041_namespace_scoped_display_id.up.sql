-- Scope the display_id unique constraint to project_id.
-- With namespace-scoped project keys, the same project key (and thus the same
-- display_id pattern like "TF-1") can exist in different namespaces. The old
-- global index would reject these as duplicates.

DROP INDEX IF EXISTS idx_work_items_display_id;
CREATE UNIQUE INDEX idx_work_items_display_id ON work_items(project_id, display_id) WHERE deleted_at IS NULL;
