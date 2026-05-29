-- Add direct-to-done transitions in the Task Workflow so items can be
-- completed without requiring a formal in_review step.
-- Also adds open → done for the same reason.

INSERT INTO workflow_transitions (id, workflow_id, from_status, to_status, name)
SELECT
    gen_random_uuid(),
    w.id,
    t.from_status,
    t.to_status,
    t.name
FROM workflows w
CROSS JOIN (VALUES
    ('in_progress', 'done',  'Complete'),
    ('open',        'done',  'Complete')
) AS t(from_status, to_status, name)
WHERE w.name = 'Task Workflow'
  AND w.is_default = true
ON CONFLICT (workflow_id, from_status, to_status) DO NOTHING;
