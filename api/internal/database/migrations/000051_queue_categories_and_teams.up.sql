CREATE TABLE queue_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_id    UUID NOT NULL REFERENCES queues(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (queue_id, name)
);

CREATE TABLE queue_teams (
    queue_id    UUID NOT NULL REFERENCES queues(id) ON DELETE CASCADE,
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (queue_id, team_id)
);

ALTER TABLE work_items ADD COLUMN category_id UUID REFERENCES queue_categories(id) ON DELETE SET NULL;
