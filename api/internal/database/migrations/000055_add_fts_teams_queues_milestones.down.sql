DROP INDEX IF EXISTS idx_milestones_search;
ALTER TABLE milestones DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_queues_search;
ALTER TABLE queues DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_teams_search;
ALTER TABLE teams DROP COLUMN IF EXISTS search_vector;
