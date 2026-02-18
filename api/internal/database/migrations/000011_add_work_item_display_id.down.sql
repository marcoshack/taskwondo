-- Remove display_id and restore original search_vector definition.
DROP INDEX IF EXISTS idx_work_items_display_id;
DROP INDEX IF EXISTS idx_work_items_search;

ALTER TABLE work_items DROP COLUMN search_vector;
ALTER TABLE work_items DROP COLUMN display_id;

-- Restore original search_vector without display_id.
ALTER TABLE work_items ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_work_items_search ON work_items USING GIN(search_vector);
