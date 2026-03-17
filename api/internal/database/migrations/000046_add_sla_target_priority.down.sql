-- Remove the priority check constraint
ALTER TABLE sla_status_targets DROP CONSTRAINT IF EXISTS sla_status_targets_priority_check;

-- Drop the new unique constraint
ALTER TABLE sla_status_targets DROP CONSTRAINT IF EXISTS sla_status_targets_project_type_workflow_status_priority_key;

-- Delete duplicated priority rows (keep only 'medium')
DELETE FROM sla_status_targets WHERE priority != 'medium';

-- Drop the priority column
ALTER TABLE sla_status_targets DROP COLUMN IF EXISTS priority;

-- Re-add original unique constraint
ALTER TABLE sla_status_targets ADD CONSTRAINT sla_status_targets_project_id_work_item_type_workflow_id_st_key
  UNIQUE(project_id, work_item_type, workflow_id, status_name);
