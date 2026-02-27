-- Replace non-unique indexes with unique partial indexes for ON CONFLICT upsert.
-- Handles both fresh installs and upgrades from 5-min resolution data.

-- Drop all possible index names (old non-unique + new unique)
DROP INDEX IF EXISTS idx_stats_project_time;
DROP INDEX IF EXISTS idx_stats_project_user_time;
DROP INDEX IF EXISTS idx_stats_project_hour;
DROP INDEX IF EXISTS idx_stats_project_user_hour;

-- Deduplicate: keep only the latest snapshot per (project, hour) for project-level rows.
DELETE FROM project_stats_snapshots a
USING project_stats_snapshots b
WHERE a.user_id IS NULL AND b.user_id IS NULL
  AND a.project_id = b.project_id
  AND date_trunc('hour', a.captured_at) = date_trunc('hour', b.captured_at)
  AND a.captured_at < b.captured_at;

-- Deduplicate: keep only the latest snapshot per (project, user, hour) for per-user rows.
DELETE FROM project_stats_snapshots a
USING project_stats_snapshots b
WHERE a.user_id IS NOT NULL AND b.user_id IS NOT NULL
  AND a.project_id = b.project_id
  AND a.user_id = b.user_id
  AND date_trunc('hour', a.captured_at) = date_trunc('hour', b.captured_at)
  AND a.captured_at < b.captured_at;

-- Truncate remaining timestamps to the hour boundary.
UPDATE project_stats_snapshots
SET captured_at = date_trunc('hour', captured_at)
WHERE captured_at != date_trunc('hour', captured_at);

CREATE UNIQUE INDEX idx_stats_project_hour ON project_stats_snapshots(project_id, captured_at) WHERE user_id IS NULL;
CREATE UNIQUE INDEX idx_stats_project_user_hour ON project_stats_snapshots(project_id, user_id, captured_at) WHERE user_id IS NOT NULL;
