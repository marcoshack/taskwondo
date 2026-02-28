CREATE TABLE work_item_watchers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_by     UUID NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (work_item_id, user_id)
);

CREATE INDEX idx_watchers_work_item ON work_item_watchers(work_item_id);
CREATE INDEX idx_watchers_user ON work_item_watchers(user_id);
