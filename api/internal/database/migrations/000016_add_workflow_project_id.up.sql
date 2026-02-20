-- Add project_id to workflows so projects can define custom workflows.
-- NULL project_id = system-wide workflow, non-NULL = project-scoped workflow.
ALTER TABLE workflows ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE CASCADE;

CREATE INDEX idx_workflows_project ON workflows(project_id);
