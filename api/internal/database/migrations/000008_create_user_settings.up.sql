CREATE TABLE user_settings (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    key        TEXT NOT NULL,
    value      JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Composite unique constraint using COALESCE for nullable project_id
CREATE UNIQUE INDEX uq_user_settings_key
    ON user_settings (user_id, COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid), key);

CREATE INDEX idx_user_settings_user_project
    ON user_settings (user_id, project_id);
