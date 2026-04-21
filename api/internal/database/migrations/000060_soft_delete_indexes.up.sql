-- Add partial indexes filtered by `deleted_at IS NULL` for tables that support
-- soft delete but whose existing foreign-key / lookup indexes do not include
-- the predicate. Queries that filter `WHERE ... AND deleted_at IS NULL` can
-- use these partial indexes to skip tombstoned rows without scanning them.
--
-- The older non-partial indexes are intentionally left in place so that
-- follow-up queries that do *not* filter soft-delete (e.g. admin undelete
-- flows) continue to benefit. A later cleanup migration can drop them if
-- redundancy becomes a storage concern.

CREATE INDEX IF NOT EXISTS idx_comments_work_item_active
    ON comments(work_item_id, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_attachments_work_item_active
    ON attachments(work_item_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_time_entries_work_item_active
    ON time_entries(work_item_id, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_time_entries_user_active
    ON time_entries(user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_projects_namespace_active
    ON projects(namespace_id)
    WHERE deleted_at IS NULL;

-- Direct indexes on deleted_at for audit / undelete flows that need to find
-- soft-deleted rows. Partial so the index only covers tombstoned rows (a
-- small fraction of each table).
CREATE INDEX IF NOT EXISTS idx_work_items_deleted
    ON work_items(deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_projects_deleted
    ON projects(deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_comments_deleted
    ON comments(deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_attachments_deleted
    ON attachments(deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_time_entries_deleted
    ON time_entries(deleted_at)
    WHERE deleted_at IS NOT NULL;
