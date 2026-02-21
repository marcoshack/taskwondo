ALTER TABLE work_items ADD COLUMN sla_target_at TIMESTAMPTZ;
CREATE INDEX idx_work_items_sla_target_at ON work_items (sla_target_at)
  WHERE sla_target_at IS NOT NULL AND deleted_at IS NULL;
