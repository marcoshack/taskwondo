DROP INDEX IF EXISTS idx_workflows_project;
ALTER TABLE workflows DROP COLUMN IF EXISTS project_id;
