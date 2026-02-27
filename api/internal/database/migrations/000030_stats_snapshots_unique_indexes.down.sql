DROP INDEX IF EXISTS idx_stats_project_hour;
DROP INDEX IF EXISTS idx_stats_project_user_hour;

CREATE INDEX idx_stats_project_time ON project_stats_snapshots(project_id, captured_at DESC) WHERE user_id IS NULL;
CREATE INDEX idx_stats_project_user_time ON project_stats_snapshots(project_id, user_id, captured_at DESC) WHERE user_id IS NOT NULL;
