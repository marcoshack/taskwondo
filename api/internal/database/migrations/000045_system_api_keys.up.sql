-- System API keys: make user_id nullable, add type/project_id/created_by columns.

-- 1. Make user_id nullable (system keys have no owning user)
ALTER TABLE api_keys ALTER COLUMN user_id DROP NOT NULL;

-- 2. Add type column (user or system), defaulting to 'user' for backward compat
ALTER TABLE api_keys ADD COLUMN type TEXT NOT NULL DEFAULT 'user'
    CHECK (type IN ('user', 'system'));

-- 3. Add project_id (reserved for future project-scoped keys)
ALTER TABLE api_keys ADD COLUMN project_id UUID REFERENCES projects(id);

-- 4. Add created_by to track which admin created a system key
ALTER TABLE api_keys ADD COLUMN created_by UUID REFERENCES users(id);

-- 5. Index for listing system keys
CREATE INDEX idx_api_keys_type ON api_keys(type) WHERE type = 'system';

-- 6. Add actor_type to work item events to distinguish user vs system key actors
ALTER TABLE work_item_events ADD COLUMN actor_type TEXT NOT NULL DEFAULT 'user'
    CHECK (actor_type IN ('user', 'system_key'));
