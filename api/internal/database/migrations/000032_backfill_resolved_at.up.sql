-- Backfill resolved_at for work items in done/cancelled statuses that were
-- resolved before the service layer started managing resolved_at on transitions.
-- Uses the latest status_changed event timestamp, falling back to updated_at.
UPDATE work_items wi
SET resolved_at = COALESCE(
    (SELECT MAX(wie.created_at)
     FROM work_item_events wie
     WHERE wie.work_item_id = wi.id
       AND wie.event_type = 'status_changed'),
    wi.updated_at
)
WHERE wi.deleted_at IS NULL
  AND wi.resolved_at IS NULL
  AND wi.status IN (
    SELECT ws.name
    FROM workflow_statuses ws
    WHERE ws.category IN ('done', 'cancelled')
  );
