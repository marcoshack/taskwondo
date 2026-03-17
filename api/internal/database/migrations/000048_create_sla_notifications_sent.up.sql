CREATE TABLE sla_notifications_sent (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id    UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    status_name     TEXT NOT NULL,
    escalation_level INT NOT NULL,
    threshold_pct   INT NOT NULL,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(work_item_id, status_name, escalation_level, threshold_pct)
);

CREATE INDEX idx_sla_notifications_sent_work_item
    ON sla_notifications_sent(work_item_id, status_name);
