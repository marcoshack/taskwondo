-- Add priority column to SLA status targets
ALTER TABLE sla_status_targets ADD COLUMN priority TEXT NOT NULL DEFAULT 'medium';

-- Expand existing rows: for each existing target, create 3 more rows (one per remaining priority)
-- keeping the same target_seconds and calendar_mode.
-- The existing row already has priority='medium' from the DEFAULT above.
INSERT INTO sla_status_targets (project_id, work_item_type, workflow_id, status_name, target_seconds, calendar_mode, priority)
SELECT project_id, work_item_type, workflow_id, status_name, target_seconds, calendar_mode, p.priority
FROM sla_status_targets
CROSS JOIN (VALUES ('critical'), ('high'), ('low')) AS p(priority)
WHERE sla_status_targets.priority = 'medium'
ON CONFLICT DO NOTHING;

-- Drop old unique constraint and create new one that includes priority
ALTER TABLE sla_status_targets DROP CONSTRAINT sla_status_targets_project_id_work_item_type_workflow_id_st_key;
ALTER TABLE sla_status_targets ADD CONSTRAINT sla_status_targets_project_type_workflow_status_priority_key
  UNIQUE(project_id, work_item_type, workflow_id, status_name, priority);

-- Remove the default now that migration is done
ALTER TABLE sla_status_targets ALTER COLUMN priority DROP DEFAULT;

-- Add check constraint for valid priorities
ALTER TABLE sla_status_targets ADD CONSTRAINT sla_status_targets_priority_check
  CHECK (priority IN ('critical', 'high', 'medium', 'low'));
