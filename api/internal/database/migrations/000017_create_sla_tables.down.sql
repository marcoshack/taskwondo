ALTER TABLE work_items ADD COLUMN sla_deadline TIMESTAMPTZ;

ALTER TABLE projects DROP COLUMN business_hours;

DROP TABLE IF EXISTS work_item_sla_elapsed;
DROP TABLE IF EXISTS sla_status_targets;
