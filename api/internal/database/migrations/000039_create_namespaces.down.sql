DROP INDEX IF EXISTS idx_projects_namespace;
ALTER TABLE projects DROP COLUMN IF EXISTS namespace_id;
DROP TABLE IF EXISTS namespace_members;
DROP INDEX IF EXISTS idx_namespaces_default;
DROP TABLE IF EXISTS namespaces;
