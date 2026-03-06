-- Remove backlog → in_progress transitions added by migration 38.

DELETE FROM workflow_transitions
WHERE from_status = 'backlog'
  AND to_status = 'in_progress'
  AND name = 'Start Work';
