-- Add FTS search_vector columns to teams, queues, and milestones tables.
-- Generated tsvector columns auto-populate for existing rows.

-- Teams: index name (weight A) + description (weight B)
ALTER TABLE teams ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_teams_search ON teams USING GIN(search_vector);

-- Queues: index name (weight A) + description (weight B)
ALTER TABLE queues ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_queues_search ON queues USING GIN(search_vector);

-- Milestones: index name (weight A) + description (weight B)
ALTER TABLE milestones ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_milestones_search ON milestones USING GIN(search_vector);
