-- Escalation list definitions
CREATE TABLE escalation_lists (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name)
);

-- Escalation levels within a list
CREATE TABLE escalation_levels (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    escalation_list_id  UUID NOT NULL REFERENCES escalation_lists(id) ON DELETE CASCADE,
    threshold_pct       INT NOT NULL CHECK (threshold_pct > 0),
    position            INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (escalation_list_id, threshold_pct)
);

-- Users assigned to each escalation level
CREATE TABLE escalation_level_users (
    escalation_level_id UUID NOT NULL REFERENCES escalation_levels(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (escalation_level_id, user_id)
);

-- Per-type escalation list assignment (like type_workflows)
CREATE TABLE type_escalation_lists (
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    work_item_type      TEXT NOT NULL,
    escalation_list_id  UUID NOT NULL REFERENCES escalation_lists(id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, work_item_type)
);
