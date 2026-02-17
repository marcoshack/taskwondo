-- Queues: inbound work channels within a project
CREATE TABLE queues (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id),
    name                TEXT NOT NULL,
    description         TEXT,
    queue_type          TEXT NOT NULL CHECK (queue_type IN ('support', 'alerts', 'feedback', 'general')),
    is_public           BOOLEAN NOT NULL DEFAULT false,
    default_priority    TEXT NOT NULL DEFAULT 'medium',
    default_assignee_id UUID REFERENCES users(id),
    workflow_id         UUID REFERENCES workflows(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (project_id, name)
);

CREATE INDEX idx_queues_project ON queues(project_id);

-- Milestones: progress tracking within a project
CREATE TABLE milestones (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id),
    name        TEXT NOT NULL,
    description TEXT,
    due_date    DATE,
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_milestones_project ON milestones(project_id);

-- Add milestone_id to work items
ALTER TABLE work_items ADD COLUMN milestone_id UUID REFERENCES milestones(id);
CREATE INDEX idx_work_items_milestone ON work_items(milestone_id) WHERE deleted_at IS NULL;

-- Add FK constraint for queue_id (column already exists from Phase 4)
ALTER TABLE work_items ADD CONSTRAINT work_items_queue_id_fkey FOREIGN KEY (queue_id) REFERENCES queues(id);
