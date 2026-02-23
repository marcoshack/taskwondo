-- Add "backlog" status to task-type workflows (identified by having an "open" status).

-- Shift existing status positions up by 1 to make room at position 0.
UPDATE workflow_statuses
SET position = position + 1
WHERE workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'open'
);

-- Insert "backlog" status at position 0 for each task workflow.
INSERT INTO workflow_statuses (id, workflow_id, name, display_name, category, position)
SELECT gen_random_uuid(), ws.workflow_id, 'backlog', 'Backlog', 'todo', 0
FROM workflow_statuses ws
WHERE ws.name = 'open'
ON CONFLICT (workflow_id, name) DO NOTHING;

-- Retarget "in_progress → open" (named "Move to Backlog") to point to actual backlog.
UPDATE workflow_transitions
SET to_status = 'backlog'
WHERE from_status = 'in_progress' AND to_status = 'open' AND name = 'Move to Backlog'
  AND workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'backlog'
  );

-- Retarget "done → open" (Reopen) to go to backlog instead.
UPDATE workflow_transitions
SET to_status = 'backlog'
WHERE from_status = 'done' AND to_status = 'open' AND name = 'Reopen'
  AND workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'backlog'
  );

-- Add new transitions involving backlog.
INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT gen_random_uuid(), ws.workflow_id, 'backlog', 'open', 'Prioritize'
FROM workflow_statuses ws WHERE ws.name = 'backlog'
ON CONFLICT (workflow_id, from_status, to_status) DO NOTHING;

INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT gen_random_uuid(), ws.workflow_id, 'backlog', 'cancelled', 'Cancel'
FROM workflow_statuses ws WHERE ws.name = 'backlog'
ON CONFLICT (workflow_id, from_status, to_status) DO NOTHING;

INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT gen_random_uuid(), ws.workflow_id, 'open', 'backlog', 'Deprioritize'
FROM workflow_statuses ws WHERE ws.name = 'backlog'
ON CONFLICT (workflow_id, from_status, to_status) DO NOTHING;
