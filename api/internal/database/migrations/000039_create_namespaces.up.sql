-- Namespaces table
CREATE TABLE namespaces (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug         TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    is_default   BOOLEAN NOT NULL DEFAULT false,
    created_by   UUID NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ensure only one default namespace exists
CREATE UNIQUE INDEX idx_namespaces_default ON namespaces (is_default) WHERE is_default = true;

-- Namespace members (owner/admin roles for namespace management)
CREATE TABLE namespace_members (
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    user_id      UUID NOT NULL REFERENCES users(id),
    role         TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (namespace_id, user_id)
);

CREATE INDEX idx_namespace_members_user ON namespace_members(user_id);

-- Add namespace_id to projects (nullable initially for backfill)
ALTER TABLE projects ADD COLUMN namespace_id UUID REFERENCES namespaces(id);

CREATE INDEX idx_projects_namespace ON projects(namespace_id);
