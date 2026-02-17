-- Work items: the central entity for tasks, tickets, bugs, feedback, and epics.
CREATE TABLE work_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id),
    queue_id            UUID,                                        -- FK deferred to Phase 7 (queues table)
    parent_id           UUID REFERENCES work_items(id),              -- self-referential for sub-tasks
    item_number         INTEGER NOT NULL,                            -- sequential within project
    type                TEXT NOT NULL CHECK (type IN ('task', 'ticket', 'bug', 'feedback', 'epic')),
    title               TEXT NOT NULL,
    description         TEXT,                                        -- markdown
    status              TEXT NOT NULL DEFAULT 'open',
    priority            TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    assignee_id         UUID REFERENCES users(id),
    reporter_id         UUID NOT NULL REFERENCES users(id),
    portal_contact_id   UUID,                                        -- FK deferred to Phase 10 (portal_contacts table)
    visibility          TEXT NOT NULL DEFAULT 'internal' CHECK (visibility IN ('internal', 'portal', 'public')),
    labels              TEXT[] NOT NULL DEFAULT '{}',
    custom_fields       JSONB NOT NULL DEFAULT '{}',
    due_date            DATE,
    sla_deadline        TIMESTAMPTZ,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,

    UNIQUE (project_id, item_number)
);

-- Core indexes
CREATE INDEX idx_work_items_project_status ON work_items(project_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_assignee ON work_items(assignee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_type ON work_items(project_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_queue ON work_items(queue_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_parent ON work_items(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_labels ON work_items USING GIN(labels) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_created ON work_items(project_id, created_at DESC) WHERE deleted_at IS NULL;

-- Full-text search
ALTER TABLE work_items ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_work_items_search ON work_items USING GIN(search_vector);

-- Work item events: audit trail for every state change.
CREATE TABLE work_item_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id    UUID NOT NULL REFERENCES work_items(id),
    actor_id        UUID REFERENCES users(id),                   -- null for system/automation events
    event_type      TEXT NOT NULL,
    field_name      TEXT,                                        -- which field changed
    old_value       TEXT,                                        -- previous value (serialized)
    new_value       TEXT,                                        -- new value (serialized)
    metadata        JSONB NOT NULL DEFAULT '{}',
    visibility      TEXT NOT NULL DEFAULT 'internal' CHECK (visibility IN ('internal', 'public')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_work_item ON work_item_events(work_item_id, created_at);
CREATE INDEX idx_events_type ON work_item_events(event_type, created_at);
