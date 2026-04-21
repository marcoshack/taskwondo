DROP INDEX IF EXISTS idx_time_entries_deleted;
DROP INDEX IF EXISTS idx_attachments_deleted;
DROP INDEX IF EXISTS idx_comments_deleted;
DROP INDEX IF EXISTS idx_projects_deleted;
DROP INDEX IF EXISTS idx_work_items_deleted;

DROP INDEX IF EXISTS idx_projects_namespace_active;
DROP INDEX IF EXISTS idx_time_entries_user_active;
DROP INDEX IF EXISTS idx_time_entries_work_item_active;
DROP INDEX IF EXISTS idx_attachments_work_item_active;
DROP INDEX IF EXISTS idx_comments_work_item_active;
