CREATE TABLE oncall_overrides (
    id               UUID        PRIMARY KEY,
    rotation_id      UUID        NOT NULL REFERENCES oncall_rotations(id) ON DELETE CASCADE,
    override_user_id UUID        NOT NULL REFERENCES users(id),
    start_at         TIMESTAMPTZ NOT NULL,
    end_at           TIMESTAMPTZ NOT NULL,
    reason           TEXT,
    created_by       UUID        NOT NULL REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT oncall_overrides_start_before_end CHECK (start_at < end_at)
);

CREATE INDEX idx_oncall_overrides_rotation_active
    ON oncall_overrides (rotation_id, start_at, end_at);
