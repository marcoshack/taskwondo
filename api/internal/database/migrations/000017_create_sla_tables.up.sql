-- SLA targets: max time per status, scoped by project + type + workflow
CREATE TABLE sla_status_targets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    work_item_type  TEXT NOT NULL,
    workflow_id     UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status_name     TEXT NOT NULL,
    target_seconds  INT NOT NULL CHECK (target_seconds > 0),
    calendar_mode   TEXT NOT NULL DEFAULT '24x7' CHECK (calendar_mode IN ('24x7', 'business_hours')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, work_item_type, workflow_id, status_name)
);

CREATE INDEX idx_sla_status_targets_project ON sla_status_targets(project_id);

-- Elapsed time per work item per status (anti-gaming accumulation)
CREATE TABLE work_item_sla_elapsed (
    work_item_id    UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    status_name     TEXT NOT NULL,
    elapsed_seconds INT NOT NULL DEFAULT 0,
    last_entered_at TIMESTAMPTZ,
    PRIMARY KEY (work_item_id, status_name)
);

-- Business hours config on projects
ALTER TABLE projects ADD COLUMN business_hours JSONB;

-- Drop unused sla_deadline (redundant with due_date)
ALTER TABLE work_items DROP COLUMN sla_deadline;
