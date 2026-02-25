CREATE TABLE saved_searches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    filters     JSONB NOT NULL DEFAULT '{}',
    view_mode   VARCHAR(10) NOT NULL DEFAULT 'list',
    position    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_saved_searches_user ON saved_searches(project_id, user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_saved_searches_shared ON saved_searches(project_id) WHERE user_id IS NULL;
