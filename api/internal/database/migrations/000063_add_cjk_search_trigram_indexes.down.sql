-- Drop the trigram indexes added for the CJK search fallback.
-- The pg_trgm extension itself is left in place: dropping it could break
-- unrelated objects on installs that had it before this migration.

DROP INDEX IF EXISTS idx_milestones_description_trgm;
DROP INDEX IF EXISTS idx_milestones_name_trgm;
DROP INDEX IF EXISTS idx_queues_description_trgm;
DROP INDEX IF EXISTS idx_queues_name_trgm;
DROP INDEX IF EXISTS idx_teams_description_trgm;
DROP INDEX IF EXISTS idx_teams_name_trgm;
DROP INDEX IF EXISTS idx_work_items_description_trgm;
DROP INDEX IF EXISTS idx_work_items_title_trgm;
