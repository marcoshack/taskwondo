ALTER TABLE projects DROP COLUMN IF EXISTS default_workflow_id;
DROP TABLE IF EXISTS workflow_transitions;
DROP TABLE IF EXISTS workflow_statuses;
DROP TABLE IF EXISTS workflows;
