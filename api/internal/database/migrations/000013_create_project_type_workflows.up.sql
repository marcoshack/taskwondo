CREATE TABLE project_type_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    work_item_type VARCHAR(20) NOT NULL,
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, work_item_type)
);

CREATE INDEX idx_ptw_project ON project_type_workflows(project_id);
