-- Add missing backlog → in_progress transition to task workflows.
-- After TF-246 swapped backlog to position 0, items now start at backlog
-- and need a direct path to in_progress.

INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT gen_random_uuid(), w.id, 'backlog', 'in_progress', 'Start Work'
FROM workflows w
JOIN workflow_statuses b ON b.workflow_id = w.id AND b.name = 'backlog'
JOIN workflow_statuses ip ON ip.workflow_id = w.id AND ip.name = 'in_progress'
WHERE NOT EXISTS (
    SELECT 1 FROM workflow_transitions t
    WHERE t.workflow_id = w.id
      AND t.from_status = 'backlog'
      AND t.to_status = 'in_progress'
);
