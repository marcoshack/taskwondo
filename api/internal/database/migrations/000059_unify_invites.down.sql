CREATE TABLE project_invites (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID NOT NULL REFERENCES projects(id),
    code           TEXT NOT NULL UNIQUE,
    role           TEXT NOT NULL CHECK (role IN ('admin', 'member', 'viewer', 'customer')),
    created_by     UUID NOT NULL REFERENCES users(id),
    invitee_email  TEXT,
    expires_at     TIMESTAMPTZ,
    max_uses       INT NOT NULL DEFAULT 0,
    use_count      INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_project_invites_project ON project_invites(project_id);

CREATE TABLE namespace_invites (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id   UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    code           TEXT NOT NULL UNIQUE,
    role           TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_by     UUID NOT NULL REFERENCES users(id),
    invitee_email  TEXT,
    expires_at     TIMESTAMPTZ,
    max_uses       INT NOT NULL DEFAULT 0,
    use_count      INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_namespace_invites_namespace ON namespace_invites(namespace_id);

INSERT INTO project_invites (id, project_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at)
SELECT id, project_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
FROM invites
WHERE project_id IS NOT NULL;

INSERT INTO namespace_invites (id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at)
SELECT id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
FROM invites
WHERE namespace_id IS NOT NULL;

DROP TABLE invites;
