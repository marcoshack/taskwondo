-- Merge project_invites and namespace_invites into a single invites table.
-- Exactly one of project_id / namespace_id is set per row; they cannot both
-- be set and cannot both be null.

CREATE TABLE invites (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID REFERENCES projects(id) ON DELETE CASCADE,
    namespace_id   UUID REFERENCES namespaces(id) ON DELETE CASCADE,
    code           TEXT NOT NULL UNIQUE,
    role           TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer', 'customer')),
    created_by     UUID NOT NULL REFERENCES users(id),
    invitee_email  TEXT,
    expires_at     TIMESTAMPTZ,
    max_uses       INT NOT NULL DEFAULT 0,
    use_count      INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT invites_target_xor CHECK (
        (project_id IS NOT NULL AND namespace_id IS NULL) OR
        (project_id IS NULL AND namespace_id IS NOT NULL)
    )
);

CREATE INDEX idx_invites_project ON invites(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_invites_namespace ON invites(namespace_id) WHERE namespace_id IS NOT NULL;

INSERT INTO invites (id, project_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at)
SELECT id, project_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
FROM project_invites;

INSERT INTO invites (id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at)
SELECT id, namespace_id, code, role, created_by, invitee_email, expires_at, max_uses, use_count, created_at
FROM namespace_invites;

DROP TABLE project_invites;
DROP TABLE namespace_invites;
