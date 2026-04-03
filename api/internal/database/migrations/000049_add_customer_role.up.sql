-- Add 'customer' role to project_members
ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_role_check;
ALTER TABLE project_members ADD CONSTRAINT project_members_role_check CHECK (role IN ('owner', 'admin', 'member', 'viewer', 'customer'));

-- Add 'customer' role to project_invites
ALTER TABLE project_invites DROP CONSTRAINT IF EXISTS project_invites_role_check;
ALTER TABLE project_invites ADD CONSTRAINT project_invites_role_check CHECK (role IN ('admin', 'member', 'viewer', 'customer'));
