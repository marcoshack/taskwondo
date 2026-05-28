ALTER TABLE work_items DROP CONSTRAINT work_items_type_check;
ALTER TABLE work_items ADD CONSTRAINT work_items_type_check CHECK (type IN ('task', 'ticket', 'bug', 'feedback', 'epic'));
