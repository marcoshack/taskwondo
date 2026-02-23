-- Swap backlog and open positions so "open" (position 0) remains the initial status
-- for new work items, while "backlog" (position 1) is available for triage.

-- Temporarily set backlog to -1 to avoid unique constraint conflicts during swap.
UPDATE workflow_statuses
SET position = -1
WHERE name = 'backlog';

-- Move open to position 0.
UPDATE workflow_statuses
SET position = 0
WHERE name = 'open'
  AND workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'backlog'
  );

-- Move backlog to position 1.
UPDATE workflow_statuses
SET position = 1
WHERE name = 'backlog';
