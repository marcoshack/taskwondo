-- Remove queue FK constraint from work items
ALTER TABLE work_items DROP CONSTRAINT IF EXISTS work_items_queue_id_fkey;

-- Remove milestone_id from work items
DROP INDEX IF EXISTS idx_work_items_milestone;
ALTER TABLE work_items DROP COLUMN IF EXISTS milestone_id;

-- Drop milestones
DROP TABLE IF EXISTS milestones;

-- Drop queues
DROP TABLE IF EXISTS queues;
