-- Phase 6: Workflows — status definitions and transition rules for work items.

-- workflows table
CREATE TABLE workflows (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- workflow_statuses table
CREATE TABLE workflow_statuses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id  UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    display_name TEXT NOT NULL,
    category     TEXT NOT NULL CHECK (category IN ('todo', 'in_progress', 'done', 'cancelled')),
    position     INTEGER NOT NULL,
    color        TEXT,

    UNIQUE (workflow_id, name)
);

CREATE INDEX idx_workflow_statuses_workflow ON workflow_statuses(workflow_id, position);

-- workflow_transitions table
CREATE TABLE workflow_transitions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    from_status TEXT NOT NULL,
    to_status   TEXT NOT NULL,
    name        TEXT,

    UNIQUE (workflow_id, from_status, to_status)
);

CREATE INDEX idx_workflow_transitions_workflow ON workflow_transitions(workflow_id);

-- Add default_workflow_id to projects (nullable for now; seed sets it)
ALTER TABLE projects ADD COLUMN default_workflow_id UUID REFERENCES workflows(id);
