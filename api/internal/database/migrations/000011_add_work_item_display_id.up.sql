-- Add display_id column (e.g. "TF-29") for searchable work item identifiers.
ALTER TABLE work_items ADD COLUMN display_id VARCHAR(20);

-- Backfill existing rows from projects.key + work_items.item_number.
UPDATE work_items SET display_id = p.key || '-' || work_items.item_number
FROM projects p WHERE work_items.project_id = p.id;

-- Make NOT NULL now that all rows are populated.
ALTER TABLE work_items ALTER COLUMN display_id SET NOT NULL;

-- Unique partial index for fast lookups and future friendly URLs (/i/TF-21).
CREATE UNIQUE INDEX idx_work_items_display_id ON work_items(display_id) WHERE deleted_at IS NULL;

-- Rebuild search_vector to include display_id (weight A, 'simple' config to avoid stemming).
-- Must DROP and re-ADD because GENERATED ALWAYS columns cannot be altered in-place.
DROP INDEX IF EXISTS idx_work_items_search;
ALTER TABLE work_items DROP COLUMN search_vector;
ALTER TABLE work_items ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(display_id, '')), 'A')
    ) STORED;
CREATE INDEX idx_work_items_search ON work_items USING GIN(search_vector);
