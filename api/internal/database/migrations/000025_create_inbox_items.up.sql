CREATE TABLE inbox_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_item_id  UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    position      INTEGER NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, work_item_id)
);

CREATE INDEX idx_inbox_items_user_position ON inbox_items(user_id, position);
