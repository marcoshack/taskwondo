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

-- Project-level queries: WHERE project_id = $1 AND user_id IS NULL ORDER BY captured_at DESC
CREATE INDEX idx_stats_project_time ON project_stats_snapshots(project_id, captured_at DESC) WHERE user_id IS NULL;

-- Per-user queries: WHERE project_id = $1 AND user_id = $2 ORDER BY captured_at DESC
CREATE INDEX idx_stats_project_user_time ON project_stats_snapshots(project_id, user_id, captured_at DESC) WHERE user_id IS NOT NULL;
