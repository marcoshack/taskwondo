-- Replace the global project key uniqueness constraint with a namespace-scoped one.
-- This allows the same project key to exist in different namespaces.

-- Drop the old global unique index
DROP INDEX IF EXISTS idx_projects_key;

-- Create a namespace-scoped unique index (key is unique within a namespace)
CREATE UNIQUE INDEX idx_projects_namespace_key ON projects(namespace_id, key) WHERE deleted_at IS NULL;
