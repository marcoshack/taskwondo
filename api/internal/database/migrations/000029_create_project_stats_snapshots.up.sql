CREATE TABLE project_stats_snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    user_id         UUID REFERENCES users(id),
    todo_count        INT NOT NULL,
    in_progress_count INT NOT NULL,
    done_count        INT NOT NULL,
    cancelled_count INT NOT NULL,
    captured_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique partial indexes enable ON CONFLICT upsert into hourly buckets.
-- Project-level (user_id IS NULL): one row per (project, hour)
CREATE UNIQUE INDEX idx_stats_project_hour ON project_stats_snapshots(project_id, captured_at) WHERE user_id IS NULL;

-- Per-user (user_id IS NOT NULL): one row per (project, user, hour)
CREATE UNIQUE INDEX idx_stats_project_user_hour ON project_stats_snapshots(project_id, user_id, captured_at) WHERE user_id IS NOT NULL;
