-- Time tracking: time entries table + estimated_seconds on work items

CREATE TABLE time_entries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id     UUID NOT NULL REFERENCES work_items(id),
    user_id          UUID NOT NULL REFERENCES users(id),
    started_at       TIMESTAMPTZ NOT NULL,
    duration_seconds INT NOT NULL CHECK (duration_seconds > 0),
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_time_entries_work_item ON time_entries(work_item_id, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_time_entries_user ON time_entries(user_id)
    WHERE deleted_at IS NULL;

ALTER TABLE work_items ADD COLUMN estimated_seconds INT;
