DROP INDEX IF EXISTS idx_work_items_sla_target_at;
ALTER TABLE work_items DROP COLUMN IF EXISTS sla_target_at;
