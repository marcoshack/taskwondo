-- Fix backlog position: ensure backlog comes before open in task-type workflows.
-- Swaps positions of backlog and open where backlog has a higher position than open.

WITH swap AS (
    SELECT
        b.id AS backlog_id, b.position AS backlog_pos,
        o.id AS open_id, o.position AS open_pos
    FROM workflow_statuses b
    JOIN workflow_statuses o ON b.workflow_id = o.workflow_id
    WHERE b.name = 'backlog' AND o.name = 'open'
      AND b.position > o.position
)
UPDATE workflow_statuses ws
SET position = CASE
    WHEN ws.id = swap.backlog_id THEN swap.open_pos
    WHEN ws.id = swap.open_id THEN swap.backlog_pos
END
FROM swap
WHERE ws.id IN (swap.backlog_id, swap.open_id);
