-- Remove transitions involving backlog.
DELETE FROM workflow_transitions
WHERE from_status = 'backlog' OR to_status = 'backlog';

-- Retarget "in_progress → backlog" back to "in_progress → open".
-- (Already deleted above, re-insert original.)
INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT gen_random_uuid(), ws.workflow_id, 'in_progress', 'open', 'Move to Backlog'
FROM workflow_statuses ws WHERE ws.name = 'open'
  AND ws.workflow_id NOT IN (
    SELECT workflow_id FROM workflow_transitions
    WHERE from_status = 'in_progress' AND to_status = 'open'
  )
ON CONFLICT (workflow_id, from_status, to_status) DO NOTHING;

-- Re-insert "done → open" (Reopen).
INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT gen_random_uuid(), ws.workflow_id, 'done', 'open', 'Reopen'
FROM workflow_statuses ws WHERE ws.name = 'open'
  AND ws.workflow_id NOT IN (
    SELECT workflow_id FROM workflow_transitions
    WHERE from_status = 'done' AND to_status = 'open'
  )
ON CONFLICT (workflow_id, from_status, to_status) DO NOTHING;

-- Remove backlog status.
DELETE FROM workflow_statuses WHERE name = 'backlog';

-- Shift positions back down by 1.
UPDATE workflow_statuses
SET position = position - 1
WHERE workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'open'
);
