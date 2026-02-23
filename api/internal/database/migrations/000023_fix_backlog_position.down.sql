-- Revert: put backlog back to position 0 and open to position 1.

UPDATE workflow_statuses
SET position = -1
WHERE name = 'open'
  AND workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'backlog'
  );

UPDATE workflow_statuses
SET position = 0
WHERE name = 'backlog';

UPDATE workflow_statuses
SET position = 1
WHERE name = 'open'
  AND workflow_id IN (
    SELECT DISTINCT workflow_id FROM workflow_statuses WHERE name = 'backlog'
  );
