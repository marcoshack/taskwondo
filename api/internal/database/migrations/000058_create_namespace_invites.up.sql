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
