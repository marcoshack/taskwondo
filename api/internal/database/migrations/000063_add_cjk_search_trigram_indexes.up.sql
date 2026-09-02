-- CJK search support.
--
-- PostgreSQL's text-search configs tokenize on whitespace, so a run of
-- Chinese/Japanese/Korean characters collapses into a single tsvector lexeme
-- and a tsquery can never match it as a substring. The repositories add an
-- ILIKE substring fallback when the query contains CJK characters; these
-- pg_trgm GIN indexes make that fallback indexable.
--
-- pg_trgm is optional (same resilience pattern as 000033/000061): when the
-- extension is unavailable — or cannot be created, e.g. a managed/external
-- Postgres without privileges on it — the indexes are skipped and the search
-- still works, just with a sequential scan for the ILIKE fallback.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_trgm') THEN
        BEGIN
            EXECUTE 'CREATE EXTENSION IF NOT EXISTS pg_trgm';
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'pg_trgm extension not available (%), skipping trigram indexes', SQLERRM;
            RETURN;
        END;

        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_work_items_title_trgm ON work_items USING GIN (title gin_trgm_ops)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_work_items_description_trgm ON work_items USING GIN (description gin_trgm_ops)';

        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_teams_name_trgm ON teams USING GIN (name gin_trgm_ops)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_teams_description_trgm ON teams USING GIN (description gin_trgm_ops)';

        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_queues_name_trgm ON queues USING GIN (name gin_trgm_ops)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_queues_description_trgm ON queues USING GIN (description gin_trgm_ops)';

        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_milestones_name_trgm ON milestones USING GIN (name gin_trgm_ops)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_milestones_description_trgm ON milestones USING GIN (description gin_trgm_ops)';
    END IF;
END
$$;
