-- Revert project_invites to original constraint
ALTER TABLE project_invites DROP CONSTRAINT IF EXISTS project_invites_role_check;
ALTER TABLE project_invites ADD CONSTRAINT project_invites_role_check CHECK (role IN ('admin', 'member', 'viewer'));

-- Revert project_members to original constraint
ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_role_check;
ALTER TABLE project_members ADD CONSTRAINT project_members_role_check CHECK (role IN ('owner', 'admin', 'member', 'viewer'));
