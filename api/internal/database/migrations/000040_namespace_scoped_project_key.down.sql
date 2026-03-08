-- Revert to global project key uniqueness
DROP INDEX IF EXISTS idx_projects_namespace_key;
CREATE UNIQUE INDEX idx_projects_key ON projects(key) WHERE deleted_at IS NULL;
