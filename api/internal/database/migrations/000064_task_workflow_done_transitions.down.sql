-- Revert: remove the direct-to-done transitions added for the Task Workflow.

DELETE FROM workflow_transitions
WHERE workflow_id = (
    SELECT id FROM workflows WHERE name = 'Task Workflow' AND is_default = true LIMIT 1
)
AND (from_status, to_status) IN (
    ('in_progress', 'done'),
    ('open',        'done')
);
