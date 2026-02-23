CREATE TABLE project_invites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id),
    code        TEXT NOT NULL UNIQUE,
    role        TEXT NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
    created_by  UUID NOT NULL REFERENCES users(id),
    expires_at  TIMESTAMPTZ,
    max_uses    INT NOT NULL DEFAULT 0,
    use_count   INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_project_invites_project ON project_invites(project_id);
