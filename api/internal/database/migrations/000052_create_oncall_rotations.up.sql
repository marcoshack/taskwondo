CREATE TABLE oncall_rotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL UNIQUE REFERENCES teams(id) ON DELETE CASCADE,
    period_days INTEGER NOT NULL DEFAULT 7,
    rotation_time TIME NOT NULL DEFAULT '12:00:00',
    timezone TEXT NOT NULL DEFAULT 'UTC',
    start_date DATE NOT NULL,
    current_user_id UUID REFERENCES users(id),
    current_position INTEGER NOT NULL DEFAULT 0,
    next_rotation_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE oncall_rotation_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rotation_id UUID NOT NULL REFERENCES oncall_rotations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    position INTEGER NOT NULL,
    UNIQUE (rotation_id, user_id)
);

CREATE TABLE oncall_rotation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rotation_id UUID NOT NULL REFERENCES oncall_rotations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
